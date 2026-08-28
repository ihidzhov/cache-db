package main

import (
	"encoding/json"
	"net/http"
)

type ServerConfig struct {
	Addr            string  `json:"addr"`
	CleanupInterval int     `json:"cleanup_interval_seconds"`
	MaxKeys         int     `json:"max_keys"`
	MaxValueBytes   int     `json:"max_value_bytes"`
	MaxMemoryBytes  int64   `json:"max_memory_bytes"`
	EvictionPolicy  string  `json:"eviction_policy"`
	RateLimit       float64 `json:"rate_limit"`
	RateLimitBurst  int     `json:"rate_limit_burst"`
	CorsOrigin      string  `json:"cors_origin"`
	AuthEnabled     bool    `json:"auth_enabled"`
}

func (s *CacheHandlers) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.config)
}
