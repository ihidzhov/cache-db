package main

import (
	"net/http"
	"strconv"
	"time"
)

func parseTTLSeconds(raw string) (seconds int, provided bool, valid bool) {
	if raw == "" {
		return 0, false, true
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < -1 {
		return 0, true, false
	}
	return seconds, true, true
}

func expirationFromSeconds(seconds int) time.Time {
	if seconds <= 0 {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(seconds) * time.Second)
}

func IncrementDecrementValue(s *CacheHandlers, w http.ResponseWriter, r *http.Request, key string, delta int) {
	_, found, err := s.cache.Increment(key, delta)

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

	w.WriteHeader(http.StatusOK)
}
