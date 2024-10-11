package user

import (
	"errors"

	"github.com/RowMur/office-games/internal/token"
	"golang.org/x/crypto/bcrypt"
)

type loginErrors struct {
	Username error
	Password error
	Error    error
}

func (u *UserService) Login(username, password string) (string, *loginErrors) {
	user, err := u.db.GetUserByUsername(username)
	if err != nil {
		return "", &loginErrors{Error: err}
	}
	if user == nil {
		return "", &loginErrors{Username: errors.New("User does not exist")}
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", &loginErrors{Password: errors.New("Invalid password")}
	}

	token, err := token.GenerateToken(user.ID, token.AuthenticationToken)
	if err != nil {
		return "", &loginErrors{Error: err}
	}
	return token, nil
}
