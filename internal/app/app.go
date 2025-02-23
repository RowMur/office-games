package app

import (
	"github.com/RowMur/office-table-tennis/internal/db"
	"github.com/RowMur/office-table-tennis/internal/officeprocessor"
)

type App struct {
	db db.Database
	op *officeprocessor.Officeprocessor
}

func NewApp(db db.Database, op *officeprocessor.Officeprocessor) *App {
	return &App{
		db: db,
		op: op,
	}
}
