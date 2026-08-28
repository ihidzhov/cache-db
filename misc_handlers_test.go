package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestStatsHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/set?key=k&value=v&ttl=60")
	doRequest(mux, "GET", "/get?key=k")
	doRequest(mux, "GET", "/get?key=missing")

	rec := doRequest(mux, "GET", "/stats")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"hits":1`) || !strings.Contains(body, `"misses":1`) {
		t.Fatalf("unexpected stats body: %s", body)
	}
}

func TestKeysHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/set?key=a&value=v&ttl=60")
	doRequest(mux, "POST", "/set?key=b&value=v&ttl=60")

	rec := doRequest(mux, "GET", "/keys?limit=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"total":2`) {
		t.Fatalf("expected total:2 in body, got %s", body)
	}
	if !strings.Contains(body, `"expires_at"`) || !strings.Contains(body, `"size_bytes"`) {
		t.Fatalf("expected expires_at/size_bytes fields in body, got %s", body)
	}
}

func TestDecrementHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/set?key=k&value=10&ttl=60")
	rec := doRequest(mux, "PUT", "/decrement?key=k")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	rec = doRequest(mux, "GET", "/get?key=k")
	if rec.Body.String() != "9" {
		t.Fatalf("expected 9, got %q", rec.Body.String())
	}
}

func TestIncrByFloatHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/set?key=price&value=9.5&ttl=60")
	rec := doRequest(mux, "PUT", "/incrbyfloat?key=price&value=0.5")
	if rec.Code != http.StatusOK || rec.Body.String() != "10" {
		t.Fatalf("expected 200/10, got %d/%q", rec.Code, rec.Body.String())
	}

	rec = doRequest(mux, "PUT", "/incrbyfloat?key=price&value=notanumber")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid amount, got %d", rec.Code)
	}
}

func TestHealthHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "GET", "/health")
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("expected 200/ok, got %d/%q", rec.Code, rec.Body.String())
	}
}

func TestReadyHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "GET", "/ready")
	if rec.Code != http.StatusOK || rec.Body.String() != "ready" {
		t.Fatalf("expected 200/ready, got %d/%q", rec.Code, rec.Body.String())
	}
}

func TestMetricsHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/set?key=k&value=v&ttl=60")

	rec := doRequest(mux, "GET", "/metrics")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"cache_keys", "cache_hits_total", "cache_approx_bytes", "cache_requests_total"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected metrics body to contain %q, got:\n%s", want, body)
		}
	}
}

func TestWrongMethodReturns405(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "GET", "/set?key=k&value=v&ttl=60")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
