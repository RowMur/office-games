package database

import "gorm.io/gorm"

type Player struct {
	gorm.Model
	Points      int `gorm:"default:500"`
	UserRefer   uint
	User        User `gorm:"foreignKey:UserRefer"`
	OfficeRefer uint
	Office      Office `gorm:"foreignKey:OfficeRefer"`
}
