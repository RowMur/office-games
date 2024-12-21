package gameprocessor

import (
	"fmt"
	"sort"
	"strconv"
)

type playerCombination struct {
	Player1 Player
	Player2 Player
	matches []uint
}

func (pc *playerCombination) MatchCount() int {
	return len(pc.matches)
}

func (pc *playerCombination) Print() string {
	return fmt.Sprintf("%s & %s (%d)", pc.Player1.User.Username, pc.Player2.User.Username, pc.MatchCount())
}

func (pc *playerCombination) PrintOtherPlayer(p Player) string {
	otherPlayer := pc.Player1
	if pc.Player1.User.ID == p.User.ID {
		otherPlayer = pc.Player2
	}

	return fmt.Sprintf("%s (%d)", otherPlayer.User.Username, pc.MatchCount())
}

type playerCombinations map[uint]map[uint]playerCombination

func newPlayerCombinations() *playerCombinations {
	return &playerCombinations{}
}

func (pcs *playerCombinations) addMatch(matchID uint, player1, player2 Player) {
	pcs.addInOneDirection(matchID, player1, player2)
	pcs.addInOneDirection(matchID, player2, player1)
}

func (pcs *playerCombinations) addInOneDirection(matchID uint, player1, player2 Player) {
	if player1.User.ID == player2.User.ID {
		return
	}

	if _, ok := (*pcs)[player1.User.ID]; !ok {
		(*pcs)[player1.User.ID] = map[uint]playerCombination{}
	}

	if _, ok := (*pcs)[player1.User.ID][player2.User.ID]; !ok {
		(*pcs)[player1.User.ID][player2.User.ID] = playerCombination{
			matches: []uint{},
			Player1: player1,
			Player2: player2,
		}
	}

	pc := (*pcs)[player1.User.ID][player2.User.ID]
	pc.matches = append((*pcs)[player1.User.ID][player2.User.ID].matches, matchID)
	(*pcs)[player1.User.ID][player2.User.ID] = pc
}

func (pcs *playerCombinations) orderedPlayerCombinations() []playerCombination {
	ordered := []playerCombination{}
	for _, playerMap := range *pcs {
		for _, pc := range playerMap {
			if pc.Player1.User.ID > pc.Player2.User.ID {
				ordered = append(ordered, pc)
			}
		}
	}

	sortPlayerCombinations(ordered)
	return ordered
}

func (pcs *playerCombinations) orderedPlayerCombinationsForUser(userId uint) []playerCombination {
	ret := []playerCombination{}
	playerMap, ok := (*pcs)[userId]
	if !ok {
		return ret
	}

	for _, pc := range playerMap {
		ret = append(ret, pc)
	}

	sortPlayerCombinations(ret)
	return ret
}

func print(pcs []playerCombination) string {
	str := ""
	for _, pc := range pcs {
		strMatchCount := strconv.Itoa(pc.MatchCount())
		str += pc.Player1.User.Username + " and " + pc.Player2.User.Username + ": " + strMatchCount + "\n"
	}

	return str
}

func sortPlayerCombinations(pcs []playerCombination) {
	sort.Slice(pcs, func(i, j int) bool {
		iCount := pcs[i].MatchCount()
		jCount := pcs[j].MatchCount()

		if iCount != jCount {
			return pcs[i].MatchCount() > pcs[j].MatchCount()
		}

		if pcs[i].Player1.User.Username != pcs[j].Player1.User.Username {
			return pcs[i].Player1.User.Username < pcs[j].Player1.User.Username
		}

		return pcs[i].Player2.User.Username < pcs[j].Player2.User.Username
	})
}
