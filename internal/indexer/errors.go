package indexer

import (
	"errors"
	"fmt"
)

// Error codes for categorizing indexer errors
const (
	ErrCodeAuthentication = "AUTH_ERROR"
	ErrCodeSearch         = "SEARCH_ERROR"
	ErrCodeDownload       = "DOWNLOAD_ERROR"
	ErrCodeConfiguration  = "CONFIG_ERROR"
	ErrCodeRateLimit      = "RATE_LIMIT_ERROR"
	ErrCodeNetwork        = "NETWORK_ERROR"
	ErrCodeParse          = "PARSE_ERROR"
	ErrCodeNotFound       = "NOT_FOUND_ERROR"
	ErrCodeTemporary      = "TEMPORARY_ERROR"
)

// IndexerError represents a categorized error from an indexer operation.
type IndexerError struct {
	Code       string // Error category code
	Message    string // Human-readable message
	IndexerID  int64  // ID of the affected indexer (0 if not applicable)
	IndexerName string // Name of the affected indexer
	Retryable  bool   // Whether the operation can be retried
	Cause      error  // Underlying error
}

// Error implements the error interface.
func (e *IndexerError) Error() string {
	if e.IndexerName != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.IndexerName, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *IndexerError) Unwrap() error {
	return e.Cause
}

// Is implements error matching for errors.Is().
func (e *IndexerError) Is(target error) bool {
	var t *IndexerError
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// Common error instances for comparison
var (
	ErrAuthentication = &IndexerError{Code: ErrCodeAuthentication, Message: "authentication failed"}
	ErrSearch         = &IndexerError{Code: ErrCodeSearch, Message: "search failed"}
	ErrDownload       = &IndexerError{Code: ErrCodeDownload, Message: "download failed"}
	ErrConfiguration  = &IndexerError{Code: ErrCodeConfiguration, Message: "configuration error"}
	ErrRateLimit      = &IndexerError{Code: ErrCodeRateLimit, Message: "rate limit exceeded"}
	ErrNetwork        = &IndexerError{Code: ErrCodeNetwork, Message: "network error"}
	ErrParse          = &IndexerError{Code: ErrCodeParse, Message: "parse error"}
	ErrNotFound       = &IndexerError{Code: ErrCodeNotFound, Message: "not found"}
	ErrTemporary      = &IndexerError{Code: ErrCodeTemporary, Message: "temporary error"}
)

// NewAuthError creates an authentication error.
func NewAuthError(indexerID int64, indexerName string, cause error) *IndexerError {
	return &IndexerError{
		Code:        ErrCodeAuthentication,
		Message:     "authentication failed",
		IndexerID:   indexerID,
		IndexerName: indexerName,
		Retryable:   false, // Auth errors usually need credential fixes
		Cause:       cause,
	}
}

// NewSearchError creates a search error.
func NewSearchError(indexerID int64, indexerName string, cause error) *IndexerError {
	return &IndexerError{
		Code:        ErrCodeSearch,
		Message:     "search failed",
		IndexerID:   indexerID,
		IndexerName: indexerName,
		Retryable:   true,
		Cause:       cause,
	}
}

// NewDownloadError creates a download error.
func NewDownloadError(indexerID int64, indexerName string, cause error) *IndexerError {
	return &IndexerError{
		Code:        ErrCodeDownload,
		Message:     "download failed",
		IndexerID:   indexerID,
		IndexerName: indexerName,
		Retryable:   true,
		Cause:       cause,
	}
}

// NewConfigError creates a configuration error.
func NewConfigError(indexerID int64, indexerName string, message string) *IndexerError {
	return &IndexerError{
		Code:        ErrCodeConfiguration,
		Message:     message,
		IndexerID:   indexerID,
		IndexerName: indexerName,
		Retryable:   false,
	}
}

// NewRateLimitError creates a rate limit error.
func NewRateLimitError(indexerID int64, indexerName string) *IndexerError {
	return &IndexerError{
		Code:        ErrCodeRateLimit,
		Message:     "rate limit exceeded",
		IndexerID:   indexerID,
		IndexerName: indexerName,
		Retryable:   true, // Can retry after backoff
	}
}

// NewNetworkError creates a network error.
func NewNetworkError(indexerID int64, indexerName string, cause error) *IndexerError {
	return &IndexerError{
		Code:        ErrCodeNetwork,
		Message:     "network error",
		IndexerID:   indexerID,
		IndexerName: indexerName,
		Retryable:   true,
		Cause:       cause,
	}
}

// NewParseError creates a parsing error.
func NewParseError(indexerID int64, indexerName string, message string, cause error) *IndexerError {
	return &IndexerError{
		Code:        ErrCodeParse,
		Message:     message,
		IndexerID:   indexerID,
		IndexerName: indexerName,
		Retryable:   false, // Parse errors are usually definition bugs
		Cause:       cause,
	}
}

// NewNotFoundError creates a not found error.
func NewNotFoundError(message string) *IndexerError {
	return &IndexerError{
		Code:      ErrCodeNotFound,
		Message:   message,
		Retryable: false,
	}
}

// NewTemporaryError creates a temporary error that should be retried.
func NewTemporaryError(indexerID int64, indexerName string, cause error) *IndexerError {
	return &IndexerError{
		Code:        ErrCodeTemporary,
		Message:     "temporary error, will retry",
		IndexerID:   indexerID,
		IndexerName: indexerName,
		Retryable:   true,
		Cause:       cause,
	}
}

// IsRetryable returns whether the error is retryable.
func IsRetryable(err error) bool {
	var indexerErr *IndexerError
	if errors.As(err, &indexerErr) {
		return indexerErr.Retryable
	}
	return false
}

// IsAuthError returns whether the error is an authentication error.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrAuthentication)
}

// IsRateLimitError returns whether the error is a rate limit error.
func IsRateLimitError(err error) bool {
	return errors.Is(err, ErrRateLimit)
}

// IsNetworkError returns whether the error is a network error.
func IsNetworkError(err error) bool {
	return errors.Is(err, ErrNetwork)
}

// GetErrorCode extracts the error code from an error.
func GetErrorCode(err error) string {
	var indexerErr *IndexerError
	if errors.As(err, &indexerErr) {
		return indexerErr.Code
	}
	return ""
}
