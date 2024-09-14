package database

import (
	"gorm.io/gorm"
)

type Office struct {
	gorm.Model
	Name       string
	AdminRefer uint
	Admin      User `gorm:"foreignKey:AdminRefer"`
	Code       string
}

func (o *Office) AddPlayer(user *User) (*Player, error) {
	newPlayer := Player{
		UserRefer:   user.ID,
		OfficeRefer: o.ID,
	}
	result := GetDB().Create(&newPlayer)
	if result.Error != nil {
		return nil, result.Error
	}

	return &newPlayer, nil
}

func (o *Office) FindPlayer(name string) (*Player, error) {
	user, err := GetUser(name)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	player := &Player{}
	result := GetDB().Where("office_refer = ?", o.ID).Where("user_refer = ?", user.ID).First(player)
	if result.Error != nil {
		return nil, result.Error
	}

	return player, nil
}

func (o *Office) GetPlayers() ([]*Player, error) {
	players := []*Player{}
	result := GetDB().Where("office_refer = ?", o.ID).Order("points desc").Preload("User").Find(&players)
	if result.Error != nil {
		return nil, result.Error
	}

	return players, nil
}

func GetOfficeByCode(code string) (*Office, error) {
	office := &Office{}
	result := GetDB().Where("code = ?", code).First(office)
	if result.Error != nil {
		return nil, result.Error
	}

	return office, nil
}
