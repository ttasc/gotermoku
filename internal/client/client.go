package client

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ttasc/gotermoku/internal/common/core"
)

// Run là điểm điều phối chính cho Client
func Run(args []string) {
	cfg := ParseConfig(args)

	// 1. Chế độ Online qua Central Server: Xác thực & Ghép trận trên Console
	if cfg.IsCentralMode {
		runCentralMatch(cfg)
		return
	}

	// 2. Chế độ truyền thống (Hotseat, Bot, LAN P2P)
	state := core.NewGameState(cfg.Rows, cfg.Cols)

	var t Transport
	if cfg.IsOnline {
		t = InitLANTransport(cfg)
		defer t.Close()
	}

	InitUI(cfg.Rows, cfg.Cols)
	defer CloseUI()

	RunEngine(cfg, state, t)
}

func runCentralMatch(cfg *Config) {
	reader := bufio.NewReader(os.Stdin)

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

	session, err := ConnectCentral(cfg)
	if err != nil {
		fmt.Printf("Online error: %v\n", err)
		os.Exit(1)
	}
	defer session.Transport.Close()

	// Đồng bộ thông số từ Server
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

	state := core.NewGameState(cfg.Rows, cfg.Cols)

	InitUI(cfg.Rows, cfg.Cols)
	defer CloseUI()

	RunEngine(cfg, state, session.Transport)
}

func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}
