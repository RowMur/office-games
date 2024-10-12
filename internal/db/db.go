package db

import (
	"errors"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	c *gorm.DB
}

var databaseSingleton = Database{}

func Init() Database {
	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Fatalf("DB_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dbUrl), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	err = db.AutoMigrate(models...)
	if err != nil {
		log.Fatalf("Error migrating models: %v", err)
	}

	databaseSingleton.c = db
	return databaseSingleton
}

func GetDB() *gorm.DB {
	return databaseSingleton.c
}

func IsRecordNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
