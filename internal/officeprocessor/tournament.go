package officeprocessor

import (
	"fmt"

	"github.com/RowMur/office-table-tennis/internal/db"
)

type tournament struct {
	tournament     db.Tournament
	IsActive       bool
	scheduledCount int
	playedCount    int
}

func (t *tournament) Link() string {
	return fmt.Sprintf("/offices/%s/tournaments/%d", t.tournament.Office.Code, t.tournament.ID)
}

func (t *tournament) Name() string {
	return t.tournament.Name
}

func (t *tournament) PlayerCount() int {
	return len(t.tournament.Participants)
}

func (t *tournament) StartDate() string {
	return t.tournament.CreatedAt.Format("02/01/06")
}

func (t *tournament) Progress() float64 {
	if t.scheduledCount == 0 {
		return 100
	}
	return float64(t.playedCount) * 100 / float64(t.scheduledCount)
}
