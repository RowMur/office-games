package gameprocessor

import (
	"time"

	"github.com/RowMur/office-table-tennis/internal/db"
)

type Player struct {
	User      db.User
	IsActive  bool
	Points    int
	WinCount  int
	LossCount int

	RecordPoints     int
	RecordPointsDate time.Time
}

func (p Player) MatchesPlayed() int {
	return p.WinCount + p.LossCount
}

func (p Player) Percentage() float64 {
	total := p.MatchesPlayed()
	percentage := 0.0
	if total > 0 {
		percentage = float64(p.WinCount) / float64(total) * 100
	}

	return percentage
}
