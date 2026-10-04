package engine

import (
	"time"

	"github.com/ttasc/gotermoku/internal/ai"
	"github.com/ttasc/gotermoku/internal/config"
	"github.com/ttasc/gotermoku/internal/core"
	"github.com/ttasc/gotermoku/internal/netio"
	"github.com/ttasc/gotermoku/internal/ui"
	"github.com/ttasc/ttbox"
)

// Run unifies Hotseat, Bot, Host, and Client loops into a single orchestration layer.
func Run(cfg *config.Config, state *core.GameState, t netio.Transport) {
	opts := buildRenderOpts(cfg)

	if cfg.IsHost {
		netio.BroadcastSync(state, t)
	}

	botMoves := make(chan [2]int, 1)
	botThinking := false

	for {
		if cfg.IsBotMode && state.CurrentTurn == core.Black && state.Winner == core.Empty && !botThinking {
			botThinking = true
			go func(s *core.GameState) {
				time.Sleep(400 * time.Millisecond)
				botMoves <- ai.GetMove(s)
			}(state)
		}

		select {
		case move := <-botMoves:
			botThinking = false
			if core.CanPlacePiece(state, move[0], move[1]) {
				core.PlacePiece(state, move[0], move[1], core.Black)
			}
		default:
		}

		if t != nil {
			processNetwork(cfg, state, t, &opts)
		}

		evt, err := ttbox.PollEventTimeout(50 * time.Millisecond)
		if err == nil {
			intent := ui.HandleInput(evt, state)
			if intent.Action == ui.ActionQuit {
				break
			}
			processIntent(cfg, intent, state, t)
		}

		ui.Render(state, opts)
	}
}

func buildRenderOpts(cfg *config.Config) ui.RenderOpts {
	opts := ui.RenderOpts{LocalColor: core.Empty, OppName: ""}
	if cfg.IsBotMode {
		opts.LocalColor = core.White
		opts.OppName = "Bot"
	} else if cfg.IsOnline {
		opts.OppName = "Opponent"
		if cfg.IsHost {
			opts.LocalColor = core.White
		} else {
			opts.LocalColor = core.Black
		}
	}
	return opts
}

func processNetwork(cfg *config.Config, state *core.GameState, t netio.Transport, opts *ui.RenderOpts) {
	select {
	case msg := <-t.Receive():
		if msg.Type == "disconnect" {
			if cfg.IsHost {
				opts.DisconnectMsg = "Opponent disconnected."
			} else {
				opts.DisconnectMsg = "Host disconnected."
			}
		} else if msg.Type == "sync" && !cfg.IsHost {
			state.Board = msg.Board
			state.Rows = len(msg.Board)
			if state.Rows > 0 {
				state.Cols = len(msg.Board[0])
			}
			state.CurrentTurn = msg.CurrentTurn
			state.Winner = msg.Winner
			state.WinningPositions = msg.WinningPositions
		} else if msg.Type == "move" && cfg.IsHost {
			if state.CurrentTurn == core.Black && core.CanPlacePiece(state, msg.X, msg.Y) {
				core.PlacePiece(state, msg.X, msg.Y, core.Black)
				netio.BroadcastSync(state, t)
			}
		} else if msg.Type == "restart" && cfg.IsHost {
			state.Reset()
			netio.BroadcastSync(state, t)
		}
	default:
	}
}

func processIntent(cfg *config.Config, intent ui.InputIntent, state *core.GameState, t netio.Transport) {
	if intent.Action == ui.ActionRestart && state.Winner != core.Empty {
		if cfg.IsOnline && !cfg.IsHost {
			t.Send(netio.NetMessage{Type: "restart"})
		} else {
			state.Reset()
			if cfg.IsHost {
				netio.BroadcastSync(state, t)
			}
		}
		return
	}

	if intent.Action == ui.ActionPlace {
		if cfg.IsOnline && !cfg.IsHost {
			if state.CurrentTurn == core.Black && core.CanPlacePiece(state, intent.X, intent.Y) {
				t.Send(netio.NetMessage{Type: "move", X: intent.X, Y: intent.Y})
			}
			return
		}

		turnAllowed := true
		colorToPlace := state.CurrentTurn

		if cfg.IsBotMode || cfg.IsHost {
			turnAllowed = (state.CurrentTurn == core.White)
			colorToPlace = core.White
		}

		if turnAllowed && core.CanPlacePiece(state, intent.X, intent.Y) {
			core.PlacePiece(state, intent.X, intent.Y, colorToPlace)
			if cfg.IsHost {
				netio.BroadcastSync(state, t)
			}
		}
	}
}
