package client

import (
	"time"

	"github.com/ttasc/gotermoku/internal/common/core"
	"github.com/ttasc/ttbox"
)

func RunEngine(cfg *Config, state *core.GameState, t Transport) {
	opts := buildRenderOpts(cfg)

	if cfg.IsHost {
		BroadcastSync(state, t)
	}

	botMoves := make(chan [2]int, 1)
	botThinking := false

	for {
		if cfg.IsBotMode && state.CurrentTurn == core.Black && state.Winner == core.Empty && !botThinking {
			botThinking = true
			go func(s *core.GameState) {
				time.Sleep(400 * time.Millisecond)
				botMoves <- GetBotMove(s)
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
			intent := HandleInput(evt, state)
			if intent.Action == ActionQuit {
				break
			}
			processIntent(cfg, intent, state, t, opts)
		}

		Render(state, opts)
	}
}

func buildRenderOpts(cfg *Config) RenderOpts {
	opts := RenderOpts{
		LocalColor:   core.Empty,
		OppName:      "",
		RoomCode:     cfg.RoomCode,
		YourElo:      cfg.YourElo,
		OppElo:       cfg.OpponentElo,
		IsPaused:     false,
		GraceSeconds: 0,
	}

	if cfg.IsBotMode {
		opts.LocalColor = core.White
		opts.OppName = "Bot"
	} else if cfg.IsOnline {
		if cfg.IsCentralMode {
			opts.LocalColor = cfg.AssignedColor
			opts.OppName = cfg.OpponentName
		} else {
			opts.OppName = "Opponent"
			if cfg.IsHost {
				opts.LocalColor = core.White
			} else {
				opts.LocalColor = core.Black
			}
		}
	}
	return opts
}

func processNetwork(cfg *Config, state *core.GameState, t Transport, opts *RenderOpts) {
	select {
	case msg := <-t.Receive():
		if msg.Type == "disconnect" {
			opts.IsPaused = false
			if msg.Message != "" {
				opts.DisconnectMsg = msg.Message
			} else if cfg.IsCentralMode {
				opts.DisconnectMsg = "Opponent disconnected from server."
			} else if cfg.IsHost {
				opts.DisconnectMsg = "Opponent disconnected."
			} else {
				opts.DisconnectMsg = "Host disconnected."
			}
		} else if msg.Type == "pause" {
			opts.IsPaused = true
			opts.GraceSeconds = msg.GraceSecondsRemaining
			opts.PauseMsg = msg.Message
		} else if msg.Type == "resume" {
			opts.IsPaused = false
			opts.PauseMsg = ""
			// Khôi phục ma trận bàn cờ cho client vừa kết nối lại
			if len(msg.Board) > 0 && !cfg.IsHost {
				state.Board = msg.Board
				state.Rows = len(msg.Board)
				state.Cols = len(msg.Board[0])
				state.CurrentTurn = msg.CurrentTurn
				state.Winner = msg.Winner
				state.WinningPositions = msg.WinningPositions
			}
		} else if msg.Type == "sync" && !cfg.IsHost {
			opts.IsPaused = msg.IsPaused
			opts.GraceSeconds = msg.GraceSecondsRemaining
			if !msg.IsPaused {
				opts.PauseMsg = ""
			}
			if msg.WhiteElo > 0 && msg.BlackElo > 0 {
				if cfg.AssignedColor == core.White {
					opts.YourElo = msg.WhiteElo
					opts.OppElo = msg.BlackElo
				} else {
					opts.YourElo = msg.BlackElo
					opts.OppElo = msg.WhiteElo
				}
			}

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
				BroadcastSync(state, t)
			}
		} else if msg.Type == "restart" && cfg.IsHost {
			state.Reset()
			BroadcastSync(state, t)
		}
	default:
	}
}

func processIntent(cfg *Config, intent InputIntent, state *core.GameState, t Transport, opts RenderOpts) {
	// Khóa không cho phép đánh cờ khi trận đấu đang tạm dừng (Grace Period)
	if opts.IsPaused && intent.Action == ActionPlace {
		return
	}

	if intent.Action == ActionRestart && state.Winner != core.Empty {
		if cfg.IsOnline && !cfg.IsHost {
			t.Send(NetMessage{Type: "restart"})
		} else {
			state.Reset()
			if cfg.IsHost {
				BroadcastSync(state, t)
			}
		}
		return
	}

	if intent.Action == ActionPlace {
		if cfg.IsOnline && !cfg.IsHost {
			allowedTurn := core.Black
			if cfg.IsCentralMode {
				allowedTurn = cfg.AssignedColor
			}

			if state.CurrentTurn == allowedTurn && core.CanPlacePiece(state, intent.X, intent.Y) {
				t.Send(NetMessage{Type: "move", X: intent.X, Y: intent.Y})
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
				BroadcastSync(state, t)
			}
		}
	}
}
