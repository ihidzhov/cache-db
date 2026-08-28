package cachedb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) Set(ctx context.Context, key, value string, ttlSeconds int) error {
	params := url.Values{"key": {key}, "value": {value}, "ttl": {strconv.Itoa(ttlSeconds)}}
	return c.call(ctx, http.MethodPost, "/v1/set", params)
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.readString(ctx, http.MethodGet, "/v1/get", url.Values{"key": {key}})
}

func (c *Client) GetJSON(ctx context.Context, key string, out interface{}) error {
	resp, err := c.do(ctx, http.MethodGet, "/v1/get", url.Values{"key": {key}, "output": {"json"}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) Delete(ctx context.Context, key string) error {
	return c.call(ctx, http.MethodDelete, "/v1/delete", url.Values{"key": {key}})
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	resp, err := c.do(ctx, http.MethodHead, "/v1/exists", url.Values{"key": {key}})
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	resp.Body.Close()
	return true, nil
}

func (c *Client) Increment(ctx context.Context, key string) error {
	return c.call(ctx, http.MethodPut, "/v1/increment", url.Values{"key": {key}})
}

func (c *Client) Decrement(ctx context.Context, key string) error {
	return c.call(ctx, http.MethodPut, "/v1/decrement", url.Values{"key": {key}})
}

func (c *Client) IncrByFloat(ctx context.Context, key string, delta float64) (string, error) {
	params := url.Values{"key": {key}, "value": {strconv.FormatFloat(delta, 'f', -1, 64)}}
	return c.readString(ctx, http.MethodPut, "/v1/incrbyfloat", params)
}
