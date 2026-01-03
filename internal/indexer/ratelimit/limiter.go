// Package ratelimit provides rate limiting for indexer operations.
package ratelimit

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// Config defines rate limit configuration.
type Config struct {
	// QueryLimit is the maximum number of queries allowed in the period
	QueryLimit int
	// QueryPeriod is the time period for query limiting
	QueryPeriod time.Duration
	// GrabLimit is the maximum number of grabs allowed in the period
	GrabLimit int
	// GrabPeriod is the time period for grab limiting
	GrabPeriod time.Duration
}

// DefaultConfig returns the default rate limit configuration.
func DefaultConfig() Config {
	return Config{
		QueryLimit:  100,
		QueryPeriod: time.Hour,
		GrabLimit:   25,
		GrabPeriod:  time.Hour,
	}
}

// Limiter tracks query/grab counts per indexer.
type Limiter struct {
	queries *sqlc.Queries
	logger  zerolog.Logger
	config  Config

	// In-memory rate limiting for immediate checks
	mu          sync.RWMutex
	queryCounts map[int64]*rateBucket
	grabCounts  map[int64]*rateBucket
}

// rateBucket tracks rate limit state for a single indexer.
type rateBucket struct {
	count     int
	resetTime time.Time
}

// NewLimiter creates a new rate limiter.
func NewLimiter(db *sql.DB, config Config, logger zerolog.Logger) *Limiter {
	return &Limiter{
		queries:     sqlc.New(db),
		logger:      logger.With().Str("component", "rate-limiter").Logger(),
		config:      config,
		queryCounts: make(map[int64]*rateBucket),
		grabCounts:  make(map[int64]*rateBucket),
	}
}

// CheckQueryLimit returns whether the indexer has reached its query limit.
func (l *Limiter) CheckQueryLimit(ctx context.Context, indexerID int64) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.getOrCreateQueryBucket(indexerID)

	// Reset if period has passed
	if time.Now().After(bucket.resetTime) {
		bucket.count = 0
		bucket.resetTime = time.Now().Add(l.config.QueryPeriod)
	}

	if bucket.count >= l.config.QueryLimit {
		l.logger.Warn().
			Int64("indexerId", indexerID).
			Int("count", bucket.count).
			Int("limit", l.config.QueryLimit).
			Msg("Query rate limit reached")
		return true, nil
	}

	return false, nil
}

// CheckGrabLimit returns whether the indexer has reached its grab limit.
func (l *Limiter) CheckGrabLimit(ctx context.Context, indexerID int64) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.getOrCreateGrabBucket(indexerID)

	// Reset if period has passed
	if time.Now().After(bucket.resetTime) {
		bucket.count = 0
		bucket.resetTime = time.Now().Add(l.config.GrabPeriod)
	}

	if bucket.count >= l.config.GrabLimit {
		l.logger.Warn().
			Int64("indexerId", indexerID).
			Int("count", bucket.count).
			Int("limit", l.config.GrabLimit).
			Msg("Grab rate limit reached")
		return true, nil
	}

	return false, nil
}

// RecordQuery records a query for rate limiting purposes.
func (l *Limiter) RecordQuery(ctx context.Context, indexerID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.getOrCreateQueryBucket(indexerID)

	// Reset if period has passed
	if time.Now().After(bucket.resetTime) {
		bucket.count = 0
		bucket.resetTime = time.Now().Add(l.config.QueryPeriod)
	}

	bucket.count++

	l.logger.Debug().
		Int64("indexerId", indexerID).
		Int("queryCount", bucket.count).
		Int("queryLimit", l.config.QueryLimit).
		Msg("Recorded query")
}

// RecordGrab records a grab for rate limiting purposes.
func (l *Limiter) RecordGrab(ctx context.Context, indexerID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.getOrCreateGrabBucket(indexerID)

	// Reset if period has passed
	if time.Now().After(bucket.resetTime) {
		bucket.count = 0
		bucket.resetTime = time.Now().Add(l.config.GrabPeriod)
	}

	bucket.count++

	l.logger.Debug().
		Int64("indexerId", indexerID).
		Int("grabCount", bucket.count).
		Int("grabLimit", l.config.GrabLimit).
		Msg("Recorded grab")
}

// GetQueryCount returns the current query count for an indexer.
func (l *Limiter) GetQueryCount(ctx context.Context, indexerID int64) (int, int, time.Time) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if bucket, exists := l.queryCounts[indexerID]; exists {
		if time.Now().After(bucket.resetTime) {
			return 0, l.config.QueryLimit, time.Now().Add(l.config.QueryPeriod)
		}
		return bucket.count, l.config.QueryLimit, bucket.resetTime
	}

	return 0, l.config.QueryLimit, time.Now().Add(l.config.QueryPeriod)
}

