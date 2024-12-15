package elo

// key being the game ID
type cache map[uint]*cacheEntry

func newCache() *cache {
	return &cache{}
}

type cacheEntry struct {
	matches map[uint]*ProcessedMatch
	elos    Elos
}

type ProcessedMatch struct {
	Participants map[uint]*ProcessedMatchParticipant
}

type ProcessedMatchParticipant struct {
	UserID        uint
	Win           bool
	PointsApplied int
}

func (c *cache) setEntry(gameId uint, newEntry *cacheEntry) {
	if c == nil {
		c = newCache()
	}

	(*c)[gameId] = newEntry
}

func (c *cache) getEntry(gameId uint) *cacheEntry {
	if c == nil {
		c = newCache()
		return nil
	}

	return (*c)[gameId]
}

func newCacheEntry() cacheEntry {
	return cacheEntry{
		matches: map[uint]*ProcessedMatch{},
		elos:    Elos{},
	}
}
