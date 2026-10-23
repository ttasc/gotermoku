package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ttasc/gotermoku/internal/config"
	"github.com/ttasc/gotermoku/internal/core"
	"github.com/ttasc/gotermoku/internal/engine"
	"github.com/ttasc/gotermoku/internal/netio"
	"github.com/ttasc/gotermoku/internal/ui"
)

func main() {
	cfg := config.Parse(os.Args[1:])

	// 1. Nếu chơi qua Central Server: thực hiện Auth & Ghép trận trước trên Console
	if cfg.IsCentralMode {
		runCentralMatch(cfg)
		return
	}

	// 2. Các chế độ truyền thống (Hotseat, Bot, LAN P2P): giữ nguyên luồng hiện tại
	state := core.NewGameState(cfg.Rows, cfg.Cols)

	var transport netio.Transport
	if cfg.IsOnline {
		transport = netio.InitTransport(cfg)
		defer transport.Close()
	}

	ui.Init(cfg.Rows, cfg.Cols)
	defer ui.Close()

	engine.Run(cfg, state, transport)
}

func runCentralMatch(cfg *config.Config) {
	reader := bufio.NewReader(os.Stdin)

	// Nhắc lệnh nhập thông tin nếu người dùng chưa truyền qua cờ CLI
	if cfg.Username == "" {
		fmt.Print("Enter username: ")
		cfg.Username = readLine(reader)
	}
	if cfg.Password == "" {
		fmt.Print("Enter password: ")
		cfg.Password = readLine(reader)
	}

	if cfg.Username == "" || cfg.Password == "" {
		fmt.Println("Error: username and password cannot be empty.")
		os.Exit(1)
	}

	// Thực hiện kết nối, xác thực và ghép trận qua Transport tầng netio
	session, err := netio.ConnectCentral(cfg)
	if err != nil {
		fmt.Printf("Online error: %v\n", err)
		os.Exit(1)
	}
	defer session.Transport.Close()

	// Đồng bộ thông số trận đấu do Server chỉ định
	cfg.Rows = session.Rows
	cfg.Cols = session.Cols
	cfg.AssignedColor = session.AssignedColor
	cfg.OpponentName = session.OpponentName

	colorStr := "White (First)"
	if cfg.AssignedColor == core.Black {
		colorStr = "Black (Second)"
	}

	fmt.Printf("\n[MATCH FOUND] Opponent: %s | You play: %s | Board: %dx%d\n",
		cfg.OpponentName, colorStr, cfg.Rows, cfg.Cols)
	fmt.Println("Entering game in 1 second...")
	time.Sleep(1 * time.Second)

	// Khởi tạo GameState và giao diện TUI bàn cờ khi trận đấu chính thức bắt đầu
	state := core.NewGameState(cfg.Rows, cfg.Cols)

	ui.Init(cfg.Rows, cfg.Cols)
	defer ui.Close()

	engine.Run(cfg, state, session.Transport)
}

func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}
