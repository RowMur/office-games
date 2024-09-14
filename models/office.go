package models

import (
	"math/rand"
	"sort"
)

type Office struct {
	Admin   *User
	Name    string
	Players []*Player
	Code    string
}

func (o *Office) SortPlayers() {
	sort.Slice(o.Players, func(i, j int) bool {
		return o.Players[i].Points > o.Players[j].Points
	})
}

func (o *Office) AddPlayer(name string) {
	o.Players = append(o.Players, newPlayer(name))
	o.SortPlayers()
}

func (o *Office) FindPlayer(name string) *Player {
	for _, p := range o.Players {
		if p.Name == name {
			return p
		}
	}

	return nil
}

func NewOffice(name string, user *User) *Office {
	player := newPlayer(user.Username)
	office := &Office{
		Admin: user,
		Name:  name,
		Players: []*Player{
			player,
		},
		Code: generateCode(),
	}

	office.SortPlayers()
	return office
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
