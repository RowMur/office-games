package elo

// key being the game ID
type cache map[uint]cacheEntry

func NewCache() *cache {
	newCache := cache{}
	return &newCache
}

type cacheEntry struct {
	matches map[uint]ProcessedMatch
	elos    Elos
}

type ProcessedMatch struct {
	Participants map[uint]ProcessedMatchParticipant
}

type ProcessedMatchParticipant struct {
	UserID        uint
	Win           bool
	PointsApplied int
}
