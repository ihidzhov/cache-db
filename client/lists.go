package cachedb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) LPush(ctx context.Context, key, value string, ttlSeconds *int) (int, error) {
	return c.push(ctx, "/v1/lpush", key, value, ttlSeconds)
}

func (c *Client) RPush(ctx context.Context, key, value string, ttlSeconds *int) (int, error) {
	return c.push(ctx, "/v1/rpush", key, value, ttlSeconds)
}

func (c *Client) push(ctx context.Context, path, key, value string, ttlSeconds *int) (int, error) {
	params := url.Values{"key": {key}, "value": {value}}
	if ttlSeconds != nil {
		params.Set("ttl", strconv.Itoa(*ttlSeconds))
	}
	resp, err := c.do(ctx, http.MethodPost, path, params)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var out struct {
		Length int `json:"length"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	return out.Length, nil
}

func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	return c.readString(ctx, http.MethodPut, "/v1/lpop", url.Values{"key": {key}})
}

func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	return c.readString(ctx, http.MethodPut, "/v1/rpop", url.Values{"key": {key}})
}

func (c *Client) LRange(ctx context.Context, key string, start, stop int) ([]string, error) {
	params := url.Values{"key": {key}, "start": {strconv.Itoa(start)}, "stop": {strconv.Itoa(stop)}}
	resp, err := c.do(ctx, http.MethodGet, "/v1/lrange", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out []string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) LLen(ctx context.Context, key string) (int, error) {
	resp, err := c.do(ctx, http.MethodGet, "/v1/llen", url.Values{"key": {key}})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var out struct {
		Length int `json:"length"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	return out.Length, nil
}
