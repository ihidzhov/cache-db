package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type CacheHandlers struct {
	cache   *Cache
	metrics *Metrics
	config  ServerConfig
}

func (s *CacheHandlers) SetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.FormValue("key")
	value := r.FormValue("value")

	seconds, provided, valid := parseTTLSeconds(r.FormValue("ttl"))
	if key == "" || value == "" || !provided || !valid {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}
	if err := s.cache.Set(key, value, expirationFromSeconds(seconds)); err == ErrValueTooLarge {
		http.Error(w, "Value too large", http.StatusRequestEntityTooLarge)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *CacheHandlers) GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	output := r.URL.Query().Get("output")

	item, found := s.cache.Get(key)
	if !found {
		s.cache.IncrementMisses()
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}
	s.cache.IncrementHits()

	if item.Type != TypeString {
		http.Error(w, "Key holds a different type, use /lrange, /llen, /hget or /hgetall", http.StatusConflict)
		return
	}

	if output == "json" {
		var jsonMap map[string]interface{}
		if err := json.Unmarshal([]byte(item.Value), &jsonMap); err != nil {
			http.Error(w, "Stored value is not valid JSON", http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jsonMap)
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(item.Value))
	}
}

func (s *CacheHandlers) SearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	search := Search{}
	searchParams, err := search.PrepareParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	matched := search.Find(searchParams, s.cache)

	total := len(matched)
	start := searchParams.Offset
	if start > total {
		start = total
	}
	end := start + searchParams.Limit
	if end > total {
		end = total
	}

	searchResult := SearchResult{
		Results: matched[start:end],
		Meta: map[string]interface{}{
			"total":    total,
			"limit":    searchParams.Limit,
			"offset":   searchParams.Offset,
			"query":    searchParams.Query,
			"sort":     searchParams.Sort,
			"has_more": end < total,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResult)
}

func (s *CacheHandlers) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")
	s.cache.Delete(key)
	w.WriteHeader(http.StatusOK)
}

func (s *CacheHandlers) StatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := s.cache.GetStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *CacheHandlers) IncrementHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.FormValue("key")
	IncrementDecrementValue(s, w, r, key, 1)
}

func (s *CacheHandlers) DecrementHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.FormValue("key")
	IncrementDecrementValue(s, w, r, key, -1)
}

func (s *CacheHandlers) IncrByFloatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.FormValue("key")
	delta, err := strconv.ParseFloat(r.FormValue("value"), 64)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	item, found, err := s.cache.IncrementFloat(key, delta)
	if !found {
		s.cache.IncrementMisses()
		http.Error(w, "Key not found", http.StatusBadRequest)
		return
	}
	if err == ErrWrongType {
		http.Error(w, "Key holds a different type", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "Something wrong with the value", http.StatusBadRequest)
		return
	}
	s.cache.IncrementHits()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(item.Value))
}

func (s *CacheHandlers) KeysHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		limit = 100
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	items := s.cache.AllItems()
	total := len(items)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	keys := make([]map[string]interface{}, 0, end-start)
	for _, item := range items[start:end] {
		var expiresAt interface{}
		if !item.Expiration.IsZero() {
			expiresAt = item.Expiration.Format(time.RFC3339)
		}
		keys = append(keys, map[string]interface{}{
			"key":        item.Key,
			"type":       string(item.Type),
			"expires_at": expiresAt,
			"size_bytes": itemSize(item),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"keys": keys,
		"meta": map[string]interface{}{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (s *CacheHandlers) ExistsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "HEAD" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")
	if !s.cache.Exists(key) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *CacheHandlers) FlushHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	s.cache.Flush()
	w.WriteHeader(http.StatusOK)
}
