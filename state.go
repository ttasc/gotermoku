package main

import "time"

// Constants representing the possible states of a board cell and the player identities.
const (
	Empty = iota
	White
	Black
)

// GameState holds the complete state of a single game session.
// It serves as the single source of truth for rendering and logic.
type GameState struct {
	Board            [BoardRows][BoardCols]uint8
	CurrentTurn      uint8         // The active player's turn (White or Black)
	MoveCount        map[uint8]int // Tracks the total number of moves made by each player
	Winner           uint8         // The winning player (Empty/0 if the game is ongoing)
	WinningPositions [][2]int      // The coordinates [x, y] of the 5 consecutive pieces that form the winning line
	SelectedX        int           // X coordinate of the currently hovered or first-clicked cell (-1 if none)
	SelectedY        int           // Y coordinate of the currently hovered or first-clicked cell (-1 if none)
	StartTime        time.Time     // The timestamp when the game session started, used for the timer
	LocalPlayerColor uint8         // The local player's color (White or Black)
	IsOnline         bool          // Indicates if the game is in online multiplayer mode
}

// NewGameState initializes and returns a fresh, clean game state.
func NewGameState() *GameState {
	return &GameState{
		// Go arrays are zero-valued by default, so all Board cells automatically start as Empty (0).
		CurrentTurn:      White,
		MoveCount:        map[uint8]int{White: 0, Black: 0},
		Winner:           Empty,
		WinningPositions: make([][2]int, 0),
		SelectedX:        -1, // -1 indicates that no valid cell is currently selected.
		SelectedY:        -1,
		StartTime:        time.Now(),
		LocalPlayerColor: White,
		IsOnline:         false,
	}
}

// Reset clears the current game state, preparing it for a new match while reusing the struct memory.
func (s *GameState) Reset() {
	s.Board = [BoardRows][BoardCols]uint8{} // Reassigning a zero-valued array resets all cells to Empty (0).
	s.CurrentTurn = White
	s.MoveCount = map[uint8]int{White: 0, Black: 0}
	s.Winner = Empty
	s.WinningPositions = make([][2]int, 0)
	s.SelectedX = -1
	s.SelectedY = -1
	s.StartTime = time.Now()
	// Note: LocalPlayerColor and IsOnline are intentionally preserved across restarts.
}
