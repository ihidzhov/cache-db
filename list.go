package main

import "time"

func (c *Cache) prepareListItem(key string, expiration *time.Time) (CacheItem, error) {
	item, found := c.data[key]
	if found && !item.Expired() {
		if item.Type != TypeList {
			return CacheItem{}, ErrWrongType
		}
	} else {
		if expiration == nil {
			return CacheItem{}, ErrTTLRequired
		}
		item = CacheItem{Key: key, Type: TypeList}
	}
	if expiration != nil {
		item.Expiration = *expiration
	}
	return item, nil
}

func (c *Cache) LPush(key, value string, expiration *time.Time) (CacheItem, error) {
	if c.config.MaxValueBytes > 0 && len(value) > c.config.MaxValueBytes {
		return CacheItem{}, ErrValueTooLarge
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	item, err := c.prepareListItem(key, expiration)
	if err != nil {
		return CacheItem{}, err
	}
	item.List = append([]string{value}, item.List...)
	c.touch(&item)
	c.storeItem(key, item)
	c.evictIfNeeded()
	return item, nil
}

func (c *Cache) RPush(key, value string, expiration *time.Time) (CacheItem, error) {
	if c.config.MaxValueBytes > 0 && len(value) > c.config.MaxValueBytes {
		return CacheItem{}, ErrValueTooLarge
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	item, err := c.prepareListItem(key, expiration)
	if err != nil {
		return CacheItem{}, err
	}
	item.List = append(item.List, value)
	c.touch(&item)
	c.storeItem(key, item)
	c.evictIfNeeded()
	return item, nil
}

func (c *Cache) LPop(key string) (string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.data[key]
	if !found || item.Expired() {
		return "", false, nil
	}
	if item.Type != TypeList {
		return "", false, ErrWrongType
	}
	if len(item.List) == 0 {
		return "", false, nil
	}

	value := item.List[0]
	item.List = item.List[1:]
	c.storeItem(key, item)
	return value, true, nil
}

func (c *Cache) RPop(key string) (string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.data[key]
	if !found || item.Expired() {
		return "", false, nil
	}
	if item.Type != TypeList {
		return "", false, ErrWrongType
	}
	if len(item.List) == 0 {
		return "", false, nil
	}

	last := len(item.List) - 1
	value := item.List[last]
	item.List = item.List[:last]
	c.storeItem(key, item)
	return value, true, nil
}

func (c *Cache) LRange(key string, start, stop int) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.data[key]
	if !found || item.Expired() {
		return []string{}, nil
	}
	if item.Type != TypeList {
		return nil, ErrWrongType
	}

	n := len(item.List)
	start = normalizeListIndex(start, n)
	stop = normalizeListIndex(stop, n)
	if stop >= n {
		stop = n - 1
	}
	if n == 0 || start > stop || start >= n {
		return []string{}, nil
	}

	result := make([]string, stop-start+1)
	copy(result, item.List[start:stop+1])
	return result, nil
}

func (c *Cache) LLen(key string) (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.data[key]
	if !found || item.Expired() {
		return 0, nil
	}
	if item.Type != TypeList {
		return 0, ErrWrongType
	}
	return len(item.List), nil
}

func normalizeListIndex(idx, n int) int {
	if idx < 0 {
		idx = n + idx
	}
	if idx < 0 {
		idx = 0
	}
	return idx
}
