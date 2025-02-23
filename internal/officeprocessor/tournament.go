package officeprocessor

import (
	"fmt"
	"slices"

	"github.com/RowMur/office-table-tennis/internal/db"
)

type Tournament struct {
	tournament     db.Tournament
	IsActive       bool
	scheduledCount int
	playedCount    int
	matches        map[uint]db.Match
	rounds         [][]db.Match
}

func newTournament(t db.Tournament) *Tournament {
	return &Tournament{
		tournament: t,
		matches:    map[uint]db.Match{},
		rounds:     [][]db.Match{},
	}
}

func (t *Tournament) Link() string {
	return fmt.Sprintf("/offices/%s/tournaments/%d", t.tournament.Office.Code, t.tournament.ID)
}

func (t *Tournament) Name() string {
	return t.tournament.Name
}

func (t *Tournament) PlayerCount() int {
	return len(t.tournament.Participants)
}

func (t *Tournament) StartDate() string {
	return t.tournament.CreatedAt.Format("02/01/06")
}

func (t *Tournament) Progress() float64 {
	if t.scheduledCount == 0 {
		return 100
	}
	return float64(t.playedCount) * 100 / float64(t.scheduledCount)
}

func (t *Tournament) Rounds() [][]db.Match {
	return t.rounds
}

func (t *Tournament) ConstructRounds() {
	rootId := uint(0)
	matchToChildren := map[uint][]uint{}
	for _, match := range t.matches {
		if match.NextMatchID == nil {
			rootId = match.ID
			continue
		}

		if _, ok := matchToChildren[*match.NextMatchID]; !ok {
			matchToChildren[*match.NextMatchID] = []uint{}
		}

		matchToChildren[*match.NextMatchID] = append(matchToChildren[*match.NextMatchID], match.ID)
		slices.Sort(matchToChildren[*match.NextMatchID])
	}

	fmt.Println(matchToChildren)

	rounds := [][]uint{
		{rootId},
	}
	for {
		lastRound := rounds[len(rounds)-1]
		nextRound := []uint{}
		for _, matchId := range lastRound {
			if children, ok := matchToChildren[matchId]; ok {
				nextRound = append(nextRound, children...)
			}
		}

		if len(nextRound) == 0 {
			break
		}

		rounds = append(rounds, nextRound)
	}

	for i := len(rounds) - 1; i >= 0; i-- {
		round := []db.Match{}
		for _, matchId := range rounds[i] {
			round = append(round, t.matches[matchId])
		}

		t.rounds = append(t.rounds, round)
	}
}
