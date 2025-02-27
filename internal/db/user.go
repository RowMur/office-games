package db

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	gorm.Model
	Username            string `gorm:"unique"`
	Email               string `gorm:"unique"`
	Password            string
	Offices             []Office `gorm:"many2many:user_offices;"`
	MatchParticipations []MatchParticipant
	Approvals           []MatchApproval
	NonPlayer           bool `default:"false"`
}

type CreateUserErrors struct {
	Username string
	Email    string
	Error    error
}

func (d Database) CreateUser(username, email, password string) *CreateUserErrors {
	user := &User{Username: username, Email: email, Password: password}
	err := d.C.Create(user).Error
	if err != nil {
		postgresErr, ok := err.(*pgconn.PgError)
		if !ok {
			return &CreateUserErrors{Error: err}
		}

		if postgresErr.SQLState() == ErrPostgresConstraintViolation {
			columnName := parsePostgresConstraintError(postgresErr)
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
	err := d.C.Where("username = ?", username).First(&user).Error
	if err != nil {
		if IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

var getUserByIdCache = map[uint]*User{}

func (d *Database) GetUserById(id uint) (*User, error) {
	if getUserByIdCache[id] != nil {
		return getUserByIdCache[id], nil
	}

	var user User
	err := d.C.Preload("Offices").First(&user, id).Error
	if err != nil {
		if IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	getUserByIdCache[id] = &user
	return &user, nil
}

func InvalidateGetUserByIdCache(id uint) {
	getUserByIdCache[id] = nil
}

type UpdateErrors map[string]string

func (d *Database) UpdateUser(id uint, updates map[string]interface{}) (*User, UpdateErrors, error) {
	user := &User{}
	err := d.C.Model(user).Where("id = ?", id).Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Updates(updates).Error
	if err != nil {
		postgresErr, ok := err.(*pgconn.PgError)
		if !ok {
			return nil, nil, err
		}

		if postgresErr.SQLState() == ErrPostgresConstraintViolation {
			columnName := parsePostgresConstraintError(postgresErr)
			for key := range updates {
				if key == columnName {
					return nil, UpdateErrors{columnName: fmt.Sprintf("%s is taken", columnName)}, nil
				}
			}
		}
	}

	InvalidateGetUserByIdCache(user.ID)
	return user, nil, nil
}

var ErrPostgresConstraintViolation = "23505"

func parsePostgresConstraintError(err *pgconn.PgError) string {
	constaintArray := strings.Split(err.ConstraintName, "_")
	columnName := constaintArray[len(constaintArray)-1]
	return columnName
}
