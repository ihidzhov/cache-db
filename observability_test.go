package main

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestObservabilityMiddlewareSetsRequestIDAndLogs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	metrics := NewMetrics()

	handler := observabilityMiddleware(metrics, logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/get", nil))

	if rec.Code != http.StatusTeapot {
		t.Fatalf("expected status to pass through, got %d", rec.Code)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("expected X-Request-Id header to be set")
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, `"status":418`) {
		t.Fatalf("expected logged status 418, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, rec.Header().Get("X-Request-Id")) {
		t.Fatalf("expected log to contain the same request id as the response header, got: %s", logOutput)
	}
}

func TestObservabilityMiddlewareDefaultsStatusOK(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	metrics := NewMetrics()

	handler := observabilityMiddleware(metrics, logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/get", nil))

	if !strings.Contains(buf.String(), `"status":200`) {
		t.Fatalf("expected implicit 200 to be logged, got: %s", buf.String())
	}
}

func TestNewRequestIDIsUnique(t *testing.T) {
	a := newRequestID()
	b := newRequestID()
	if a == b {
		t.Fatal("expected two generated request IDs to differ")
	}
	if len(a) == 0 {
		t.Fatal("expected non-empty request id")
	}
}
