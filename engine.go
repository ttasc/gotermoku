package main

import (
	"time"
	"github.com/ttasc/ttbox"
)

// RunLocalLoop handles pure offline, hotseat gameplay.
func RunLocalLoop(state *GameState) {
	opts := RenderOpts{LocalColor: Empty, OppName: ""} // Both players play on this terminal

	for {
		evt, err := ttbox.PollEventTimeout(50 * time.Millisecond)
		if err == nil {
			intent := HandleInput(evt, state)
			if intent.Action == ActionQuit { break }
			if intent.Action == ActionRestart && state.Winner != Empty {
				state.Reset()
			}
			if intent.Action == ActionPlace && canPlacePiece(state, intent.X, intent.Y) {
				placePiece(state, intent.X, intent.Y, state.CurrentTurn)
			}
		}
		Render(state, opts)
	}
}

// RunBotLoop orchestrates the local player vs AI.
func RunBotLoop(state *GameState) {
	opts := RenderOpts{LocalColor: White, OppName: "Bot"}
	botMoves := make(chan [2]int, 1)
	botThinking := false

	for {
		// 1. Bot Routine
		if state.CurrentTurn == Black && state.Winner == Empty && !botThinking {
			botThinking = true
			go func(s *GameState) {
				time.Sleep(400 * time.Millisecond)
				botMoves <- getBotMove(s)
			}(state)
		}

		select {
		case move := <-botMoves:
			botThinking = false
			if canPlacePiece(state, move[0], move[1]) {
				placePiece(state, move[0], move[1], Black)
			}
		default:
		}

		// 2. Input Routine
		evt, err := ttbox.PollEventTimeout(50 * time.Millisecond)
		if err == nil {
			intent := HandleInput(evt, state)
			if intent.Action == ActionQuit { break }
			if intent.Action == ActionRestart && state.Winner != Empty {
				state.Reset()
			}
			// Only allow local placement if it's White's turn
			if intent.Action == ActionPlace && state.CurrentTurn == White && canPlacePiece(state, intent.X, intent.Y) {
				placePiece(state, intent.X, intent.Y, White)
			}
		}
		Render(state, opts)
	}
}

// RunHostLoop enforces the Server as the single source of truth.
func RunHostLoop(state *GameState, t GameTransport) {
	opts := RenderOpts{LocalColor: White, OppName: "Opponent"}
	broadcastSync(state, t) // Initial sync to client

	for {
		// 1. Network Routine
		select {
		case msg := <-t.Receive():
			if msg.Type == "disconnect" {
				opts.DisconnectMsg = "Opponent disconnected."
			} else if msg.Type == "move" && state.CurrentTurn == Black && canPlacePiece(state, msg.X, msg.Y) {
				placePiece(state, msg.X, msg.Y, Black)
				broadcastSync(state, t)
			} else if msg.Type == "restart" {
				state.Reset()
				broadcastSync(state, t)
			}
		default:
		}

		// 2. Input Routine
		evt, err := ttbox.PollEventTimeout(50 * time.Millisecond)
		if err == nil {
			intent := HandleInput(evt, state)
			if intent.Action == ActionQuit { break }
			if intent.Action == ActionRestart && state.Winner != Empty {
				state.Reset()
				broadcastSync(state, t)
			}
			if intent.Action == ActionPlace && state.CurrentTurn == White && canPlacePiece(state, intent.X, intent.Y) {
				placePiece(state, intent.X, intent.Y, White)
				broadcastSync(state, t)
			}
		}
		Render(state, opts)
	}
}

// RunClientLoop acts purely as a dumb terminal. Mutates state only via syncs.
func RunClientLoop(state *GameState, t GameTransport) {
	opts := RenderOpts{LocalColor: Black, OppName: "Opponent"}

	for {
		// 1. Network Routine
		select {
		case msg := <-t.Receive():
			if msg.Type == "disconnect" {
				opts.DisconnectMsg = "Host disconnected."
			} else if msg.Type == "sync" {
				state.Board = msg.Board
				state.Rows = len(msg.Board)
				if state.Rows > 0 { state.Cols = len(msg.Board[0]) }
				state.CurrentTurn = msg.CurrentTurn
				state.Winner = msg.Winner
				state.WinningPositions = msg.WinningPositions
			}
		default:
		}

		// 2. Input Routine
		evt, err := ttbox.PollEventTimeout(50 * time.Millisecond)
		if err == nil {
			intent := HandleInput(evt, state)
			if intent.Action == ActionQuit { break }
			if intent.Action == ActionRestart && state.Winner != Empty {
				t.Send(NetMessage{Type: "restart"})
			}
			if intent.Action == ActionPlace && state.CurrentTurn == Black && canPlacePiece(state, intent.X, intent.Y) {
				// Client does NOT apply the move. It politely requests the Host to do it.
				t.Send(NetMessage{Type: "move", X: intent.X, Y: intent.Y})
			}
		}
		Render(state, opts)
	}
}

func broadcastSync(state *GameState, t GameTransport) {
	t.Send(NetMessage{
		Type:             "sync",
		Board:            state.Board,
		CurrentTurn:      state.CurrentTurn,
		Winner:           state.Winner,
		WinningPositions: state.WinningPositions,
	})
}
