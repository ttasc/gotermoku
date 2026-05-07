package main

import (
	"fmt"
	"os"
)

func main() {
	cfg := ParseConfig()

	state := NewGameState(cfg.Rows, cfg.Cols)
	state.IsOnline = cfg.IsOnline
	state.LocalPlayerColor = White

	var netMgr *NetworkManager
	if cfg.IsOnline {
		var err error
		if cfg.IsHost {
			fmt.Printf("Starting Host... Waiting for client to connect on port %s...\n", cfg.Port)
			netMgr, err = HostGame(cfg.Port)
		} else {
			address := fmt.Sprintf("%s:%s", cfg.JoinAddr, cfg.Port)
			fmt.Printf("Connecting to Host at %s...\n", address)
			netMgr, err = JoinGame(address)
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
