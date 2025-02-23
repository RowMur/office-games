package app

import (
	"github.com/RowMur/office-table-tennis/internal/db"
	"github.com/RowMur/office-table-tennis/internal/gameprocessor"
)

type App struct {
	db db.Database
	gp *gameprocessor.GameProcessor
}

func NewApp(db db.Database, gp *gameprocessor.GameProcessor) *App {
	return &App{
		db: db,
		gp: gp,
	}
}
