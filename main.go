package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var startTime time.Time

func main() {
	startTime = time.Now()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	addr := flag.String("addr", envOrDefault("CACHE_ADDR", ":8080"), "address to listen on (env CACHE_ADDR)")
	cleanupInterval := flag.Int("cleanup-interval", envOrDefaultInt("CACHE_CLEANUP_INTERVAL", 20), "seconds between background sweeps for expired keys (env CACHE_CLEANUP_INTERVAL)")
	maxKeys := flag.Int("max-keys", envOrDefaultInt("CACHE_MAX_KEYS", 0), "maximum number of keys before eviction kicks in, 0 = unlimited (env CACHE_MAX_KEYS)")
	maxValueBytes := flag.Int("max-value-bytes", envOrDefaultInt("CACHE_MAX_VALUE_BYTES", 0), "maximum size in bytes for a single stored value, 0 = unlimited (env CACHE_MAX_VALUE_BYTES)")
	maxMemoryBytes := flag.Int64("max-memory-bytes", envOrDefaultInt64("CACHE_MAX_MEMORY_BYTES", 0), "approximate maximum total size in bytes of all stored keys/values before eviction kicks in, 0 = unlimited (env CACHE_MAX_MEMORY_BYTES)")
	evictionPolicy := flag.String("eviction-policy", envOrDefault("CACHE_EVICTION_POLICY", "lru"), "eviction policy once max-keys or max-memory-bytes is reached: lru, lfu, or random (env CACHE_EVICTION_POLICY)")
	apiKey := flag.String("api-key", envOrDefault("CACHE_API_KEY", ""), "require this API key (X-API-Key or Authorization: Bearer header) on every request except /health and /ready, empty = auth disabled (env CACHE_API_KEY)")
	rateLimit := flag.Float64("rate-limit", envOrDefaultFloat("CACHE_RATE_LIMIT", 0), "max requests per second per client, identified by API key or IP, 0 = disabled (env CACHE_RATE_LIMIT)")
	rateLimitBurst := flag.Int("rate-limit-burst", envOrDefaultInt("CACHE_RATE_LIMIT_BURST", 10), "burst size for -rate-limit (env CACHE_RATE_LIMIT_BURST)")
	corsOrigin := flag.String("cors-origin", envOrDefault("CACHE_CORS_ORIGIN", ""), "value for Access-Control-Allow-Origin, e.g. * or https://example.com, empty = CORS disabled (env CACHE_CORS_ORIGIN)")
	flag.Parse()

	var policy EvictionPolicy
	switch *evictionPolicy {
	case "lru":
		policy = EvictionLRU
	case "lfu":
		policy = EvictionLFU
	case "random":
		policy = EvictionRandom
	default:
		log.Fatalf("invalid -eviction-policy %q: must be lru, lfu, or random", *evictionPolicy)
	}

	cache := NewCache(CacheConfig{
		MaxKeys:         *maxKeys,
		MaxValueBytes:   *maxValueBytes,
		MaxTotalBytes:   *maxMemoryBytes,
		EvictionPolicy:  policy,
		CleanupInterval: time.Duration(*cleanupInterval) * time.Second,
	})
	defer cache.StopAutoCleanup()

	var limiter *RateLimiter
	if *rateLimit > 0 {
		limiter = NewRateLimiter(*rateLimit, float64(*rateLimitBurst))
		defer limiter.Stop()
	}

	server := &CacheHandlers{
		cache:   cache,
		metrics: NewMetrics(),
		config: ServerConfig{
			Addr:            *addr,
			CleanupInterval: *cleanupInterval,
			MaxKeys:         *maxKeys,
			MaxValueBytes:   *maxValueBytes,
			MaxMemoryBytes:  *maxMemoryBytes,
			EvictionPolicy:  string(policy),
			RateLimit:       *rateLimit,
			RateLimitBurst:  *rateLimitBurst,
			CorsOrigin:      *corsOrigin,
			AuthEnabled:     *apiKey != "",
		},
	}
	mux := newMux(server)

	var handler http.Handler = mux
	handler = authMiddleware(*apiKey, handler)
	handler = rateLimitMiddleware(limiter, handler)
	handler = corsMiddleware(*corsOrigin, handler)
	handler = observabilityMiddleware(server.metrics, logger, handler)

	httpServer := &http.Server{
		Addr:    *addr,
		Handler: handler,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", *addr)
		serverErr <- httpServer.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	case <-stop:
		logger.Info("shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	}
}
