package main

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
)

const AutoCleanUpInterval = 20

var (
	ErrWrongType     = errors.New("key holds a different type")
	ErrTTLRequired   = errors.New("ttl is required when creating a new key")
	ErrValueTooLarge = errors.New("value exceeds the configured size limit")
)

type ItemType string

const (
	TypeString ItemType = "string"
	TypeList   ItemType = "list"
	TypeHash   ItemType = "hash"
)

type EvictionPolicy string

const (
	EvictionLRU    EvictionPolicy = "lru"
	EvictionLFU    EvictionPolicy = "lfu"
	EvictionRandom EvictionPolicy = "random"
)

type CacheConfig struct {
	MaxKeys         int
	MaxValueBytes   int
	MaxTotalBytes   int64
	EvictionPolicy  EvictionPolicy
	CleanupInterval time.Duration
}

type CacheForm struct {
	Key   string                 `json:"key"`
	Value string                 `json:"value"`
	Data  map[string]interface{} `json:"data"`
	TTL   int                    `json:"ttl"`
}

type CacheItem struct {
	Key          string
	Type         ItemType
	Value        string            `json:",omitempty"`
	List         []string          `json:",omitempty"`
	Hash         map[string]string `json:",omitempty"`
	Expiration   time.Time
	LastAccessed time.Time `json:"-"`
	AccessCount  int64     `json:"-"`
}

func (i CacheItem) Expired() bool {
	return !i.Expiration.IsZero() && time.Now().After(i.Expiration)
}

func itemSize(item CacheItem) int64 {
	size := int64(len(item.Key)) + int64(len(item.Value))
	for _, v := range item.List {
		size += int64(len(v))
	}
	for k, v := range item.Hash {
		size += int64(len(k)) + int64(len(v))
	}
	return size
}

type Cache struct {
	data        map[string]CacheItem
	mu          sync.RWMutex
	stopChan    chan struct{}
	stats       CacheStats
	statsMu     sync.Mutex
	config      CacheConfig
	approxBytes int64
}

func (c *Cache) storeItem(key string, item CacheItem) {
	if old, existed := c.data[key]; existed {
		c.approxBytes -= itemSize(old)
	}
	c.approxBytes += itemSize(item)
	c.data[key] = item
}

func (c *Cache) removeItem(key string) {
	if old, existed := c.data[key]; existed {
		c.approxBytes -= itemSize(old)
		delete(c.data, key)
	}
}

func (c *Cache) IncrementHits() {
	c.statsMu.Lock()
	c.stats.Hits++
	c.statsMu.Unlock()
}

func (c *Cache) IncrementMisses() {
	c.statsMu.Lock()
	c.stats.Misses++
	c.statsMu.Unlock()
}

func NewCache(config CacheConfig) *Cache {
	if config.EvictionPolicy == "" {
		config.EvictionPolicy = EvictionLRU
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = AutoCleanUpInterval * time.Second
	}
	cache := &Cache{
		data:     make(map[string]CacheItem),
		stopChan: make(chan struct{}),
		config:   config,
	}
	go cache.startAutoCleanup(config.CleanupInterval)
	return cache
}

func (c *Cache) Set(key, value string, ttl time.Time) error {
	if c.config.MaxValueBytes > 0 && len(value) > c.config.MaxValueBytes {
		return ErrValueTooLarge
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	item := CacheItem{
		Key:        key,
		Type:       TypeString,
		Value:      value,
		Expiration: ttl,
	}
	c.touch(&item)
	c.storeItem(key, item)
	c.evictIfNeeded()
	return nil
}

func (c *Cache) Increment(key string, delta int) (CacheItem, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.data[key]
	if !found || item.Expired() {
		return CacheItem{}, false, nil
	}
	if item.Type != TypeString {
		return CacheItem{}, true, ErrWrongType
	}

	intValue, err := strconv.Atoi(item.Value)
	if err != nil {
		return CacheItem{}, true, fmt.Errorf("value for key %q is not an integer: %w", key, err)
	}

	item.Value = strconv.Itoa(intValue + delta)
	c.storeItem(key, item)
	return item, true, nil
}

func (c *Cache) IncrementFloat(key string, delta float64) (CacheItem, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.data[key]
	if !found || item.Expired() {
		return CacheItem{}, false, nil
	}
	if item.Type != TypeString {
		return CacheItem{}, true, ErrWrongType
	}

	floatValue, err := strconv.ParseFloat(item.Value, 64)
	if err != nil {
		return CacheItem{}, true, fmt.Errorf("value for key %q is not a number: %w", key, err)
	}

	item.Value = strconv.FormatFloat(floatValue+delta, 'f', -1, 64)
	c.storeItem(key, item)
	return item, true, nil
}

func (c *Cache) Get(key string) (CacheItem, bool) {
	if c.config.MaxKeys > 0 || c.config.MaxTotalBytes > 0 {
		c.mu.Lock()
		defer c.mu.Unlock()

		item, found := c.data[key]
		if !found || item.Expired() {
			return CacheItem{}, false
		}
		c.touch(&item)
		c.data[key] = item
		return item, true
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.data[key]
	if !found || item.Expired() {
		return CacheItem{}, false
	}
	return item, true
}

func (c *Cache) Exists(key string) bool {
	_, found := c.Get(key)
	return found
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeItem(key)
}

func (c *Cache) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]CacheItem)
	c.approxBytes = 0
}

func (c *Cache) AllItems() []CacheItem {
	c.mu.RLock()
	defer c.mu.RUnlock()

	items := make([]CacheItem, 0, len(c.data))
	for _, item := range c.data {
		if item.Expired() {
			continue
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})

	return items
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

	for key, item := range c.data {
		if item.Expired() {
			c.removeItem(key)
		}
	}
}

func (c *Cache) StopAutoCleanup() {
	close(c.stopChan)
}

func (c *Cache) Healthy(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		c.mu.Lock()
		c.mu.Unlock()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (c *Cache) GetStats() CacheStats {
	c.mu.RLock()
	totalKeys := len(c.data)
	approxBytes := c.approxBytes
	c.mu.RUnlock()

	c.statsMu.Lock()
	hits, misses, evictions := c.stats.Hits, c.stats.Misses, c.stats.Evictions
	c.statsMu.Unlock()

	return CacheStats{
		TotalKeys:   totalKeys,
		ApproxBytes: approxBytes,
		Hits:        hits,
		Misses:      misses,
		Evictions:   evictions,
		Uptime:      getUptimeFormatted(),
	}
}
