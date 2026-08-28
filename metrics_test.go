package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestMetricsObserveAndWriteText(t *testing.T) {
	m := NewMetrics()
	m.Observe("/get", "200", 0.001)
	m.Observe("/get", "404", 0.2)
	m.Observe("/get", "200", 0.001)

	var buf bytes.Buffer
	m.WriteText(&buf)
	out := buf.String()

	for _, want := range []string{
		`cache_requests_total{path="/get",status="200"} 2`,
		`cache_requests_total{path="/get",status="404"} 1`,
		`cache_request_duration_seconds_count{path="/get"} 3`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestFormatFloat(t *testing.T) {
	if got := formatFloat(0.005); got != "0.005" {
		t.Fatalf("expected 0.005, got %s", got)
	}
	if got := formatFloat(1); got != "1" {
		t.Fatalf("expected 1, got %s", got)
	}
}
