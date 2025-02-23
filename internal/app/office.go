package app

import (
	"errors"

	"github.com/RowMur/office-table-tennis/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (a *App) GetOfficeByCode(code string) (*db.Office, error) {
	office := &db.Office{}
	err := a.db.C.Where("code = ?", code).
		Preload("Players", func(db *gorm.DB) *gorm.DB {
			return db.Order("LOWER(username)")
		}).
		Preload("Matches", func(d *gorm.DB) *gorm.DB {
			return d.Where("state = ?", db.MatchStateApproved).Order("created_at DESC")
		}).
		Preload("Matches.Participants.User").
		Preload("Matches.Creator").
		Preload("Matches.Approvals").
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
	err := a.db.C.Where("code = ?", code).Preload("Players").First(office).Error
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

	tx := a.db.C.Begin()
	err = tx.Model(office).Association("Players").Append(user)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	db.InvalidateGetUserByIdCache(user.ID)
	return nil, nil
}

func (a *App) CreateOffice(admin *db.User, name string) (*db.Office, error) {
	tx := a.db.C.Begin()

	office := &db.Office{Name: name, AdminRefer: admin.ID}
	err := tx.Create(office).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	db.InvalidateGetUserByIdCache(admin.ID)
	return office, nil
}
