package models

type User struct {
	ID       int
	Username string
}

var ID = 0

func NewUser(username string) *User {
	ID++
	return &User{
		ID:       ID,
		Username: username,
	}
}
