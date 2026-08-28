package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func newTestServer() *CacheHandlers {
	return &CacheHandlers{cache: newTestCache(), metrics: NewMetrics()}
}

func doRequest(mux http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestSetHandlerValid(t *testing.T) {
	s := newTestServer()
	rec := doRequest(newMux(s), "POST", "/set?key=k&value=v&ttl=60")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSetHandlerMissingParams(t *testing.T) {
	s := newTestServer()
	rec := doRequest(newMux(s), "POST", "/set?key=k&ttl=60")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing value, got %d", rec.Code)
	}
}

func TestSetHandlerPermanentTTL(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "POST", "/set?key=k&value=v&ttl=0")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for ttl=0, got %d", rec.Code)
	}

	rec = doRequest(mux, "POST", "/set?key=k2&value=v&ttl=-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for ttl=-1, got %d", rec.Code)
	}

	rec = doRequest(mux, "POST", "/set?key=k3&value=v&ttl=-2")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for ttl=-2, got %d", rec.Code)
	}
}

func TestGetHandlerFlow(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "GET", "/get?key=missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing key, got %d", rec.Code)
	}

	doRequest(mux, "POST", "/set?key=k&value=hello&ttl=60")
	rec = doRequest(mux, "GET", "/get?key=k")
	if rec.Code != http.StatusOK || rec.Body.String() != "hello" {
		t.Fatalf("expected 200/hello, got %d/%q", rec.Code, rec.Body.String())
	}
}

func TestGetHandlerJSONOutput(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/set?key=k&value="+`{"a":1}`+"&ttl=60")
	rec := doRequest(mux, "GET", "/get?key=k&output=json")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	doRequest(mux, "POST", "/set?key=bad&value=not-json&ttl=60")
	rec = doRequest(mux, "GET", "/get?key=bad&output=json")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid JSON, got %d", rec.Code)
	}
}

func TestGetHandlerWrongType(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/lpush?key=k&value=v&ttl=60")
	rec := doRequest(mux, "GET", "/get?key=k")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for wrong type, got %d", rec.Code)
	}
}

func TestDeleteHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/set?key=k&value=v&ttl=60")
	rec := doRequest(mux, "DELETE", "/delete?key=k")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	rec = doRequest(mux, "GET", "/get?key=k")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected key to be gone after delete, got %d", rec.Code)
	}
}

func TestExistsHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/set?key=k&value=v&ttl=60")

	rec := doRequest(mux, "HEAD", "/exists?key=k")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	rec = doRequest(mux, "HEAD", "/exists?key=missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestFlushHandler(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/set?key=k&value=v&ttl=60")
	rec := doRequest(mux, "DELETE", "/flush")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	rec = doRequest(mux, "GET", "/get?key=k")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected cache to be empty after flush, got %d", rec.Code)
	}
}

func TestSearchHandlerHasMore(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	for _, k := range []string{"a", "b", "c"} {
		doRequest(mux, "POST", "/set?key="+k+"&value=v&ttl=60")
	}

	rec := doRequest(mux, "GET", "/search?limit=2")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"has_more":true`) {
		t.Fatalf(`expected "has_more":true in body, got %s`, body)
	}
}

func TestIncrementHandlerHTTPConcurrent(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/set?key=counter&value=0&ttl=60")

	const goroutines = 30
	const perGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				rec := doRequest(mux, "PUT", "/increment?key=counter")
				if rec.Code != http.StatusOK {
					t.Errorf("unexpected status %d", rec.Code)
				}
			}
		}()
	}
	wg.Wait()

	rec := doRequest(mux, "GET", "/get?key=counter")
	want := strconv.Itoa(goroutines * perGoroutine)
	if rec.Body.String() != want {
		t.Fatalf("expected %s after concurrent increments, got %s", want, rec.Body.String())
	}
}

func TestVersionedRoutesAlias(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	doRequest(mux, "POST", "/v1/set?key=k&value=v&ttl=60")
	rec := doRequest(mux, "GET", "/get?key=k")
	if rec.Code != http.StatusOK || rec.Body.String() != "v" {
		t.Fatalf("expected /v1/set and /get to share state, got %d/%q", rec.Code, rec.Body.String())
	}

	rec = doRequest(mux, "GET", "/v1/get?key=k")
	if rec.Code != http.StatusOK || rec.Body.String() != "v" {
		t.Fatalf("expected /v1/get to work, got %d/%q", rec.Code, rec.Body.String())
	}
}

func TestHealthReadyNotVersioned(t *testing.T) {
	s := newTestServer()
	mux := newMux(s)

	rec := doRequest(mux, "GET", "/v1/health")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected /v1/health to not exist, got %d", rec.Code)
	}
}
