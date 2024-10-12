package app

import "gorm.io/gorm"

type App struct {
	db *gorm.DB
}

func NewApp(db *gorm.DB) *App {
	return &App{
		db: db,
	}
}
