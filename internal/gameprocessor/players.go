package gameprocessor

import (
	"time"

	"github.com/RowMur/office-games/internal/db"
)

type Player struct {
	User      db.User
	Points    int
	WinCount  int
	LossCount int

	RecordPoints     int
	RecordPointsDate time.Time
}

func (p Player) matchesPlayed() int {
	return p.WinCount + p.LossCount
}

func (p Player) Percentage() float64 {
	total := p.matchesPlayed()
	percentage := 0.0
	if total > 0 {
		percentage = float64(p.WinCount) / float64(total) * 100
	}

	return percentage
}
