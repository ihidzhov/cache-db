# cachedb

A Go client for [cache-db](https://github.com/ihidzhov/cache-db). Import it when you want another Go program to read and write a cache-db server without hand-rolling HTTP requests.

```
go get github.com/ihidzhov/cache-db/client
```

```go
import (
	"context"
	"log"

	cachedb "github.com/ihidzhov/cache-db/client"
)

func main() {
	c := cachedb.New("http://localhost:8080", cachedb.WithAPIKey("changeme"))
	ctx := context.Background()

	if err := c.Set(ctx, "user:1", `{"name":"alice"}`, 3600); err != nil {
		log.Fatal(err)
	}

	value, err := c.Get(ctx, "user:1")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(value)
}
```

## Construction

`New(baseURL string, opts ...Option) *Client` — options:

* `WithAPIKey(key string)` — sent as `X-API-Key` on every request; only needed if the server was started with `-api-key`.
* `WithHTTPClient(*http.Client)` — bring your own transport, proxy, TLS config, etc.
* `WithTimeout(time.Duration)` — shorthand for setting the default client's timeout (default `10s`).

## Method surface

One method per endpoint, all taking `context.Context` as the first argument, all hitting the server's `/v1` routes:

| Category | Methods |
|----------|---------|
| Strings  | `Set`, `Get`, `GetJSON`, `Delete`, `Exists`, `Increment`, `Decrement`, `IncrByFloat` |
| Lists    | `LPush`, `RPush`, `LPop`, `RPop`, `LRange`, `LLen` |
| Hashes   | `HSet`, `HGet`, `HGetAll`, `HDel` |
| Search   | `Search(ctx, SearchOptions)` |
| Admin    | `Keys`, `Stats`, `Config`, `Seed`, `Flush`, `Health`, `Ready`, `RawMetrics` |

See [the main README's API Reference](../README.md#api-reference) for what each endpoint actually does, what `ttl` values mean, and the full set of `SearchOptions` fields — the method names and parameters here mirror it 1:1.

## TTL

`ttlSeconds int` follows the server's rules exactly: a positive number of seconds, or `0` / `cachedb.NeverExpire` (`-1`) for a key that never expires.

For lists and hashes, TTL is a `*int` instead: pass a pointer when creating a new key (required) or to refresh an existing key's expiration, or `nil` to leave the current expiration untouched on an update.

## Errors

A non-2xx response comes back as `*cachedb.APIError{StatusCode, Status, Body}`. Two helpers cover the common cases instead of comparing status codes or parsing error text yourself:

```go
value, err := c.Get(ctx, "missing")
if cachedb.IsNotFound(err) {
    // key doesn't exist
}

if code, ok := cachedb.StatusCode(err); ok {
    // code is e.g. 409 (wrong type), 413 (value too large), 429 (rate limited)...
}
```

`Exists` uses this internally — a `404` from `/exists` is not treated as an error, it's mapped to `(false, nil)`.

## Why this lives in the main repo

Same module, same version, one CI — no risk of the client drifting out of sync with what the server actually accepts. Importing `.../client` doesn't pull the server or dashboard code into your binary; Go only compiles what's actually referenced.
