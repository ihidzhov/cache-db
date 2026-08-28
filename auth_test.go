package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidAPIKeyHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/get?key=k", nil)
	req.Header.Set("X-API-Key", "secret")
	if !validAPIKey(req, "secret") {
		t.Fatal("expected valid X-API-Key to be accepted")
	}
}

func TestValidAPIKeyBearer(t *testing.T) {
	req := httptest.NewRequest("GET", "/get?key=k", nil)
	req.Header.Set("Authorization", "Bearer secret")
	if !validAPIKey(req, "secret") {
		t.Fatal("expected valid Bearer token to be accepted")
	}
}

func TestValidAPIKeyRejectsWrongOrMissing(t *testing.T) {
	req := httptest.NewRequest("GET", "/get?key=k", nil)
	if validAPIKey(req, "secret") {
		t.Fatal("expected missing key to be rejected")
	}

	req.Header.Set("X-API-Key", "wrong")
	if validAPIKey(req, "secret") {
		t.Fatal("expected wrong key to be rejected")
	}
}

func TestAuthMiddlewareDisabledWhenKeyEmpty(t *testing.T) {
	handler := authMiddleware("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/get", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when auth disabled, got %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsWithoutKey(t *testing.T) {
	handler := authMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/get", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareExemptsProbes(t *testing.T) {
	handler := authMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/health", "/ready", "/dashboard"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected %s to be exempt from auth, got %d", path, rec.Code)
		}
	}
}
