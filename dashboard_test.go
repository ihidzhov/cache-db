package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestDashboardHandlerServesHTML(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "GET", "/dashboard")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "<title>cache-db dashboard</title>") {
		t.Fatal("expected dashboard HTML body to contain the page title")
	}
}

func TestDashboardHandlerWrongMethod(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "POST", "/dashboard")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestDashboardNotVersioned(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "GET", "/v1/dashboard")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected /v1/dashboard to not exist, got %d", rec.Code)
	}
}

func TestDashboardExemptFromAuth(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)
	handler := authMiddleware("secret", mux)

	rec := doRequest(handler, "GET", "/dashboard")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected /dashboard to be exempt from auth, got %d", rec.Code)
	}

	rec = doRequest(handler, "GET", "/config")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected /config to still require auth, got %d", rec.Code)
	}
}
