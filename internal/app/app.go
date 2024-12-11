package app

import (
	"github.com/RowMur/office-games/internal/db"
)

type App struct {
	db db.Database
}

func NewApp(db db.Database) *App {
	return &App{
		db: db,
	}
}
