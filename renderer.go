package main

import (
	"fmt"
	"time"

	"github.com/ttasc/ttbox"
)

// Modern character design for the board pieces and grid.
const (
	CharDot          = '·' // Grid intersection point.
	CharWhite        = 'O' // White piece
	CharBlack        = 'X' // Black piece
	CharLeftBracket  = '['
	CharRightBracket = ']'
)

// Modern and minimalist xterm-256 color palette (Pastel & Minimalist).
const (
	ColorBoardGrid  = 239 // Dark gray for the grid.
	ColorWhitePiece = 255 // Pure white.
	ColorBlackPiece = 245 // Light gray (suitable for both dark and light terminal backgrounds).

	ColorSelValid   = 39  // Bright blue (indicates a valid selection).
	ColorSelInvalid = 196 // Red (when selecting none-Empty position)
	ColorWin        = 114 // Pastel green (indicates winning pieces).

	ColorText     = 250 // Light gray text.
	ColorTextDim  = 240 // Dim gray text (for the inactive player).
	ColorBgActive = 236 // Dark gray background (highlights the active player).
	ColorBgModal  = 235 // Background color for the popup modal.
)

// Render draws the current game state, and presents it to the screen.
func Render(state *GameState) {
	ttbox.Clear()

	drawStatusline(state)
	drawBoard(state)

	if state.Winner != Empty {
		drawEndgamePopup(state)
	}

	drawControlsGuide()

	ttbox.Present()
}

// drawBoard renders the game grid, placed pieces, and the active selection cursor.
func drawBoard(state *GameState) {
	boardWidth := state.Cols * CellWidth
	boardHeight := state.Rows

	w, h := ttbox.Size()
	offsetX := (w - boardWidth) / 2
	offsetY := (h - boardHeight) / 2

	for y := range state.Rows {
		for x := range state.Cols {
			ch := CharDot
			fg := ColorBoardGrid
			bg := ttbox.ColorDefault

			// Determine the piece representation.
			switch state.Board[y][x] {
			case White:
				ch = CharWhite
				fg = ColorWhitePiece
			case Black:
				ch = CharBlack
				fg = ColorBlackPiece
			}

			// Check if this cell is part of the winning sequence.
			isWinPos := false
			if state.Winner != Empty {
				for _, pos := range state.WinningPositions {
					if x == pos[0] && y == pos[1] {
						isWinPos = true
						break
					}
				}
			}

			if isWinPos {
				fg = ColorWin
			}

			screenX := offsetX + (x * 3)
			screenY := offsetY + y

			// Handle cursor selection effects.
			leftChar, rightChar := ' ', ' '
			bracketFg := ColorSelValid

			if x == state.SelectedX && y == state.SelectedY {
				leftChar = CharLeftBracket
				rightChar = CharRightBracket

				// Selection effect turn red if not Empty
				if state.Board[y][x] != Empty {
					bracketFg = ColorSelInvalid
				}
			}

			// Draw the left bracket or a space.
			ttbox.SetCell(screenX-1, screenY, leftChar, bracketFg, bg)

			// Draw the piece or grid intersection point in bold.
			ttbox.SetAttr(true, false, false, false)
			ttbox.SetCell(screenX, screenY, ch, fg, bg)
			ttbox.ResetAttr() // Reset text formatting.

			// Draw the right bracket or a space.
			ttbox.SetCell(screenX+1, screenY, rightChar, bracketFg, bg)
		}
	}
}

