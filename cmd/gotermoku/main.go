package main

import (
	"os"

	"github.com/ttasc/gotermoku/internal/config"
	"github.com/ttasc/gotermoku/internal/core"
	"github.com/ttasc/gotermoku/internal/engine"
	"github.com/ttasc/gotermoku/internal/netio"
	"github.com/ttasc/gotermoku/internal/ui"
)

func main() {
	cfg := config.Parse(os.Args[1:])
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
