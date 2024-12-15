package elo

import (
	"fmt"
	"sort"
	"time"

	"github.com/RowMur/office-games/internal/db"
)

type EloService struct {
	db    db.Database
	cache *cache
}

func NewEloService(db db.Database) *EloService {
	return &EloService{
		db:    db,
		cache: newCache(),
	}
}

type Elo struct {
	User      db.User
	Points    int
	WinCount  int
	LossCount int

	RecordPoints     int
	RecordPointsDate time.Time
}

func (e Elo) MatchesPlayed() int {
	return e.WinCount + e.LossCount
}

func (e Elo) Percentage() float64 {
	total := e.MatchesPlayed()
	percentage := 0.0
	if total > 0 {
		percentage = float64(e.WinCount) / float64(total) * 100
	}

	return percentage
}

type Elos []Elo

func (elos Elos) RecordElo() *Elo {
	recordElo := &Elo{}
	for _, elo := range elos {
		if elo.RecordPoints > recordElo.RecordPoints {
			recordElo = &elo
		}
	}

	return recordElo
}

func (es *EloService) GetElos(gameId uint) (Elos, error) {
	startTime := time.Now()
	defer func() {
		fmt.Printf("GetElos: %s\n", time.Now().Sub(startTime))
	}()

	if es.cache != nil {
		entry := es.cache.getEntry(gameId)
		if entry != nil {
			return entry.elos, nil
		}
	}

	matches := []db.Match{}
	err := es.db.C.Where("game_id = ?", gameId).
		Where("state NOT IN (?)", db.MatchStatePending).
		Order("created_at").
		Preload("Participants.User").
		Find(&matches).Error

	if err != nil {
		return nil, err
	}

	ce := newCacheEntry()

	elos := map[uint]Elo{}
	for _, match := range matches {
		cachedMatch := ProcessedMatch{
			Participants: map[uint]*ProcessedMatchParticipant{},
		}

		winners := []Elo{}
		losers := []Elo{}
		for _, participant := range match.Participants {
			if _, ok := elos[participant.UserID]; !ok {
				elos[participant.UserID] = Elo{
					User:   participant.User,
					Points: 400,
				}
			}

			if participant.Result == "win" {
				winners = append(winners, elos[participant.UserID])
			} else {
				losers = append(losers, elos[participant.UserID])
			}
		}

		pointsGainLoss := CalculatePointsGainLoss(winners, losers, 1.0)
		for _, winner := range winners {
			winner.WinCount++
			pointsToApply := pointsGainLoss

			if winner.MatchesPlayed() < 20 {
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
			elos[winner.User.ID] = winner
		}
		for _, loser := range losers {
			loser.LossCount++
			pointsToApply := pointsGainLoss

			if loser.MatchesPlayed() < 20 {
				pointsToApply = pointsToApply * 2
			}

			if loser.Points-pointsToApply < 200 {
				pointsToApply = loser.Points - 200
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
			elos[loser.User.ID] = loser
		}

		ce.matches[match.ID] = &cachedMatch
	}

	elosSlice := Elos{}
	for _, elo := range elos {
		elosSlice = append(elosSlice, elo)
	}

	sort.Slice(elosSlice, func(i, j int) bool {
		iElo := elosSlice[i]
		jElo := elosSlice[j]
		if iElo.Points != jElo.Points {
			return iElo.Points > jElo.Points
		}

		if iElo.WinCount != jElo.WinCount {
			return iElo.WinCount > jElo.WinCount
		}

		if iElo.LossCount != jElo.LossCount {
			return iElo.LossCount < jElo.LossCount
		}

		return iElo.User.Username > jElo.User.Username
	})

	ce.elos = elosSlice
	es.cache.setEntry(gameId, &ce)
	return elosSlice, nil
}

func (es EloService) InvalidateEloCache(gameId uint) {
	(*es.cache)[gameId] = nil
}

func (es EloService) GetMatch(gameId uint, matchId uint) *ProcessedMatch {
	matchesCache := (*es.cache)[gameId].matches
	if matchesCache == nil {
		return nil
	}

	return matchesCache[matchId]
}
