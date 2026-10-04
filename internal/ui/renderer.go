package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/ttasc/gotermoku/internal/core"
	"github.com/ttasc/ttbox"
)

const CellWidth = 3

const (
	CharDot          = '·'
	CharWhite        = 'O'
	CharBlack        = 'X'
	CharLeftBracket  = '['
	CharRightBracket = ']'
)

const (
	ColorDefault    int = ttbox.ColorDefault
	ColorBoardGrid  int = 239
	ColorWhitePiece int = 255
	ColorBlackPiece int = 245
	ColorSelValid   int = 39
	ColorSelInvalid int = 196
	ColorWin        int = 114
	ColorText       int = 250
	ColorTextDim    int = 240
	ColorBgActive   int = 236
	ColorBgModal    int = 235
)

type RenderOpts struct {
	LocalColor    uint8
	OppName       string
	DisconnectMsg string
}

func Init(rows, cols int) {
	if err := ttbox.Init(); err != nil {
		fmt.Printf("initializing TUI: %v\n", err)
		os.Exit(1)
	}

	termW, termH := ttbox.Size()
	maxCols := termW / CellWidth
	maxRows := termH - 6

	if cols > maxCols || rows > maxRows {
		ttbox.Close()
		fmt.Printf("terminal size too small. Max capacity: %dx%d\n", maxRows, maxCols)
		os.Exit(1)
	}
	ttbox.EnableMouse()
}

func Close() {
	ttbox.DisableMouse()
	ttbox.Close()
}

func Render(state *core.GameState, opts RenderOpts) {
	ttbox.Clear()

	drawStatusline(state, opts)
	drawBoard(state)

	if state.Winner != core.Empty {
		drawEndgameBanner(state)
	} else if opts.DisconnectMsg != "" {
		drawDisconnectBanner(opts.DisconnectMsg)
	} else {
		drawControlsGuide()
	}

	ttbox.Present()
}

func drawBoard(state *core.GameState) {
	w, h := ttbox.Size()
	offsetX := (w - (state.Cols * CellWidth)) / 2
	offsetY := (h - state.Rows) / 2

	for y := 0; y < state.Rows; y++ {
		for x := 0; x < state.Cols; x++ {
			drawCell(state, x, y, offsetX, offsetY)
		}
	}
}

