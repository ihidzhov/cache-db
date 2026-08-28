package main

import (
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func seedSearchCache(t *testing.T) *Cache {
	t.Helper()
	c := newTestCache()
	ttl := time.Now().Add(time.Minute)
	c.Set("user:1", "alice", ttl)
	c.Set("user:2", "bob", ttl)
	c.Set("order:1", "widget", ttl)
	c.Set("counter:a", "5", ttl)
	c.Set("counter:b", "15", ttl)
	c.LPush("mylist", "x", &ttl)
	return c
}

func find(t *testing.T, c *Cache, rawQuery string) []CacheItem {
	t.Helper()
	req := httptest.NewRequest("GET", "/search?"+rawQuery, nil)
	s := Search{}
	sp, err := s.PrepareParams(req)
	if err != nil {
		t.Fatalf("PrepareParams returned error: %v", err)
	}
	return s.Find(sp, c)
}

func TestSearchExcludesNonStringTypes(t *testing.T) {
	c := seedSearchCache(t)
	defer c.StopAutoCleanup()

	results := find(t, c, "")
	for _, r := range results {
		if r.Type != TypeString {
			t.Fatalf("search returned non-string item: %+v", r)
		}
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 string items, got %d", len(results))
	}
}

func TestSearchQueryContains(t *testing.T) {
	c := seedSearchCache(t)
	defer c.StopAutoCleanup()

	results := find(t, c, "query=ali")
	if len(results) != 1 || results[0].Key != "user:1" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestSearchStartsWithEndsWith(t *testing.T) {
	c := seedSearchCache(t)
	defer c.StopAutoCleanup()

	results := find(t, c, "startswith=al")
	if len(results) != 1 || results[0].Key != "user:1" {
		t.Fatalf("unexpected startswith results: %+v", results)
	}

	results = find(t, c, "endswith=dget")
	if len(results) != 1 || results[0].Key != "order:1" {
		t.Fatalf("unexpected endswith results: %+v", results)
	}
}

func TestSearchNumericFilters(t *testing.T) {
	c := seedSearchCache(t)
	defer c.StopAutoCleanup()

	results := find(t, c, "gt=10")
	if len(results) != 1 || results[0].Key != "counter:b" {
		t.Fatalf("unexpected gt results: %+v", results)
	}

	results = find(t, c, "lt=10")
	if len(results) != 1 || results[0].Key != "counter:a" {
		t.Fatalf("unexpected lt results: %+v", results)
	}

	results = find(t, c, "eq=5")
	if len(results) != 1 || results[0].Key != "counter:a" {
		t.Fatalf("unexpected eq results: %+v", results)
	}
}

func TestSearchKeyStartsWith(t *testing.T) {
	c := seedSearchCache(t)
	defer c.StopAutoCleanup()

	results := find(t, c, "keystartswith=user:")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
}

func TestSearchKeyPattern(t *testing.T) {
	c := seedSearchCache(t)
	defer c.StopAutoCleanup()

	results := find(t, c, "keypattern=counter:*")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
}

func TestSearchRegex(t *testing.T) {
	c := seedSearchCache(t)
	defer c.StopAutoCleanup()

	v := url.Values{"regex": {`^\d+$`}}
	results := find(t, c, v.Encode())
	if len(results) != 2 {
		t.Fatalf("expected 2 numeric-looking results, got %d: %+v", len(results), results)
	}
}

func TestSearchInvalidRegexReturnsError(t *testing.T) {
	req := httptest.NewRequest("GET", "/search?regex=(", nil)
	s := Search{}
	if _, err := s.PrepareParams(req); err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestSearchSort(t *testing.T) {
	c := seedSearchCache(t)
	defer c.StopAutoCleanup()

	results := find(t, c, "keystartswith=user:&sort=desc")
	if len(results) != 2 || results[0].Value < results[1].Value {
		t.Fatalf("expected descending order, got %+v", results)
	}
}

func TestSearchDefaultLimitOffset(t *testing.T) {
	req := httptest.NewRequest("GET", "/search", nil)
	s := Search{}
	sp, err := s.PrepareParams(req)
	if err != nil {
		t.Fatalf("PrepareParams error: %v", err)
	}
	if sp.Limit != 10 || sp.Offset != 0 || sp.Sort != "asc" {
		t.Fatalf("unexpected defaults: limit=%d offset=%d sort=%s", sp.Limit, sp.Offset, sp.Sort)
	}
}
