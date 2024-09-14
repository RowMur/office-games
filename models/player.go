package models

type Player struct {
	UserID int
	Name   string
	Points int
}

func newPlayer(name string) *Player {
	return &Player{
		Name:   name,
		Points: 500,
	}
}
