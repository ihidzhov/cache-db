package main

import (
	"fmt"
	"net/http"
	"time"
)

func (s *CacheHandlers) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	stats := s.cache.GetStats()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	fmt.Fprintln(w, "# HELP cache_keys Current number of keys in the cache.")
	fmt.Fprintln(w, "# TYPE cache_keys gauge")
	fmt.Fprintf(w, "cache_keys %d\n", stats.TotalKeys)

	fmt.Fprintln(w, "# HELP cache_approx_bytes Approximate total size in bytes of all stored keys and values.")
	fmt.Fprintln(w, "# TYPE cache_approx_bytes gauge")
	fmt.Fprintf(w, "cache_approx_bytes %d\n", stats.ApproxBytes)

	fmt.Fprintln(w, "# HELP cache_hits_total Total cache hits.")
	fmt.Fprintln(w, "# TYPE cache_hits_total counter")
	fmt.Fprintf(w, "cache_hits_total %d\n", stats.Hits)

	fmt.Fprintln(w, "# HELP cache_misses_total Total cache misses.")
	fmt.Fprintln(w, "# TYPE cache_misses_total counter")
	fmt.Fprintf(w, "cache_misses_total %d\n", stats.Misses)

	fmt.Fprintln(w, "# HELP cache_evictions_total Total cache evictions.")
	fmt.Fprintln(w, "# TYPE cache_evictions_total counter")
	fmt.Fprintf(w, "cache_evictions_total %d\n", stats.Evictions)

	fmt.Fprintln(w, "# HELP cache_uptime_seconds Server uptime in seconds.")
	fmt.Fprintln(w, "# TYPE cache_uptime_seconds counter")
	fmt.Fprintf(w, "cache_uptime_seconds %d\n", int(time.Since(startTime).Seconds()))

	s.metrics.WriteText(w)
}
