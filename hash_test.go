package main

import (
	"testing"
	"time"
)

func TestHSetHGet(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	ttl := time.Now().Add(time.Minute)
	if _, err := c.HSet("h", "a", "1", &ttl); err != nil {
		t.Fatalf("HSet failed: %v", err)
	}
	if _, err := c.HSet("h", "b", "2", nil); err != nil {
		t.Fatalf("HSet with nil ttl on existing key failed: %v", err)
	}

	v, found, err := c.HGet("h", "a")
	if !found || err != nil || v != "1" {
		t.Fatalf("HGet: got v=%q found=%v err=%v", v, found, err)
	}

	if _, found, _ := c.HGet("h", "missing-field"); found {
		t.Fatal("expected missing field to report not found")
	}
}

func TestHashRequiresTTLOnCreate(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	if _, err := c.HSet("h", "a", "1", nil); err != ErrTTLRequired {
		t.Fatalf("expected ErrTTLRequired, got %v", err)
	}
}

func TestHGetAll(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	ttl := time.Now().Add(time.Minute)
	c.HSet("h", "a", "1", &ttl)
	c.HSet("h", "b", "2", nil)

	fields, err := c.HGetAll("h")
	if err != nil {
		t.Fatalf("HGetAll failed: %v", err)
	}
	if len(fields) != 2 || fields["a"] != "1" || fields["b"] != "2" {
		t.Fatalf("unexpected fields: %v", fields)
	}

	fields, err = c.HGetAll("nope")
	if err != nil || len(fields) != 0 {
		t.Fatalf("expected empty map for missing key, got %v (err=%v)", fields, err)
	}
}

func TestHDel(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	ttl := time.Now().Add(time.Minute)
	c.HSet("h", "a", "1", &ttl)

	deleted, err := c.HDel("h", "a")
	if !deleted || err != nil {
		t.Fatalf("expected field to be deleted, got deleted=%v err=%v", deleted, err)
	}

	deleted, err = c.HDel("h", "a")
	if deleted || err != nil {
		t.Fatalf("expected second delete to report not-found, got deleted=%v err=%v", deleted, err)
	}
}

func TestHashWrongType(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	c.Set("k", "v", time.Now().Add(time.Minute))

	if _, err := c.HSet("k", "a", "1", nil); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from HSet, got %v", err)
	}
	if _, _, err := c.HGet("k", "a"); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from HGet, got %v", err)
	}
	if _, err := c.HGetAll("k"); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from HGetAll, got %v", err)
	}
	if _, err := c.HDel("k", "a"); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from HDel, got %v", err)
	}
}
