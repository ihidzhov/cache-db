package cachedb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

type Stats struct {
	TotalKeys   int    `json:"total_keys"`
	ApproxBytes int64  `json:"approx_bytes"`
	Hits        int    `json:"hits"`
	Misses      int    `json:"misses"`
	Evictions   int    `json:"evictions"`
	Uptime      string `json:"uptime"`
}

type Config struct {
	Addr            string  `json:"addr"`
	CleanupInterval int     `json:"cleanup_interval_seconds"`
	MaxKeys         int     `json:"max_keys"`
	MaxValueBytes   int     `json:"max_value_bytes"`
	MaxMemoryBytes  int64   `json:"max_memory_bytes"`
	EvictionPolicy  string  `json:"eviction_policy"`
	RateLimit       float64 `json:"rate_limit"`
	RateLimitBurst  int     `json:"rate_limit_burst"`
	CorsOrigin      string  `json:"cors_origin"`
	AuthEnabled     bool    `json:"auth_enabled"`
}

type KeyInfo struct {
	Key       string  `json:"key"`
	Type      string  `json:"type"`
	ExpiresAt *string `json:"expires_at"`
	SizeBytes int64   `json:"size_bytes"`
}

type KeysPage struct {
	Keys []KeyInfo `json:"keys"`
	Meta struct {
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

func (c *Client) decodeJSON(ctx context.Context, method, path string, params url.Values, out interface{}) error {
	resp, err := c.do(ctx, method, path, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) Stats(ctx context.Context) (*Stats, error) {
	var out Stats
	if err := c.decodeJSON(ctx, http.MethodGet, "/v1/stats", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Config(ctx context.Context) (*Config, error) {
	var out Config
	if err := c.decodeJSON(ctx, http.MethodGet, "/v1/config", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Keys(ctx context.Context, limit, offset int) (*KeysPage, error) {
	params := url.Values{"limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)}}
	var out KeysPage
	if err := c.decodeJSON(ctx, http.MethodGet, "/v1/keys", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Flush(ctx context.Context) error {
	return c.call(ctx, http.MethodDelete, "/v1/flush", nil)
}

func (c *Client) Seed(ctx context.Context) (int, error) {
	var out struct {
		Writes int `json:"writes"`
	}
	if err := c.decodeJSON(ctx, http.MethodPost, "/v1/seed", nil, &out); err != nil {
		return 0, err
	}
	return out.Writes, nil
}

func (c *Client) Health(ctx context.Context) error {
	return c.call(ctx, http.MethodGet, "/health", nil)
}

func (c *Client) Ready(ctx context.Context) error {
	return c.call(ctx, http.MethodGet, "/ready", nil)
}

func (c *Client) RawMetrics(ctx context.Context) (string, error) {
	return c.readString(ctx, http.MethodGet, "/metrics", nil)
}
