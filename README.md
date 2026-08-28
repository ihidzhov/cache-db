# cache-db

An HTTP key-value store written in Go — strings, lists, and hashes, each with a TTL. Similar in spirit to Memcached, but with a search endpoint, a built-in web dashboard, and an official Go client.

---

### 📦 Already have a Go project and just want to talk to cache-db?

```
go get github.com/ihidzhov/cache-db/client
```

Skip everything below and go straight to **[client/README.md](client/README.md)** for the client library docs, or [cmd/cli](#go-client--cli) for a shell CLI built on it. No need to hand-roll HTTP requests.

---

## What this is

A single Go binary that stores key-value data in memory and serves it over plain HTTP. Point it at a port, `curl` (or use the [Go client](#go-client--cli)) to read/write, and optionally open the [dashboard](#dashboard) in a browser to see what's in it. Restarting the process clears everything — this is a cache, not a database with a WAL.

**Highlights:**

* Three data types — strings, lists, hashes — each with an independent TTL (or none at all)
* Search over string values: substring/prefix/suffix/regex/numeric filters, key glob matching, sorting, pagination
* Atomic counters (`increment`/`decrement`, integer or float)
* Optional bounded-cache mode: cap total keys and/or memory, evict LRU/LFU/random — off by default, so it's an unbounded open cache until you turn it on
* A web dashboard, a Go client library, and a CLI — three ways in besides raw `curl`
* Prometheus metrics, structured JSON logs, `/health`/`/ready` probes
* Optional API key auth, rate limiting, and CORS — all off by default
* A `/v1`-versioned API, Docker image, and a `-race`-clean test suite

## Contents

* [Quickstart](#quickstart)
* [Configuration](#configuration)
* [Dashboard](#dashboard)
* [Go client & CLI](#go-client--cli)
* [Data types](#data-types)
* [API Reference](#api-reference)
* [Observability](#observability)
* [Errors](#errors)
* [Testing](#testing)
* [Version History](#version-history)
* [License](#license)

## Quickstart

Requires Go 1.27+.

```
git clone https://github.com/ihidzhov/cache-db.git cache-server
cd cache-server
go run .
```

Or with Docker: `docker compose up --build`.

Then either open **`http://localhost:8080/dashboard`** in a browser, or:

```
curl --location 'http://localhost:8080/set' --form 'key="hello"' --form 'value="world"' --form 'ttl="300"'
curl --location 'http://localhost:8080/get?key=hello'
# → world
```

That's the whole loop: `/set` writes a key with a TTL in seconds, `/get` reads it back. Everything else in this README builds on those two calls.

## Configuration

Every setting is a CLI flag, each with a matching environment variable as a fallback — useful for containers, where you set env vars instead of editing a command line. Precedence: **flag > env var > default**.

| Flag                 | Env var                   | Default | Meaning                                                        |
|----------------------|----------------------------|---------|-----------------------------------------------------------------|
| `-addr`              | `CACHE_ADDR`               | `:8080` | Address to listen on                                            |
| `-cleanup-interval`  | `CACHE_CLEANUP_INTERVAL`   | `20`    | Seconds between background sweeps for expired keys              |
| `-max-keys`          | `CACHE_MAX_KEYS`           | `0`     | Max number of keys before eviction kicks in (`0` = unlimited)   |
| `-max-value-bytes`   | `CACHE_MAX_VALUE_BYTES`    | `0`     | Max size in bytes for a single stored value (`0` = unlimited)   |
| `-max-memory-bytes`  | `CACHE_MAX_MEMORY_BYTES`   | `0`     | Approx. max total size in bytes of all stored keys/values (`0` = unlimited) |
| `-eviction-policy`   | `CACHE_EVICTION_POLICY`    | `lru`   | `lru`, `lfu`, or `random` — which key to evict once a limit above is hit |
| `-api-key`           | `CACHE_API_KEY`            | `""`    | Require this key on every request except `/health`/`/ready`/`/dashboard` (empty = auth disabled) |
| `-rate-limit`        | `CACHE_RATE_LIMIT`         | `0`     | Max requests/sec per client, identified by API key or IP (`0` = disabled) |
| `-rate-limit-burst`  | `CACHE_RATE_LIMIT_BURST`   | `10`    | Burst size for `-rate-limit`                                    |
| `-cors-origin`       | `CACHE_CORS_ORIGIN`        | `""`    | Value for `Access-Control-Allow-Origin` (e.g. `*`); empty = CORS disabled |

Every limit defaults to off. `-max-keys` and `-max-memory-bytes` can be combined — whichever is hit first triggers eviction. The server shuts down gracefully on `SIGINT`/`SIGTERM`, waiting up to 10s for in-flight requests.

```
go run . -max-keys=10000 -max-memory-bytes=536870912 -eviction-policy=lfu
go run . -api-key=changeme -rate-limit=20 -rate-limit-burst=40 -cors-origin='*'
```

**Do you need `-api-key`?** It's one shared secret, not a user/permission system. On an internal network (a VPC, a cluster's internal mesh, localhost) leave it unset — an unauthenticated Memcached or Redis on that same network wouldn't be any safer, network isolation is the real boundary. Turn it on once the server is reachable from outside that boundary, pair it with `-rate-limit`, and put TLS in front of it (the server itself only speaks plain HTTP).

**Docker:**

```
docker build -t cache-db .
docker run --rm -p 8080:8080 cache-db
```

Multi-stage build (Go + minimal Alpine, non-root user, `HEALTHCHECK` against `/health`). `docker-compose.yml` lists every env var above.

## Dashboard

`http://localhost:8080/dashboard` — a built-in admin UI, one HTML page embedded in the binary (`web/dashboard.html`), served same-origin (no CORS setup needed). Plain HTML/CSS/JS, no framework, works offline.

* **Key browser** — every key with its type; filter by name, or switch to "by value" mode to search string contents via `/search`. Click a key to view it (JSON values are pretty-printed and highlighted), delete it, or add a new string key from the form at the bottom.
* **Overview** — live key count, size, hits, misses, hit ratio, evictions, with an activity sparkline.
* **Config** — a read-only snapshot of the running server's flags (never the API key itself).
* **Seed demo data** — one click populates a realistic dataset covering all three types, so a fresh cache isn't empty. Safe to click more than once.
* **Flush all** — wipes the cache, behind a confirmation dialog.

If `-api-key` is set, the page still loads, but its API calls need the key entered in the ⚙ settings popover first (kept in `localStorage`, never leaves the browser).

Prefer scripting the seed step? `go run ./cmd/seed` does the same thing over HTTP (`-flush` to reset first, `-repeat=30s` to keep re-seeding).

## Go client & CLI

**`client/`** is a Go SDK for this server, in the same module: `go get github.com/ihidzhov/cache-db/client`. Full docs, including error handling and TTL semantics, are in **[client/README.md](client/README.md)** — the short version:

```go
c := cachedb.New("http://localhost:8080", cachedb.WithAPIKey("changeme"))
ctx := context.Background()

c.Set(ctx, "user:1", `{"name":"alice"}`, 3600)
value, err := c.Get(ctx, "user:1")
length, err := c.RPush(ctx, "queue:jobs", "send-email", nil)
stats, err := c.Stats(ctx)
```

One method per endpoint, all take `context.Context`, all hit `/v1`. Errors come back typed (`cachedb.IsNotFound(err)`, `cachedb.StatusCode(err)`) instead of needing to parse strings.

**`cmd/cli`** wraps the same library for shell use:

```
go run ./cmd/cli set foo bar 60
go run ./cmd/cli get foo
go run ./cmd/cli stats
go run ./cmd/cli -h
```

## Data types

A key is exactly one type, fixed the first time it's created: `string` (`/set`), `list` (`/lpush`/`/rpush`), or `hash` (`/hset`). Using the wrong operation on an existing key returns `409` — e.g. `/lpush` on a key created with `/set` fails until that key is gone. `/set` is the one exception: it always overwrites a key outright, whatever type it held before.

**TTL** is seconds: positive expires the key after that long; `0` or `-1` means it never expires; any other negative value is rejected. `/set`, and a brand-new list/hash key, require `ttl`. On an *existing* list/hash key, `ttl` is optional on further `/lpush`/`/rpush`/`/hset` calls — pass it to refresh the expiration, omit it to leave the current one alone.

## API Reference

Every endpoint also works under `/v1` (`/v1/get` alongside `/get`) — same handler. Build new integrations against `/v1`; the unprefixed paths stay for backward compatibility. A future breaking change would land as `/v2`. `/health`, `/ready`, `/dashboard`, `/metrics` are infra endpoints and stay unversioned.

| Method | Path            | Body / Query                                                    | Description                                  |
|--------|-----------------|-------------------------------------------------------------------|-----------------------------------------------|
| POST   | `/set`          | form-data: `key`, `value`, `ttl`                                   | Store a string value                          |
| GET    | `/get`          | query: `key`, optional `output=json`                               | Retrieve a string value                       |
| GET    | `/search`       | query: see [Search parameters](#search-parameters)                 | Search stored string values                   |
| DELETE | `/delete`       | query: `key`                                                       | Remove a key                                  |
| HEAD   | `/exists`       | query: `key`                                                       | `200` if the key exists, `404` otherwise      |
| GET    | `/keys`         | query: `limit`, `offset`                                           | List keys with their type, TTL, and size      |
| DELETE | `/flush`        | —                                                                   | Remove every key                              |
| GET    | `/stats`        | —                                                                   | Cache statistics                              |
| PUT    | `/increment`    | form-data: `key`                                                   | Increment an integer value by 1               |
| PUT    | `/decrement`    | form-data: `key`                                                   | Decrement an integer value by 1               |
| PUT    | `/incrbyfloat`  | form-data: `key`, `value` (delta, may be negative)                 | Increment a numeric value by a float amount   |
| POST   | `/lpush`        | form-data: `key`, `value`, optional `ttl`                          | Push a value to the front of a list           |
| POST   | `/rpush`        | form-data: `key`, `value`, optional `ttl`                          | Push a value to the back of a list            |
| PUT    | `/lpop`         | query: `key`                                                       | Pop and return the first element of a list    |
| PUT    | `/rpop`         | query: `key`                                                       | Pop and return the last element of a list     |
| GET    | `/lrange`       | query: `key`, `start`, `stop`                                      | Return elements `start..stop` (inclusive)     |
| GET    | `/llen`         | query: `key`                                                       | Return the length of a list                   |
| POST   | `/hset`         | form-data: `key`, `field`, `value`, optional `ttl`                 | Set a field in a hash                         |
| GET    | `/hget`         | query: `key`, `field`                                              | Get a single field from a hash                |
| GET    | `/hgetall`      | query: `key`                                                       | Get every field in a hash                     |
| DELETE | `/hdel`         | query: `key`, `field`                                              | Delete a field from a hash                    |
| GET    | `/config`       | —                                                                   | Non-secret runtime configuration              |
| POST   | `/seed`         | —                                                                   | Populate the cache with demo data             |
| GET    | `/health`       | —                                                                   | Liveness probe (not versioned)                |
| GET    | `/ready`        | —                                                                   | Readiness probe (not versioned)               |
| GET    | `/dashboard`    | —                                                                   | Built-in web UI (not versioned, exempt from auth) |
| GET    | `/metrics`      | —                                                                   | Prometheus text-format metrics (not versioned) |

### Strings — `/set`, `/get`

```
curl --location 'http://localhost:8080/set' --form 'key="key"' --form 'value="value"' --form 'ttl="300"'
curl --location 'http://localhost:8080/get?key=key'
```

Values are stored as plain strings — to store JSON, store the JSON-encoded string and read it back with `output=json` (`422` if it isn't valid JSON). `/get` returns `404` if missing/expired, `409` if the key holds a list or hash. `ttl="0"`/`"-1"` for a permanent key.

### Search — `GET /search`

Filters combine with AND; omit what you don't need. Only string values are searched.

#### Search parameters

| Param           | Type   | Meaning                                                |
|-----------------|--------|-----------------------------------------------------------|
| `query`         | string | Case-insensitive substring match against the value        |
| `contains`      | string | Same as `query`, kept as a separate alias                 |
| `startswith`    | string | Value must start with this prefix                         |
| `endswith`      | string | Value must end with this suffix                           |
| `regex`         | string | Value must match this RE2 regular expression (`400` if invalid) |
| `gt` / `lt` / `eq` | number | Value (parsed as a number) compared against this       |
| `keystartswith` | string | Key must start with this prefix                            |
| `keypattern`    | string | Key must match this glob pattern (`*`, `?`, `[abc]`, as in `path.Match`) |
| `sort`          | `asc`/`desc` | Sort results by value (default `asc`)                |
| `limit` / `offset` | int | Pagination (default `10` / `0`)                           |

```
curl --location 'http://localhost:8080/search?contains=err&sort=desc&limit=20'
curl --location 'http://localhost:8080/search?keypattern=user:*'
```

```json
{
  "results": [{ "Key": "key", "Value": "value", "Expiration": "..." }],
  "meta": { "total": 1, "limit": 10, "offset": 0, "query": "", "sort": "asc", "has_more": false }
}
```

### Delete / exists / keys / flush

```
curl --location --request DELETE 'http://localhost:8080/delete?key=key'
curl -I 'http://localhost:8080/exists?key=key'
curl --location 'http://localhost:8080/keys?limit=50&offset=0'
curl --location --request DELETE 'http://localhost:8080/flush'
```

`/delete` and `/exists` work regardless of type. `/keys` lists keys sorted alphabetically (stable pagination), each with its type, `expires_at` (`null` if permanent), and approximate `size_bytes`. `/flush` wipes everything immediately — no confirmation, use with care.

### Counters

```
curl --location --request PUT 'http://localhost:8080/increment' --form 'key="key"'
curl --location --request PUT 'http://localhost:8080/incrbyfloat' --form 'key="price"' --form 'value="-1.75"'
```

The stored value must parse as a number. Both `/increment`/`/decrement` and `/incrbyfloat` are atomic under concurrent calls on the same key.

### Lists and hashes

```
curl --location 'http://localhost:8080/rpush' --form 'key="mylist"' --form 'value="b"' --form 'ttl="300"'
curl --location 'http://localhost:8080/lrange?key=mylist&start=0&stop=-1'
curl --location 'http://localhost:8080/hset' --form 'key="myhash"' --form 'field="a"' --form 'value="1"' --form 'ttl="300"'
curl --location 'http://localhost:8080/hgetall?key=myhash'
```

`start`/`stop` in `/lrange` work like Redis's `LRANGE` — `0` is the first element, negative counts from the end (`-1` is the last).

### Stats — `GET /stats`

```json
{
  "total_keys": 3,
  "approx_bytes": 128,
  "hits": 12,
  "misses": 2,
  "evictions": 0,
  "uptime": "0 days, 00:04:12"
}
```

`hits`/`misses` update on `/get`, `/increment`, `/decrement`, `/incrbyfloat`. `approx_bytes` is the running total of key+value bytes across all types — what `-max-memory-bytes` compares against.

## Observability

* **Logging** — every request logged to stdout as one JSON line via [`log/slog`](https://pkg.go.dev/log/slog), with a `request_id` also echoed back as the `X-Request-Id` response header.
* **Metrics — `GET /metrics`** — hand-rolled Prometheus text format: `cache_requests_total{path,status}`, `cache_request_duration_seconds{path}` (histogram), `cache_hits_total`/`cache_misses_total`, `cache_evictions_total`, `cache_keys`, `cache_approx_bytes`, `cache_uptime_seconds`.
* **Health — `GET /health` / `GET /ready`** — `/health` is a plain liveness check. `/ready` additionally checks the cache's internal lock responds within 200ms (`503` if not). Point a Kubernetes `livenessProbe` at the first, `readinessProbe` at the second.

## Errors

| Status | When                                                                                     |
|--------|--------------------------------------------------------------------------------------------|
| 400    | Missing/invalid params, an invalid `regex`, or a non-numeric value on a counter endpoint   |
| 401    | Missing/invalid API key while `-api-key` is set                                            |
| 404    | Key (or field) not found or expired                                                        |
| 405    | Wrong HTTP method for the endpoint                                                          |
| 409    | The key exists but holds a different type than the operation expects                       |
| 413    | The value exceeds `-max-value-bytes`                                                       |
| 422    | `/get?output=json` requested but the stored value isn't valid JSON                         |
| 429    | Client exceeded `-rate-limit`                                                              |

## Testing

```
go test ./...
go test ./... -race -cover
```

Covers the cache/list/hash/eviction/search logic directly, the HTTP handlers via `net/http/httptest`, the Go client against a real server subprocess, and concurrency tests (e.g. hundreds of goroutines hitting `/increment` on the same key). Always run with `-race`.

## Version History

* **v0.1.0** — first tagged release. Strings/lists/hashes with per-key TTL; search (substring/prefix/suffix/regex/numeric, key glob matching, sort, pagination); atomic int/float counters; optional bounded-cache eviction (LRU/LFU/random); optional API key auth, rate limiting, and CORS; structured logging, Prometheus metrics, health probes; a `/v1`-versioned API; a built-in web dashboard; a Go client library and CLI; a Docker image; a unit + concurrency test suite.
* Pre-tag history (`/set`/`/get`/`/delete`/`/stats`, then `/increment`/`/decrement`, then an early `/search`) — see the [commit log](https://github.com/ihidzhov/cache-db/commits/main).

## License

MIT — see [LICENSE](LICENSE).
