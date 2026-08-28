package main

import (
	"fmt"
	"math/rand"
	"time"
)

func seedExpiration(seconds int) time.Time {
	if seconds <= 0 {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(seconds) * time.Second)
}

func seedDemoData(c *Cache) int {
	writes := 0

	users := []struct{ name, email string }{
		{"alice", "alice@example.com"},
		{"bob", "bob@example.com"},
		{"carol", "carol@example.com"},
		{"dave", "dave@example.com"},
		{"erin", "erin@example.com"},
	}
	for i, u := range users {
		key := fmt.Sprintf("user:%d", i+1)
		value := fmt.Sprintf(`{"name":%q,"email":%q,"active":true}`, u.name, u.email)
		c.Set(key, value, seedExpiration(3600))
		writes++
	}

	counters := map[string]string{
		"counter:pageviews": "184320",
		"counter:signups":   "742",
		"counter:errors":    "3",
	}
	for k, v := range counters {
		c.Set(k, v, time.Time{})
		writes++
	}

	c.Set("config:app", `{"theme":"dark","locale":"en-US","version":"1.4.2","features":["dark_mode","beta_search"],"limits":{"max_upload_mb":25,"rate_limit_per_min":100}}`, time.Time{})
	writes++

	flags := map[string]string{
		"config:feature.dark_mode":   "true",
		"config:feature.beta_search": "false",
		"config:max_upload_mb":       "25",
	}
	for k, v := range flags {
		c.Set(k, v, time.Time{})
		writes++
	}

	for i := 0; i < 4; i++ {
		key := fmt.Sprintf("session:%08x", rand.Uint32())
		c.Set(key, fmt.Sprintf("user:%d", 1+rand.Intn(len(users))), seedExpiration(120))
		writes++
	}

	c.Delete("queue:jobs")
	jobs := []string{"send-email:welcome", "resize-image:1234", "generate-report:q3", "sync-inventory", "cleanup-tmp"}
	jobsTTL := seedExpiration(600)
	for _, j := range jobs {
		c.RPush("queue:jobs", j, &jobsTTL)
		writes++
	}

	c.Delete("recent:logins")
	loginsTTL := seedExpiration(600)
	for _, name := range []string{"alice", "bob", "carol"} {
		c.LPush("recent:logins", name, &loginsTTL)
		writes++
	}

	products := []struct{ id, name, price, stock string }{
		{"1001", "Wireless Mouse", "19.99", "142"},
		{"1002", "Mechanical Keyboard", "89.50", "37"},
		{"1003", "USB-C Hub", "24.00", "88"},
	}
	for _, p := range products {
		key := "product:" + p.id
		c.Delete(key)
		ttl := seedExpiration(3600)
		c.HSet(key, "name", p.name, &ttl)
		c.HSet(key, "price", p.price, nil)
		c.HSet(key, "stock", p.stock, nil)
		writes += 3
	}

	c.Delete("settings:app")
	permanent := time.Time{}
	c.HSet("settings:app", "theme", "dark", &permanent)
	c.HSet("settings:app", "locale", "en-US", nil)
	c.HSet("settings:app", "version", "1.4.2", nil)
	writes += 3

	return writes
}
