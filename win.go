package main

// winDirections defines the 4 directional vectors (dx, dy) used to check for 5 consecutive pieces.
var winDirections =[4][2]int{
	{1, 0},  // Horizontal
	{0, 1},  // Vertical
	{1, 1},  // Main diagonal (\)
	{1, -1}, // Anti-diagonal (/)
}

// checkWin evaluates if the most recently placed piece results in a victory.
// It optimizes performance by only scanning the immediate surrounding axes of the placed piece
// rather than iterating over the entire board.
func checkWin(state *GameState, lastX, lastY int) {
	color := state.Board[lastY][lastX]
	if color == Empty {
		return
	}

	for _, dir := range winDirections {
		count := 1
		// Maximum possible contiguous line length in a single check is 9 (1 origin + 4 forward + 4 backward).
		positions := make([][2]int, 0, 9)
		positions = append(positions, [2]int{lastX, lastY})

		// 1. Scan in the positive direction (+dx, +dy).
		for i := 1; i < 5; i++ {
			nx := lastX + (dir[0] * i)
			ny := lastY + (dir[1] * i)

			// Stop scanning if the boundary is hit, or if a different colored/empty cell is encountered.
			if nx < 0 || nx >= BoardCols || ny < 0 || ny >= BoardRows || state.Board[ny][nx] != color {
				break
			}
			count++
			positions = append(positions, [2]int{nx, ny})
		}

		// 2. Scan in the negative direction (-dx, -dy).
		for i := 1; i < 5; i++ {
			nx := lastX - (dir[0] * i)
			ny := lastY - (dir[1] * i)

			if nx < 0 || nx >= BoardCols || ny < 0 || ny >= BoardRows || state.Board[ny][nx] != color {
				break
			}
			count++
			positions = append(positions, [2]int{nx, ny})
		}

		// 3. Victory condition met if 5 or more consecutive pieces are found.
		if count >= 5 {
			state.Winner = color
			state.WinningPositions = positions
			return // Halt further scanning since a win has already been confirmed.
		}
	}
}
