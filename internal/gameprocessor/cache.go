package gameprocessor

type cache map[uint]*Game

func newCache() *cache {
	return &cache{}
}

func (c *cache) setEntry(gameId uint, newEntry *Game) {
	if c == nil {
		c = newCache()
	}

	(*c)[gameId] = newEntry
}

func (c *cache) getEntry(gameId uint) *Game {
	if c == nil {
		c = newCache()
		return nil
	}

	return (*c)[gameId]
}
