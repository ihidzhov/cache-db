# cache-db

A small in-memory key-value server written in Go — strings, lists, and hashes, each with a TTL, over a simple HTTP API. Similar in spirit to Memcached, with some things Memcached doesn't have: a search endpoint, a built-in web dashboard, and a Go client library.

> **Writing Go code that needs to read/write this cache?** Don't hand-roll HTTP calls — `go get github.com/ihidzhov/cache-db/client` and use the Go client library instead. It's a thin wrapper with one method per endpoint, typed errors, and full docs in **[client/README.md](client/README.md)**. There's also a CLI (`cmd/cli`) built on the same library for scripting from a shell — see [Go client & CLI](#go-client--cli) below.

## Features

* Strings, lists, and hashes, each with an optional per-key TTL (or no expiration at all)
* Search over string values — substring/prefix/suffix/regex/numeric filters, key glob/prefix matching, sorting, pagination
* Atomic increment/decrement (integer or float), safe under concurrent requests
* Optional bounded-cache mode — max keys and/or max total memory, with LRU/LFU/random eviction — off by default
* A built-in web dashboard (`/dashboard`) and a Go client + CLI, so you're never stuck writing raw HTTP calls
* Structured JSON request logging, Prometheus `/metrics`, `/health`/`/ready` probes
* Optional API key auth, per-client rate limiting, and CORS — all off by default
* Versioned API (`/v1/...`), graceful shutdown, a Docker image, and a unit + concurrency (`-race`) test suite

Everything lives in process memory — restarting the server clears the cache. A background sweeper removes expired keys automatically, in addition to expiry being enforced on read.

## Contents

