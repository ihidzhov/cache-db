package cachedb_test

import (
	"context"
	"testing"

	cachedb "github.com/ihidzhov/cache-db/client"
)

func newClient(t *testing.T) *cachedb.Client {
	t.Helper()
	c := cachedb.New(baseURL)
	if err := c.Flush(context.Background()); err != nil {
		t.Fatalf("failed to flush before test: %v", err)
	}
	return c
}

func TestSetGetDelete(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	if err := c.Set(ctx, "k", "v", 60); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, err := c.Get(ctx, "k")
	if err != nil || value != "v" {
		t.Fatalf("Get: value=%q err=%v", value, err)
	}

	ok, err := c.Exists(ctx, "k")
	if err != nil || !ok {
		t.Fatalf("Exists: ok=%v err=%v", ok, err)
	}

	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	ok, err = c.Exists(ctx, "k")
	if err != nil || ok {
		t.Fatalf("expected key gone after delete: ok=%v err=%v", ok, err)
	}
}

func TestGetMissingKeyReturnsAPIError(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	_, err := c.Get(ctx, "missing")
	if err == nil {
		t.Fatal("expected an error for a missing key")
	}
	if !cachedb.IsNotFound(err) {
		t.Fatalf("expected IsNotFound(err) to be true, got %v", err)
	}
	if code, ok := cachedb.StatusCode(err); !ok || code != 404 {
		t.Fatalf("expected status 404, got %d (ok=%v)", code, ok)
	}
}

func TestPermanentTTL(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	if err := c.Set(ctx, "k", "v", cachedb.NeverExpire); err != nil {
		t.Fatalf("Set with NeverExpire failed: %v", err)
	}
	value, err := c.Get(ctx, "k")
	if err != nil || value != "v" {
		t.Fatalf("Get: value=%q err=%v", value, err)
	}
}

func TestIncrementDecrement(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	c.Set(ctx, "counter", "10", 60)
	if err := c.Increment(ctx, "counter"); err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if err := c.Decrement(ctx, "counter"); err != nil {
		t.Fatalf("Decrement failed: %v", err)
	}
	value, _ := c.Get(ctx, "counter")
	if value != "10" {
		t.Fatalf("expected 10, got %s", value)
	}
}

func TestIncrByFloat(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	c.Set(ctx, "price", "9.5", 60)
	value, err := c.IncrByFloat(ctx, "price", 0.5)
	if err != nil || value != "10" {
		t.Fatalf("IncrByFloat: value=%q err=%v", value, err)
	}
}

func TestListOperations(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	ttl := 60
	length, err := c.RPush(ctx, "l", "a", &ttl)
	if err != nil || length != 1 {
		t.Fatalf("RPush: length=%d err=%v", length, err)
	}
	length, err = c.RPush(ctx, "l", "b", nil)
	if err != nil || length != 2 {
		t.Fatalf("RPush: length=%d err=%v", length, err)
	}
	length, err = c.LPush(ctx, "l", "z", nil)
	if err != nil || length != 3 {
		t.Fatalf("LPush: length=%d err=%v", length, err)
	}

	values, err := c.LRange(ctx, "l", 0, -1)
	if err != nil {
		t.Fatalf("LRange failed: %v", err)
	}
	want := []string{"z", "a", "b"}
	if len(values) != len(want) {
		t.Fatalf("got %v, want %v", values, want)
	}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("got %v, want %v", values, want)
		}
	}

	n, err := c.LLen(ctx, "l")
	if err != nil || n != 3 {
		t.Fatalf("LLen: n=%d err=%v", n, err)
	}

	popped, err := c.LPop(ctx, "l")
	if err != nil || popped != "z" {
		t.Fatalf("LPop: popped=%q err=%v", popped, err)
	}
	popped, err = c.RPop(ctx, "l")
	if err != nil || popped != "b" {
		t.Fatalf("RPop: popped=%q err=%v", popped, err)
	}
}

func TestHashOperations(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	ttl := 60
	if err := c.HSet(ctx, "h", "a", "1", &ttl); err != nil {
		t.Fatalf("HSet failed: %v", err)
	}
	if err := c.HSet(ctx, "h", "b", "2", nil); err != nil {
		t.Fatalf("HSet failed: %v", err)
	}

	value, err := c.HGet(ctx, "h", "a")
	if err != nil || value != "1" {
		t.Fatalf("HGet: value=%q err=%v", value, err)
	}

	fields, err := c.HGetAll(ctx, "h")
	if err != nil || len(fields) != 2 {
		t.Fatalf("HGetAll: fields=%v err=%v", fields, err)
	}

	if err := c.HDel(ctx, "h", "a"); err != nil {
		t.Fatalf("HDel failed: %v", err)
	}
	if _, err := c.HGet(ctx, "h", "a"); !cachedb.IsNotFound(err) {
		t.Fatalf("expected not-found after HDel, got %v", err)
	}
}

func TestSearch(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	c.Set(ctx, "user:1", "alice", 60)
	c.Set(ctx, "user:2", "bob", 60)

	result, err := c.Search(ctx, cachedb.SearchOptions{KeyStartsWith: "user:"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
}

func TestStatsConfigKeysFlush(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	c.Set(ctx, "a", "1", 60)
	c.Set(ctx, "b", "2", 60)

	stats, err := c.Stats(ctx)
	if err != nil || stats.TotalKeys != 2 {
		t.Fatalf("Stats: %+v err=%v", stats, err)
	}

	cfg, err := c.Config(ctx)
	if err != nil {
		t.Fatalf("Config failed: %v", err)
	}
	if cfg.EvictionPolicy == "" {
		t.Fatal("expected a non-empty eviction policy")
	}

	page, err := c.Keys(ctx, 10, 0)
	if err != nil || page.Meta.Total != 2 {
		t.Fatalf("Keys: %+v err=%v", page, err)
	}

	if err := c.Flush(ctx); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
	stats, _ = c.Stats(ctx)
	if stats.TotalKeys != 0 {
		t.Fatalf("expected 0 keys after flush, got %d", stats.TotalKeys)
	}
}

func TestSeedAndHealthReady(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	writes, err := c.Seed(ctx)
	if err != nil || writes == 0 {
		t.Fatalf("Seed: writes=%d err=%v", writes, err)
	}

	if err := c.Health(ctx); err != nil {
		t.Fatalf("Health failed: %v", err)
	}
	if err := c.Ready(ctx); err != nil {
		t.Fatalf("Ready failed: %v", err)
	}
}

func TestAPIKeyAuth(t *testing.T) {
	addr, stop, err := launchServer(serverBinPath, "-api-key=secret")
	if err != nil {
		t.Fatalf("failed to start authed server: %v", err)
	}
	defer stop()
	ctx := context.Background()

	unauthenticated := cachedb.New(addr)
	if err := unauthenticated.Set(ctx, "k", "v", 60); err == nil {
		t.Fatal("expected an error without an API key")
	} else if code, ok := cachedb.StatusCode(err); !ok || code != 401 {
		t.Fatalf("expected 401, got %d (ok=%v)", code, ok)
	}

	authenticated := cachedb.New(addr, cachedb.WithAPIKey("secret"))
	if err := authenticated.Set(ctx, "k", "v", 60); err != nil {
		t.Fatalf("expected authenticated request to succeed, got %v", err)
	}
}
