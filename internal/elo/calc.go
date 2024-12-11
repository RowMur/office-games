package elo

import (
	"math"
)

func CalculatePointsGainLoss(winners, losers []Elo, multiplier float64) int {
	summedWinnerElo := float64(0)
	for _, winner := range winners {
		summedWinnerElo += float64(winner.Points)
	}
	avgWinnerElo := summedWinnerElo / float64(len(winners))

	summedLoserElo := float64(0)
	for _, loser := range losers {
		summedLoserElo += float64(loser.Points)
	}
	avgLoserElo := summedLoserElo / float64(len(losers))

	expectedScore := calculateExpectedScore(avgWinnerElo, avgLoserElo)
	pointsGainLoss := calculatePointsGainLoss(expectedScore, 1, multiplier)

	return pointsGainLoss
}

func calculateExpectedScore(elo1, elo2 float64) float64 {
	return 1 / (1 + math.Pow(10, ((elo2-elo1)/400)))
}

func calculatePointsGainLoss(expectedScore float64, actualScore float64, multiplier float64) int {
	return int(math.Round(multiplier * 32 * (actualScore - expectedScore)))
}
