package main

import (
	"net/http"
	"strconv"
	"strings"
)

type Search struct{}

type SearchParams struct {
	Query      string
	GT         string
	LT         string
	EQ         string
	Contains   string
	EndsWith   string
	StartsWith string
	Limit      int
	Offset     int
	Filters    string
	Sort       string
}

type SearchResult struct {
	Results []CacheItem
	Meta    map[string]interface{}
}

func (s *Search) PrepareParams(r *http.Request) SearchParams {
	var seachParams SearchParams
	seachParams.Query = r.URL.Query().Get("query")
	seachParams.GT = r.URL.Query().Get("gt")
	seachParams.LT = r.URL.Query().Get("lt")
	seachParams.EQ = r.URL.Query().Get("eq")
	seachParams.Contains = r.URL.Query().Get("contains")
	seachParams.EndsWith = r.URL.Query().Get("endswith")
	seachParams.StartsWith = r.URL.Query().Get("startswith")
	seachParams.Filters = r.URL.Query().Get("filters")

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		limit = 10
	}
	seachParams.Limit = limit

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}
	seachParams.Offset = offset

	sort := r.URL.Query().Get("sort")
	if sort != "desc" && sort != "asc" {
		sort = "asc"
	}
	seachParams.Sort = sort

	return seachParams
}

func (s *Search) Find(sp SearchParams, c *Cache) []CacheItem {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []CacheItem
	for _, item := range c.data {
		if strings.Contains(strings.ToLower(item.Value), strings.ToLower(sp.Query)) {
			results = append(results, item)
		}
	}

	return results
}
