package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ttasc/ttbox"
)

func initUI(state *GameState) error {
	if err := ttbox.Init(); err != nil {
		return fmt.Errorf("initializing TUI: %w", err)
	}

	termW, termH := ttbox.Size()
	maxCols := termW / CellWidth
	maxRows := termH - 6

	if state.Cols > maxCols || state.Rows > maxRows {
		ttbox.Close()
		return fmt.Errorf("terminal size too small. Max capacity: %dx%d", maxRows, maxCols)
	}
	ttbox.EnableMouse()
	return nil
}

func RunGame(state *GameState, netMgr *NetworkManager) {
	if err := initUI(state); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer ttbox.Close()
	defer ttbox.DisableMouse()

	if state.IsOnline && netMgr != nil && netMgr.IsHost {
		broadcastSync(state, netMgr)
	}

	var disconnectMsg string
	defer func() {
		if disconnectMsg != "" {
			fmt.Printf("\n\nNOTICE: %s\n\n", disconnectMsg)
		}
	}()

	isRunning := true
	botMoves := make(chan [2]int, 1)
	botThinking := false

	for isRunning {
		tickBot(state, botMoves, &botThinking)

		select {
		case move := <-botMoves:
			botThinking = false
			placePiece(state, move[0], move[1], Black)
		default:
		}

		if netMgr != nil {
			if !processNetworkEvents(state, netMgr, &disconnectMsg) {
				isRunning = false
				continue
			}
		}

		evt, err := ttbox.PollEventTimeout(500 * time.Millisecond)
		if err == nil {
			isRunning = processInputEvent(evt, state, netMgr)
		}

		Render(state)
	}
}

func tickBot(state *GameState, botMoves chan<- [2]int, botThinking *bool) {
	if state.IsBotMode && state.CurrentTurn == Black && state.Winner == Empty && !*botThinking {
		*botThinking = true
		go func(s *GameState) {
			time.Sleep(400 * time.Millisecond)
			botMoves <- getBotMove(s)
		}(state)
	}
}

func processNetworkEvents(state *GameState, netMgr *NetworkManager, disconnectMsg *string) bool {
	select {
	case msg := <-netMgr.Incoming:
		if msg.Type == "disconnect" {
			*disconnectMsg = "The opponent has disconnected."
			return false
		}
		handleNetworkMessage(msg, state, netMgr)
	default:
	}
	return true
}

func processInputEvent(evt ttbox.Event, state *GameState, netMgr *NetworkManager) bool {
	if evt.Type == ttbox.EventKey {
		if evt.Key == ttbox.KeyEscape || evt.Key == ttbox.KeyCtrlC || evt.Ch == 'q' || evt.Ch == 'Q' {
			return false
		}
		if state.Winner != Empty && (evt.Ch == 'r' || evt.Ch == 'R') {
			requestRestart(state, netMgr)
		} else if handleKeyboard(evt, state) {
			tryLocalMove(state, netMgr, state.SelectedX, state.SelectedY)
		}
	} else if evt.Type == ttbox.EventMouse {
		if x, y, action := handleMouse(evt, state); action {
			tryLocalMove(state, netMgr, x, y)
		}
	}
	return true
}

func tryLocalMove(state *GameState, netMgr *NetworkManager, x, y int) {
	if (netMgr != nil || state.IsBotMode) && state.CurrentTurn != state.LocalPlayerColor {
		return
	}
	if !canPlacePiece(state, x, y) {
		return
	}

	if netMgr != nil && !netMgr.IsHost {
		netMgr.Send(NetMessage{Type: "move", X: x, Y: y})
	} else {
		placePiece(state, x, y, state.CurrentTurn)
		if netMgr != nil && netMgr.IsHost {
			broadcastSync(state, netMgr)
		}
	}
}

func requestRestart(state *GameState, netMgr *NetworkManager) {
	if netMgr != nil {
		if netMgr.IsHost {
			state.Reset()
			broadcastSync(state, netMgr)
		} else {
			netMgr.Send(NetMessage{Type: "restart"})
		}
	} else {
		state.Reset()
	}
}

func broadcastSync(state *GameState, netMgr *NetworkManager) {
	netMgr.Send(NetMessage{
		Type:             "sync",
		Board:            state.Board,
		CurrentTurn:      state.CurrentTurn,
		Winner:           state.Winner,
		WinningPositions: state.WinningPositions,
	})
}

func handleNetworkMessage(msg NetMessage, state *GameState, netMgr *NetworkManager) {
	switch msg.Type {
	case "move":
		if netMgr.IsHost && state.CurrentTurn == Black && canPlacePiece(state, msg.X, msg.Y) {
			placePiece(state, msg.X, msg.Y, Black)
			broadcastSync(state, netMgr)
		}
	case "sync":
		if !netMgr.IsHost {
			state.Board = msg.Board
			state.Rows = len(msg.Board)
			if state.Rows > 0 {
				state.Cols = len(msg.Board[0])
			}
			state.CurrentTurn = msg.CurrentTurn
			state.Winner = msg.Winner
			state.WinningPositions = msg.WinningPositions
		}
	case "restart":
		if netMgr.IsHost {
			state.Reset()
			broadcastSync(state, netMgr)
		}
	}
}
