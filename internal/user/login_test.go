package user_test

import (
	"testing"

	"github.com/RowMur/office-games/internal/user"
	"golang.org/x/crypto/bcrypt"
)

func TestLogin(t *testing.T) {
	t.Run("user doesn't exist", func(t *testing.T) {
		u := user.NewUserService(NewMockDB(nil))
		_, err := u.Login("username", "password")
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("passwords don't match", func(t *testing.T) {
		u := user.NewUserService(NewMockDB([]mockUser{
			{username: "username", email: "email", password: "password"},
		}))
		_, errs := u.Login("username", "wrongpassword")
		if errs == nil || errs.Password == nil {
			t.Errorf("expected password error, got nil")
		}
	})

	t.Run("successful login", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "secret")

		password := "password"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		u := user.NewUserService(NewMockDB([]mockUser{
			{username: "username", email: "email", password: string(hashedPassword)},
		}))
		token, errs := u.Login("username", "password")
		if errs != nil {
			t.Errorf("expected no errors, got %v", errs)
		}
		if token == "" {
			t.Errorf("expected token, got empty string")
		}
	})
}
