package metadata

import (
	"sync"
	"time"
)

// Cache provides in-memory caching with TTL for metadata results.
type Cache struct {
	mu       sync.RWMutex
	items    map[string]cacheItem
	ttl      time.Duration
	maxItems int
}

type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

// CacheConfig holds cache configuration.
type CacheConfig struct {
	TTL      time.Duration
	MaxItems int
}

// DefaultCacheConfig returns default cache configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		TTL:      15 * time.Minute,
		MaxItems: 1000,
	}
}

// NewCache creates a new cache with the given configuration.
func NewCache(cfg CacheConfig) *Cache {
	if cfg.TTL == 0 {
		cfg.TTL = 15 * time.Minute
	}
	if cfg.MaxItems == 0 {
		cfg.MaxItems = 1000
	}

	c := &Cache{
		items:    make(map[string]cacheItem),
		ttl:      cfg.TTL,
		maxItems: cfg.MaxItems,
	}

	// Start background cleanup goroutine
	go c.cleanup()

	return c
}

// Get retrieves an item from the cache.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		return nil, false
	}

	return item.value, true
}

// Set stores an item in the cache.
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest items if at capacity
	if len(c.items) >= c.maxItems {
		c.evictOldest()
	}

	c.items[key] = cacheItem{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// SetWithTTL stores an item with a custom TTL.
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxItems {
		c.evictOldest()
	}

	c.items[key] = cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes an item from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]cacheItem)
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// evictOldest removes the oldest 10% of items (must be called with lock held).
func (c *Cache) evictOldest() {
	// Simple eviction: remove expired items first
	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
		}
	}

	// If still at capacity, remove oldest 10%
	if len(c.items) >= c.maxItems {
		toRemove := c.maxItems / 10
		if toRemove < 1 {
			toRemove = 1
		}

		var oldest []string
		var oldestTimes []time.Time

		for key, item := range c.items {
			if len(oldest) < toRemove {
				oldest = append(oldest, key)
				oldestTimes = append(oldestTimes, item.expiresAt)
			} else {
				// Find if this item is older than any in our list
				for i, t := range oldestTimes {
					if item.expiresAt.Before(t) {
						oldest[i] = key
						oldestTimes[i] = item.expiresAt
						break
					}
				}
			}
		}

		for _, key := range oldest {
			delete(c.items, key)
		}
	}
}

// cleanup periodically removes expired items.
func (c *Cache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// GetMovieResults retrieves cached movie search results.
func (c *Cache) GetMovieResults(key string) ([]MovieResult, bool) {
	val, ok := c.Get(key)
	if !ok {
		return nil, false
	}
	results, ok := val.([]MovieResult)
	return results, ok
}

// GetMovieResult retrieves a cached movie result.
func (c *Cache) GetMovieResult(key string) (*MovieResult, bool) {
	val, ok := c.Get(key)
	if !ok {
		return nil, false
	}
	result, ok := val.(*MovieResult)
	return result, ok
}

// GetSeriesResults retrieves cached series search results.
func (c *Cache) GetSeriesResults(key string) ([]SeriesResult, bool) {
	val, ok := c.Get(key)
	if !ok {
		return nil, false
	}
	results, ok := val.([]SeriesResult)
	return results, ok
}

// GetSeriesResult retrieves a cached series result.
func (c *Cache) GetSeriesResult(key string) (*SeriesResult, bool) {
	val, ok := c.Get(key)
	if !ok {
		return nil, false
	}
	result, ok := val.(*SeriesResult)
	return result, ok
}
