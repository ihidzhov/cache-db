package main

import (
	"sync"
	"testing"
	"time"
)

func TestLPushRPushOrder(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	ttl := time.Now().Add(time.Minute)
	if _, err := c.RPush("l", "b", &ttl); err != nil {
		t.Fatalf("RPush failed: %v", err)
	}
	if _, err := c.RPush("l", "c", nil); err != nil {
		t.Fatalf("RPush failed: %v", err)
	}
	if _, err := c.LPush("l", "a", nil); err != nil {
		t.Fatalf("LPush failed: %v", err)
	}

	values, err := c.LRange("l", 0, -1)
	if err != nil {
		t.Fatalf("LRange failed: %v", err)
	}
	want := []string{"a", "b", "c"}
	if len(values) != len(want) {
		t.Fatalf("got %v, want %v", values, want)
	}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("got %v, want %v", values, want)
		}
	}
}

func TestListRequiresTTLOnCreate(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	if _, err := c.LPush("l", "a", nil); err != ErrTTLRequired {
		t.Fatalf("expected ErrTTLRequired, got %v", err)
	}
}

func TestListTTLOptionalOnUpdate(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	ttl := time.Now().Add(time.Minute)
	if _, err := c.LPush("l", "a", &ttl); err != nil {
		t.Fatalf("LPush failed: %v", err)
	}
	if _, err := c.LPush("l", "b", nil); err != nil {
		t.Fatalf("expected ttl to be optional on existing key, got %v", err)
	}
}

func TestLPopRPop(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	ttl := time.Now().Add(time.Minute)
	c.RPush("l", "a", &ttl)
	c.RPush("l", "b", nil)
	c.RPush("l", "c", nil)

	v, found, err := c.LPop("l")
	if !found || err != nil || v != "a" {
		t.Fatalf("LPop: got v=%q found=%v err=%v", v, found, err)
	}

	v, found, err = c.RPop("l")
	if !found || err != nil || v != "c" {
		t.Fatalf("RPop: got v=%q found=%v err=%v", v, found, err)
	}

	length, err := c.LLen("l")
	if err != nil || length != 1 {
		t.Fatalf("expected length 1, got %d (err=%v)", length, err)
	}
}

func TestPopEmptyOrMissingList(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	if _, found, err := c.LPop("nope"); found || err != nil {
		t.Fatalf("expected not found, no error; got found=%v err=%v", found, err)
	}

	ttl := time.Now().Add(time.Minute)
	c.RPush("l", "a", &ttl)
	c.LPop("l")
	if _, found, err := c.LPop("l"); found || err != nil {
		t.Fatalf("expected pop on empty list to report not found; got found=%v err=%v", found, err)
	}
}

func TestLRangeNegativeIndices(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	ttl := time.Now().Add(time.Minute)
	for i, v := range []string{"a", "b", "c", "d", "e"} {
		if i == 0 {
			c.RPush("l", v, &ttl)
		} else {
			c.RPush("l", v, nil)
		}
	}

	values, _ := c.LRange("l", -2, -1)
	if len(values) != 2 || values[0] != "d" || values[1] != "e" {
		t.Fatalf("expected [d e], got %v", values)
	}

	values, _ = c.LRange("l", 0, -1)
	if len(values) != 5 {
		t.Fatalf("expected all 5 elements, got %v", values)
	}

	values, _ = c.LRange("l", 10, 20)
	if len(values) != 0 {
		t.Fatalf("expected empty slice for out-of-range indices, got %v", values)
	}
}

func TestListWrongType(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("k", "v", time.Now().Add(time.Minute))

	if _, err := c.LPush("k", "x", nil); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from LPush, got %v", err)
	}
	if _, _, err := c.LPop("k"); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from LPop, got %v", err)
	}
	if _, err := c.LRange("k", 0, -1); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from LRange, got %v", err)
	}
	if _, err := c.LLen("k"); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from LLen, got %v", err)
	}
}

func TestConcurrentPushPop(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	ttl := time.Now().Add(time.Minute)
	c.RPush("l", "seed", &ttl)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.RPush("l", "x", nil)
		}()
		go func() {
			defer wg.Done()
			c.LPop("l")
		}()
	}
	wg.Wait()
}
