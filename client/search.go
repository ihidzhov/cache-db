package cachedb

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type SearchOptions struct {
	Query         string
	Contains      string
	StartsWith    string
	EndsWith      string
	Regex         string
	GT            string
	LT            string
	EQ            string
	KeyStartsWith string
	KeyPattern    string
	Sort          string
	Limit         int
	Offset        int
}

type SearchItem struct {
	Key        string    `json:"Key"`
	Type       string    `json:"Type"`
	Value      string    `json:"Value"`
	Expiration time.Time `json:"Expiration"`
}

type SearchResult struct {
	Results []SearchItem           `json:"results"`
	Meta    map[string]interface{} `json:"meta"`
}

func (c *Client) Search(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	params := url.Values{}
	set := func(key, value string) {
		if value != "" {
			params.Set(key, value)
		}
	}
	set("query", opts.Query)
	set("contains", opts.Contains)
	set("startswith", opts.StartsWith)
	set("endswith", opts.EndsWith)
	set("regex", opts.Regex)
	set("gt", opts.GT)
	set("lt", opts.LT)
	set("eq", opts.EQ)
	set("keystartswith", opts.KeyStartsWith)
	set("keypattern", opts.KeyPattern)
	set("sort", opts.Sort)
	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		params.Set("offset", strconv.Itoa(opts.Offset))
	}

	var out SearchResult
	if err := c.decodeJSON(ctx, http.MethodGet, "/v1/search", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
