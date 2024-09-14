package main

import (
	"github.com/RowMur/office-games/internal/database"
	"github.com/RowMur/office-games/internal/server"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	database.Init()
	server.NewServer().Run()
}
