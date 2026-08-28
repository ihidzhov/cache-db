package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimiterAllowsBurstThenBlocks(t *testing.T) {
	rl := NewRateLimiter(1, 3)
	defer rl.Stop()

	for i := 0; i < 3; i++ {
		if !rl.Allow("client") {
			t.Fatalf("expected request %d within burst to be allowed", i)
		}
	}
	if rl.Allow("client") {
		t.Fatal("expected request beyond burst to be blocked")
	}
}

func TestRateLimiterRefillsOverTime(t *testing.T) {
	rl := NewRateLimiter(100, 1)
	defer rl.Stop()

	if !rl.Allow("client") {
		t.Fatal("expected first request to be allowed")
	}
	if rl.Allow("client") {
		t.Fatal("expected second immediate request to be blocked")
	}

	time.Sleep(20 * time.Millisecond)

	if !rl.Allow("client") {
		t.Fatal("expected request to be allowed after refill")
	}
}

func TestRateLimiterPerClientIsolation(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	defer rl.Stop()

	if !rl.Allow("a") {
		t.Fatal("expected client a's first request to be allowed")
	}
	if !rl.Allow("b") {
		t.Fatal("expected client b to have its own independent bucket")
	}
	if rl.Allow("a") {
		t.Fatal("expected client a's second immediate request to be blocked")
	}
}

func TestRateLimiterCleanupRemovesStaleBuckets(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	defer rl.Stop()

	rl.Allow("stale")

	rl.mu.Lock()
	rl.buckets["stale"].lastSeen = time.Now().Add(-time.Hour)
	rl.mu.Unlock()

	rl.cleanup()

	rl.mu.Lock()
	_, stillThere := rl.buckets["stale"]
	rl.mu.Unlock()

	if stillThere {
		t.Fatal("expected stale bucket to be removed by cleanup")
	}
}

func TestRateLimitMiddlewareBlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	defer rl.Stop()

	handler := rateLimitMiddleware(rl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/get", nil)
	req.RemoteAddr = "1.2.3.4:5555"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected first request to pass, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on second request, got %d", rec.Code)
	}
}

func TestRateLimitMiddlewareExemptsProbes(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	defer rl.Stop()
	rl.Allow("ip:1.2.3.4")

	handler := rateLimitMiddleware(rl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	req.RemoteAddr = "1.2.3.4:5555"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected /health to be exempt from rate limiting, got %d", rec.Code)
	}
}

func TestRateLimitKeyPrefersAPIKeyOverIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/get", nil)
	req.RemoteAddr = "1.2.3.4:5555"
	req.Header.Set("X-API-Key", "secret")

	if got := rateLimitKey(req); got != "key:secret" {
		t.Fatalf("expected key:secret, got %q", got)
	}

	req2 := httptest.NewRequest("GET", "/get", nil)
	req2.RemoteAddr = "1.2.3.4:5555"
	if got := rateLimitKey(req2); got != "ip:1.2.3.4" {
		t.Fatalf("expected ip:1.2.3.4, got %q", got)
	}
}

func TestRateLimiterConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(1000, 1000)
	defer rl.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.Allow("shared")
		}()
	}
	wg.Wait()
}
