package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func (s *CacheHandlers) push(w http.ResponseWriter, r *http.Request, push func(key, value string, expiration *time.Time) (CacheItem, error)) {
	if r.Method != "POST" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.FormValue("key")
	value := r.FormValue("value")
	if key == "" || value == "" {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	seconds, provided, valid := parseTTLSeconds(r.FormValue("ttl"))
	if !valid {
		http.Error(w, "Invalid ttl", http.StatusBadRequest)
		return
	}
	var expiration *time.Time
	if provided {
		e := expirationFromSeconds(seconds)
		expiration = &e
	}

	item, err := push(key, value, expiration)
	if err == ErrTTLRequired {
		http.Error(w, "ttl is required when creating a new key", http.StatusBadRequest)
		return
	}
	if err == ErrWrongType {
		http.Error(w, "Key holds a different type", http.StatusConflict)
		return
	}
	if err == ErrValueTooLarge {
		http.Error(w, "Value too large", http.StatusRequestEntityTooLarge)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"length": len(item.List)})
}

func (s *CacheHandlers) LPushHandler(w http.ResponseWriter, r *http.Request) {
	s.push(w, r, s.cache.LPush)
}

func (s *CacheHandlers) RPushHandler(w http.ResponseWriter, r *http.Request) {
	s.push(w, r, s.cache.RPush)
}

func (s *CacheHandlers) pop(w http.ResponseWriter, r *http.Request, pop func(key string) (string, bool, error)) {
	if r.Method != "PUT" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")

	value, found, err := pop(key)
	if err == ErrWrongType {
		http.Error(w, "Key holds a different type", http.StatusConflict)
		return
	}
	if !found {
		http.Error(w, "Key not found or list empty", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(value))
}

func (s *CacheHandlers) LPopHandler(w http.ResponseWriter, r *http.Request) {
	s.pop(w, r, s.cache.LPop)
}

func (s *CacheHandlers) RPopHandler(w http.ResponseWriter, r *http.Request) {
	s.pop(w, r, s.cache.RPop)
}

func (s *CacheHandlers) LRangeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")
	start, errStart := strconv.Atoi(r.URL.Query().Get("start"))
	stop, errStop := strconv.Atoi(r.URL.Query().Get("stop"))
	if errStart != nil || errStop != nil {
		http.Error(w, "Invalid start/stop", http.StatusBadRequest)
		return
	}

	values, err := s.cache.LRange(key, start, stop)
	if err == ErrWrongType {
		http.Error(w, "Key holds a different type", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(values)
}

func (s *CacheHandlers) LLenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")

	length, err := s.cache.LLen(key)
	if err == ErrWrongType {
		http.Error(w, "Key holds a different type", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"length": length})
}
