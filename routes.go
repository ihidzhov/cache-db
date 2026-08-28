package main

import "net/http"

func newMux(server *CacheHandlers) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", server.HealthHandler)
	mux.HandleFunc("/ready", server.ReadyHandler)
	mux.HandleFunc("/metrics", server.MetricsHandler)
	mux.HandleFunc("/dashboard", server.DashboardHandler)

	apiRoutes := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/config", server.ConfigHandler},
		{"/seed", server.SeedHandler},
		{"/set", server.SetHandler},
		{"/get", server.GetHandler},
		{"/search", server.SearchHandler},
		{"/delete", server.DeleteHandler},
		{"/stats", server.StatsHandler},
		{"/increment", server.IncrementHandler},
		{"/decrement", server.DecrementHandler},
		{"/incrbyfloat", server.IncrByFloatHandler},
		{"/keys", server.KeysHandler},
		{"/exists", server.ExistsHandler},
		{"/flush", server.FlushHandler},
		{"/lpush", server.LPushHandler},
		{"/rpush", server.RPushHandler},
		{"/lpop", server.LPopHandler},
		{"/rpop", server.RPopHandler},
		{"/lrange", server.LRangeHandler},
		{"/llen", server.LLenHandler},
		{"/hset", server.HSetHandler},
		{"/hget", server.HGetHandler},
		{"/hgetall", server.HGetAllHandler},
		{"/hdel", server.HDelHandler},
	}

	for _, rt := range apiRoutes {
		mux.HandleFunc(rt.path, rt.handler)
		mux.HandleFunc("/v1"+rt.path, rt.handler)
	}

	return mux
}
