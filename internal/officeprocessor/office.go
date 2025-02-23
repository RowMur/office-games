package officeprocessor

import (
	"sort"
)

type Office struct {
	matches                map[uint]*processedMatch
	players                map[uint]Player
	playerPairings         *playerCombinations
	playerOpposingPairings *playerCombinations
}

func newOffice() Office {
	return Office{
		matches:                map[uint]*processedMatch{},
		players:                map[uint]Player{},
		playerPairings:         newPlayerCombinations(),
		playerOpposingPairings: newPlayerCombinations(),
	}
}

func (o *Office) MatchesPlayed() int {
	return len(o.matches)
}

func (o *Office) RecordElo() (player *Player) {
	recordHolder := &Player{}
	for _, player := range o.players {
		if player.RecordPoints > recordHolder.RecordPoints {
			recordHolder = &player
		}
	}

	return recordHolder
}

func (o *Office) MostPlayedPlayer() (player *Player) {
	mostPlayed := &Player{}
	for _, player := range o.players {
		if player.MatchesPlayed() > mostPlayed.MatchesPlayed() {
			mostPlayed = &player
		}
	}

	return mostPlayed
}

func (o *Office) HighestRankedPlayer() (player *Player) {
	rankedPlayers := o.RankedPlayers()
	if len(rankedPlayers) > 0 {
		return &rankedPlayers[0]
	}

	return nil
}

func (o *Office) PlayerCountCounts() map[int]int {
	counts := map[int]int{}
	for _, match := range o.matches {
		count := len(match.Participants)
		if _, ok := counts[count]; !ok {
			counts[count] = 0
		}

		counts[count]++
	}

	return counts
}

func (o *Office) MostCommonPairing() *playerCombination {
	pairings := o.playerPairings.orderedPlayerCombinations()
	if len(pairings) > 0 {
		return &pairings[0]
	}

	return nil
}

func (o *Office) MostCommonOpposingPairing() *playerCombination {
	pairings := o.playerOpposingPairings.orderedPlayerCombinations()
	if len(pairings) > 0 {
		return &pairings[0]
	}

	return nil
}

func (o *Office) MostCommonPairingForPlayer(p Player) *playerCombination {
	pairings := o.playerPairings.orderedPlayerCombinationsForUser(p.User.ID)
	if len(pairings) > 0 {
		return &pairings[0]
	}

	return nil
}

func (o *Office) MostCommonOpponentForPlayer(p Player) *playerCombination {
	pairings := o.playerOpposingPairings.orderedPlayerCombinationsForUser(p.User.ID)
	if len(pairings) > 0 {
		return &pairings[0]
	}

	return nil
}

func (o *Office) GetMatch(matchId uint) *processedMatch {
	return o.matches[matchId]
}

func (o *Office) GetPlayer(userId uint) *Player {
	player, ok := o.players[userId]
	if !ok {
		return nil
	}

	return &player
}

func (o *Office) RankedPlayers() []Player {
	players := []Player{}
	for _, player := range o.players {
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
