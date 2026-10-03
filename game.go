package main

// canPlacePiece ensures the target cell is valid and empty.
func canPlacePiece(state *GameState, x, y int) bool {
	if state.Winner != Empty {
		return false
	}
	if x < 0 || x >= state.Cols || y < 0 || y >= state.Rows {
		return false
	}
	return state.Board[y][x] == Empty
}

// placePiece commits a piece to the board, updates the win state, and toggles the turn.
func placePiece(state *GameState, x, y int, color uint8) {
	state.Board[y][x] = color
	state.MoveCount[color]++

	updateWinState(state, x, y)

	if state.Winner == Empty {
		if state.CurrentTurn == White {
			state.CurrentTurn = Black
		} else {
			state.CurrentTurn = White
		}
	}
}

// updateWinState evaluates if the last move resulted in a win.
func updateWinState(state *GameState, lastX, lastY int) {
	color := state.Board[lastY][lastX]
	if color == Empty {
		return
	}

	for _, dir := range winDirections {
		c1, _, p1 := scanRay(state.Board, state.Cols, state.Rows, lastX, lastY, dir[0], dir[1], color)
		c2, _, p2 := scanRay(state.Board, state.Cols, state.Rows, lastX, lastY, -dir[0], -dir[1], color)

		if c1+c2+1 >= 5 {
			state.Winner = color
			state.WinningPositions = make([][2]int, 0, c1+c2+1)
			state.WinningPositions = append(state.WinningPositions, [2]int{lastX, lastY})
			state.WinningPositions = append(state.WinningPositions, p1...)
			state.WinningPositions = append(state.WinningPositions, p2...)
			return
		}
	}
}
