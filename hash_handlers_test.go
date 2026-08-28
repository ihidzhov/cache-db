package main

import (
	"net/http"
	"testing"
)

func TestHSetHGetHandlers(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "POST", "/hset?key=h&field=a&value=1&ttl=60")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(mux, "GET", "/hget?key=h&field=a")
	if rec.Code != http.StatusOK || rec.Body.String() != "1" {
		t.Fatalf("expected 200/1, got %d/%q", rec.Code, rec.Body.String())
	}

	rec = doRequest(mux, "GET", "/hget?key=h&field=missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing field, got %d", rec.Code)
	}
}

func TestHGetAllHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/hset?key=h&field=a&value=1&ttl=60")
	doRequest(mux, "POST", "/hset?key=h&field=b&value=2")

	rec := doRequest(mux, "GET", "/hgetall?key=h")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != `{"a":"1","b":"2"}`+"\n" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestHDelHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/hset?key=h&field=a&value=1&ttl=60")

	rec := doRequest(mux, "DELETE", "/hdel?key=h&field=a")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	rec = doRequest(mux, "DELETE", "/hdel?key=h&field=a")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for already-deleted field, got %d", rec.Code)
	}
}

func TestHashHandlersWrongType(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/set?key=k&value=v&ttl=60")

	rec := doRequest(mux, "POST", "/hset?key=k&field=a&value=1")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}
