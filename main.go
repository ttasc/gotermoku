package main

import (
	"fmt"
	"os"
)

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
	state.IsOnline = cfg.IsOnline
	state.IsBotMode = cfg.IsBotMode
	state.LocalPlayerColor = White

	var netMgr *NetworkManager
	if cfg.IsOnline {
		if cfg.IsHost {
			fmt.Printf("Starting Host... Waiting for client to connect on port %s...\n", cfg.Port)
			netMgr, err = HostGame(cfg.Port)
		} else {
			addr := fmt.Sprintf("%s:%s", cfg.JoinAddr, cfg.Port)
			fmt.Printf("Connecting to Host at %s...\n", addr)
			netMgr, err = JoinGame(addr)
			state.LocalPlayerColor = Black
		}

		if err != nil {
			fmt.Printf("Network error: %v\n", err)
			os.Exit(1)
		}
		defer netMgr.Close()
	}

	RunGame(state, netMgr)
}
