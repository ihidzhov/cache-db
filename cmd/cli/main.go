package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	cachedb "github.com/ihidzhov/cache-db/client"
)

func main() {
	addr := flag.String("addr", "http://localhost:8080", "cache-db base URL")
	apiKey := flag.String("api-key", "", "API key, if the server has -api-key set")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	c := cachedb.New(*addr, cachedb.WithAPIKey(*apiKey))
	ctx := context.Background()

	cmd, rest := args[0], args[1:]
	if err := run(ctx, c, cmd, rest); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `cache-db-cli [-addr=http://localhost:8080] [-api-key=...] <command> [args]

Commands:
  set <key> <value> [ttl=300]
  get <key>
  del <key>
  exists <key>
  incr <key>
  decr <key>
  incrbyfloat <key> <delta>
  lpush <key> <value> [ttl]
  rpush <key> <value> [ttl]
  lpop <key>
  rpop <key>
  lrange <key> <start> <stop>
  llen <key>
  hset <key> <field> <value> [ttl]
  hget <key> <field>
  hgetall <key>
  hdel <key> <field>
  keys [limit] [offset]
  search <query>
  stats
  config
  seed
  flush
`)
}

func run(ctx context.Context, c *cachedb.Client, cmd string, args []string) error {
	switch cmd {
	case "set":
		if len(args) < 2 {
			return fmt.Errorf("usage: set <key> <value> [ttl]")
		}
		ttl := 300
		if len(args) > 2 {
			var err error
			ttl, err = strconv.Atoi(args[2])
			if err != nil {
				return err
			}
		}
		return c.Set(ctx, args[0], args[1], ttl)

	case "get":
		if len(args) < 1 {
			return fmt.Errorf("usage: get <key>")
		}
		value, err := c.Get(ctx, args[0])
		if err != nil {
			return err
		}
		fmt.Println(value)
		return nil

	case "del":
		if len(args) < 1 {
			return fmt.Errorf("usage: del <key>")
		}
		return c.Delete(ctx, args[0])

	case "exists":
		if len(args) < 1 {
			return fmt.Errorf("usage: exists <key>")
		}
		ok, err := c.Exists(ctx, args[0])
		if err != nil {
			return err
		}
		fmt.Println(ok)
		return nil

	case "incr":
		if len(args) < 1 {
			return fmt.Errorf("usage: incr <key>")
		}
		return c.Increment(ctx, args[0])

	case "decr":
		if len(args) < 1 {
			return fmt.Errorf("usage: decr <key>")
		}
		return c.Decrement(ctx, args[0])

	case "incrbyfloat":
		if len(args) < 2 {
			return fmt.Errorf("usage: incrbyfloat <key> <delta>")
		}
		delta, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return err
		}
		value, err := c.IncrByFloat(ctx, args[0], delta)
		if err != nil {
			return err
		}
		fmt.Println(value)
		return nil

	case "lpush", "rpush":
		if len(args) < 2 {
			return fmt.Errorf("usage: %s <key> <value> [ttl]", cmd)
		}
		var ttl *int
		if len(args) > 2 {
			v, err := strconv.Atoi(args[2])
			if err != nil {
				return err
			}
			ttl = &v
		}
		var length int
		var err error
		if cmd == "lpush" {
			length, err = c.LPush(ctx, args[0], args[1], ttl)
		} else {
			length, err = c.RPush(ctx, args[0], args[1], ttl)
		}
		if err != nil {
			return err
		}
		fmt.Println(length)
		return nil

	case "lpop", "rpop":
		if len(args) < 1 {
			return fmt.Errorf("usage: %s <key>", cmd)
		}
		var value string
		var err error
		if cmd == "lpop" {
			value, err = c.LPop(ctx, args[0])
		} else {
			value, err = c.RPop(ctx, args[0])
		}
		if err != nil {
			return err
		}
		fmt.Println(value)
		return nil

	case "lrange":
		if len(args) < 3 {
			return fmt.Errorf("usage: lrange <key> <start> <stop>")
		}
		start, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		stop, err := strconv.Atoi(args[2])
		if err != nil {
			return err
		}
		values, err := c.LRange(ctx, args[0], start, stop)
		if err != nil {
			return err
		}
		return printJSON(values)

	case "llen":
		if len(args) < 1 {
			return fmt.Errorf("usage: llen <key>")
		}
		length, err := c.LLen(ctx, args[0])
		if err != nil {
			return err
		}
		fmt.Println(length)
		return nil

	case "hset":
		if len(args) < 3 {
			return fmt.Errorf("usage: hset <key> <field> <value> [ttl]")
		}
		var ttl *int
		if len(args) > 3 {
			v, err := strconv.Atoi(args[3])
			if err != nil {
				return err
			}
			ttl = &v
		}
		return c.HSet(ctx, args[0], args[1], args[2], ttl)

	case "hget":
		if len(args) < 2 {
			return fmt.Errorf("usage: hget <key> <field>")
		}
		value, err := c.HGet(ctx, args[0], args[1])
		if err != nil {
			return err
		}
		fmt.Println(value)
		return nil

	case "hgetall":
		if len(args) < 1 {
			return fmt.Errorf("usage: hgetall <key>")
		}
		fields, err := c.HGetAll(ctx, args[0])
		if err != nil {
			return err
		}
		return printJSON(fields)

	case "hdel":
		if len(args) < 2 {
			return fmt.Errorf("usage: hdel <key> <field>")
		}
		return c.HDel(ctx, args[0], args[1])

	case "keys":
		limit, offset := 100, 0
		if len(args) > 0 {
			v, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}
			limit = v
		}
		if len(args) > 1 {
			v, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}
			offset = v
		}
		page, err := c.Keys(ctx, limit, offset)
		if err != nil {
			return err
		}
		return printJSON(page)

	case "search":
		if len(args) < 1 {
			return fmt.Errorf("usage: search <query>")
		}
		result, err := c.Search(ctx, cachedb.SearchOptions{Query: strings.Join(args, " ")})
		if err != nil {
			return err
		}
		return printJSON(result)

	case "stats":
		stats, err := c.Stats(ctx)
		if err != nil {
			return err
		}
		return printJSON(stats)

	case "config":
		cfg, err := c.Config(ctx)
		if err != nil {
			return err
		}
		return printJSON(cfg)

	case "seed":
		writes, err := c.Seed(ctx)
		if err != nil {
			return err
		}
		fmt.Println(writes, "writes")
		return nil

	case "flush":
		return c.Flush(ctx)

	default:
		usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
