package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func (s *CacheHandlers) HSetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.FormValue("key")
	field := r.FormValue("field")
	value := r.FormValue("value")
	if key == "" || field == "" {
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

	_, err := s.cache.HSet(key, field, value, expiration)
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

	w.WriteHeader(http.StatusOK)
}

func (s *CacheHandlers) HGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")
	field := r.URL.Query().Get("field")

	value, found, err := s.cache.HGet(key, field)
	if err == ErrWrongType {
		http.Error(w, "Key holds a different type", http.StatusConflict)
		return
	}
	if !found {
		http.Error(w, "Field not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(value))
}

func (s *CacheHandlers) HGetAllHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")

	fields, err := s.cache.HGetAll(key)
	if err == ErrWrongType {
		http.Error(w, "Key holds a different type", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fields)
}

func (s *CacheHandlers) HDelHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")
	field := r.URL.Query().Get("field")

	deleted, err := s.cache.HDel(key, field)
	if err == ErrWrongType {
		http.Error(w, "Key holds a different type", http.StatusConflict)
		return
	}
	if !deleted {
		http.Error(w, "Field not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}
