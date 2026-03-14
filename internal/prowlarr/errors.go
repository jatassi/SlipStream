package prowlarr

import (
	"errors"
)

var (
	// Configuration errors
	ErrNotConfigured    = errors.New("prowlarr is not configured")
	ErrInvalidURL       = errors.New("invalid prowlarr URL")
	ErrInvalidAPIKey    = errors.New("invalid or missing API key")
	ErrValidationFailed = errors.New("prowlarr configuration validation failed")

	// Connection errors
	ErrConnectionFailed      = errors.New("prowlarr connection failed")
	ErrConnectionTimeout     = errors.New("prowlarr connection timed out")
	ErrSSLVerificationFailed = errors.New("SSL certificate verification failed")

	// Search errors
	ErrSearchFailed      = errors.New("prowlarr search failed")
	ErrSearchTimeout     = errors.New("prowlarr search timed out")
	ErrNoIndexersEnabled = errors.New("no indexers enabled in prowlarr")

	// Download/Grab errors
	ErrDownloadFailed   = errors.New("prowlarr download failed")
	ErrDownloadNotFound = errors.New("download not found")

	// Rate limiting errors
	ErrRateLimited = errors.New("prowlarr rate limit exceeded")

	// Not found errors
	ErrNotFound = errors.New("not found")
)
