package main

import "time"

const (
	Empty = iota
	White
	Black
)

const CellWidth = 3

type GameState struct {
	Board            [][]uint8
	Rows             int
	Cols             int
	CurrentTurn      uint8
	MoveCount        map[uint8]int
	Winner           uint8
	WinningPositions [][2]int
	SelectedX        int
	SelectedY        int
	StartTime        time.Time
	LocalPlayerColor uint8
	IsOnline         bool
	IsBotMode        bool
}

func NewGameState(rows, cols int) *GameState {
	board := make([][]uint8, rows)
	for i := range board {
		board[i] = make([]uint8, cols)
	}

	return &GameState{
		Board:            board,
		Rows:             rows,
		Cols:             cols,
		CurrentTurn:      White,
		MoveCount:        map[uint8]int{White: 0, Black: 0},
		WinningPositions: make([][2]int, 0),
		SelectedX:        -1,
		SelectedY:        -1,
		StartTime:        time.Now(),
		LocalPlayerColor: White,
	}
}

func (s *GameState) Reset() {
	for y := 0; y < s.Rows; y++ {
		for x := 0; x < s.Cols; x++ {
			s.Board[y][x] = Empty
		}
	}
	s.CurrentTurn = White
	s.MoveCount = map[uint8]int{White: 0, Black: 0}
	s.Winner = Empty
	s.WinningPositions = make([][2]int, 0)
	s.SelectedX = -1
	s.SelectedY = -1
	s.StartTime = time.Now()
}

func (s *GameState) IsWinPos(x, y int) bool {
	if s.Winner == Empty {
		return false
	}
	for _, pos := range s.WinningPositions {
		if x == pos[0] && y == pos[1] {
			return true
		}
	}
	return false
}
