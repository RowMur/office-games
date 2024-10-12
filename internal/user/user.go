package user

import (
	"github.com/RowMur/office-games/internal/db"
)

type database interface {
	CreateUser(username, email, password string) *db.CreateUserErrors
	GetUserByUsername(username string) (*db.User, error)
	UpdateUser(id uint, updates map[string]interface{}) error
}

type UserService struct {
	db database
}

func NewUserService(db database) *UserService {
	return &UserService{
		db: db,
	}
}