// drawStatusline renders the top information bar, including the players, active turn indicator, and elapsed time.
func drawStatusline(state *GameState) {
	w, h := ttbox.Size()
	if w == 0 || h == 0 {
		return
	}

	// Position the status line cleanly two rows above the board.
	offsetY := (h - state.Rows) / 2
	y := max(offsetY-2, 0) // Ensure it does not overflow if the terminal is too small.

	if y != 0 {
		ttbox.DrawTextCenter(1, " G O T E R M O K U ", ColorText, ttbox.ColorDefault)
	}

	// Calculate elapsed time.
	elapsed := time.Since(state.StartTime)
	hours := int(elapsed.Hours())
	mins := int(elapsed.Minutes()) % 60
	secs := int(elapsed.Seconds()) % 60
	timerText := fmt.Sprintf("  %02d:%02d:%02d  ", hours, mins, secs)

	// Center the entire status line horizontally.
	centerX := w / 2

	whiteLabel := " WHITE "
	blackLabel := " BLACK "

	// Append tags to identify which player is playing on the local terminal.
	if state.IsOnline {
		if state.LocalPlayerColor == White {
			whiteLabel = " WHITE (You) "
			blackLabel = " BLACK (Opp) "
		} else {
			whiteLabel = " WHITE (Opp) "
			blackLabel = " BLACK (You) "
		}
	} else if state.IsBotMode {
		whiteLabel = " WHITE (You) "
		blackLabel = " BLACK (Bot) "
	}

	whiteText := fmt.Sprintf(" %c -%s", CharWhite, whiteLabel)
	blackText := fmt.Sprintf(" %c -%s", CharBlack, blackLabel)

	whiteFg, whiteBg := ColorTextDim, ttbox.ColorDefault
	blackFg, blackBg := ColorTextDim, ttbox.ColorDefault

	// Highlight the active player's turn.
	switch state.CurrentTurn {
	case White:
		whiteFg, whiteBg = ColorWhitePiece, ColorBgActive
	case Black:
		blackFg, blackBg = ColorWhitePiece, ColorBgActive
	}

	// 1. Draw Player 1 (White) to the left of the timer.
	p1X := centerX - (len(timerText) / 2) - len(whiteText)
	for i, ch := range whiteText {
		ttbox.SetCell(p1X+i, y, ch, whiteFg, whiteBg)
	}

	// 2. Draw the Timer (Center).
	ttbox.DrawTextCenter(y, timerText, ColorText, ttbox.ColorDefault)

	// 3. Draw Player 2 (Black) to the right of the timer.
	p2X := centerX + (len(timerText) / 2) + (len(timerText) % 2)
	for i, ch := range blackText {
		ttbox.SetCell(p2X+i, y, ch, blackFg, blackBg)
	}

	// 4. Draw Turn Indicator directly under the status line (Online only)
	if (state.IsOnline || state.IsBotMode) && state.Winner == Empty {
		var turnIndicator string
		colorStr := "WHITE"
		if state.CurrentTurn == Black {
			colorStr = "BLACK"
		}

		if state.CurrentTurn == state.LocalPlayerColor {
			turnIndicator = fmt.Sprintf(" YOUR TURN (%s) ", colorStr)
			ttbox.DrawTextCenter(y+1, turnIndicator, ColorSelValid, ttbox.ColorDefault)
		} else {
			oppName := "OPPONENT'S"
			if state.IsBotMode {
				oppName = "BOT'S"
			}
			turnIndicator = fmt.Sprintf(" %s TURN (%s) ", oppName, colorStr)
			ttbox.DrawTextCenter(y+1, turnIndicator, ColorTextDim, ttbox.ColorDefault)
		}
	}
}

// drawControlsGuide renders the bottom instructions bar to help players with keybindings.
func drawControlsGuide() {
	w, h := ttbox.Size()
	if w == 0 || h == 0 {
		return
	}

	guideText := " Move(h, j, k, l; arrows)   Place(space, enter; left-click twice)   Quit(Ctrl+C, Esc) "
	ttbox.DrawTextCenter(h-1, guideText, ColorText, ttbox.ColorDefault)
}

// drawEndgamePopup displays a modal dialogue when the game concludes, showing the winner and restart instructions.
func drawEndgamePopup(state *GameState) {
	w, h := ttbox.Size()

	msgFg := ColorWin
	msg := " WHITE WINS! "
	if state.Winner == Black {
		msg = " BLACK WINS! "
	}
	subMsg := "[R] Play Again   [ESC] Exit "

	// Modal dimensions.
	boxW := len(subMsg) + 6
	boxH := 4

	x := (w - boxW) / 2

	// --- SMART POSITIONING ---
	// Calculate the average Y position of the winning pieces to avoid covering them with the popup.
	avgWinY := 0
	if len(state.WinningPositions) > 0 {
		for _, pos := range state.WinningPositions {
			avgWinY += pos[1] // Accumulate the Y coordinates of the winning pieces.
		}
		avgWinY /= len(state.WinningPositions)
	}

	boardHeight := state.Rows
	offsetY := (h - boardHeight) / 2
	var y int

	// If the winning pieces are in the top half of the board, push the popup to the bottom.
	if avgWinY < state.Rows/2 {
		y = offsetY + boardHeight
		// If the terminal is too short, force the popup to sit at the bottom edge.
		if y+boxH > h {
			y = h - boxH
		}
	} else {
		// If the winning pieces are in the bottom half of the board, push the popup to the top.
		// If the terminal is too short, force the popup to sit at the top edge.
		y = max(offsetY-boxH-1, 0)
	}

	// Clear the modal background area.
	ttbox.SetColor(ColorText, ColorBgModal)
	ttbox.Fill(x, y, boxW, boxH, ' ')

	// Draw the modal border box.
	ttbox.SetColor(ColorBoardGrid, ColorBgModal)
	ttbox.Box(x, y, boxW, boxH)

	// Draw the bold winning message.
	ttbox.SetAttr(true, false, false, false)
	ttbox.DrawTextCenter(y+1, msg, msgFg, ColorBgModal)
	ttbox.ResetAttr()

	// Draw the instructional sub-message.
	ttbox.DrawTextCenter(y+2, subMsg, ColorTextDim, ColorBgModal)

	// Reset to default system colors.
	ttbox.SetColor(ttbox.ColorDefault, ttbox.ColorDefault)
}
