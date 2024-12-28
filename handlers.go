package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
	ttlFormated := time.Now().Add(time.Duration(int(ttl.Seconds())) * time.Second)
	s.cache.Set(key, value, ttlFormated)
	w.WriteHeader(http.StatusOK)
}

func (s *CacheHandlers) GetHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	output := r.URL.Query().Get("output")

	s.cache.stats.IncrementHits()

	item, found := s.cache.Get(key)
	if !found {
		s.cache.stats.IncrementMisses()
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	if output == "json" {
		w.Header().Set("Content-Type", "application/json")
		var jsonMap map[string]interface{}
		json.Unmarshal([]byte(item.Value), &jsonMap)
		json.NewEncoder(w).Encode(jsonMap)
	} else {
		w.Header().Set("Content-Type", "plain/text")
		json.NewEncoder(w).Encode(item.Value)
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

func (s *CacheHandlers) IncrementHandler(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	s.cache.stats.IncrementHits()

	item, found := s.cache.Get(key)
	fmt.Println(item, found)
	if !found {
		s.cache.stats.IncrementMisses()
		http.Error(w, "Key not found", http.StatusBadRequest)
		return
	}

	intValue, err := StringToInt(item.Value)
	if err != nil {
		http.Error(w, "Something wrong with the value", http.StatusBadRequest)
		return
	}
	intValue = intValue + 1
	item.Value = strconv.Itoa(intValue)
	s.cache.Set(key, item.Value, item.Expiration)

	w.WriteHeader(http.StatusOK)
}
