package elo

import (
	"math"
)

func CalculateNewElos(winnerElo, loserElo int) (int, int) {
	floatWinnerElo := float64(winnerElo)
	floatLoserElo := float64(loserElo)

	expectedWinnerScore := calculateExpectedScore(floatWinnerElo, floatLoserElo)
	expectedLoserScore := calculateExpectedScore(floatLoserElo, floatWinnerElo)

	newWinnerElo := calculateNewElo(floatWinnerElo, expectedWinnerScore, 1)
	newLoserElo := calculateNewElo(floatLoserElo, expectedLoserScore, 0)

	return newWinnerElo, newLoserElo
}

func calculateExpectedScore(elo1, elo2 float64) float64 {
	return 1 / (1 + math.Pow(10, ((elo2-elo1)/400)))
}

func calculateNewElo(prevElo float64, expectedScore float64, actualScore float64) int {
	return int(prevElo + 32*(actualScore-expectedScore))
}
