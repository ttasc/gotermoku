package main

import (
	"math/rand"
	"time"
)

// getBotMove evaluates the board and returns the best coordinates for the bot.
func getBotMove(state *GameState) [2]int {
	bestScore := -1
	var bestMoves [][2]int

	for y := 0; y < state.Rows; y++ {
		for x := 0; x < state.Cols; x++ {
			if state.Board[y][x] == Empty {
				// Đánh giá điểm tấn công (Đen) và phòng thủ (Trắng)
				attackScore := evaluateCell(state, x, y, Black)
				defenseScore := evaluateCell(state, x, y, White)

				// Tổng điểm của ô cờ. Ưu tiên nếu có cơ hội thắng ngay (Attack >= 100000)
				totalScore := attackScore + defenseScore
				if attackScore >= 100000 {
					totalScore += 50000
				}

				if totalScore > bestScore {
					bestScore = totalScore
					bestMoves = [][2]int{{x, y}}
				} else if totalScore == bestScore {
					bestMoves = append(bestMoves, [2]int{x, y})
				}
			}
		}
	}

	// Nếu bàn cờ trống trơn, đánh vào giữa
	if bestScore == -1 || len(bestMoves) == 0 {
		return [2]int{state.Cols / 2, state.Rows / 2}
	}

	// Chọn ngẫu nhiên 1 trong các ô có điểm cao nhất để lối đánh đa dạng
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return bestMoves[rng.Intn(len(bestMoves))]
}

// evaluateCell tính điểm hình học nếu đặt quân cờ vào tọa độ x, y
func evaluateCell(state *GameState, x, y int, color uint8) int {
	score := 0
	for _, dir := range winDirections {
		consecutive := 1
		openEnds := 0

		// Quét chiều dương
		nx, ny := x+dir[0], y+dir[1]
		for range 4 {
			if nx < 0 || nx >= state.Cols || ny < 0 || ny >= state.Rows {
				break
			}
			if state.Board[ny][nx] == color {
				consecutive++
			} else if state.Board[ny][nx] == Empty {
				openEnds++
				break
			} else {
				break
			}
			nx += dir[0]
			ny += dir[1]
		}

		// Quét chiều âm
		nx, ny = x-dir[0], y-dir[1]
		for range 4 {
			if nx < 0 || nx >= state.Cols || ny < 0 || ny >= state.Rows {
				break
			}
			if state.Board[ny][nx] == color {
				consecutive++
			} else if state.Board[ny][nx] == Empty {
				openEnds++
				break
			} else {
				break
			}
			nx -= dir[0]
			ny -= dir[1]
		}

		// Chấm điểm dựa trên số quân liên tiếp và số đầu mở
		if consecutive >= 5 {
			score += 100000 // Thắng chắc
		} else if consecutive == 4 {
			if openEnds == 2 { score += 10000 } else if openEnds == 1 { score += 1000 }
		} else if consecutive == 3 {
			if openEnds == 2 { score += 1000 } else if openEnds == 1 { score += 100 }
		} else if consecutive == 2 {
			if openEnds == 2 { score += 100 } else if openEnds == 1 { score += 10 }
		}
	}
	return score
}
