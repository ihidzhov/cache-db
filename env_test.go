package main

import "testing"

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_STR", "")
	if got := envOrDefault("TEST_STR", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}

	t.Setenv("TEST_STR", "value")
	if got := envOrDefault("TEST_STR", "fallback"); got != "value" {
		t.Fatalf("expected value, got %q", got)
	}
}

func TestEnvOrDefaultInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	if got := envOrDefaultInt("TEST_INT", 0); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}

	t.Setenv("TEST_INT", "not-a-number")
	if got := envOrDefaultInt("TEST_INT", 7); got != 7 {
		t.Fatalf("expected fallback 7 for invalid int, got %d", got)
	}
}

func TestEnvOrDefaultInt64(t *testing.T) {
	t.Setenv("TEST_INT64", "9999999999")
	if got := envOrDefaultInt64("TEST_INT64", 0); got != 9999999999 {
		t.Fatalf("expected 9999999999, got %d", got)
	}
}

func TestEnvOrDefaultFloat(t *testing.T) {
	t.Setenv("TEST_FLOAT", "1.5")
	if got := envOrDefaultFloat("TEST_FLOAT", 0); got != 1.5 {
		t.Fatalf("expected 1.5, got %v", got)
	}

	t.Setenv("TEST_FLOAT", "")
	if got := envOrDefaultFloat("TEST_FLOAT", 2.5); got != 2.5 {
		t.Fatalf("expected fallback 2.5, got %v", got)
	}
}
