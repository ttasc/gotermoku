package main

import (
	"github.com/ttasc/ttbox"
)

// tryPlacePiece handles the core logic for placing a piece on the board.
// It is a shared function utilized by both keyboard and mouse input handlers.
func tryPlacePiece(state *GameState, netMgr *NetworkManager, playerColor uint8, boardX, boardY int) bool {
	if state.Winner != Empty {
		return false
	}

	// Check if it is the player's turn (applicable for online multiplayer).
	if netMgr != nil && state.CurrentTurn != playerColor {
		return false
	}

	// Reject the move if the target cell is already occupied.
	if state.Board[boardY][boardX] != Empty {
		return false
	}

	if netMgr != nil && !netMgr.IsHost {
		// Client sends the move coordinates to the Host.
		netMgr.Send(NetMessage{Type: "move", X: boardX, Y: boardY})
	} else {
		// Host or Offline mode updates the local game state immediately.
		state.Board[boardY][boardX] = state.CurrentTurn
		state.MoveCount[state.CurrentTurn]++

		checkWin(state, boardX, boardY)

		if state.Winner == Empty {
			if state.CurrentTurn == White {
				state.CurrentTurn = Black
			} else {
				state.CurrentTurn = White
			}
		}

		if netMgr != nil && netMgr.IsHost {
			broadcastSync(state, netMgr)
		}
	}
	return true
}

// handleKeyboard processes keyboard events, managing cursor movement and piece placement.
func handleKeyboard(evt ttbox.Event, state *GameState, netMgr *NetworkManager, playerColor uint8) {
	if state.Winner != Empty {
		return
	}

	// If the cursor is hidden (-1) because the mouse hasn't been used or the game just started,
	// center the cursor on the board upon the first directional key press.
	if state.SelectedX < 0 || state.SelectedY < 0 {
		state.SelectedX = state.Cols / 2
		state.SelectedY = state.Rows / 2
	}

	moved := false

	// Handle standard arrow keys and Enter.
	switch evt.Key {
	case ttbox.KeyArrowUp:
		state.SelectedY--
		moved = true
	case ttbox.KeyArrowDown:
		state.SelectedY++
		moved = true
	case ttbox.KeyArrowLeft:
		state.SelectedX--
		moved = true
	case ttbox.KeyArrowRight:
		state.SelectedX++
		moved = true
	case ttbox.KeyEnter:
		tryPlacePiece(state, netMgr, playerColor, state.SelectedX, state.SelectedY)
	default:
		// Handle Vim-style movement keys (HJKL) and Spacebar.
		switch evt.Ch {
		case 'k', 'K':
			state.SelectedY--
			moved = true
		case 'j', 'J':
			state.SelectedY++
			moved = true
		case 'h', 'H':
			state.SelectedX--
			moved = true
		case 'l', 'L':
			state.SelectedX++
			moved = true
		case ' ':
			tryPlacePiece(state, netMgr, playerColor, state.SelectedX, state.SelectedY)
		}
	}

	// Prevent the selection cursor from moving out of the board boundaries.
	if moved {
		if state.SelectedX < 0 {
			state.SelectedX = 0
		} else if state.SelectedX >= state.Cols {
			state.SelectedX = state.Cols - 1
		}

		if state.SelectedY < 0 {
			state.SelectedY = 0
		} else if state.SelectedY >= state.Rows {
			state.SelectedY = state.Rows - 1
		}
	}
}

// handleMouse processes mouse events, translating screen coordinates to board coordinates.
// It implements a two-click mechanic: click once to focus a cell, click again to place a piece.
func handleMouse(evt ttbox.Event, state *GameState, netMgr *NetworkManager, playerColor uint8) {
	if !evt.Press || evt.Button != ttbox.MouseLeft {
		return
	}

	boardWidth := state.Cols * CellWidth
	boardHeight := state.Rows
	w, h := ttbox.Size()
	offsetX := (w - boardWidth) / 2
	offsetY := (h - boardHeight) / 2

	relativeX := evt.X - offsetX
	relativeY := evt.Y - offsetY

	// Hide the cursor if the click is outside the board area.
	if relativeX < 0 || relativeX >= boardWidth || relativeY < 0 || relativeY >= boardHeight {
		state.SelectedX = -1
		state.SelectedY = -1
		return
	}

	boardX := relativeX / CellWidth
	boardY := relativeY

	// Core mechanic: First click focuses the cell, second click on the same cell places the piece.
	if state.SelectedX == boardX && state.SelectedY == boardY {
		success := tryPlacePiece(state, netMgr, playerColor, boardX, boardY)
		if success {
			// Hide the selection cursor after successfully placing a piece via mouse.
			state.SelectedX = -1
			state.SelectedY = -1
		}
	} else {
		// First click: Move the selection cursor to the target cell.
		state.SelectedX = boardX
		state.SelectedY = boardY
	}
}
