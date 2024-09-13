package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
)

type Player struct {
	Name   string
	Points int
}

func newPlayer(name string) *Player {
	return &Player{
		Name:   name,
		Points: rand.Intn(1000),
	}
}

type Office struct {
	Name    string
	Players []*Player
}

func newOffice(name string) *Office {
	players := []*Player{
		newPlayer("John"),
		newPlayer("Jane"),
		newPlayer("Jack"),
		newPlayer("Jill"),
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].Points > players[j].Points
	})
	return &Office{
		Name:    name,
		Players: players,
	}
}

func main() {
	office := newOffice("Office")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		page(*office).Render(context.Background(), w)
	})
	fmt.Println("Server is running on port 8080")
	http.ListenAndServe(":8080", nil)
}
