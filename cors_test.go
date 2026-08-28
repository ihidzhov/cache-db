package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorsMiddlewareDisabledByDefault(t *testing.T) {
	handler := corsMiddleware("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/get", nil))
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("expected no CORS headers when disabled")
	}
}

func TestCorsMiddlewareSetsHeaders(t *testing.T) {
	handler := corsMiddleware("https://example.com", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/get", nil))
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("expected origin header, got %q", got)
	}
}

func TestCorsMiddlewareHandlesPreflight(t *testing.T) {
	called := false
	handler := corsMiddleware("*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("OPTIONS", "/set", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", rec.Code)
	}
	if called {
		t.Fatal("expected preflight to short-circuit before reaching the next handler")
	}
}
