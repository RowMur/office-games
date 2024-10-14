package app

import (
	"errors"

	"github.com/RowMur/office-games/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (a *App) GetOfficeByCode(code string) (*db.Office, error) {
	office := &db.Office{}
	err := a.db.Where("code = ?", code).
		Preload("Players", func(db *gorm.DB) *gorm.DB {
			return db.Order("username NOCASE")
		}).
		Preload(clause.Associations).
		First(office).Error
	if err != nil {
		if db.IsRecordNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	return office, nil
}

func (a *App) JoinOffice(user *db.User, code string) (error, error) {
	office := &db.Office{}
	err := a.db.Where("code = ?", code).Preload("Players").Preload("Games").First(office).Error
	if err != nil {
		if db.IsRecordNotFoundError(err) {
			return errors.New("Office not found"), nil
		}

		return nil, err
	}

	// Check if user is already in the office
	for _, player := range office.Players {
		if player.ID == user.ID {
			return nil, nil
		}
	}

	tx := a.db.Begin()
	err = tx.Model(office).Association("Players").Append(user)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	var initRankingsForEachOfficeGame []db.Ranking
	for _, game := range office.Games {
		initRankingsForEachOfficeGame = append(initRankingsForEachOfficeGame, db.Ranking{UserID: user.ID, GameID: game.ID})
	}
	if len(initRankingsForEachOfficeGame) > 0 {
		err = tx.Model(user).Association("Rankings").Append(initRankingsForEachOfficeGame)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	tx.Commit()
	return nil, nil
}

func (a *App) CreateOffice(admin *db.User, name string) (*db.Office, error) {
	office := &db.Office{Name: name, AdminRefer: admin.ID}
	err := a.db.Create(office).Error
	if err != nil {
		return nil, err
	}

	return office, nil
}
