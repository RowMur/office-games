package main

import (
	"log"

	"github.com/RowMur/office-games/database"
	"github.com/RowMur/office-games/server"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	database.Init()
	server.NewServer().Run()
}
