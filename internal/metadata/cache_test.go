package metadata

import (
	"testing"
	"time"
)

func TestCache_SetGet(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Minute, MaxItems: 100})

	cache.Set("key1", "value1")

	val, ok := cache.Get("key1")
	if !ok {
		t.Error("expected key1 to exist")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}
}

func TestCache_GetMissing(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Minute, MaxItems: 100})

	_, ok := cache.Get("nonexistent")
	if ok {
		t.Error("expected key to not exist")
	}
}

func TestCache_Expiration(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: 50 * time.Millisecond, MaxItems: 100})

	cache.Set("key1", "value1")

	// Should exist immediately
	_, ok := cache.Get("key1")
	if !ok {
		t.Error("expected key1 to exist immediately")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, ok = cache.Get("key1")
	if ok {
		t.Error("expected key1 to be expired")
	}
}

func TestCache_SetWithTTL(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Hour, MaxItems: 100})

	cache.SetWithTTL("key1", "value1", 50*time.Millisecond)

	// Should exist immediately
	_, ok := cache.Get("key1")
	if !ok {
		t.Error("expected key1 to exist immediately")
	}

	// Wait for custom TTL
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, ok = cache.Get("key1")
	if ok {
		t.Error("expected key1 to be expired with custom TTL")
	}
}

func TestCache_Delete(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Minute, MaxItems: 100})

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, ok := cache.Get("key1")
	if ok {
		t.Error("expected key1 to be deleted")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Minute, MaxItems: 100})

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("expected cache to be empty, got %d items", cache.Len())
	}
}

func TestCache_Len(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Minute, MaxItems: 100})

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	if cache.Len() != 2 {
		t.Errorf("expected 2 items, got %d", cache.Len())
	}
}

func TestCache_Eviction(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Minute, MaxItems: 5})

	// Add more items than max
	for i := 0; i < 10; i++ {
		cache.Set(string(rune('a'+i)), i)
	}

	// Should have evicted some items
	if cache.Len() > 5 {
		t.Errorf("expected at most 5 items, got %d", cache.Len())
	}
}

func TestCache_GetMovieResults(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Minute, MaxItems: 100})

	results := []MovieResult{
		{ID: 1, Title: "Movie 1"},
		{ID: 2, Title: "Movie 2"},
	}
	cache.Set("movies:search:test", results)

	got, ok := cache.GetMovieResults("movies:search:test")
	if !ok {
		t.Error("expected results to exist")
	}
	if len(got) != 2 {
		t.Errorf("expected 2 results, got %d", len(got))
	}
	if got[0].Title != "Movie 1" {
		t.Errorf("expected Movie 1, got %s", got[0].Title)
	}
}

func TestCache_GetMovieResult(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Minute, MaxItems: 100})

	result := &MovieResult{ID: 1, Title: "Movie 1"}
	cache.Set("movies:1", result)

	got, ok := cache.GetMovieResult("movies:1")
	if !ok {
		t.Error("expected result to exist")
	}
	if got.Title != "Movie 1" {
		t.Errorf("expected Movie 1, got %s", got.Title)
	}
}

func TestCache_GetSeriesResults(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Minute, MaxItems: 100})

	results := []SeriesResult{
		{ID: 1, Title: "Series 1"},
		{ID: 2, Title: "Series 2"},
	}
	cache.Set("series:search:test", results)

	got, ok := cache.GetSeriesResults("series:search:test")
	if !ok {
		t.Error("expected results to exist")
	}
	if len(got) != 2 {
		t.Errorf("expected 2 results, got %d", len(got))
	}
}

func TestCache_GetSeriesResult(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Minute, MaxItems: 100})

	result := &SeriesResult{ID: 1, Title: "Series 1"}
	cache.Set("series:1", result)

	got, ok := cache.GetSeriesResult("series:1")
	if !ok {
		t.Error("expected result to exist")
	}
	if got.Title != "Series 1" {
		t.Errorf("expected Series 1, got %s", got.Title)
	}
}

func TestCache_TypeMismatch(t *testing.T) {
	cache := NewCache(CacheConfig{TTL: time.Minute, MaxItems: 100})

	// Store a string
	cache.Set("key", "string value")

	// Try to get as MovieResult
	_, ok := cache.GetMovieResult("key")
	if ok {
		t.Error("expected type mismatch to return false")
	}
}
