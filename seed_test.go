package main

import (
	"net/http"
	"testing"
)

func TestSeedDemoDataPopulatesAllTypes(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	writes := seedDemoData(c)
	if writes == 0 {
		t.Fatal("expected seedDemoData to report a non-zero write count")
	}

	items := c.AllItems()
	types := map[ItemType]int{}
	for _, item := range items {
		types[item.Type]++
	}
	if types[TypeString] == 0 || types[TypeList] == 0 || types[TypeHash] == 0 {
		t.Fatalf("expected all three types to be seeded, got %v", types)
	}

	if !c.Exists("user:1") {
		t.Fatal("expected user:1 to be seeded")
	}
	length, _ := c.LLen("queue:jobs")
	if length == 0 {
		t.Fatal("expected queue:jobs to be seeded with entries")
	}
	fields, _ := c.HGetAll("settings:app")
	if len(fields) == 0 {
		t.Fatal("expected settings:app hash to be seeded")
	}
}

func TestSeedDemoDataIsRepeatable(t *testing.T) {
	c := newTestCache()
	defer c.StopAutoCleanup()

	seedDemoData(c)
	before, _ := c.LLen("queue:jobs")

	seedDemoData(c)
	after, _ := c.LLen("queue:jobs")

	if before != after {
		t.Fatalf("expected queue:jobs length to stay stable across reseeds, got %d then %d", before, after)
	}
}

func TestSeedHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "POST", "/seed")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(mux, "GET", "/keys?limit=200")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSeedHandlerWrongMethod(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "GET", "/seed")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestSeedVersionedAlias(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "POST", "/v1/seed")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected /v1/seed to work, got %d", rec.Code)
	}
}
