package elo

import (
	"sort"

	"github.com/RowMur/office-games/internal/db"
)

type EloService struct {
	db    db.Database
	cache *cache
}

func NewEloService(db db.Database) *EloService {
	return &EloService{
		db:    db,
		cache: NewCache(),
	}
}

type Elo struct {
	User      db.User
	Points    int
	Rank      int
	WinCount  int
	LossCount int
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

func (es *EloService) GetElos(gameId uint) (Elos, error) {
	if es.cache != nil {
		e := (*es.cache)[gameId].elos
		if e != nil {
			return e, nil
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

	newCacheEntry := cacheEntry{
		matches: map[uint]ProcessedMatch{},
	}

	elos := map[uint]Elo{}
	for _, match := range matches {
		cachedMatch := ProcessedMatch{
			Participants: map[uint]ProcessedMatchParticipant{},
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

			cachedMatch.Participants[winner.User.ID] = ProcessedMatchParticipant{
				UserID:        winner.User.ID,
				Win:           true,
				PointsApplied: pointsToApply,
			}
			winner.Points += pointsToApply
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

			cachedMatch.Participants[loser.User.ID] = ProcessedMatchParticipant{
				UserID:        loser.User.ID,
				Win:           false,
				PointsApplied: pointsToApply,
			}
			loser.Points -= pointsToApply
			elos[loser.User.ID] = loser
		}

		newCacheEntry.matches[match.ID] = cachedMatch
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

	newCacheEntry.elos = elosSlice
	(*es.cache)[gameId] = newCacheEntry
	return elosSlice, nil
}

func (es EloService) InvalidateEloCache(gameId uint) {
	(*es.cache)[gameId] = cacheEntry{}
}

func (es EloService) GetMatch(gameId uint, matchId uint) *ProcessedMatch {
	matchesCache := (*es.cache)[gameId].matches
	if matchesCache == nil {
		return nil
	}

	match := matchesCache[matchId]
	return &match
}
