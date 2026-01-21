package prowlarr

import (
	"errors"
	"fmt"
)

var (
	// Configuration errors
	ErrNotConfigured    = errors.New("prowlarr is not configured")
	ErrInvalidURL       = errors.New("invalid prowlarr URL")
	ErrInvalidAPIKey    = errors.New("invalid or missing API key")
	ErrValidationFailed = errors.New("prowlarr configuration validation failed")

	// Connection errors
	ErrConnectionFailed   = errors.New("prowlarr connection failed")
	ErrConnectionTimeout  = errors.New("prowlarr connection timed out")
	ErrSSLVerificationFailed = errors.New("SSL certificate verification failed")

	// Search errors
	ErrSearchFailed      = errors.New("prowlarr search failed")
	ErrSearchTimeout     = errors.New("prowlarr search timed out")
	ErrNoIndexersEnabled = errors.New("no indexers enabled in prowlarr")

	// Download/Grab errors
	ErrDownloadFailed    = errors.New("prowlarr download failed")
	ErrDownloadNotFound  = errors.New("download not found")

	// Rate limiting errors
	ErrRateLimited = errors.New("prowlarr rate limit exceeded")
)

// ProwlarrError wraps an error with additional context.
type ProwlarrError struct {
	Op      string // Operation that failed (e.g., "search", "download", "connect")
	Err     error  // Underlying error
	Message string // Additional context message
}

func (e *ProwlarrError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("prowlarr %s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("prowlarr %s: %v", e.Op, e.Err)
}

func (e *ProwlarrError) Unwrap() error {
	return e.Err
}

// WrapError creates a ProwlarrError wrapping the given error.
func WrapError(op string, err error, message string) error {
	if err == nil {
		return nil
	}
	return &ProwlarrError{
		Op:      op,
		Err:     err,
		Message: message,
	}
}

// IsNotConfigured returns true if the error indicates Prowlarr is not configured.
func IsNotConfigured(err error) bool {
	return errors.Is(err, ErrNotConfigured)
}

// IsConnectionError returns true if the error is a connection-related error.
func IsConnectionError(err error) bool {
	return errors.Is(err, ErrConnectionFailed) ||
		errors.Is(err, ErrConnectionTimeout) ||
		errors.Is(err, ErrSSLVerificationFailed)
}

// IsSearchError returns true if the error is a search-related error.
func IsSearchError(err error) bool {
	return errors.Is(err, ErrSearchFailed) ||
		errors.Is(err, ErrSearchTimeout) ||
		errors.Is(err, ErrNoIndexersEnabled)
}

// IsRateLimited returns true if the error indicates rate limiting.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}
