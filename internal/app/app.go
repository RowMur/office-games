package app

import (
	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/elo"
)

type App struct {
	db db.Database
	es *elo.EloService
}

func NewApp(db db.Database, es *elo.EloService) *App {
	return &App{
		db: db,
		es: es,
	}
}
