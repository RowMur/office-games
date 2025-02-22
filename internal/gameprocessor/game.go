package gameprocessor

import (
	"sort"
)

type Game struct {
	matches                map[uint]*processedMatch
	players                map[uint]Player
	playerPairings         *playerCombinations
	playerOpposingPairings *playerCombinations
}

func newGame() Game {
	return Game{
		matches:                map[uint]*processedMatch{},
		players:                map[uint]Player{},
		playerPairings:         newPlayerCombinations(),
		playerOpposingPairings: newPlayerCombinations(),
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

func (g *Game) MostPlayedPlayer() (player *Player) {
	mostPlayed := &Player{}
	for _, player := range g.players {
		if player.MatchesPlayed() > mostPlayed.MatchesPlayed() {
			mostPlayed = &player
		}
	}

	return mostPlayed
}

func (g *Game) HighestRankedPlayer() (player *Player) {
	rankedPlayers := g.RankedPlayers()
	if len(rankedPlayers) > 0 {
		return &rankedPlayers[0]
	}

	return nil
}

func (g *Game) PlayerCountCounts() map[int]int {
	counts := map[int]int{}
	for _, match := range g.matches {
		count := len(match.Participants)
		if _, ok := counts[count]; !ok {
			counts[count] = 0
		}

		counts[count]++
	}

	return counts
}

func (g *Game) MostCommonPairing() *playerCombination {
	pairings := g.playerPairings.orderedPlayerCombinations()
	if len(pairings) > 0 {
		return &pairings[0]
	}

	return nil
}

func (g *Game) MostCommonOpposingPairing() *playerCombination {
	pairings := g.playerOpposingPairings.orderedPlayerCombinations()
	if len(pairings) > 0 {
		return &pairings[0]
	}

	return nil
}

func (g *Game) MostCommonPairingForPlayer(p Player) *playerCombination {
	pairings := g.playerPairings.orderedPlayerCombinationsForUser(p.User.ID)
	if len(pairings) > 0 {
		return &pairings[0]
	}

	return nil
}

func (g *Game) MostCommonOpponentForPlayer(p Player) *playerCombination {
	pairings := g.playerOpposingPairings.orderedPlayerCombinationsForUser(p.User.ID)
	if len(pairings) > 0 {
		return &pairings[0]
	}

	return nil
}

func (g *Game) GetMatch(matchId uint) *processedMatch {
	return g.matches[matchId]
}

func (g *Game) GetPlayer(userId uint) *Player {
	player, ok := g.players[userId]
	if !ok {
		return nil
	}

	return &player
}

func (g *Game) RankedPlayers() []Player {
	players := []Player{}
	for _, player := range g.players {
		if player.IsActive {
			players = append(players, player)
		}
	}

	sort.Slice(players, func(i, j int) bool {
		playerI := players[i]
		playerJ := players[j]
		if playerI.Points != playerJ.Points {
			return playerI.Points > playerJ.Points
		}
		if playerI.WinCount != playerJ.WinCount {
			return playerI.WinCount > playerJ.WinCount
		}
		if playerI.LossCount != playerJ.LossCount {
			return playerI.LossCount < playerJ.LossCount
		}
		return playerI.User.Username > playerJ.User.Username
	})

	return players
}
