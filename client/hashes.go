package cachedb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) HSet(ctx context.Context, key, field, value string, ttlSeconds *int) error {
	params := url.Values{"key": {key}, "field": {field}, "value": {value}}
	if ttlSeconds != nil {
		params.Set("ttl", strconv.Itoa(*ttlSeconds))
	}
	return c.call(ctx, http.MethodPost, "/v1/hset", params)
}

func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.readString(ctx, http.MethodGet, "/v1/hget", url.Values{"key": {key}, "field": {field}})
}

func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	resp, err := c.do(ctx, http.MethodGet, "/v1/hgetall", url.Values{"key": {key}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out := map[string]string{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) HDel(ctx context.Context, key, field string) error {
	return c.call(ctx, http.MethodDelete, "/v1/hdel", url.Values{"key": {key}, "field": {field}})
}
