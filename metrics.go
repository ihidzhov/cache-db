package main

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"sync"
)

var latencyBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type pathStatusKey struct {
	Path   string
	Status string
}

type Metrics struct {
	mu            sync.Mutex
	requestsTotal map[pathStatusKey]int64
	durationSum   map[string]float64
	durationCount map[string]int64
	bucketCounts  map[string][]int64
}

func NewMetrics() *Metrics {
	return &Metrics{
		requestsTotal: make(map[pathStatusKey]int64),
		durationSum:   make(map[string]float64),
		durationCount: make(map[string]int64),
		bucketCounts:  make(map[string][]int64),
	}
}

func (m *Metrics) Observe(path, status string, seconds float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestsTotal[pathStatusKey{Path: path, Status: status}]++
	m.durationSum[path] += seconds
	m.durationCount[path]++

	counts, ok := m.bucketCounts[path]
	if !ok {
		counts = make([]int64, len(latencyBuckets))
		m.bucketCounts[path] = counts
	}
	for i, le := range latencyBuckets {
		if seconds <= le {
			counts[i]++
		}
	}
}

func (m *Metrics) WriteText(w io.Writer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fmt.Fprintln(w, "# HELP cache_requests_total Total HTTP requests handled, by path and status code.")
	fmt.Fprintln(w, "# TYPE cache_requests_total counter")
	reqKeys := make([]pathStatusKey, 0, len(m.requestsTotal))
	for k := range m.requestsTotal {
		reqKeys = append(reqKeys, k)
	}
	sort.Slice(reqKeys, func(i, j int) bool {
		if reqKeys[i].Path != reqKeys[j].Path {
			return reqKeys[i].Path < reqKeys[j].Path
		}
		return reqKeys[i].Status < reqKeys[j].Status
	})
	for _, k := range reqKeys {
		fmt.Fprintf(w, "cache_requests_total{path=%q,status=%q} %d\n", k.Path, k.Status, m.requestsTotal[k])
	}

	fmt.Fprintln(w, "# HELP cache_request_duration_seconds Request latency in seconds, by path.")
	fmt.Fprintln(w, "# TYPE cache_request_duration_seconds histogram")
	paths := make([]string, 0, len(m.durationCount))
	for p := range m.durationCount {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		counts := m.bucketCounts[p]
		var cumulative int64
		for i, le := range latencyBuckets {
			cumulative += counts[i]
			fmt.Fprintf(w, "cache_request_duration_seconds_bucket{path=%q,le=%q} %d\n", p, formatFloat(le), cumulative)
		}
		fmt.Fprintf(w, "cache_request_duration_seconds_bucket{path=%q,le=\"+Inf\"} %d\n", p, m.durationCount[p])
		fmt.Fprintf(w, "cache_request_duration_seconds_sum{path=%q} %s\n", p, formatFloat(m.durationSum[p]))
		fmt.Fprintf(w, "cache_request_duration_seconds_count{path=%q} %d\n", p, m.durationCount[p])
	}
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
