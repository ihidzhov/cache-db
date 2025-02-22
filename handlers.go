package main

import (
	"encoding/json"
	"fmt"
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
	ttlFormated := time.Now().Add(time.Duration(int(ttl.Seconds())) * time.Second)
	s.cache.Set(key, value, ttlFormated)
	w.WriteHeader(http.StatusOK)
}

func (s *CacheHandlers) GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

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

func (s *CacheHandlers) SearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	//	output := r.URL.Query().Get("output")

	seach := Search{}
	searchParams := seach.PrepareParams(r)
	fmt.Println(searchParams)
	results := seach.Find(searchParams, s.cache)
	fmt.Println(results)

	searchResult := SearchResult{
		Results: results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResult)

	// "results": results,
	//        "meta": map[string]interface{}{
	//            "total":  len(results),
	//            "limit":  limit,
	//            "offset": offset,
	//            "query":  query,
	//            "sort":   sort,
	//        }

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
