package main

import (
	"fmt"
	"time"

	"github.com/ttasc/ttbox"
)

const (
	CharDot          = '·'
	CharWhite        = 'O'
	CharBlack        = 'X'
	CharLeftBracket  = '['
	CharRightBracket = ']'
)

const (
	ColorBoardGrid  = 239
	ColorWhitePiece = 255
	ColorBlackPiece = 245
	ColorSelValid   = 39
	ColorSelInvalid = 196
	ColorWin        = 114
	ColorText       = 250
	ColorTextDim    = 240
	ColorBgActive   = 236
	ColorBgModal    = 235
)

func Render(state *GameState) {
	ttbox.Clear()

	drawStatusline(state)
	drawBoard(state)

	if state.Winner != Empty {
		drawEndgameBanner(state)
	} else {
		drawControlsGuide()
	}

	ttbox.Present()
}

func drawBoard(state *GameState) {
	w, h := ttbox.Size()
	offsetX := (w - (state.Cols * CellWidth)) / 2
	offsetY := (h - state.Rows) / 2

	for y := 0; y < state.Rows; y++ {
		for x := 0; x < state.Cols; x++ {
			drawCell(state, x, y, offsetX, offsetY)
		}
	}
}

func drawCell(state *GameState, x, y, offsetX, offsetY int) {
	ch, fg, bg := CharDot, ColorBoardGrid, ttbox.ColorDefault
	isWinPos := state.IsWinPos(x, y)

	switch state.Board[y][x] {
	case White:
		ch, fg = CharWhite, ColorWhitePiece
	case Black:
		ch, fg = CharBlack, ColorBlackPiece
	}

	if isWinPos {
		fg, bg = ColorBgModal, ColorWin
	}

	screenX, screenY := offsetX+(x*CellWidth), offsetY+y
	drawCursor(state, x, y, screenX, screenY, isWinPos, bg)

	ttbox.SetAttr(true, false, false, false)
	ttbox.SetCell(screenX, screenY, ch, fg, bg)
	ttbox.ResetAttr()
}

func drawCursor(state *GameState, x, y, screenX, screenY int, isWinPos bool, bg int) {
	leftChar, rightChar := ' ', ' '
	bracketFg := ColorSelValid

	if x == state.SelectedX && y == state.SelectedY {
		leftChar, rightChar = CharLeftBracket, CharRightBracket
		if state.Board[y][x] != Empty {
			bracketFg = ColorSelInvalid
		}
	}

	if isWinPos && leftChar != ' ' {
		bracketFg = ColorBgModal
	}

	ttbox.SetCell(screenX-1, screenY, leftChar, bracketFg, bg)
	ttbox.SetCell(screenX+1, screenY, rightChar, bracketFg, bg)
}

func drawStatusline(state *GameState) {
	w, h := ttbox.Size()
	if w == 0 || h == 0 {
		return
	}

	y := max((h-state.Rows)/2-2, 0)
	if y != 0 {
		ttbox.DrawTextCenter(1, " G O T E R M O K U ", ColorText, ttbox.ColorDefault)
	}

	timerText := formatTimer(time.Since(state.StartTime))
	centerX := w / 2

	whiteText, blackText := getPlayerLabels(state)
	whiteFg, whiteBg, blackFg, blackBg := getPlayerColors(state)

	p1X := centerX - (len(timerText) / 2) - len(whiteText)
	for i, ch := range whiteText {
		ttbox.SetCell(p1X+i, y, ch, whiteFg, whiteBg)
	}

	ttbox.DrawTextCenter(y, timerText, ColorText, ttbox.ColorDefault)

	p2X := centerX + (len(timerText) / 2) + (len(timerText) % 2)
	for i, ch := range blackText {
		ttbox.SetCell(p2X+i, y, ch, blackFg, blackBg)
	}

	drawTurnIndicator(state, y+1)
}

func formatTimer(elapsed time.Duration) string {
	hours, mins, secs := int(elapsed.Hours()), int(elapsed.Minutes())%60, int(elapsed.Seconds())%60
	return fmt.Sprintf("  %02d:%02d:%02d  ", hours, mins, secs)
}

func getPlayerLabels(state *GameState) (string, string) {
	wLabel, bLabel := " WHITE ", " BLACK "
	if state.IsOnline {
		if state.LocalPlayerColor == White {
			wLabel, bLabel = " WHITE (You) ", " BLACK (Opp) "
		} else {
			wLabel, bLabel = " WHITE (Opp) ", " BLACK (You) "
		}
	} else if state.IsBotMode {
		wLabel, bLabel = " WHITE (You) ", " BLACK (Bot) "
	}
	return fmt.Sprintf(" %c -%s", CharWhite, wLabel), fmt.Sprintf(" %c -%s", CharBlack, bLabel)
}

func getPlayerColors(state *GameState) (int, int, int, int) {
	wFg, wBg, bFg, bBg := ColorTextDim, ttbox.ColorDefault, ColorTextDim, ttbox.ColorDefault
	if state.CurrentTurn == White {
		wFg, wBg = ColorWhitePiece, ColorBgActive
	} else {
		bFg, bBg = ColorWhitePiece, ColorBgActive
	}
	return wFg, wBg, bFg, bBg
}

func drawTurnIndicator(state *GameState, y int) {
	if (!state.IsOnline && !state.IsBotMode) || state.Winner != Empty {
		return
	}

	colorStr := "WHITE"
	if state.CurrentTurn == Black {
		colorStr = "BLACK"
	}

	if state.CurrentTurn == state.LocalPlayerColor {
		ttbox.DrawTextCenter(y, fmt.Sprintf(" YOUR TURN (%s) ", colorStr), ColorSelValid, ttbox.ColorDefault)
	} else {
		oppName := "OPPONENT'S"
		if state.IsBotMode {
			oppName = "BOT'S"
		}
		ttbox.DrawTextCenter(y, fmt.Sprintf(" %s TURN (%s) ", oppName, colorStr), ColorTextDim, ttbox.ColorDefault)
	}
}

func drawControlsGuide() {
	_, h := ttbox.Size()
	ttbox.DrawTextCenter(h-1, " Move(h, j, k, l; arrows)   Place(space, enter; left-click twice)   Quit(Ctrl+C, Esc) ", ColorText, ttbox.ColorDefault)
}

func drawEndgameBanner(state *GameState) {
	_, h := ttbox.Size()
	msg := " * WHITE WINS! * "
	if state.Winner == Black {
		msg = " * BLACK WINS! * "
	}

	ttbox.SetAttr(true, false, false, false)
	ttbox.DrawTextCenter(h-2, msg, ColorWin, ttbox.ColorDefault)
	ttbox.ResetAttr()
	ttbox.DrawTextCenter(h-1, " [R] Play Again   [ESC] Exit ", ColorTextDim, ttbox.ColorDefault)
}
