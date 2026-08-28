package main

import (
	"fmt"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Search struct{}

type SearchParams struct {
	Query         string
	GT            string
	LT            string
	EQ            string
	Contains      string
	EndsWith      string
	StartsWith    string
	KeyStartsWith string
	KeyPattern    string
	Regex         *regexp.Regexp
	Limit         int
	Offset        int
	Sort          string
}

type SearchResult struct {
	Results []CacheItem            `json:"results"`
	Meta    map[string]interface{} `json:"meta"`
}

func (s *Search) PrepareParams(r *http.Request) (SearchParams, error) {
	var sp SearchParams
	sp.Query = r.URL.Query().Get("query")
	sp.GT = r.URL.Query().Get("gt")
	sp.LT = r.URL.Query().Get("lt")
	sp.EQ = r.URL.Query().Get("eq")
	sp.Contains = r.URL.Query().Get("contains")
	sp.EndsWith = r.URL.Query().Get("endswith")
	sp.StartsWith = r.URL.Query().Get("startswith")
	sp.KeyStartsWith = r.URL.Query().Get("keystartswith")
	sp.KeyPattern = r.URL.Query().Get("keypattern")

	if raw := r.URL.Query().Get("regex"); raw != "" {
		re, err := regexp.Compile(raw)
		if err != nil {
			return SearchParams{}, fmt.Errorf("invalid regex: %w", err)
		}
		sp.Regex = re
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		limit = 10
	}
	sp.Limit = limit

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}
	sp.Offset = offset

	sortOrder := r.URL.Query().Get("sort")
	if sortOrder != "desc" && sortOrder != "asc" {
		sortOrder = "asc"
	}
	sp.Sort = sortOrder

	return sp, nil
}

func (s *Search) matches(sp SearchParams, key, value string) bool {
	if sp.Query != "" && !strings.Contains(strings.ToLower(value), strings.ToLower(sp.Query)) {
		return false
	}
	if sp.Contains != "" && !strings.Contains(strings.ToLower(value), strings.ToLower(sp.Contains)) {
		return false
	}
	if sp.StartsWith != "" && !strings.HasPrefix(value, sp.StartsWith) {
		return false
	}
	if sp.EndsWith != "" && !strings.HasSuffix(value, sp.EndsWith) {
		return false
	}
	if sp.GT != "" && !numericCompare(value, sp.GT, func(a, b float64) bool { return a > b }) {
		return false
	}
	if sp.LT != "" && !numericCompare(value, sp.LT, func(a, b float64) bool { return a < b }) {
		return false
	}
	if sp.EQ != "" && !numericCompare(value, sp.EQ, func(a, b float64) bool { return a == b }) {
		return false
	}
	if sp.Regex != nil && !sp.Regex.MatchString(value) {
		return false
	}
	if sp.KeyStartsWith != "" && !strings.HasPrefix(key, sp.KeyStartsWith) {
		return false
	}
	if sp.KeyPattern != "" {
		matched, err := path.Match(sp.KeyPattern, key)
		if err != nil || !matched {
			return false
		}
	}
	return true
}

func numericCompare(value, threshold string, cmp func(a, b float64) bool) bool {
	valueNum, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return false
	}
	thresholdNum, err := strconv.ParseFloat(threshold, 64)
	if err != nil {
		return false
	}
	return cmp(valueNum, thresholdNum)
}

func (s *Search) Find(sp SearchParams, c *Cache) []CacheItem {
	c.mu.RLock()
	matched := make([]CacheItem, 0, len(c.data))
	for _, item := range c.data {
		if item.Expired() || item.Type != TypeString {
			continue
		}
		if s.matches(sp, item.Key, item.Value) {
			matched = append(matched, item)
		}
	}
	c.mu.RUnlock()

	sort.Slice(matched, func(i, j int) bool {
		if sp.Sort == "desc" {
			return matched[i].Value > matched[j].Value
		}
		return matched[i].Value < matched[j].Value
	})

	return matched
}
