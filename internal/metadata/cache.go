package metadata

import (
	"net/url"
	"sync"
	"time"
)

// Cache is one Node's in-memory answer cache: a bounded LRU of bodies, each
// with the time it was stored. It drops everything when a call carries a
// generation other than the one it holds, so clearing every holder needs no
// tracking of offline ones.
type Cache struct {
	mu      sync.Mutex
	limit   int
	gen     uint64
	entries map[string]*cached
	// recent is the ring's sentinel: recent.next is the most recently used.
	recent cached
}

type cached struct {
	key        string
	body       []byte
	at         time.Time
	prev, next *cached
}

func newCache(limit int) *Cache {
	c := &Cache{limit: limit, entries: map[string]*cached{}}
	c.recent.prev, c.recent.next = &c.recent, &c.recent
	return c
}

func (c *Cache) Sync(generation uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if generation != c.gen {
		c.gen = generation
		clear(c.entries)
		c.recent.prev, c.recent.next = &c.recent, &c.recent
	}
}

// Get answers the body stored under key when it is younger than maxAge.
func (c *Cache) Get(key string, maxAge time.Duration) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || time.Since(e.at) >= maxAge {
		return nil, false
	}
	c.touch(e)
	return e.body, true
}

func (c *Cache) Put(key string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		e = &cached{key: key}
		c.entries[key] = e
	}
	e.body, e.at = body, time.Now()
	c.touch(e)
	if len(c.entries) > c.limit {
		oldest := c.recent.prev
		unlink(oldest)
		delete(c.entries, oldest.key)
	}
}

func (c *Cache) touch(e *cached) {
	if e.next != nil {
		unlink(e)
	}
	e.prev, e.next = &c.recent, c.recent.next
	c.recent.next.prev = e
	c.recent.next = e
}

func unlink(e *cached) {
	e.prev.next, e.next.prev = e.next, e.prev
}

// cacheKey is provider, URL and sorted query. Request.URL never holds the
// key, which the Line adds only to what it sends.
func cacheKey(p ProviderID, u *url.URL) string {
	return string(p) + " " + u.Scheme + "://" + u.Host + u.EscapedPath() + "?" + u.Query().Encode()
}
