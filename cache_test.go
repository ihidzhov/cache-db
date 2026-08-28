package main

import (
	"sync"
	"testing"
	"time"
)

func newTestCache() *Cache {
	c := NewCache(CacheConfig{})
	return c
}

func TestSetGet(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	if err := c.Set("k", "v", time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	item, found := c.Get("k")
	if !found {
		t.Fatal("expected key to be found")
	}
	if item.Value != "v" {
		t.Fatalf("expected value %q, got %q", "v", item.Value)
	}
	if item.Type != TypeString {
		t.Fatalf("expected type %q, got %q", TypeString, item.Type)
	}
}

func TestGetMissingKey(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	if _, found := c.Get("nope"); found {
		t.Fatal("expected key to not be found")
	}
}

func TestGetExpiredKey(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("k", "v", time.Now().Add(-time.Second))

	if _, found := c.Get("k"); found {
		t.Fatal("expected expired key to not be found")
	}
}

func TestPermanentTTL(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("permanent", "v", time.Time{})

	item, found := c.Get("permanent")
	if !found {
		t.Fatal("expected permanent key to be found")
	}
	if item.Expired() {
		t.Fatal("permanent key (zero Expiration) should never be considered expired")
	}
}

func TestDeleteAndExists(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("k", "v", time.Now().Add(time.Minute))
	if !c.Exists("k") {
		t.Fatal("expected key to exist after Set")
	}

	c.Delete("k")
	if c.Exists("k") {
		t.Fatal("expected key to not exist after Delete")
	}
}

func TestFlush(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("a", "1", time.Now().Add(time.Minute))
	c.Set("b", "2", time.Now().Add(time.Minute))

	c.Flush()

	items := c.AllItems()
	if len(items) != 0 {
		t.Fatalf("expected 0 items after Flush, got %d", len(items))
	}
	if c.approxBytes != 0 {
		t.Fatalf("expected approxBytes to be 0 after Flush, got %d", c.approxBytes)
	}
}

func TestAllItemsSortedAndExcludesExpired(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("c", "3", time.Now().Add(time.Minute))
	c.Set("a", "1", time.Now().Add(time.Minute))
	c.Set("b", "2", time.Now().Add(time.Minute))
	c.Set("expired", "x", time.Now().Add(-time.Second))

	items := c.AllItems()
	if len(items) != 3 {
		t.Fatalf("expected 3 non-expired items, got %d", len(items))
	}
	for i, want := range []string{"a", "b", "c"} {
		if items[i].Key != want {
			t.Fatalf("expected sorted order, item %d = %q, want %q", i, items[i].Key, want)
		}
	}
}

func TestIncrementDecrement(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("counter", "10", time.Now().Add(time.Minute))

	item, found, err := c.Increment("counter", 1)
	if !found || err != nil {
		t.Fatalf("unexpected result: found=%v err=%v", found, err)
	}
	if item.Value != "11" {
		t.Fatalf("expected 11, got %s", item.Value)
	}

	item, found, err = c.Increment("counter", -5)
	if !found || err != nil {
		t.Fatalf("unexpected result: found=%v err=%v", found, err)
	}
	if item.Value != "6" {
		t.Fatalf("expected 6, got %s", item.Value)
	}
}

func TestIncrementMissingKey(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	_, found, err := c.Increment("nope", 1)
	if found || err != nil {
		t.Fatalf("expected not found and no error, got found=%v err=%v", found, err)
	}
}

func TestIncrementNonIntegerValue(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("k", "not-a-number", time.Now().Add(time.Minute))
	_, found, err := c.Increment("k", 1)
	if !found || err == nil {
		t.Fatalf("expected found=true and an error, got found=%v err=%v", found, err)
	}
}

func TestIncrementWrongType(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	ttl := time.Now().Add(time.Minute)
	if _, err := c.LPush("k", "v", &ttl); err != nil {
		t.Fatalf("LPush failed: %v", err)
	}

	_, found, err := c.Increment("k", 1)
	if !found || err != ErrWrongType {
		t.Fatalf("expected ErrWrongType, got found=%v err=%v", found, err)
	}
}

func TestIncrementFloat(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("price", "9.5", time.Now().Add(time.Minute))

	item, found, err := c.IncrementFloat("price", 0.25)
	if !found || err != nil {
		t.Fatalf("unexpected result: found=%v err=%v", found, err)
	}
	if item.Value != "9.75" {
		t.Fatalf("expected 9.75, got %s", item.Value)
	}

	item, _, _ = c.IncrementFloat("price", -1.75)
	if item.Value != "8" {
		t.Fatalf("expected 8, got %s", item.Value)
	}
}

func TestSetValueTooLarge(t *testing.T) {
	c := NewCache(CacheConfig{MaxValueBytes: 5})
	defer c.StopAutoCleanup()

	if err := c.Set("k", "0123456789", time.Now().Add(time.Minute)); err != ErrValueTooLarge {
		t.Fatalf("expected ErrValueTooLarge, got %v", err)
	}
	if err := c.Set("k", "ok", time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("expected value within limit to succeed, got %v", err)
	}
}

func TestConcurrentIncrementIsAtomic(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("counter", "0", time.Now().Add(time.Minute))

	const goroutines = 50
	const incrementsEach = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsEach; j++ {
				if _, _, err := c.Increment("counter", 1); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	item, found := c.Get("counter")
	if !found {
		t.Fatal("expected counter to exist")
	}
	want := "1000"
	if item.Value != want {
		t.Fatalf("lost updates under concurrency: got %s, want %s", item.Value, want)
	}
}

func TestConcurrentStatsCounters(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	const goroutines = 50
	const opsEach = 20

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsEach; j++ {
				c.IncrementHits()
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < opsEach; j++ {
				c.IncrementMisses()
			}
		}()
	}
	wg.Wait()

	stats := c.GetStats()
	want := goroutines * opsEach
	if stats.Hits != want {
		t.Errorf("hits: got %d, want %d", stats.Hits, want)
	}
	if stats.Misses != want {
		t.Errorf("misses: got %d, want %d", stats.Misses, want)
	}
}

func TestConcurrentGetSetDifferentKeys(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := string(rune('a' + n%26))
			c.Set(key, "v", time.Now().Add(time.Minute))
			c.Get(key)
			c.Exists(key)
		}(i)
	}
	wg.Wait()
}

func TestCleanupExpiredItems(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("live", "v", time.Now().Add(time.Minute))
	c.Set("dead", "v", time.Now().Add(-time.Second))

	c.cleanupExpiredItems()

	c.mu.RLock()
	_, deadStillInMap := c.data["dead"]
	_, liveStillInMap := c.data["live"]
	c.mu.RUnlock()

	if deadStillInMap {
		t.Fatal("expected expired key to be removed from the underlying map by cleanup")
	}
	if !liveStillInMap {
		t.Fatal("expected live key to survive cleanup")
	}
	if c.approxBytes <= 0 {
		t.Fatalf("expected approxBytes to still reflect the live key, got %d", c.approxBytes)
	}
}

func TestCleanupIntervalConfigured(t *testing.T) {
	c := NewCache(CacheConfig{CleanupInterval: 5 * time.Millisecond})
	defer c.StopAutoCleanup()

	c.Set("dead", "v", time.Now().Add(-time.Second))
	time.Sleep(30 * time.Millisecond)

	if c.Exists("dead") {
		t.Fatal("expected background sweep to have removed the expired key")
	}
}

func TestHealthy(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	if !c.Healthy(time.Second) {
		t.Fatal("expected cache to report healthy")
	}
}