func drawCell(state *core.GameState, x, y, offsetX, offsetY int) {
	ch := CharDot
	fg, bg := ColorBoardGrid, ColorDefault
	isWinPos := state.IsWinPos(x, y)

	switch state.Board[y][x] {
	case core.White:
		ch, fg = CharWhite, ColorWhitePiece
	case core.Black:
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

func drawCursor(state *core.GameState, x, y, screenX, screenY int, isWinPos bool, bg int) {
	leftChar, rightChar := ' ', ' '
	bracketFg := ColorSelValid

	if x == state.SelectedX && y == state.SelectedY {
		leftChar, rightChar = CharLeftBracket, CharRightBracket
		if state.Board[y][x] != core.Empty {
			bracketFg = ColorSelInvalid
		}
	}

	if isWinPos && leftChar != ' ' {
		bracketFg = ColorBgModal
	}

	ttbox.SetCell(screenX-1, screenY, leftChar, bracketFg, bg)
	ttbox.SetCell(screenX+1, screenY, rightChar, bracketFg, bg)
}

func drawStatusline(state *core.GameState, opts RenderOpts) {
	w, h := ttbox.Size()
	if w == 0 || h == 0 {
		return
	}

	y := max((h-state.Rows)/2-2, 0)
	if y != 0 {
		ttbox.DrawTextCenter(1, " G O T E R M O K U ", ColorText, ColorDefault)
	}

	drawTimer(state, w, y, opts)
	drawTurnIndicator(state, opts, y+1)
}

func drawTimer(state *core.GameState, w, y int, opts RenderOpts) {
	elapsed := time.Since(state.StartTime)
	timerText := fmt.Sprintf("  %02d:%02d:%02d  ", int(elapsed.Hours()), int(elapsed.Minutes())%60, int(elapsed.Seconds())%60)
	centerX := w / 2

	whiteText, blackText := getPlayerLabels(opts)
	whiteFg, whiteBg, blackFg, blackBg := getPlayerColors(state)

	p1X := centerX - (len(timerText) / 2) - len(whiteText)
	for i, ch := range whiteText {
		ttbox.SetCell(p1X+i, y, ch, whiteFg, whiteBg)
	}

	ttbox.DrawTextCenter(y, timerText, ColorText, ColorDefault)

	p2X := centerX + (len(timerText) / 2) + (len(timerText) % 2)
	for i, ch := range blackText {
		ttbox.SetCell(p2X+i, y, ch, blackFg, blackBg)
	}
}

func getPlayerLabels(opts RenderOpts) (string, string) {
	wLabel, bLabel := " WHITE ", " BLACK "
	if opts.OppName != "" {
		if opts.LocalColor == core.White {
			wLabel, bLabel = " WHITE (You) ", fmt.Sprintf(" BLACK (%s) ", opts.OppName)
		} else if opts.LocalColor == core.Black {
			wLabel, bLabel = fmt.Sprintf(" WHITE (%s) ", opts.OppName), " BLACK (You) "
		}
	}
	return fmt.Sprintf(" %c -%s", CharWhite, wLabel), fmt.Sprintf(" %c -%s", CharBlack, bLabel)
}

func getPlayerColors(state *core.GameState) (int, int, int, int) {
	wFg, wBg, bFg, bBg := ColorTextDim, ColorDefault, ColorTextDim, ColorDefault
	if state.CurrentTurn == core.White {
		wFg, wBg = ColorWhitePiece, ColorBgActive
	} else {
		bFg, bBg = ColorWhitePiece, ColorBgActive
	}
	return wFg, wBg, bFg, bBg
}

func drawTurnIndicator(state *core.GameState, opts RenderOpts, y int) {
	if opts.OppName == "" || state.Winner != core.Empty {
		return
	}

	colorStr := "WHITE"
	if state.CurrentTurn == core.Black {
		colorStr = "BLACK"
	}

	if state.CurrentTurn == opts.LocalColor {
		ttbox.DrawTextCenter(y, fmt.Sprintf(" YOUR TURN (%s) ", colorStr), ColorSelValid, ColorDefault)
	} else {
		ttbox.DrawTextCenter(y, fmt.Sprintf(" %s'S TURN (%s) ", opts.OppName, colorStr), ColorTextDim, ColorDefault)
	}
}

func drawControlsGuide() {
	_, h := ttbox.Size()
	ttbox.DrawTextCenter(h-1, " Move(h, j, k, l; arrows)   Place(space, enter; left-click twice)   Quit(Ctrl+C, Esc) ", ColorText, ColorDefault)
}

func drawEndgameBanner(state *core.GameState) {
	_, h := ttbox.Size()
	msg := " * WHITE WINS! * "
	if state.Winner == core.Black {
		msg = " * BLACK WINS! * "
	}
	ttbox.SetAttr(true, false, false, false)
	ttbox.DrawTextCenter(h-2, msg, ColorWin, ColorDefault)
	ttbox.ResetAttr()
	ttbox.DrawTextCenter(h-1, " [R] Play Again   [ESC] Exit ", ColorTextDim, ColorDefault)
}

func drawDisconnectBanner(msg string) {
	_, h := ttbox.Size()
	ttbox.SetAttr(true, false, false, false)
	ttbox.DrawTextCenter(h-2, msg, ColorSelInvalid, ColorDefault)
	ttbox.ResetAttr()
	ttbox.DrawTextCenter(h-1, " [ESC] Exit ", ColorTextDim, ColorDefault)
}
