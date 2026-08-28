package main

import (
	"testing"
	"time"
)

func TestGetUptimeFormatted(t *testing.T) {
	original := startTime
	defer func() { startTime = original }()

	startTime = time.Now().Add(-(25*time.Hour + 3*time.Minute + 4*time.Second))

	got := getUptimeFormatted()
	want := "1 days, 01:03:"
	if len(got) < len(want) || got[:len(want)] != want {
		t.Fatalf("unexpected uptime format: %q, want prefix %q", got, want)
	}
}
