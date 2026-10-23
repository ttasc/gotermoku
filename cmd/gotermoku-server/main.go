package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/ttasc/gotermoku/internal/netio/pb"
	"github.com/ttasc/gotermoku/internal/server"
)

func main() {
	port := flag.String("port", "4444", "TCP port for central server")
	dbPath := flag.String("db", "users.json", "Path to flat-file JSON user storage")
	rows := flag.Int("rows", 20, "Default board rows for online matches")
	cols := flag.Int("cols", 30, "Default board cols for online matches")
	flag.Parse()

	fmt.Printf("[SERVER] Initializing Gotermoku Central Server...\n")

	// 1. Khởi tạo kho lưu trữ người dùng
	userStore, err := server.NewUserStore(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing user storage: %v\n", err)
		os.Exit(1)
	}

	// 2. Khởi tạo các module dịch vụ hạt nhân
	authMgr := server.NewAuthManager(userStore)
	roomMgr := server.NewRoomManager()
	matchmaker := server.NewMatchmaker(roomMgr, *rows, *cols)
	centralSrv := server.NewCentralServer(authMgr, matchmaker, roomMgr)

	// 3. Khởi tạo Socket lắng nghe TCP
	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error binding port %s: %v\n", *port, err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterGomokuServiceServer(grpcServer, centralSrv)

	// 4. Lắng nghe tín hiệu dừng POSIX chuẩn (SIGINT, SIGTERM)
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
