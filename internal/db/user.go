package db

import (
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username            string `gorm:"unique"`
	Email               string `gorm:"unique"`
	Password            string
	Offices             []Office `gorm:"many2many:user_offices;"`
	Rankings            []Ranking
	MatchParticipations []MatchParticipant
	Approvals           []MatchApproval
}

type CreateUserErrors struct {
	Username string
	Email    string
	Error    error
}

func (d *Database) CreateUser(username, email, password string) *CreateUserErrors {
	user := &User{Username: username, Email: email, Password: password}
	err := d.c.Create(user).Error
	if err != nil {
		postgresErr, ok := err.(*pgconn.PgError)
		if !ok {
			return &CreateUserErrors{Error: err}
		}

		if postgresErr.SQLState() == "23505" {
			constaintArray := strings.Split(postgresErr.ConstraintName, "_")
			columnName := constaintArray[len(constaintArray)-1]
			if columnName == "username" {
				return &CreateUserErrors{Username: "Username is taken"}
			}
			if columnName == "email" {
				return &CreateUserErrors{Email: "Email is taken"}
			}
		}

		return &CreateUserErrors{Error: err}
	}

	return nil
}

func (d *Database) GetUserByUsername(username string) (*User, error) {
	var user User
	err := d.c.Where("username = ?", username).First(&user).Error
	if err != nil {
		if IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (d *Database) GetUserById(id uint) (*User, error) {
	var user User
	err := d.c.Preload("Offices").First(&user, id).Error
	if err != nil {
		if IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}
