package officeprocessor

import (
	"fmt"
	"time"

	"github.com/RowMur/office-table-tennis/internal/db"
)

const (
	matchesWithDoublePoints = 20
	eloStartingPoints       = 400
	eloLowerBound           = 200
)

type Officeprocessor struct {
	db    db.Database
	cache *cache
}

func Newofficeprocessor(db db.Database) *Officeprocessor {
	return &Officeprocessor{
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

func (op *Officeprocessor) Process(officeId uint) (*Office, error) {
	startTime := time.Now()
	defer func() {
		fmt.Printf("GetElos: %s\n", time.Since(startTime))
	}()

	if op.cache != nil {
		entry := op.cache.getEntry(officeId)
		if entry != nil {
			return entry, nil
		}
	}

	return op.process(officeId)
}

func (op *Officeprocessor) process(officeId uint) (*Office, error) {
	matches := []db.Match{}
	err := op.db.C.Where("office_id  = ?", officeId).
		Order("created_at").
		Preload("Participants.User").
		Find(&matches).Error
	if err != nil {
		return nil, err
	}

	tournaments := []db.Tournament{}
	err = op.db.C.Where("office_id = ?", officeId).
		Preload("Office").
		Preload("Participants").
		Find(&tournaments).Error
	if err != nil {
		return nil, err
	}

	o := newOffice()

	for _, t := range tournaments {
		o.tournaments[t.ID] = *newTournament(t)
	}

	players := map[uint]Player{}
	for _, match := range matches {
		var t Tournament
		var tournamentOk bool
		if match.TournamentID != nil {
			t, tournamentOk = o.tournaments[*match.TournamentID]
			t.matches[match.ID] = match
			o.tournaments[*match.TournamentID] = t
		}

		if match.State == db.MatchStateScheduled && match.TournamentID != nil {
			if tournamentOk {
				t.IsActive = true
				t.scheduledCount++
				o.tournaments[*match.TournamentID] = t
			}
		}

		if match.State != db.MatchStateApproved {
			continue
		}

		if tournamentOk {
			t.playedCount++
			o.tournaments[*match.TournamentID] = t
		}

		cachedMatch := processedMatch{
			Participants: map[uint]*ProcessedMatchParticipant{},
		}

		timeSinceMatch := time.Since(match.CreatedAt)
		activatePlayers := timeSinceMatch < 8*7*24*time.Hour // 8 weeks

		winners := []Player{}
		losers := []Player{}
		for _, participant := range match.Participants {

			if player, ok := players[participant.UserID]; !ok {
				players[participant.UserID] = Player{
					User:     participant.User,
					Points:   eloStartingPoints,
					IsActive: activatePlayers,
				}
			} else {
				if !player.IsActive && activatePlayers {
					player.IsActive = true
				}
				players[participant.UserID] = player
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
					o.playerPairings.addMatch(match.ID, winner, w)
				}
			}
			for _, l := range losers {
				o.playerOpposingPairings.addMatch(match.ID, winner, l)
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
					o.playerPairings.addMatch(match.ID, loser, l)
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

		o.matches[match.ID] = &cachedMatch
	}

	for _, t := range o.tournaments {
		t.ConstructRounds()
		o.tournaments[t.tournament.ID] = t
	}

	o.players = players
	op.cache.setEntry(officeId, &o)
	return &o, nil
}

func (op Officeprocessor) InvalidateOfficeCache(officeId uint) {
	(*op.cache)[officeId] = nil
}
