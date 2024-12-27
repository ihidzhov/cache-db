package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type CacheHandlers struct {
	cache *Cache
}

func (s *CacheHandlers) SetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.FormValue("key")
	value := r.FormValue("value")

	ttl, _ := time.ParseDuration(r.FormValue("ttl") + "s")
	if key == "" || value == "" || ttl <= 0 {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	s.cache.Set(key, value, int(ttl.Seconds()))
	w.WriteHeader(http.StatusOK)
}

func (s *CacheHandlers) GetHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	output := r.URL.Query().Get("output")

	s.cache.stats.IncrementHits()

	value, found := s.cache.Get(key)
	if !found {
		s.cache.stats.IncrementMisses()
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	if output == "json" {
		w.Header().Set("Content-Type", "application/json")
		var jsonMap map[string]interface{}
		json.Unmarshal([]byte(value), &jsonMap)
		json.NewEncoder(w).Encode(jsonMap)
	} else {
		w.Header().Set("Content-Type", "plain/text")
		json.NewEncoder(w).Encode(value)
	}
}

func (s *CacheHandlers) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	s.cache.Delete(key)
	w.WriteHeader(http.StatusOK)
}

func (s *CacheHandlers) StatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := s.cache.GetStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}