package main

import (
	"fmt"
	"time"
)

type CacheStats struct {
	TotalKeys   int    `json:"total_keys"`
	ApproxBytes int64  `json:"approx_bytes"`
	Hits        int    `json:"hits"`
	Misses      int    `json:"misses"`
	Evictions   int    `json:"evictions"`
	Uptime      string `json:"uptime"`
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
