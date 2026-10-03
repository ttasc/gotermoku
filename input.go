package main

import "github.com/ttasc/ttbox"

// handleKeyboard updates the cursor state and returns true if an action (placement) is triggered.
func handleKeyboard(evt ttbox.Event, state *GameState) bool {
	if state.Winner != Empty {
		return false
	}
	if state.SelectedX < 0 || state.SelectedY < 0 {
		state.SelectedX, state.SelectedY = state.Cols/2, state.Rows/2
	}

	moved := false
	action := false

	switch evt.Key {
	case ttbox.KeyArrowUp: state.SelectedY--; moved = true
	case ttbox.KeyArrowDown: state.SelectedY++; moved = true
	case ttbox.KeyArrowLeft: state.SelectedX--; moved = true
	case ttbox.KeyArrowRight: state.SelectedX++; moved = true
	case ttbox.KeyEnter: action = true
	default:
		switch evt.Ch {
		case 'k', 'K': state.SelectedY--; moved = true
		case 'j', 'J': state.SelectedY++; moved = true
		case 'h', 'H': state.SelectedX--; moved = true
		case 'l', 'L': state.SelectedX++; moved = true
		case ' ': action = true
		}
	}

	if moved {
		state.SelectedX = max(0, min(state.SelectedX, state.Cols-1))
		state.SelectedY = max(0, min(state.SelectedY, state.Rows-1))
	}
	return action
}

// handleMouse translates screen clicks to board coordinates.
// Returns (x, y, true) if a valid piece placement is requested.
func handleMouse(evt ttbox.Event, state *GameState) (int, int, bool) {
	if !evt.Press || evt.Button != ttbox.MouseLeft {
		return 0, 0, false
	}

	w, h := ttbox.Size()
	offsetX := (w - (state.Cols * CellWidth)) / 2
	offsetY := (h - state.Rows) / 2

	relX, relY := evt.X-offsetX, evt.Y-offsetY
	if relX < 0 || relX >= state.Cols*CellWidth || relY < 0 || relY >= state.Rows {
		state.SelectedX, state.SelectedY = -1, -1
		return 0, 0, false
	}

	boardX, boardY := relX/CellWidth, relY

	if state.SelectedX == boardX && state.SelectedY == boardY {
		state.SelectedX, state.SelectedY = -1, -1
		return boardX, boardY, true
	}

	state.SelectedX, state.SelectedY = boardX, boardY
	return 0, 0, false
}
