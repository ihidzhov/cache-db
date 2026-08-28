package main

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func isPublicPath(path string) bool {
	return path == "/health" || path == "/ready" || path == "/dashboard"
}

func authMiddleware(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" || isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if !validAPIKey(r, apiKey) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validAPIKey(r *http.Request, apiKey string) bool {
	if key := r.Header.Get("X-API-Key"); key != "" {
		return constantTimeEqual(key, apiKey)
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return constantTimeEqual(strings.TrimPrefix(auth, "Bearer "), apiKey)
	}
	return false
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
