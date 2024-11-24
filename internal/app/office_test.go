package app_test

import (
	"fmt"
	"testing"

	"github.com/RowMur/office-games/internal/app"
	"github.com/RowMur/office-games/internal/db"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var c *gorm.DB

func setup(t *testing.T) {
	fmt.Println("Setting up")
	if c != nil {
		return
	}

	var err error
	c, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = c.AutoMigrate(db.Models...)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	fmt.Println("Setup done")
}

func TestJoinOffice(t *testing.T) {
	setup(t)
	t.Run("It should return an error if the office code does not exist", func(t *testing.T) {
		app := app.NewApp(c)

		user := &db.User{Username: "username", Email: "email", Password: "password"}
		c.Create(user)
		defer c.Unscoped().Delete(user)

		userErr, _ := app.JoinOffice(user, "code")
		if userErr == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("It shouldn't add a duplicate user to the office", func(t *testing.T) {
		app := app.NewApp(c)

		user := &db.User{Username: "username", Email: "email", Password: "password"}
		c.Create(user)
		defer c.Unscoped().Delete(user)

		office := &db.Office{AdminRefer: user.ID}
		c.Create(office)
		defer c.Unscoped().Delete(office)

		userErr, err := app.JoinOffice(user, office.Code)
		if userErr != nil {
			t.Error("didn't expect an error, got one")
		}
		if err != nil {
			t.Error("didn't expect an error, got one")
		}

		players := []db.User{}
		c.Model(office).Association("Players").Find(&players)
		if len(players) != 1 {
			t.Errorf("expected 1 player, got %d", len(players))
		}
	})

	t.Run("It should initialise rankings for each game", func(t *testing.T) {
		app := app.NewApp(c)

		officeAdmin := &db.User{Username: "username", Email: "email", Password: "password"}
		c.Create(officeAdmin)
		defer c.Unscoped().Delete(officeAdmin)

		user := &db.User{Username: "user", Email: "userEmail", Password: "password"}
		c.Create(user)
		defer c.Unscoped().Delete(user)

		office := &db.Office{AdminRefer: officeAdmin.ID}
		c.Create(office)
		defer c.Unscoped().Delete(office)

		game1 := &db.Game{Name: "game1", OfficeID: office.ID}
		c.Create(game1)
		defer c.Unscoped().Delete(game1)

		userErr, err := app.JoinOffice(user, office.Code)
		if userErr != nil || err != nil {
			t.Error("didn't expect an error, got one")
		}

		rankings := []db.Ranking{}
		c.Model(user).Association("Rankings").Find(&rankings)
		if len(rankings) != 1 {
			t.Errorf("expected 1 rankings, got %d", len(rankings))
		}
	})

	return
}

func TestCreateOffice(t *testing.T) {
	setup(t)
	t.Run("It should create an office", func(t *testing.T) {
		app := app.NewApp(c)

		user := &db.User{Username: "username", Email: "email", Password: "password"}
		c.Create(user)
		defer c.Unscoped().Delete(user)

		office, err := app.CreateOffice(user, "office")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if office == nil {
			t.Error("expected an office, got nil")
		}

		if office.AdminRefer != user.ID {
			t.Errorf("expected admin to be %d, got %d", user.ID, office.AdminRefer)
		}
	})

	t.Run("It should create a default game", func(t *testing.T) {
		app := app.NewApp(c)

		user := &db.User{Username: "username", Email: "email", Password: "password"}
		c.Create(user)
		defer c.Unscoped().Delete(user)

		office, err := app.CreateOffice(user, "office")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if office == nil {
			t.Error("expected an office, got nil")
		}

		games := []db.Game{}
		c.Model(office).Association("Games").Find(&games)
		if len(games) != 1 {
			t.Errorf("expected 1 game, got %d", len(games))
		}

		c.Model(games[0]).Association("Rankings").Find(&games[0].Rankings)
		if len(games[0].Rankings) != 1 {
			t.Errorf("expected 1 ranking, got %d", len(games[0].Rankings))
		}
		if games[0].Rankings[0].UserID != user.ID {
			t.Errorf("expected user id to be %d, got %d", user.ID, games[0].Rankings[0].UserID)
		}
	})
}
