package main

import "time"

func (c *Cache) prepareHashItem(key string, expiration *time.Time) (CacheItem, error) {
	item, found := c.data[key]
	if found && !item.Expired() {
		if item.Type != TypeHash {
			return CacheItem{}, ErrWrongType
		}
	} else {
		if expiration == nil {
			return CacheItem{}, ErrTTLRequired
		}
		item = CacheItem{Key: key, Type: TypeHash}
	}
	if item.Hash == nil {
		item.Hash = make(map[string]string)
	}
	if expiration != nil {
		item.Expiration = *expiration
	}
	return item, nil
}

func (c *Cache) HSet(key, field, value string, expiration *time.Time) (CacheItem, error) {
	if c.config.MaxValueBytes > 0 && len(value) > c.config.MaxValueBytes {
		return CacheItem{}, ErrValueTooLarge
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	item, err := c.prepareHashItem(key, expiration)
	if err != nil {
		return CacheItem{}, err
	}
	item.Hash[field] = value
	c.touch(&item)
	c.storeItem(key, item)
	c.evictIfNeeded()
	return item, nil
}

func (c *Cache) HGet(key, field string) (string, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.data[key]
	if !found || item.Expired() {
		return "", false, nil
	}
	if item.Type != TypeHash {
		return "", false, ErrWrongType
	}

	value, ok := item.Hash[field]
	return value, ok, nil
}

func (c *Cache) HGetAll(key string) (map[string]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.data[key]
	if !found || item.Expired() {
		return map[string]string{}, nil
	}
	if item.Type != TypeHash {
		return nil, ErrWrongType
	}

	result := make(map[string]string, len(item.Hash))
	for k, v := range item.Hash {
		result[k] = v
	}
	return result, nil
}

func (c *Cache) HDel(key, field string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.data[key]
	if !found || item.Expired() {
		return false, nil
	}
	if item.Type != TypeHash {
		return false, ErrWrongType
	}

	_, existed := item.Hash[field]
	delete(item.Hash, field)
	c.storeItem(key, item)
	return existed, nil
}
