package core

import "math"

const (
	DefaultElo = 1200
	KFactor    = 32
	MinElo     = 100
)

// CalculateElo tính điểm Elo mới cho người chơi A và người chơi B.
// scoreA nhận các giá trị: 1.0 (A thắng), 0.5 (Hòa), 0.0 (B thắng).
func CalculateElo(ratingA, ratingB int, scoreA float64) (newA, newB int) {
	expectedA := 1.0 / (1.0 + math.Pow(10.0, float64(ratingB-ratingA)/400.0))
	expectedB := 1.0 - expectedA

	scoreB := 1.0 - scoreA

	deltaA := float64(KFactor) * (scoreA - expectedA)
	deltaB := float64(KFactor) * (scoreB - expectedB)

	newA = ratingA + int(math.Round(deltaA))
	newB = ratingB + int(math.Round(deltaB))

	if newA < MinElo {
		newA = MinElo
	}
	if newB < MinElo {
		newB = MinElo
	}

	return newA, newB
}
