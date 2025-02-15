package main

import (
	"net/http"
	"strconv"
)

func StringToInt(str string) (int, error) {
	num, err := strconv.Atoi(str)
	if err != nil {
		return 0, err
	}

	return num, nil
}

func IncrementDecrementValue(s *CacheHandlers, w http.ResponseWriter, r *http.Request, key string, what int) {
	s.cache.stats.IncrementHits()

	item, found := s.cache.Get(key)

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
	intValue = intValue + what
	item.Value = strconv.Itoa(intValue)
	s.cache.Set(key, item.Value, item.Expiration)

	w.WriteHeader(http.StatusOK)
}
