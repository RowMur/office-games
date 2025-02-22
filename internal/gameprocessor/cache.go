package gameprocessor

type cache map[uint]*Game

func newCache() *cache {
	return &cache{}
}

func (c *cache) setEntry(officeId uint, newEntry *Game) {
	if c == nil {
		c = newCache()
	}

	(*c)[officeId] = newEntry
}

func (c *cache) getEntry(officeId uint) *Game {
	if c == nil {
		c = newCache()
		return nil
	}

	return (*c)[officeId]
}
