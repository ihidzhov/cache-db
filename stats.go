package main

import (
	"fmt"
	"time"
)

type CacheStats struct {
	TotalKeys int    `json:"total_keys"`
	Hits      int    `json:"hits"`
	Misses    int    `json:"misses"`
	Uptime    string `json:"uptime"`
}

func (s *CacheStats) IncrementHits() {
	s.Hits = s.Hits + 1
}

func (s *CacheStats) IncrementMisses() {
	s.Misses = s.Misses + 1
}

func getUptimeFormatted() string {
	uptime := time.Since(startTime)

	days := uptime / (24 * time.Hour)
	uptime -= days * (24 * time.Hour)

	hours := uptime / time.Hour
	uptime -= hours * time.Hour

	minutes := uptime / time.Minute
	uptime -= minutes * time.Minute

	seconds := uptime / time.Second

	return fmt.Sprintf("%d days, %02d:%02d:%02d", days, hours, minutes, seconds)
}
