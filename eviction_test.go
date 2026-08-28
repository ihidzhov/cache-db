package main

import (
	"strconv"
	"testing"
	"time"
)

func TestNoEvictionByDefault(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	for i := 0; i < 1000; i++ {
		c.Set("key"+strconv.Itoa(i), "v", time.Now().Add(time.Minute))
	}

	stats := c.GetStats()
	if stats.Evictions != 0 {
		t.Fatalf("expected no evictions with unlimited config, got %d", stats.Evictions)
	}
}

func TestLRUEviction(t *testing.T) {
	c := NewCache(CacheConfig{MaxKeys: 2, EvictionPolicy: EvictionLRU})
	defer c.StopAutoCleanup()

	c.Set("a", "1", time.Now().Add(time.Minute))
	c.Set("b", "2", time.Now().Add(time.Minute))

	// touch "a" so "b" becomes the least recently used
	c.Get("a")
	time.Sleep(2 * time.Millisecond)

	c.Set("c", "3", time.Now().Add(time.Minute))

	if c.Exists("b") {
		t.Fatal("expected least-recently-used key 'b' to be evicted")
	}
	if !c.Exists("a") || !c.Exists("c") {
		t.Fatal("expected 'a' and 'c' to survive eviction")
	}

	stats := c.GetStats()
	if stats.Evictions != 1 {
		t.Fatalf("expected 1 eviction, got %d", stats.Evictions)
	}
}

func TestLFUEviction(t *testing.T) {
	c := NewCache(CacheConfig{MaxKeys: 2, EvictionPolicy: EvictionLFU})
	defer c.StopAutoCleanup()

	c.Set("x", "1", time.Now().Add(time.Minute))
	c.Get("x")
	c.Get("x")

	c.Set("y", "2", time.Now().Add(time.Minute))

	c.Set("z", "3", time.Now().Add(time.Minute))

	if c.Exists("y") {
		t.Fatal("expected least-frequently-used key 'y' to be evicted")
	}
	if !c.Exists("x") || !c.Exists("z") {
		t.Fatal("expected 'x' and 'z' to survive eviction")
	}
}

func TestRandomEvictionRespectsLimit(t *testing.T) {
	c := NewCache(CacheConfig{MaxKeys: 3, EvictionPolicy: EvictionRandom})
	defer c.StopAutoCleanup()

	for i := 0; i < 10; i++ {
		c.Set(string(rune('a'+i)), "v", time.Now().Add(time.Minute))
	}

	items := c.AllItems()
	if len(items) != 3 {
		t.Fatalf("expected exactly 3 keys under max-keys=3, got %d", len(items))
	}
}

func TestMaxTotalBytesEviction(t *testing.T) {
	c := NewCache(CacheConfig{MaxTotalBytes: 20, EvictionPolicy: EvictionLRU})
	defer c.StopAutoCleanup()

	c.Set("a", "0123456789", time.Now().Add(time.Minute))
	time.Sleep(2 * time.Millisecond)
	c.Set("b", "0123456789", time.Now().Add(time.Minute))
	time.Sleep(2 * time.Millisecond)
	c.Set("c", "0123456789", time.Now().Add(time.Minute))

	if c.approxBytes > 20 {
		t.Fatalf("expected approxBytes <= 20 after eviction, got %d", c.approxBytes)
	}
	if c.Exists("a") {
		t.Fatal("expected oldest key 'a' to be evicted first")
	}
}