// GetGrabCount returns the current grab count for an indexer.
func (l *Limiter) GetGrabCount(ctx context.Context, indexerID int64) (int, int, time.Time) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if bucket, exists := l.grabCounts[indexerID]; exists {
		if time.Now().After(bucket.resetTime) {
			return 0, l.config.GrabLimit, time.Now().Add(l.config.GrabPeriod)
		}
		return bucket.count, l.config.GrabLimit, bucket.resetTime
	}

	return 0, l.config.GrabLimit, time.Now().Add(l.config.GrabPeriod)
}

// GetLimits returns the current rate limit status for an indexer.
func (l *Limiter) GetLimits(ctx context.Context, indexerID int64) *LimitStatus {
	queryCount, queryLimit, queryReset := l.GetQueryCount(ctx, indexerID)
	grabCount, grabLimit, grabReset := l.GetGrabCount(ctx, indexerID)

	return &LimitStatus{
		IndexerID:       indexerID,
		QueryCount:      queryCount,
		QueryLimit:      queryLimit,
		QueryResetTime:  queryReset,
		GrabCount:       grabCount,
		GrabLimit:       grabLimit,
		GrabResetTime:   grabReset,
		QueryLimited:    queryCount >= queryLimit,
		GrabLimited:     grabCount >= grabLimit,
	}
}

// Reset clears the rate limit state for an indexer.
func (l *Limiter) Reset(indexerID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.queryCounts, indexerID)
	delete(l.grabCounts, indexerID)

	l.logger.Info().
		Int64("indexerId", indexerID).
		Msg("Reset rate limits")
}

// ResetAll clears all rate limit state.
func (l *Limiter) ResetAll() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.queryCounts = make(map[int64]*rateBucket)
	l.grabCounts = make(map[int64]*rateBucket)

	l.logger.Info().Msg("Reset all rate limits")
}

// CheckDatabaseLimits checks rate limits using the database history.
// This is more accurate but slower than in-memory checks.
func (l *Limiter) CheckDatabaseLimits(ctx context.Context, indexerID int64) (*LimitStatus, error) {
	// Calculate the start time for the query period
	queryStart := time.Now().Add(-l.config.QueryPeriod)
	grabStart := time.Now().Add(-l.config.GrabPeriod)

	// Count queries from database
	queryCount, err := l.queries.CountIndexerQueries(ctx, sqlc.CountIndexerQueriesParams{
		IndexerID: indexerID,
		CreatedAt: sql.NullTime{Time: queryStart, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count queries: %w", err)
	}

	// Count grabs from database
	grabCount, err := l.queries.CountIndexerGrabs(ctx, sqlc.CountIndexerGrabsParams{
		IndexerID: indexerID,
		CreatedAt: sql.NullTime{Time: grabStart, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count grabs: %w", err)
	}

	return &LimitStatus{
		IndexerID:       indexerID,
		QueryCount:      int(queryCount),
		QueryLimit:      l.config.QueryLimit,
		QueryResetTime:  time.Now().Add(l.config.QueryPeriod),
		GrabCount:       int(grabCount),
		GrabLimit:       l.config.GrabLimit,
		GrabResetTime:   time.Now().Add(l.config.GrabPeriod),
		QueryLimited:    int(queryCount) >= l.config.QueryLimit,
		GrabLimited:     int(grabCount) >= l.config.GrabLimit,
	}, nil
}

// getOrCreateQueryBucket gets or creates a query rate bucket.
func (l *Limiter) getOrCreateQueryBucket(indexerID int64) *rateBucket {
	if bucket, exists := l.queryCounts[indexerID]; exists {
		return bucket
	}

	bucket := &rateBucket{
		count:     0,
		resetTime: time.Now().Add(l.config.QueryPeriod),
	}
	l.queryCounts[indexerID] = bucket
	return bucket
}

// getOrCreateGrabBucket gets or creates a grab rate bucket.
func (l *Limiter) getOrCreateGrabBucket(indexerID int64) *rateBucket {
	if bucket, exists := l.grabCounts[indexerID]; exists {
		return bucket
	}

	bucket := &rateBucket{
		count:     0,
		resetTime: time.Now().Add(l.config.GrabPeriod),
	}
	l.grabCounts[indexerID] = bucket
	return bucket
}

// LimitStatus represents the current rate limit status for an indexer.
type LimitStatus struct {
	IndexerID      int64     `json:"indexerId"`
	QueryCount     int       `json:"queryCount"`
	QueryLimit     int       `json:"queryLimit"`
	QueryResetTime time.Time `json:"queryResetTime"`
	GrabCount      int       `json:"grabCount"`
	GrabLimit      int       `json:"grabLimit"`
	GrabResetTime  time.Time `json:"grabResetTime"`
	QueryLimited   bool      `json:"queryLimited"`
	GrabLimited    bool      `json:"grabLimited"`
}
