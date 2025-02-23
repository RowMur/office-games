package app

import (
	"errors"
	"math"

	"math/rand/v2"

	"github.com/RowMur/office-table-tennis/internal/db"
)

var (
	ErrTournamentNameEmpty            = errors.New("name cannot be empty")
	ErrTournamentInvalidParticipant   = errors.New("invalid participant")
	ErrTournamentDuplicateParticipant = errors.New("duplicate participant")
	ErrInvalidNumberOfParticipants    = errors.New("invalid number of participants")
	ErrUnauthorized                   = errors.New("must be office admin")
)

func (a *App) CreateTournament(creator *db.User, name string, office db.Office, participants []uint) (*db.Tournament, error) {
	if name == "" {
		return nil, ErrTournamentNameEmpty
	}

	if creator.ID != office.AdminRefer {
		return nil, ErrUnauthorized
	}

	validParticipants := make(map[uint]bool)
	for _, participant := range office.Players {
		if participant.NonPlayer {
			continue
		}

		validParticipants[participant.ID] = true
	}

	uniqueParticipants := make(map[uint]bool)
	for _, participant := range participants {
		if _, ok := validParticipants[participant]; !ok {
			return nil, ErrTournamentInvalidParticipant
		}
		if _, ok := uniqueParticipants[participant]; ok {
			return nil, ErrTournamentDuplicateParticipant
		}

		uniqueParticipants[participant] = true
	}

	participantCount := len(participants)
	if participantCount&(participantCount-1) != 0 {
		return nil, ErrInvalidNumberOfParticipants
	}

	for i := range participants {
		j := rand.IntN(i + 1)
		participants[i], participants[j] = participants[j], participants[i]
	}

	tx := a.db.C.Begin()
	defer tx.Rollback()

	tournament := &db.Tournament{Name: name, OfficeID: office.ID}
	err := a.db.C.Create(tournament).Error
	if err != nil {
		return nil, err
	}

	var tournamentParticipants []db.User
	err = a.db.C.Model(&db.User{}).Where("id IN ?", participants).Find(&tournamentParticipants).Error
	if err != nil {
		return nil, err
	}

	err = tx.Model(tournament).Association("Participants").Append(tournamentParticipants)
	if err != nil {
		return nil, err
	}

	nextRoundMatches := []db.Match{}
	for roundWidth := 1; roundWidth < participantCount; roundWidth *= 2 {
		nOfParticipantsInRound := roundWidth * 2
		thisRoundMatches := []db.Match{}
		for i := 0; i < roundWidth; i++ {
			indexOfNextRoundMatch := int(math.Trunc(float64(i) / 2))
			var nextRoundMatch *db.Match
			if (len(nextRoundMatches) - 1) >= indexOfNextRoundMatch {
				nextRoundMatch = &nextRoundMatches[indexOfNextRoundMatch]
			}

			var match *db.Match
			if nOfParticipantsInRound < participantCount {
				match, err = a.ScheduleMatch(tx, creator, office, tournament, []uint{}, []uint{}, nextRoundMatch)
			} else {
				match, err = a.ScheduleMatch(tx, creator, office, tournament, []uint{participants[i*2]}, []uint{participants[i*2+1]}, nextRoundMatch)
			}
			if err != nil {
				return nil, err
			}

			thisRoundMatches = append(thisRoundMatches, *match)
		}

		nextRoundMatches = thisRoundMatches
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, err
	}

	return tournament, nil
}
