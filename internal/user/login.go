package user

import (
	"github.com/RowMur/office-games/internal/token"
	"golang.org/x/crypto/bcrypt"
)

type loginErrors struct {
	Username string
	Password string
	Error    error
}

func (u *UserService) Login(username, password string) (string, *loginErrors) {
	if username == "" {
		return "", &loginErrors{Username: "Username is required"}
	}
	if password == "" {
		return "", &loginErrors{Password: "Password is required"}
	}

	user, err := u.db.GetUserByUsername(username)
	if err != nil {
		return "", &loginErrors{Error: err}
	}
	if user == nil {
		return "", &loginErrors{Username: "User does not exist"}
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", &loginErrors{Password: "Invalid password"}
	}

	token, err := token.GenerateToken(user.ID, token.AuthenticationToken)
	if err != nil {
		return "", &loginErrors{Error: err}
	}
	return token, nil
}
