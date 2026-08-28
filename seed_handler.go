package main

import (
	"encoding/json"
	"net/http"
)

func (s *CacheHandlers) SeedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	writes := seedDemoData(s.cache)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"writes": writes})
}
