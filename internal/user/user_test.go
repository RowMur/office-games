package user_test

import (
	"errors"

	"github.com/RowMur/office-games/internal/db"
)

type mockUser struct {
	username string
	email    string
	password string
}

type mockDB struct {
	users []mockUser
}

func NewMockDB(mockUsers []mockUser) *mockDB {
	return &mockDB{
		users: mockUsers,
	}
}

func (m *mockDB) CreateUser(username, email, password string) *db.CreateUserErrors {
	for _, u := range m.users {
		if u.username == username {
			return &db.CreateUserErrors{Username: "Username is taken"}
		}
		if u.email == email {
			return &db.CreateUserErrors{Email: "Email is taken"}
		}
	}
	m.users = append(m.users, mockUser{username: username, email: email, password: password})
	return nil
}

func (m *mockDB) GetUserByUsername(username string) (*db.User, error) {
	for _, u := range m.users {
		if u.username == username {
			return &db.User{Username: u.username, Email: u.email, Password: u.password}, nil
		}
	}

	return nil, errors.New("not found")
}

func (m *mockDB) UpdateUser(id uint, updates map[string]interface{}) (*db.User, db.UpdateErrors, error) {
	return nil, nil, nil
}
