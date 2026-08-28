package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestConfigHandler(t *testing.T) {
	s := newTestServer()
	s.config = ServerConfig{
		Addr:           ":8080",
		MaxKeys:        100,
		EvictionPolicy: "lru",
		AuthEnabled:    true,
	}
	mux := newMux(s)

	rec := doRequest(mux, "GET", "/config")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var got ServerConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.MaxKeys != 100 || got.EvictionPolicy != "lru" || !got.AuthEnabled {
		t.Fatalf("unexpected config: %+v", got)
	}
}

func TestConfigHandlerWrongMethod(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "POST", "/config")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestConfigVersionedAlias(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "GET", "/v1/config")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected /v1/config to work, got %d", rec.Code)
	}
}
