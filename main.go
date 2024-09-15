package main

import (
	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/server"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	db.Init()
	server.NewServer().Run()
}
