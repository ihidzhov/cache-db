package main

import (
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	rate     float64
	burst    float64
	buckets  map[string]*bucket
	stopChan chan struct{}
}

func NewRateLimiter(ratePerSecond, burst float64) *RateLimiter {
	rl := &RateLimiter{
		rate:     ratePerSecond,
		burst:    burst,
		buckets:  make(map[string]*bucket),
		stopChan: make(chan struct{}),
	}
	go rl.startCleanup(5 * time.Minute)
	return rl
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &bucket{tokens: rl.burst - 1, lastSeen: now}
		return true
	}

	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens = math.Min(rl.burst, b.tokens+elapsed*rl.rate)
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (rl *RateLimiter) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopChan:
			return
		}
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for key, b := range rl.buckets {
		if b.lastSeen.Before(cutoff) {
			delete(rl.buckets, key)
		}
	}
}

func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}

func rateLimitMiddleware(limiter *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limiter == nil || isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if !limiter.Allow(rateLimitKey(r)) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rateLimitKey(r *http.Request) string {
	if key := r.Header.Get("X-API-Key"); key != "" {
		return "key:" + key
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return "key:" + strings.TrimPrefix(auth, "Bearer ")
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "ip:" + r.RemoteAddr
	}
	return "ip:" + host
}
