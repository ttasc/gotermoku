package main

var winDirections = [4][2]int{
	{1, 0},  // Horizontal
	{0, 1},  // Vertical
	{1, 1},  // Main diagonal
	{1, -1}, // Anti-diagonal
}

// scanRay scans from (x,y) in direction (dx,dy) looking for pieces of the target color.
// Returns the number of consecutive pieces, the number of open ends (0 or 1), and positions.
func scanRay(board [][]uint8, cols, rows, x, y, dx, dy int, color uint8) (count int, openEnd int, pos [][2]int) {
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
