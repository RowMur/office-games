package main

import (
	"github.com/RowMur/office-games/internal/server"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	server.NewServer().Run()
}
