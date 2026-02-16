package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	DefaultIPRequestsPerMinute = 10
	DefaultIPWindowDuration    = time.Minute
	DefaultMaxFailedAttempts   = 5
	DefaultLockoutDuration     = 15 * time.Minute
	MaxLockoutDuration         = time.Hour
)

type ipBucket struct {
	count     int64
	resetTime time.Time
}

type accountLockout struct {
	failedAttempts int
	lockedUntil    time.Time
	lockoutCount   int
}

type AuthLimiter struct {
	mu              sync.RWMutex
	ipBuckets       map[string]*ipBucket
	accountLockouts map[string]*accountLockout

	ipLimit             int64
	ipWindow            time.Duration
	maxFailedAttempts   int
	baseLockoutDuration time.Duration
}

func NewAuthLimiter() *AuthLimiter {
	return &AuthLimiter{
		ipBuckets:           make(map[string]*ipBucket),
		accountLockouts:     make(map[string]*accountLockout),
		ipLimit:             DefaultIPRequestsPerMinute,
		ipWindow:            DefaultIPWindowDuration,
		maxFailedAttempts:   DefaultMaxFailedAttempts,
		baseLockoutDuration: DefaultLockoutDuration,
	}
}

func (l *AuthLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			if !l.allowIP(ip) {
				return echo.NewHTTPError(http.StatusTooManyRequests, "too many requests, please try again later")
			}

			return next(c)
		}
	}
}

func (l *AuthLimiter) allowIP(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	bucket, exists := l.ipBuckets[ip]
	if !exists || now.After(bucket.resetTime) {
		l.ipBuckets[ip] = &ipBucket{
			count:     1,
			resetTime: now.Add(l.ipWindow),
		}
		return true
	}

	if bucket.count >= l.ipLimit {
		return false
	}

	bucket.count++
	return true
}

func (l *AuthLimiter) IsAccountLocked(username string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	lockout, exists := l.accountLockouts[username]
	if !exists {
		return false
	}

	return time.Now().Before(lockout.lockedUntil)
}

func (l *AuthLimiter) GetLockoutRemaining(username string) time.Duration {
	l.mu.RLock()
	defer l.mu.RUnlock()

	lockout, exists := l.accountLockouts[username]
	if !exists {
		return 0
	}

	remaining := time.Until(lockout.lockedUntil)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (l *AuthLimiter) RecordFailedAttempt(username string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	lockout, exists := l.accountLockouts[username]
	if !exists {
		lockout = &accountLockout{}
		l.accountLockouts[username] = lockout
	}

	if time.Now().After(lockout.lockedUntil) && lockout.failedAttempts >= l.maxFailedAttempts {
		lockout.failedAttempts = 0
	}

	lockout.failedAttempts++

	if lockout.failedAttempts >= l.maxFailedAttempts {
		lockout.lockoutCount++
		duration := l.baseLockoutDuration * time.Duration(lockout.lockoutCount)
		if duration > MaxLockoutDuration {
			duration = MaxLockoutDuration
		}
		lockout.lockedUntil = time.Now().Add(duration)
	}
}

func (l *AuthLimiter) RecordSuccessfulLogin(username string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.accountLockouts, username)
}

func (l *AuthLimiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	for ip, bucket := range l.ipBuckets {
		if now.After(bucket.resetTime) {
			delete(l.ipBuckets, ip)
		}
	}

	for username, lockout := range l.accountLockouts {
		if now.After(lockout.lockedUntil) && lockout.failedAttempts < l.maxFailedAttempts {
			delete(l.accountLockouts, username)
		}
	}
}

func (l *AuthLimiter) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			l.Cleanup()
		}
	}()
}
