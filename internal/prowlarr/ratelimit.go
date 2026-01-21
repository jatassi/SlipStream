package prowlarr

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// RateLimiter implements adaptive rate limiting for Prowlarr requests.
// It starts with no rate limiting and backs off when receiving 429 errors.
type RateLimiter struct {
	mu               sync.Mutex
	logger           zerolog.Logger
	minDelay         time.Duration
	maxDelay         time.Duration
	currentDelay     time.Duration
	lastRequest      time.Time
	consecutiveOK    int
	backoffFactor    float64
	recoveryRequests int
}

// RateLimiterConfig configures the rate limiter behavior.
type RateLimiterConfig struct {
	MinDelay         time.Duration
	MaxDelay         time.Duration
	BackoffFactor    float64
	RecoveryRequests int
	Logger           zerolog.Logger
}

// DefaultRateLimiterConfig returns sensible defaults for Prowlarr rate limiting.
func DefaultRateLimiterConfig(logger zerolog.Logger) RateLimiterConfig {
	return RateLimiterConfig{
		MinDelay:         0,              // No initial delay
		MaxDelay:         30 * time.Second,
		BackoffFactor:    2.0,
		RecoveryRequests: 5, // Reduce delay after 5 successful requests
		Logger:           logger,
	}
}

// NewRateLimiter creates a new adaptive rate limiter.
func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		logger:           cfg.Logger.With().Str("component", "prowlarr-ratelimiter").Logger(),
		minDelay:         cfg.MinDelay,
		maxDelay:         cfg.MaxDelay,
		currentDelay:     cfg.MinDelay,
		backoffFactor:    cfg.BackoffFactor,
		recoveryRequests: cfg.RecoveryRequests,
	}
}

// Wait blocks until the rate limit allows a request.
// Returns immediately if no rate limiting is active.
func (r *RateLimiter) Wait() {
	r.mu.Lock()
	delay := r.currentDelay
	lastReq := r.lastRequest
	r.mu.Unlock()

	if delay == 0 {
		return
	}

	elapsed := time.Since(lastReq)
	if elapsed < delay {
		sleepTime := delay - elapsed
		r.logger.Debug().
			Dur("delay", sleepTime).
			Msg("rate limiting: waiting before request")
		time.Sleep(sleepTime)
	}
}

// RecordSuccess should be called after a successful request.
// It gradually reduces the rate limit if enough successful requests occur.
func (r *RateLimiter) RecordSuccess() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastRequest = time.Now()
	r.consecutiveOK++

	if r.currentDelay > r.minDelay && r.consecutiveOK >= r.recoveryRequests {
		oldDelay := r.currentDelay
		r.currentDelay = time.Duration(float64(r.currentDelay) / r.backoffFactor)
		if r.currentDelay < r.minDelay {
			r.currentDelay = r.minDelay
		}
		r.consecutiveOK = 0

		r.logger.Debug().
			Dur("oldDelay", oldDelay).
			Dur("newDelay", r.currentDelay).
			Msg("rate limit recovered")
	}
}

// RecordRateLimited should be called when a 429 response is received.
// It increases the delay between requests.
func (r *RateLimiter) RecordRateLimited() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastRequest = time.Now()
	r.consecutiveOK = 0

	oldDelay := r.currentDelay
	if r.currentDelay == 0 {
		r.currentDelay = 1 * time.Second // Start with 1 second
	} else {
		r.currentDelay = time.Duration(float64(r.currentDelay) * r.backoffFactor)
	}

	if r.currentDelay > r.maxDelay {
		r.currentDelay = r.maxDelay
	}

	r.logger.Warn().
		Dur("oldDelay", oldDelay).
		Dur("newDelay", r.currentDelay).
		Msg("rate limited: backing off")
}

// RecordError should be called after a non-429 error.
// It resets the consecutive success counter but doesn't increase delay.
func (r *RateLimiter) RecordError() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastRequest = time.Now()
	r.consecutiveOK = 0
}

// GetCurrentDelay returns the current delay between requests.
func (r *RateLimiter) GetCurrentDelay() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.currentDelay
}

// Reset clears the rate limiter state.
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.currentDelay = r.minDelay
	r.consecutiveOK = 0
	r.lastRequest = time.Time{}

	r.logger.Info().Msg("rate limiter reset")
}

// IsRateLimited returns whether the rate limiter is currently active.
func (r *RateLimiter) IsRateLimited() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.currentDelay > 0
}
