package database

import (
	"math/rand"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string
}

func CreateUser(username string) (*User, error) {
	user := &User{
		Username: username,
	}

	result := GetDB().Create(user)
	if result.Error != nil {
		return nil, result.Error
	}

	return user, nil
}

func (u User) CreateOffice(name string) (*Office, error) {
	office := &Office{
		Name:       name,
		AdminRefer: u.ID,
		Code:       generateCode(),
	}

	result := GetDB().Create(office)
	if result.Error != nil {
		return nil, result.Error
	}

	office.AddPlayer(&u)
	return office, nil
}

func (u User) GetOffices() ([]*Office, error) {
	players := []Player{}
	result := GetDB().Where("user_refer = ?", u.ID).Find(&players)
	if result.Error != nil {
		return nil, result.Error
	}

	officeIDs := make([]uint, len(players))
	for i, player := range players {
		officeIDs[i] = player.OfficeRefer
	}

	offices := []*Office{}
	result = GetDB().Where("id IN ?", officeIDs).Find(&offices)
	if result.Error != nil {
		return nil, result.Error
	}

	return offices, nil
}

func generateCode() string {
	lengthOfCode := 6
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	code := make([]rune, lengthOfCode)

	for i := range lengthOfCode {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}

func GetUser(username string) (*User, error) {
	user := &User{}
	result := GetDB().Where("username = ?", username).First(user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	return user, nil
}
