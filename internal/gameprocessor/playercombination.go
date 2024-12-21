package gameprocessor

import (
	"sort"
	"strconv"
)

type playerCombination struct {
	player1 Player
	player2 Player
	matches []uint
}

func (pc *playerCombination) matchCount() int {
	return len(pc.matches)
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
			player1: player1,
			player2: player2,
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
			if pc.player1.User.ID > pc.player2.User.ID {
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
		strMatchCount := strconv.Itoa(pc.matchCount())
		str += pc.player1.User.Username + " and " + pc.player2.User.Username + ": " + strMatchCount + "\n"
	}

	return str
}

func sortPlayerCombinations(pcs []playerCombination) {
	sort.Slice(pcs, func(i, j int) bool {
		iCount := pcs[i].matchCount()
		jCount := pcs[j].matchCount()

		if iCount != jCount {
			return pcs[i].matchCount() > pcs[j].matchCount()
		}

		if pcs[i].player1.User.Username != pcs[j].player1.User.Username {
			return pcs[i].player1.User.Username < pcs[j].player1.User.Username
		}

		return pcs[i].player2.User.Username < pcs[j].player2.User.Username
	})
}
