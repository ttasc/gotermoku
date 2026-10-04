package core

var WinDirections = [4][2]int{
	{1, 0},  // Horizontal
	{0, 1},  // Vertical
	{1, 1},  // Main diagonal
	{1, -1}, // Anti-diagonal
}

func ScanRay(board [][]uint8, cols, rows, x, y, dx, dy int, color uint8) (count int, openEnd int, pos [][2]int) {
	for i := 1; i <= 4; i++ {
		nx, ny := x+(dx*i), y+(dy*i)
		if nx < 0 || nx >= cols || ny < 0 || ny >= rows {
			break
		}
		if board[ny][nx] == color {
			count++
			pos = append(pos, [2]int{nx, ny})
		} else if board[ny][nx] == Empty {
			openEnd = 1
			break
		} else {
			break
		}
	}
	return
}

func CanPlacePiece(state *GameState, x, y int) bool {
	if state.Winner != Empty {
		return false
	}
	if x < 0 || x >= state.Cols || y < 0 || y >= state.Rows {
		return false
	}
	return state.Board[y][x] == Empty
}

func PlacePiece(state *GameState, x, y int, color uint8) {
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

func updateWinState(state *GameState, lastX, lastY int) {
	color := state.Board[lastY][lastX]
	if color == Empty {
		return
	}

	for _, dir := range WinDirections {
		c1, _, p1 := ScanRay(state.Board, state.Cols, state.Rows, lastX, lastY, dir[0], dir[1], color)
		c2, _, p2 := ScanRay(state.Board, state.Cols, state.Rows, lastX, lastY, -dir[0], -dir[1], color)

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
