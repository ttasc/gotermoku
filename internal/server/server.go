package server

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/ttasc/gotermoku/internal/common/pb"
)

// Run khởi động và điều phối toàn bộ vòng đời của Central Server
func Run(args []string) {
	fs := flag.NewFlagSet("gotermoku-server", flag.ExitOnError)
	port := fs.String("port", "4444", "TCP port for central server")
	dbPath := fs.String("db", "gotermoku.db", "Path to bbolt database file")
	rows := fs.Int("rows", 20, "Default board rows for online matches")
	cols := fs.Int("cols", 30, "Default board cols for online matches")

	// Cờ cấu hình bảo mật TLS
	enableTLS := fs.Bool("tls", false, "Enable TLS encryption")
	certFile := fs.String("cert", "cert.pem", "Path to TLS certificate file")
	keyFile := fs.String("key", "key.pem", "Path to TLS private key file")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing server flags: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[SERVER] Initializing Gotermoku Central Server...\n")

	// 1. Khởi tạo cơ sở dữ liệu bbolt nhúng
	userStore, err := NewUserStore(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}

	// 2. Khởi tạo các module dịch vụ hạt nhân
	authMgr := NewAuthManager(userStore)
	roomMgr := NewRoomManager()
	matchmaker := NewMatchmaker(roomMgr, authMgr, *rows, *cols)
	centralSrv := NewCentralServer(authMgr, matchmaker, roomMgr)

	// 3. Khởi tạo Socket lắng nghe TCP
	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error binding port %s: %v\n", *port, err)
		_ = userStore.Close()
		os.Exit(1)
	}

	// 4. Cấu hình gRPC Keepalive & HTTP/2 Ping (Phát hiện rớt mạng tầng transport)
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute,
		Time:              10 * time.Second, // Gửi HTTP/2 Ping sau 10s không có dữ liệu
		Timeout:           3 * time.Second,  // Chờ tối đa 3s nhận Ping ACK
	}
	kaPolicy := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}

	serverOpts := []grpc.ServerOption{
		grpc.KeepaliveParams(kaParams),
		grpc.KeepaliveEnforcementPolicy(kaPolicy),
		grpc.UnaryInterceptor(UnaryAuthInterceptor(authMgr)),
		grpc.StreamInterceptor(StreamAuthInterceptor(authMgr)),
	}

	// 5. Cấu hình TLS (nếu bật)
	if *enableTLS {
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading TLS certificates: %v\n", err)
			_ = userStore.Close()
			os.Exit(1)
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
		fmt.Println("[SECURITY] TLS encryption enabled.")
	} else {
		fmt.Println("[SECURITY] Running in plaintext mode (insecure).")
	}

	grpcServer := grpc.NewServer(serverOpts...)
	pb.RegisterGomokuServiceServer(grpcServer, centralSrv)

	// 6. Bắt tín hiệu dừng POSIX chuẩn (SIGINT, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n[SERVER] Received signal %v. Gracefully shutting down...\n", sig)
		grpcServer.GracefulStop()
		_ = userStore.Close()
		fmt.Println("[SERVER] Database and server closed cleanly.")
		os.Exit(0)
	}()

	fmt.Printf("[SERVER] Listening on port %s (board: %dx%d, db: %s)\n", *port, *rows, *cols, *dbPath)
	if err := grpcServer.Serve(listener); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		_ = userStore.Close()
		os.Exit(1)
	}
}
