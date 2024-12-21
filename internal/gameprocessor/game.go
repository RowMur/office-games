package gameprocessor

import (
	"sort"
)

type Game struct {
	matches map[uint]*processedMatch
	players map[uint]Player
}

func newGame() Game {
	return Game{
		matches: map[uint]*processedMatch{},
		players: map[uint]Player{},
	}
}

func (g *Game) MatchesPlayed() int {
	return len(g.matches)
}

func (g *Game) RecordElo() (player *Player) {
	recordHolder := &Player{}
	for _, player := range g.players {
		if player.RecordPoints > recordHolder.RecordPoints {
			recordHolder = &player
		}
	}

	return recordHolder
}

func (g *Game) GetMatch(matchId uint) *processedMatch {
	return g.matches[matchId]
}

func (g *Game) RankedPlayers() []Player {
	players := []Player{}
	for _, player := range g.players {
		players = append(players, player)
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].Points > players[j].Points
	})

	return players
}
