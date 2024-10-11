package user_test

import (
	"testing"

	"github.com/RowMur/office-games/internal/user"
)

func TestCreateUser(t *testing.T) {
	t.Run("empty fields", func(t *testing.T) {
		u := user.NewUserService(NewMockDB(nil))
		errs := u.CreateUser("", "email", "password", "password")
		if errs == nil || errs.Username == nil {
			t.Errorf("expected username error, got nil")
		}
	})

	t.Run("passwords do not match", func(t *testing.T) {
		u := user.NewUserService(NewMockDB(nil))
		errs := u.CreateUser("username", "email", "password", "confirm")
		if errs == nil || errs.Confirm == nil {
			t.Errorf("expected confirm error, got nil")
		}
	})

	t.Run("username taken error", func(t *testing.T) {
		u := user.NewUserService(NewMockDB([]mockUser{
			{username: "username", email: "email", password: "password"},
		}))
		errs := u.CreateUser("username", "email", "password", "password")
		if errs == nil || errs.Username == nil {
			t.Errorf("expected username error error, got nil")
		}
	})

	t.Run("user gets stored", func(t *testing.T) {
		m := NewMockDB(nil)
		u := user.NewUserService(m)
		errs := u.CreateUser("username", "email", "password", "password")
		if errs != nil {
			t.Errorf("expected no errors, got %v", errs)
		}
		if len(m.users) != 1 {
			t.Errorf("expected 1 user, got %d", len(m.users))
		}
		if m.users[0].username != "username" {
			t.Errorf("expected username 'username', got %s", m.users[0].username)
		}
		if m.users[0].email != "email" {
			t.Errorf("expected email 'email', got %s", m.users[0].email)
		}
	})
}
