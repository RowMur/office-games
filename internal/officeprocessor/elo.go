package officeprocessor

import (
	"math"
)

const (
	handicapMultiplier = 0.25
	kFactor            = 32
)

func calculatePointsGainLoss(winners, losers []Player, multiplier float64) int {
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
	pointsGainLoss := calculatePointsGainLossFromExpected(expectedScore, 1, multiplier)

	return pointsGainLoss
}

func CalculateHandicapPointsGain() int {
	basePointsGain := calculatePointsGainLossFromExpected(0.5, 1, handicapMultiplier)
	return basePointsGain
}

func calculateExpectedScore(elo1, elo2 float64) float64 {
	return 1 / (1 + math.Pow(10, ((elo2-elo1)/400)))
}

func calculatePointsGainLossFromExpected(expectedScore float64, actualScore float64, multiplier float64) int {
	return int(math.Round(multiplier * kFactor * (actualScore - expectedScore)))
}
