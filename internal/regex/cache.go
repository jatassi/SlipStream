package regex

import "sync"

// cacheSize bounds Cached: TRaSH's whole catalog is about 1,300 title
// patterns, so this holds every pasted format with room to spare.
const cacheSize = 8192

type cacheKey struct {
	pattern string
	o       Options
}

type cacheEntry struct {
	re   *Regex
	used uint64
}

var cache = struct {
	sync.Mutex
	at   map[cacheKey]*cacheEntry
	tick uint64
}{at: map[cacheKey]*cacheEntry{}}

// Cached is Compile once per pattern and Options for the process. Fifty
// pasted formats that share a pattern compile it once. The cache holds at
// most 8,192 patterns and drops the least recently used.
func Cached(pattern string, o Options) (*Regex, error) {
	k := cacheKey{pattern, o}
	if re := lookup(k); re != nil {
		return re, nil
	}
	re, err := Compile(pattern, o)
	if err != nil {
		return nil, err
	}
	cache.Lock()
	defer cache.Unlock()
	cache.tick++
	if e, ok := cache.at[k]; ok {
		e.used = cache.tick
		return e.re, nil
	}
	cache.at[k] = &cacheEntry{re: re, used: cache.tick}
	if len(cache.at) > cacheSize {
		evictOldest()
	}
	return re, nil
}

func lookup(k cacheKey) *Regex {
	cache.Lock()
	defer cache.Unlock()
	e, ok := cache.at[k]
	if !ok {
		return nil
	}
	cache.tick++
	e.used = cache.tick
	return e.re
}

// evictOldest scans every entry. It runs once per compile past the bound,
// and the compile costs more than the scan.
func evictOldest() {
	var oldest cacheKey
	least := cache.tick
	for k, e := range cache.at {
		if e.used < least {
			oldest, least = k, e.used
		}
	}
	delete(cache.at, oldest)
}
