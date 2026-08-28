package main

import (
	"net/http"
	"testing"
)

func TestRPushLPopRPopLRangeLLenHandlers(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "POST", "/rpush?key=l&value=a&ttl=60")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	doRequest(mux, "POST", "/rpush?key=l&value=b")
	doRequest(mux, "POST", "/rpush?key=l&value=c")

	rec = doRequest(mux, "GET", "/llen?key=l")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"length":3}`+"\n" {
		t.Fatalf("unexpected llen response: %d %q", rec.Code, rec.Body.String())
	}

	rec = doRequest(mux, "GET", "/lrange?key=l&start=0&stop=-1")
	if rec.Code != http.StatusOK || rec.Body.String() != `["a","b","c"]`+"\n" {
		t.Fatalf("unexpected lrange response: %d %q", rec.Code, rec.Body.String())
	}

	rec = doRequest(mux, "PUT", "/lpop?key=l")
	if rec.Code != http.StatusOK || rec.Body.String() != "a" {
		t.Fatalf("unexpected lpop response: %d %q", rec.Code, rec.Body.String())
	}

	rec = doRequest(mux, "PUT", "/rpop?key=l")
	if rec.Code != http.StatusOK || rec.Body.String() != "c" {
		t.Fatalf("unexpected rpop response: %d %q", rec.Code, rec.Body.String())
	}
}

func TestLRangeHandlerInvalidParams(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "GET", "/lrange?key=l&start=abc&stop=1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-numeric start, got %d", rec.Code)
	}
}

func TestPopHandlerMissingKey(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "PUT", "/lpop?key=missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestPushHandlerRequiresTTLOnCreate(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "POST", "/lpush?key=l&value=a")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when ttl missing on new key, got %d", rec.Code)
	}
}
