package elo

import (
	"sort"

	"github.com/RowMur/office-games/internal/db"
)

type EloService struct {
	db db.Database
}

func NewEloService(db db.Database) *EloService {
	return &EloService{
		db: db,
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
	matches := []db.Match{}

	err := es.db.C.Where("game_id = ?", gameId).
		Where("state NOT IN (?)", db.MatchStatePending).
		Order("created_at").
		Preload("Participants.User").
		Find(&matches).Error

	if err != nil {
		return nil, err
	}

	elos := map[uint]Elo{}
	for _, match := range matches {
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
				loser.Points = 200
			} else {
				loser.Points -= pointsToApply
			}

			elos[loser.User.ID] = loser
		}
	}

	elosSlice := Elos{}
	for _, elo := range elos {
		elosSlice = append(elosSlice, elo)
	}

	sort.Slice(elosSlice, func(i, j int) bool {
		return elosSlice[i].Points > elosSlice[j].Points
	})

	return elosSlice, nil
}
