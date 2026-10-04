package main

import "github.com/ttasc/ttbox"

const (
	ActionNone = iota
	ActionQuit
	ActionPlace
	ActionRestart
)

type InputIntent struct {
	Action int
	X, Y   int
}

// HandleInput translates raw physical events into logical intents.
func HandleInput(evt ttbox.Event, state *GameState) InputIntent {
	if evt.Type == ttbox.EventKey {
		return handleKeyboard(evt, state)
	} else if evt.Type == ttbox.EventMouse {
		return handleMouse(evt, state)
	}
	return InputIntent{Action: ActionNone}
}

func handleKeyboard(evt ttbox.Event, state *GameState) InputIntent {
	if evt.Key == ttbox.KeyEscape || evt.Key == ttbox.KeyCtrlC || evt.Ch == 'q' || evt.Ch == 'Q' {
		return InputIntent{Action: ActionQuit}
	}
	if state.Winner != Empty && (evt.Ch == 'r' || evt.Ch == 'R') {
		return InputIntent{Action: ActionRestart}
	}

	if state.SelectedX < 0 || state.SelectedY < 0 {
		state.SelectedX, state.SelectedY = state.Cols/2, state.Rows/2
	}

	moved, place := false, false

	switch evt.Key {
	case ttbox.KeyArrowUp: state.SelectedY--; moved = true
	case ttbox.KeyArrowDown: state.SelectedY++; moved = true
	case ttbox.KeyArrowLeft: state.SelectedX--; moved = true
	case ttbox.KeyArrowRight: state.SelectedX++; moved = true
	case ttbox.KeyEnter: place = true
	default:
		switch evt.Ch {
		case 'k', 'K': state.SelectedY--; moved = true
		case 'j', 'J': state.SelectedY++; moved = true
		case 'h', 'H': state.SelectedX--; moved = true
		case 'l', 'L': state.SelectedX++; moved = true
		case ' ': place = true
		}
	}

	if moved {
		state.SelectedX = max(0, min(state.SelectedX, state.Cols-1))
		state.SelectedY = max(0, min(state.SelectedY, state.Rows-1))
	}

	if place {
		return InputIntent{Action: ActionPlace, X: state.SelectedX, Y: state.SelectedY}
	}
	return InputIntent{Action: ActionNone}
}

func handleMouse(evt ttbox.Event, state *GameState) InputIntent {
	if !evt.Press || evt.Button != ttbox.MouseLeft {
		return InputIntent{Action: ActionNone}
	}

	w, h := ttbox.Size()
	offsetX := (w - (state.Cols * CellWidth)) / 2
	offsetY := (h - state.Rows) / 2

	relX, relY := evt.X-offsetX, evt.Y-offsetY
	if relX < 0 || relX >= state.Cols*CellWidth || relY < 0 || relY >= state.Rows {
		state.SelectedX, state.SelectedY = -1, -1
		return InputIntent{Action: ActionNone}
	}

	boardX, boardY := relX/CellWidth, relY
	if state.SelectedX == boardX && state.SelectedY == boardY {
		state.SelectedX, state.SelectedY = -1, -1
		return InputIntent{Action: ActionPlace, X: boardX, Y: boardY}
	}

	state.SelectedX, state.SelectedY = boardX, boardY
	return InputIntent{Action: ActionNone}
}
