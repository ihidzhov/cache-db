package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", "http://localhost:8080", "cache-db base URL")
	apiKey := flag.String("api-key", "", "API key, if the target server has -api-key set")
	flush := flag.Bool("flush", false, "flush the cache before seeding")
	repeat := flag.Duration("repeat", 0, "if set, re-seed on this interval instead of running once (e.g. 30s)")
	flag.Parse()

	base := strings.TrimRight(*addr, "/")
	client := &http.Client{Timeout: 10 * time.Second}

	call := func(method, path string) {
		req, err := http.NewRequest(method, base+path, nil)
		if err != nil {
			log.Fatal(err)
		}
		if *apiKey != "" {
			req.Header.Set("X-API-Key", *apiKey)
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			log.Fatalf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(body)))
		}
		fmt.Printf("%s %s -> %s: %s", method, path, resp.Status, body)
	}

	run := func() {
		if *flush {
			call("DELETE", "/flush")
		}
		call("POST", "/seed")
	}

	run()

	if *repeat > 0 {
		for range time.Tick(*repeat) {
			run()
		}
	}
}
