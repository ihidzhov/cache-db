package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

var startTime time.Time

func main() {
	startTime = time.Now()

	cache := NewCache()
	defer cache.StopAutoCleanup()

	server := &CacheHandlers{cache: cache}

	http.HandleFunc("/set", server.SetHandler)
	http.HandleFunc("/get", server.GetHandler)
	http.HandleFunc("/delete", server.DeleteHandler)
	http.HandleFunc("/stats", server.StatsHandler)
	http.HandleFunc("/increment", server.IncrementHandler)
	http.HandleFunc("/decrement", server.DecrementHandler)

	fmt.Println("HTTP cache server with TTL running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
