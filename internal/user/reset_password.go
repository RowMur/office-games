package user

import (
	"golang.org/x/crypto/bcrypt"
)

type resetPasswordsErrors struct {
	Password string
	Confirm  string
	Error    error
}

func (u *UserService) ResetPassword(userId uint, password, confirm string) *resetPasswordsErrors {
	if password == "" {
		return &resetPasswordsErrors{Password: "Password is required"}
	}
	if confirm == "" {
		return &resetPasswordsErrors{Confirm: "Confirm password is required"}
	}

	if password != confirm {
		return &resetPasswordsErrors{Confirm: "Passwords do not match"}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return &resetPasswordsErrors{Error: err}
	}

	err = u.db.UpdateUser(userId, map[string]interface{}{"password": string(hashedPassword)})
	if err != nil {
		return &resetPasswordsErrors{Error: err}
	}

	return nil
}