* [Getting Started](#getting-started)
* [Dashboard](#dashboard)
* [Go client & CLI](#go-client--cli)
* [Data types](#data-types)
* [API Reference](#api-reference)
* [Observability](#observability)
* [Errors](#errors)
* [Testing](#testing)
* [Version History](#version-history)
* [License](#license)

## Getting Started

Requires Go 1.27+.

```
git clone https://github.com/ihidzhov/cache-db.git cache-server
cd cache-server
go run .
```

Or with Docker:

```
docker compose up --build
```

Then open `http://localhost:8080/dashboard`, or:

```
curl --location 'http://localhost:8080/set' --form 'key="hello"' --form 'value="world"' --form 'ttl="300"'
curl --location 'http://localhost:8080/get?key=hello'
```

### Configuration

Every setting is a CLI flag, each with a matching environment variable as a fallback default — useful for containers. Precedence: **explicit flag > env var > built-in default**.

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

**Every limit is off by default** — the server behaves as an unbounded, open cache until you turn one of these on. `-max-keys` and `-max-memory-bytes` can be set together; whichever is hit first triggers eviction. The server shuts down gracefully on `SIGINT`/`SIGTERM`, waiting up to 10s for in-flight requests to finish.

```
go run . -max-keys=10000 -max-memory-bytes=536870912 -eviction-policy=lfu
go run . -api-key=changeme -rate-limit=20 -rate-limit-burst=40 -cors-origin='*'
```

### Auth, rate limiting, and CORS

All three are independent and off by default.

* **Auth** — when `-api-key` (or `CACHE_API_KEY`) is set, every request needs an `X-API-Key: <key>` or `Authorization: Bearer <key>` header, or it gets `401`. `/health`, `/ready`, and `/dashboard` are always exempt. The key is compared in constant time.
* **Rate limiting** — a token-bucket limiter per client (API key if present, else IP). Over the limit gets `429`. `/health` and `/ready` are exempt.
* **CORS** — when `-cors-origin` is set, responses get `Access-Control-Allow-*` headers, and `OPTIONS` preflight gets an immediate `204` before auth/rate-limiting run (browsers don't attach the API key to preflight requests).

**Do you need auth?** This isn't a user/permission system — `-api-key` is one shared secret, closer to a service credential. On an internal network (a VPC, a cluster's internal mesh, localhost) leave it unset; an unauthenticated Memcached or Redis on that same network wouldn't be any safer, so network isolation is the real boundary. Set it when the server is reachable from outside that boundary — pair it with `-rate-limit`, and put TLS termination in front of it, since the server itself only speaks plain HTTP.

### Docker

```
docker build -t cache-db .
docker run --rm -p 8080:8080 cache-db
```

The image is a multi-stage build (Go builder + minimal Alpine runtime, so `wget` is available for the container's own `HEALTHCHECK` against `/health`), runs as a non-root user. `docker-compose.yml` lists every env var above — override what you need in its `environment:` block, or via `-e CACHE_...=...` on `docker run`.

## Dashboard

Open `http://localhost:8080/dashboard` for a built-in admin UI — a single HTML page embedded in the binary (`web/dashboard.html`, via `go:embed`), served same-origin so it never needs `-cors-origin`. Plain HTML/CSS/JS, no framework, no external requests, works fully offline.

* **Key browser** (left) — every key with its type, filterable by name or, in "by value" mode, backed by `/search` (string values only). Click a key to view it: plain text for strings (pretty-printed and syntax-highlighted if it happens to be JSON), an ordered list for lists, a table for hashes, plus size and expiration. Delete from the detail view, or add a new string key from the form at the bottom of the list.
* **Overview** — live key count, approximate size, hits, misses, hit ratio, and evictions, with a small sparkline of activity per interval.
* **Config** — a read-only snapshot of `/config`: every flag this instance was started with, and whether auth is on (never the key itself).
* **Seed demo data** — populates a small realistic dataset (users, counters, feature flags, short-lived sessions, a job queue, product hashes) covering all three types, so there's something to click around on a fresh cache. Safe to click repeatedly.
* **Flush all** — clears the whole cache, behind a confirmation.

If `-api-key` is set, the dashboard page itself still loads, but its calls to the API won't succeed until you open the ⚙ settings popover and paste in the key — kept in `localStorage`, sent as `X-API-Key`, never leaves the browser.

Prefer the command line for seeding? `go run ./cmd/seed` does the same thing over HTTP:

```
go run ./cmd/seed
go run ./cmd/seed -addr=http://localhost:8080 -api-key=changeme -flush
go run ./cmd/seed -repeat=30s
```

## Go client & CLI

Full docs: **[client/README.md](client/README.md)**.

```
go get github.com/ihidzhov/cache-db/client
```

```go
c := cachedb.New("http://localhost:8080", cachedb.WithAPIKey("changeme"))
ctx := context.Background()

c.Set(ctx, "user:1", `{"name":"alice"}`, 3600)
value, err := c.Get(ctx, "user:1")
length, err := c.RPush(ctx, "queue:jobs", "send-email", nil)
stats, err := c.Stats(ctx)
```

One method per endpoint, all versioned against `/v1`, all taking `context.Context`. Non-2xx responses come back as `*cachedb.APIError`; check them with `cachedb.IsNotFound(err)` / `cachedb.StatusCode(err)` instead of parsing strings. It lives in this repo (not a separate one) so it stays in lockstep with the server — importing it doesn't pull the server or dashboard code into your binary.

`cmd/cli` is a thin command-line wrapper over the same library:

```
go run ./cmd/cli set foo bar 60
go run ./cmd/cli get foo
go run ./cmd/cli stats
go run ./cmd/cli -h
```

## Data types

Every key holds exactly one type, picked the first time it's created: `string` (via `/set`), `list` (via `/lpush`/`/rpush`), or `hash` (via `/hset`). An operation for a different type against an existing key returns `409 Conflict` — e.g. `/lpush` on a key created with `/set` fails until that key is deleted or expires. `/set` is the exception: it always overwrites the key outright, regardless of its previous type.

**TTL** is given in seconds: a positive number expires the key after that many seconds; `0` or `-1` means the key never expires; anything else negative is rejected. `/set` and a brand-new list/hash key require `ttl`. On an existing list/hash key, `ttl` is optional on subsequent `/lpush`, `/rpush`, or `/hset` calls — pass it to refresh the expiration, or omit it to leave the current one untouched.

## API Reference

Every endpoint below also works under `/v1` (e.g. `/v1/get` alongside `/get`) — same handler, same behavior. `/v1` is what to build new integrations against; the unprefixed paths are kept for backward compatibility and aren't going away. A future breaking change would land as `/v2`. `/health`, `/ready`, `/dashboard`, and `/metrics` are infrastructure endpoints and intentionally not versioned.

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

### Store a value — `POST /set`

```
curl --location 'http://localhost:8080/set' --form 'key="key"' --form 'value="value"' --form 'ttl="300"'
```

Values are stored as plain strings — if you want JSON, store the JSON-encoded string and read it back with `output=json`. Use `ttl="0"` or `ttl="-1"` for a key that never expires.

### Read a value — `GET /get`

```
curl --location 'http://localhost:8080/get?key=key'
```

Returns `404` if the key is missing or expired, `409` if it holds a list or hash.

```
curl --location 'http://localhost:8080/get?key=key&output=json'
```

Decodes the value as JSON; fails with `422` if the stored value isn't valid JSON.

### Search — `GET /search`

Filters are combined with AND; any filter you omit is ignored. Only string values are searched.

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

`/delete` and `/exists` work regardless of key type. `/keys` returns keys sorted alphabetically (stable pagination), each with its type, `expires_at` (`null` if permanent), and approximate `size_bytes`. `/flush` removes everything immediately — no confirmation, use with care.

### Counters

The stored value must parse as a number. Both `/increment`/`/decrement` and `/incrbyfloat` are atomic — safe to call concurrently against the same key.

```
curl --location --request PUT 'http://localhost:8080/increment' --form 'key="key"'
curl --location --request PUT 'http://localhost:8080/incrbyfloat' --form 'key="price"' --form 'value="-1.75"'
```

### Lists and hashes

```
curl --location 'http://localhost:8080/rpush' --form 'key="mylist"' --form 'value="b"' --form 'ttl="300"'
curl --location 'http://localhost:8080/lrange?key=mylist&start=0&stop=-1'
curl --location 'http://localhost:8080/hset' --form 'key="myhash"' --form 'field="a"' --form 'value="1"' --form 'ttl="300"'
curl --location 'http://localhost:8080/hgetall?key=myhash'
```

`start`/`stop` in `/lrange` work like Redis's `LRANGE`: `0` is the first element, negative indices count from the end (`-1` is the last).

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

**Logging** — every request is logged to stdout as one JSON line via [`log/slog`](https://pkg.go.dev/log/slog), including a `request_id` that's also echoed back as the `X-Request-Id` response header, so a client-reported issue can be matched to its log line.

**Metrics — `GET /metrics`** — hand-rolled Prometheus text format (no external dependency): `cache_requests_total{path,status}`, `cache_request_duration_seconds{path}` (histogram), `cache_hits_total`/`cache_misses_total` (derive hit ratio with a `rate()` query), `cache_evictions_total`, `cache_keys`, `cache_approx_bytes`, `cache_uptime_seconds`.

**Health — `GET /health` and `GET /ready`** — `/health` always returns `200` once the process is up. `/ready` additionally checks the cache's internal lock is responsive within 200ms, returning `503` if not. Point a Kubernetes `livenessProbe` at `/health` and a `readinessProbe` at `/ready`.

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

Covers the cache/list/hash/eviction/search logic directly, the HTTP handlers via `net/http/httptest`, the Go client against a real server subprocess, and dedicated concurrency tests (e.g. hundreds of goroutines hitting `/increment` on the same key). Always run with `-race`.

## Version History

* **Unreleased**
    * **Client tooling** — a Go client library (`client/`, `go get github.com/ihidzhov/cache-db/client`) and a CLI built on it (`cmd/cli`); fixed the module path so the package is actually `go get`-able
    * **Dashboard** — a built-in web UI at `/dashboard` (key browser with JSON pretty-printing, live stats, config panel, one-click demo seeding), backed by new `/config` and `/seed` endpoints, also usable standalone via `go run ./cmd/seed`
    * **Data model** — lists and hashes alongside strings; permanent keys (`ttl=0`/`-1`) instead of TTL always being mandatory; atomic integer and float increment/decrement
    * **Search** — real filtering (substring/prefix/suffix/regex/numeric on values, prefix/glob on keys), sorting, and pagination, instead of returning every stored key
    * **Bounded caches** — optional `-max-keys` and `-max-memory-bytes` with LRU/LFU/random eviction, and a per-value size limit, all opt-in
    * **Security** — optional API key auth, per-client rate limiting, and CORS, all off by default
    * **Observability** — structured JSON request logging, a Prometheus `/metrics` endpoint, `/health`/`/ready` probes
    * **Operations** — every flag has a `CACHE_*` env var fallback, graceful shutdown, a Docker image, and a `/v1`-versioned API with unprefixed paths kept as aliases
    * **Correctness fixes** — `/increment`/`/decrement` are now atomic (no lost updates under concurrent requests on the same key); `hits`/`misses` are now safe under concurrent access and a miss no longer also counts as a hit; `/get?output=json` returns `422` instead of silently responding with `null` on invalid JSON
    * Added a unit + concurrency (`-race`) test suite covering all of the above
* **0.2**
    * [`87a8a5b`](https://github.com/ihidzhov/cache-db/commit/87a8a5b) — reject requests using the wrong HTTP method
    * [`f014218`](https://github.com/ihidzhov/cache-db/commit/f014218), [`9dac52f`](https://github.com/ihidzhov/cache-db/commit/9dac52f) — added `/increment` and `/decrement`
    * [`321ffb5`](https://github.com/ihidzhov/cache-db/commit/321ffb5) — initial `/search` skeleton
* **0.1**
    * [`30a637a`](https://github.com/ihidzhov/cache-db/commit/30a637a) — initial release: `/set`, `/get`, `/delete`, `/stats`

See the [full commit history](https://github.com/ihidzhov/cache-db/commits/main) for details.

## License

MIT — see [LICENSE](LICENSE).
