package startup

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// RetryConfig configures the exponential backoff retry behavior.
type RetryConfig struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	MaxAttempts  int
	Multiplier   float64
}

// DefaultRetryConfig returns sensible defaults for network retry.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		InitialDelay: 5 * time.Second,
		MaxDelay:     5 * time.Minute,
		MaxAttempts:  5,
		Multiplier:   2.0,
	}
}

// IsNetworkError checks if an error is likely due to network unavailability.
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	var dnsErr *net.DNSError
	if errors.As(err, &netErr) || errors.As(err, &dnsErr) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	networkIndicators := []string{
		"connection refused",
		"no such host",
		"timeout",
		"network is unreachable",
		"no route to host",
		"host is down",
		"dial tcp",
		"dial udp",
		"i/o timeout",
		"connection reset",
		"temporary failure in name resolution",
	}
	for _, indicator := range networkIndicators {
		if strings.Contains(errStr, indicator) {
			return true
		}
	}

	return false
}

// WithRetry executes fn with exponential backoff retry for network errors only.
// Non-network errors fail immediately without retry.
func WithRetry(ctx context.Context, name string, cfg RetryConfig, fn func() error, logger *zerolog.Logger) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			if attempt > 1 {
				logger.Info().Str("operation", name).Int("attempt", attempt).Msg("operation succeeded after retry")
			}
			return nil
		}

		lastErr = err

		if !IsNetworkError(err) {
			logger.Error().Err(err).Str("operation", name).Msg("non-network error, not retrying")
			return err
		}

		if attempt == cfg.MaxAttempts {
			break
		}

		delay = waitAndBackoff(ctx, logger, name, attempt, cfg, delay, err)
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	logger.Error().Err(lastErr).Str("operation", name).Int("attempts", cfg.MaxAttempts).
		Msg("operation failed after all retries")
	return lastErr
}

func waitAndBackoff(ctx context.Context, logger *zerolog.Logger, name string, attempt int, cfg RetryConfig, delay time.Duration, err error) time.Duration {
	logger.Warn().
		Err(err).
		Str("operation", name).
		Int("attempt", attempt).
		Int("maxAttempts", cfg.MaxAttempts).
		Dur("nextRetryIn", delay).
		Msg("network error, will retry")

	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}

	next := time.Duration(float64(delay) * cfg.Multiplier)
	if next > cfg.MaxDelay {
		next = cfg.MaxDelay
	}
	return next
}
