package main

import "time"

func (c *Cache) touch(item *CacheItem) {
	if c.config.MaxKeys <= 0 && c.config.MaxTotalBytes <= 0 {
		return
	}
	item.LastAccessed = time.Now()
	item.AccessCount++
}

func (c *Cache) overLimit() bool {
	if c.config.MaxKeys > 0 && len(c.data) > c.config.MaxKeys {
		return true
	}
	if c.config.MaxTotalBytes > 0 && c.approxBytes > c.config.MaxTotalBytes {
		return true
	}
	return false
}

func (c *Cache) evictIfNeeded() {
	for c.overLimit() {
		victimKey := c.selectVictim()
		if victimKey == "" {
			return
		}
		c.removeItem(victimKey)
		c.statsMu.Lock()
		c.stats.Evictions++
		c.statsMu.Unlock()
	}
}

func (c *Cache) selectVictim() string {
	var victimKey string

	switch c.config.EvictionPolicy {
	case EvictionLFU:
		var lowestCount int64
		var lowestAccess time.Time
		first := true
		for k, item := range c.data {
			if first || item.AccessCount < lowestCount ||
				(item.AccessCount == lowestCount && item.LastAccessed.Before(lowestAccess)) {
				victimKey = k
				lowestCount = item.AccessCount
				lowestAccess = item.LastAccessed
				first = false
			}
		}
	case EvictionRandom:
		for k := range c.data {
			victimKey = k
			break
		}
	default:
		var oldest time.Time
		first := true
		for k, item := range c.data {
			if first || item.LastAccessed.Before(oldest) {
				victimKey = k
				oldest = item.LastAccessed
				first = false
			}
		}
	}

	return victimKey
}
