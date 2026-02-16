package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
)

const (
	DefaultRequestsPerMinute = 60
	DefaultWindowDuration    = time.Minute
)

type userBucket struct {
	count     int64
	resetTime time.Time
}

type SearchLimiter struct {
	mu       sync.RWMutex
	buckets  map[int64]*userBucket
	limit    int64
	window   time.Duration
	getLimit func() int64
}

func NewSearchLimiter(limitGetter func() int64) *SearchLimiter {
	return &SearchLimiter{
		buckets:  make(map[int64]*userBucket),
		limit:    DefaultRequestsPerMinute,
		window:   DefaultWindowDuration,
		getLimit: limitGetter,
	}
}

func (l *SearchLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := portalmw.GetPortalUser(c)
			if claims == nil {
				return next(c)
			}

			if !l.allow(claims.UserID) {
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}

			return next(c)
		}
	}
}

func (l *SearchLimiter) allow(userID int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	limit := l.limit
	if l.getLimit != nil {
		limit = l.getLimit()
	}

	bucket, exists := l.buckets[userID]
	if !exists || now.After(bucket.resetTime) {
		l.buckets[userID] = &userBucket{
			count:     1,
			resetTime: now.Add(l.window),
		}
		return true
	}

	if bucket.count >= limit {
		return false
	}

	bucket.count++
	return true
}

func (l *SearchLimiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for userID, bucket := range l.buckets {
		if now.After(bucket.resetTime) {
			delete(l.buckets, userID)
		}
	}
}

func (l *SearchLimiter) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			l.Cleanup()
		}
	}()
}
