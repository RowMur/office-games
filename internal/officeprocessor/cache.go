package officeprocessor

type cache map[uint]*Office

func newCache() *cache {
	return &cache{}
}

func (c *cache) setEntry(officeId uint, newEntry *Office) {
	if c == nil {
		c = newCache()
	}

	(*c)[officeId] = newEntry
}

func (c *cache) getEntry(officeId uint) *Office {
	if c == nil {
		c = newCache()
		return c.getEntry(officeId)
	}

	return (*c)[officeId]
}
