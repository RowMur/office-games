package user

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type createUserErrors struct {
	Username error
	Email    error
	Password error
	Confirm  error
	Error    error
}

var errRequired = errors.New("required")
var errPasswordsDoNotMatch = errors.New("passwords do not match")

func (u *UserService) CreateUser(username, email, password, confirm string) *createUserErrors {
	if username == "" {
		return &createUserErrors{Username: errRequired}
	}
	if email == "" {
		return &createUserErrors{Email: errRequired}
	}
	if password == "" {
		return &createUserErrors{Password: errRequired}
	}
	if confirm == "" {
		return &createUserErrors{Confirm: errRequired}
	}
	if password != confirm {
		return &createUserErrors{Confirm: errPasswordsDoNotMatch}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return &createUserErrors{Error: err}
	}

	errs := u.db.CreateUser(username, email, string(hashedPassword))
	if errs != nil {
		return &createUserErrors{
			Username: errs.Username,
			Email:    errs.Email,
			Error:    errs.Error,
		}
	}

	return nil
}
