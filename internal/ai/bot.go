package ai

import (
	"math/rand"
	"time"
	"github.com/ttasc/gotermoku/internal/core"
)

func GetMove(state *core.GameState) [2]int {
	bestScore := -1
	var bestMoves [][2]int

	for y := 0; y < state.Rows; y++ {
		for x := 0; x < state.Cols; x++ {
			if state.Board[y][x] == core.Empty {
				attackScore := evaluateCell(state, x, y, core.Black)
				defenseScore := evaluateCell(state, x, y, core.White)

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

	if bestScore == -1 || len(bestMoves) == 0 {
		return [2]int{state.Cols / 2, state.Rows / 2}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return bestMoves[rng.Intn(len(bestMoves))]
}

func evaluateCell(state *core.GameState, x, y int, color uint8) int {
	score := 0
	for _, dir := range core.WinDirections {
		c1, o1, _ := core.ScanRay(state.Board, state.Cols, state.Rows, x, y, dir[0], dir[1], color)
		c2, o2, _ := core.ScanRay(state.Board, state.Cols, state.Rows, x, y, -dir[0], -dir[1], color)
		consecutive := c1 + c2 + 1
		openEnds := o1 + o2
		score += calculateScore(consecutive, openEnds)
	}
	return score
}

func calculateScore(consecutive, openEnds int) int {
	if consecutive >= 5 { return 100000 }
	if consecutive == 4 {
		if openEnds == 2 { return 10000 }
		if openEnds == 1 { return 1000 }
	}
	if consecutive == 3 {
		if openEnds == 2 { return 1000 }
		if openEnds == 1 { return 100 }
	}
	if consecutive == 2 {
		if openEnds == 2 { return 100 }
		if openEnds == 1 { return 10 }
	}
	return 0
}
