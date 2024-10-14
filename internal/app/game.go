package app

import (
	"github.com/RowMur/office-games/internal/db"
	"gorm.io/gorm"
)

func (a *App) GetGameById(id string, pendingMatches bool) (*db.Game, error) {
	game := db.Game{}

	query := a.db.Where("id = ?", id).
		Preload("Office.Players", func(db *gorm.DB) *gorm.DB {
			return db.Order("username")
		}).
		Preload("Rankings", func(db *gorm.DB) *gorm.DB {
			return db.Order("Rankings.points DESC")
		}).
		Preload("Rankings.User").
		Preload("Matches.Participants.User").
		Preload("Matches.Creator").
		Preload("Matches.Approvals")
	sortMatchesFunc := func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}
	if pendingMatches {
		query = query.Preload("Matches", "state = ?", db.MatchStatePending, sortMatchesFunc)
	} else {
		query = query.Preload("Matches", "state NOT IN (?)", db.MatchStatePending, sortMatchesFunc)
	}

	err := query.First(&game).Error
	if err != nil {
		return nil, err
	}

	return &game, nil
}
