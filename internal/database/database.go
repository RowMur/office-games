package database

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type database struct {
	db *gorm.DB
}

var databaseSingleton = database{}

func Init() {
	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Fatalf("DB_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dbUrl), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	err = db.AutoMigrate(&User{}, &Office{}, &Player{})
	if err != nil {
		log.Fatalf("Error migrating models: %v", err)
	}

	databaseSingleton.db = db
}

func GetDB() *gorm.DB {
	return databaseSingleton.db
}
