package gameprocessor

import (
	"fmt"
	"time"

	"github.com/RowMur/office-games/internal/db"
)

const (
	matchesWithDoublePoints = 20
	eloStartingPoints       = 400
	eloLowerBound           = 200
)

type GameProcessor struct {
	db    db.Database
	cache *cache
}

func NewGameProcessor(db db.Database) *GameProcessor {
	return &GameProcessor{
		db:    db,
		cache: newCache(),
	}
}

type processedMatch struct {
	Participants map[uint]*ProcessedMatchParticipant
}

type ProcessedMatchParticipant struct {
	UserID        uint
	Win           bool
	PointsApplied int
}

func (gp *GameProcessor) Process(gameId uint) (*Game, error) {
	startTime := time.Now()
	defer func() {
		fmt.Printf("GetElos: %s\n", time.Now().Sub(startTime))
	}()

	if gp.cache != nil {
		entry := gp.cache.getEntry(gameId)
		if entry != nil {
			return entry, nil
		}
	}

	return gp.process(gameId)
}

func (gp *GameProcessor) process(gameId uint) (*Game, error) {
	matches := []db.Match{}
	err := gp.db.C.Where("game_id = ?", gameId).
		Where("state NOT IN (?)", db.MatchStatePending).
		Order("created_at").
		Preload("Participants.User").
		Find(&matches).Error

	if err != nil {
		return nil, err
	}

	g := newGame()

	players := map[uint]Player{}
	for _, match := range matches {
		cachedMatch := processedMatch{
			Participants: map[uint]*ProcessedMatchParticipant{},
		}

		winners := []Player{}
		losers := []Player{}
		for _, participant := range match.Participants {
			if _, ok := players[participant.UserID]; !ok {
				players[participant.UserID] = Player{
					User:   participant.User,
					Points: eloStartingPoints,
				}
			}

			if participant.Result == "win" {
				winners = append(winners, players[participant.UserID])
			} else {
				losers = append(losers, players[participant.UserID])
			}
		}

		pointsGainLoss := calculatePointsGainLoss(winners, losers, 1.0)
		if match.IsHandicap {
			pointsGainLoss = CalculateHandicapPointsGain()
		}

		for _, winner := range winners {
			for _, w := range winners {
				if w.User.ID > winner.User.ID {
					g.playerPairings.addMatch(match.ID, winner, w)
				}
			}
			for _, l := range losers {
				g.playerOpposingPairings.addMatch(match.ID, winner, l)
			}

			winner.WinCount++
			pointsToApply := pointsGainLoss

			if winner.MatchesPlayed() < matchesWithDoublePoints {
				pointsToApply = pointsToApply * 2
			}

			cachedMatch.Participants[winner.User.ID] = &ProcessedMatchParticipant{
				UserID:        winner.User.ID,
				Win:           true,
				PointsApplied: pointsToApply,
			}
			winner.Points += pointsToApply
			if winner.Points > winner.RecordPoints {
				winner.RecordPoints = winner.Points
				winner.RecordPointsDate = match.CreatedAt
			}
			players[winner.User.ID] = winner
		}
		for _, loser := range losers {
			for _, l := range losers {
				if l.User.ID > loser.User.ID {
					g.playerPairings.addMatch(match.ID, loser, l)
				}
			}

			loser.LossCount++
			pointsToApply := pointsGainLoss

			if loser.MatchesPlayed() < matchesWithDoublePoints {
				pointsToApply = pointsToApply * 2
			}

			if loser.Points-pointsToApply < eloLowerBound {
				pointsToApply = loser.Points - eloLowerBound
			}

			cachedMatch.Participants[loser.User.ID] = &ProcessedMatchParticipant{
				UserID:        loser.User.ID,
				Win:           false,
				PointsApplied: pointsToApply,
			}
			loser.Points -= pointsToApply
			if loser.Points > loser.RecordPoints {
				loser.RecordPoints = loser.Points
				loser.RecordPointsDate = match.CreatedAt
			}
			players[loser.User.ID] = loser
		}

		g.matches[match.ID] = &cachedMatch
	}

	g.players = players
	gp.cache.setEntry(gameId, &g)
	return &g, nil
}

func (gp GameProcessor) InvalidateGameCache(gameId uint) {
	(*gp.cache)[gameId] = nil
}
