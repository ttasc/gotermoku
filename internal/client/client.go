package client

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ttasc/gotermoku/internal/common/core"
)

func Run(args []string) {
	cfg := ParseConfig(args)

	// 1. Chế độ Online qua Central Server
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

	// Đồng bộ thông số từ Session
	cfg.Rows = session.Rows
	cfg.Cols = session.Cols
	cfg.AssignedColor = session.AssignedColor
	cfg.OpponentName = session.OpponentName
	cfg.YourElo = session.YourElo
	cfg.OpponentElo = session.OpponentElo
	cfg.RoomCode = session.RoomCode

	colorStr := "White (First)"
	if cfg.AssignedColor == core.Black {
		colorStr = "Black (Second)"
	}

	fmt.Println("\n==================================================")
	fmt.Printf(" [MATCH READY] Match ID: %s\n", session.MatchID)
	fmt.Printf(" (Tip: Save this Match ID to rejoin if disconnected: --rejoin %s)\n", session.MatchID)
	fmt.Printf(" Opponent: %s (%d Elo) | You: %s (%d Elo)\n",
		cfg.OpponentName, cfg.OpponentElo, colorStr, cfg.YourElo)
	fmt.Printf(" Board Size: %dx%d\n", cfg.Rows, cfg.Cols)
	fmt.Println("==================================================")
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
