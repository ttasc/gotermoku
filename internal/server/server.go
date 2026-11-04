package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"math/big"
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
	port := fs.String("port", "3333", "TCP port for central server")
	dbPath := fs.String("db", "gotermoku.db", "Path to bbolt database file")
	rows := fs.Int("rows", 20, "Default board rows for online matches")
	cols := fs.Int("cols", 30, "Default board cols for online matches")

	// Cờ cấu hình bảo mật TLS
	enableTLS := fs.Bool("tls", false, "Enable TLS encryption")
	certFile := fs.String("cert", "", "Path to TLS certificate file (optional)")
	keyFile := fs.String("key", "", "Path to TLS private key file (optional)")

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

	// 4. Cấu hình gRPC Keepalive & HTTP/2 Ping
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute,
		Time:              10 * time.Second, // Gửi HTTP/2 Ping sau 10s không có dữ liệu
		Timeout:           3 * time.Second,  // Chờ tối đa 3s nhận Ping ACK
	}
	kaPolicy := keepalive.EnforcementPolicy{
		MinTime:             1 * time.Second,
		PermitWithoutStream: true,
	}

	serverOpts := []grpc.ServerOption{
		grpc.KeepaliveParams(kaParams),
		grpc.KeepaliveEnforcementPolicy(kaPolicy),
		grpc.UnaryInterceptor(UnaryAuthInterceptor(authMgr)),
		grpc.StreamInterceptor(StreamAuthInterceptor(authMgr)),
	}

	// 5. Cấu hình TLS (Hỗ trợ nạp từ file hoặc tự động sinh trong RAM)
	if *enableTLS {
		if *certFile != "" && *keyFile != "" {
			// Nạp chứng chỉ chính thức từ file đĩa (Production / Let's Encrypt)
			creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading TLS certificates: %v\n", err)
				_ = userStore.Close()
				os.Exit(1)
			}
			serverOpts = append(serverOpts, grpc.Creds(creds))
			fmt.Printf("[SECURITY] TLS enabled (Loaded from %s, %s).\n", *certFile, *keyFile)
		} else {
			// Tự động sinh chứng chỉ ECDSA tạm thời trong RAM (Zero-Config TLS)
			tlsCert, err := generateEphemeralCertificate()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating in-memory TLS certificate: %v\n", err)
				_ = userStore.Close()
				os.Exit(1)
			}
			serverOpts = append(serverOpts, grpc.Creds(credentials.NewServerTLSFromCert(&tlsCert)))
			fmt.Println("[SECURITY] TLS enabled (Zero-Config: Generated ephemeral ECDSA certificate in-memory).")
		}
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

// generateEphemeralCertificate tự sinh cặp khóa và chứng chỉ ECDSA P-256 trực tiếp trong RAM
func generateEphemeralCertificate() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate ecdsa key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate serial: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Gotermoku In-Memory Server"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}, nil
}
