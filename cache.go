package main

import (
	"sync"
	"time"
)

const AutoCleanUpInterval = 20

type CacheForm struct {
	Key   string                 `json:"key"`
	Value string                 `json:"value"`
	Data  map[string]interface{} `json:"data"`
	TTL   int                    `json:"ttl"`
}

type CacheItem struct {
	Value      string
	Expiration time.Time
}

type Cache struct {
	data     map[string]CacheItem
	mu       sync.RWMutex
	stopChan chan struct{}
	stats    CacheStats
}

func NewCache() *Cache {
	cache := &Cache{
		data:     make(map[string]CacheItem),
		stopChan: make(chan struct{}),
		stats:    CacheStats{TotalKeys: 0, Hits: 0, Misses: 0},
	}
	go cache.startAutoCleanup(AutoCleanUpInterval * time.Second)
	return cache
}

func (c *Cache) Set(key, value string, ttl time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = CacheItem{
		Value:      value,
		Expiration: ttl,
	}
}

func (c *Cache) Get(key string) (CacheItem, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.data[key]
	if !found || time.Now().After(item.Expiration) {
		return CacheItem{}, false
	}
	return item, true
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *Cache) startAutoCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpiredItems()
		case <-c.stopChan:
			return
		}
	}
}

func (c *Cache) cleanupExpiredItems() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.data {
		if item.Expiration.Before(now) {
			delete(c.data, key)
		}
	}
}

func (c *Cache) StopAutoCleanup() {
	close(c.stopChan)
}

func (c *Cache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		TotalKeys: len(c.data),
		Hits:      c.stats.Hits,
		Misses:    c.stats.Misses,
		Uptime:    getUptimeFormatted(),
	}
}
