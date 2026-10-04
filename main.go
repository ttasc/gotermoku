package main

import (
	"fmt"
	"net"
	"os"

	"github.com/ttasc/ttbox"
)

func initTUI(rows, cols int) error {
	if err := ttbox.Init(); err != nil {
		return fmt.Errorf("initializing TUI: %w", err)
	}

	termW, termH := ttbox.Size()
	maxCols := termW / CellWidth
	maxRows := termH - 6

	if cols > maxCols || rows > maxRows {
		ttbox.Close()
		return fmt.Errorf("terminal size too small. Max capacity: %dx%d", maxRows, maxCols)
	}

	ttbox.EnableMouse()
	return nil
}

func main() {
	cfg, err := ParseConfig(os.Args[1:])
	if err != nil {
		if err.Error() == "help" {
			PrintUsage()
			os.Exit(0)
		}
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	state := NewGameState(cfg.Rows, cfg.Cols)

	// Routing setup
	var transport GameTransport
	if cfg.IsOnline {
		if cfg.IsHost {
			fmt.Printf("Starting Host... Waiting for client to connect on port %s...\n", cfg.Port)
			ln, err := net.Listen("tcp", ":"+cfg.Port)
			if err != nil { fmt.Printf("Network error: %v\n", err); os.Exit(1) }
			conn, err := ln.Accept()
			if err != nil { fmt.Printf("Accept error: %v\n", err); os.Exit(1) }
			ln.Close()
			transport = NewTCPTransport(conn)

		} else {
			addr := fmt.Sprintf("%s:%s", cfg.JoinAddr, cfg.Port)
			fmt.Printf("Connecting to Host at %s...\n", addr)
			conn, err := net.Dial("tcp", addr)
			if err != nil { fmt.Printf("Network error: %v\n", err); os.Exit(1) }
			transport = NewTCPTransport(conn)
		}
		defer transport.Close()
	}

	if err := initTUI(cfg.Rows, cfg.Cols); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer ttbox.Close()
	defer ttbox.DisableMouse()

	// Launch the specific game loop
	if cfg.IsOnline {
		if cfg.IsHost {
			RunHostLoop(state, transport)
		} else {
			RunClientLoop(state, transport)
		}
	} else if cfg.IsBotMode {
		RunBotLoop(state)
	} else {
		RunLocalLoop(state)
	}
}
