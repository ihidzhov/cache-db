package main

import (
	_ "embed"
	"net/http"
)

//go:embed web/dashboard.html
var dashboardHTML []byte

func (s *CacheHandlers) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(dashboardHTML)
}
