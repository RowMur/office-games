package elo

import (
	"math"
)

func CalculatePointsGainLoss(winnerElo, loserElo int) (int, float64) {
	floatWinnerElo := float64(winnerElo)
	floatLoserElo := float64(loserElo)

	expectedScore := calculateExpectedScore(floatWinnerElo, floatLoserElo)
	pointsGainLoss := calculatePointsGainLoss(expectedScore, 1)

	return pointsGainLoss, expectedScore
}

func calculateExpectedScore(elo1, elo2 float64) float64 {
	return 1 / (1 + math.Pow(10, ((elo2-elo1)/400)))
}

func calculatePointsGainLoss(expectedScore float64, actualScore float64) int {
	return int(math.Round(32 * (actualScore - expectedScore)))
}
