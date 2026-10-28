package server

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/ttasc/gotermoku/internal/common/pb"
)

// Run khởi động và điều phối toàn bộ vòng đời của Central Server
func Run(args []string) {
	fs := flag.NewFlagSet("gotermoku-server", flag.ExitOnError)
	port := fs.String("port", "4444", "TCP port for central server")
	dbPath := fs.String("db", "users.json", "Path to flat-file JSON user storage")
	rows := fs.Int("rows", 20, "Default board rows for online matches")
	cols := fs.Int("cols", 30, "Default board cols for online matches")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing server flags: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[SERVER] Initializing Gotermoku Central Server...\n")

	// 1. Khởi tạo kho lưu trữ người dùng
	userStore, err := NewUserStore(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing user storage: %v\n", err)
		os.Exit(1)
	}

	// 2. Khởi tạo các module dịch vụ hạt nhân
	authMgr := NewAuthManager(userStore)
	roomMgr := NewRoomManager()
	matchmaker := NewMatchmaker(roomMgr, *rows, *cols)
	centralSrv := NewCentralServer(authMgr, matchmaker, roomMgr)

	// 3. Khởi tạo Socket lắng nghe TCP
	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error binding port %s: %v\n", *port, err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterGomokuServiceServer(grpcServer, centralSrv)

	// 4. Bắt tín hiệu dừng POSIX chuẩn (SIGINT, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n[SERVER] Received signal %v. Gracefully shutting down...\n", sig)
		grpcServer.GracefulStop()
		fmt.Println("[SERVER] Server stopped cleanly.")
		os.Exit(0)
	}()

	fmt.Printf("[SERVER] Listening on port %s (board: %dx%d, db: %s)\n", *port, *rows, *cols, *dbPath)
	if err := grpcServer.Serve(listener); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
