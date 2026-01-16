// Package status provides indexer health tracking and status management.
package status

import (
	"time"
)

// IndexerStatus represents the health status of an indexer.
type IndexerStatus struct {
	IndexerID         int64      `json:"indexerId"`
	IndexerName       string     `json:"indexerName,omitempty"`
	InitialFailure    *time.Time `json:"initialFailure,omitempty"`
	MostRecentFailure *time.Time `json:"mostRecentFailure,omitempty"`
	EscalationLevel   int        `json:"escalationLevel"`
	DisabledTill      *time.Time `json:"disabledTill,omitempty"`
	LastRSSSync       *time.Time `json:"lastRssSync,omitempty"`
	LastSearch        *time.Time `json:"lastSearch,omitempty"`
	IsDisabled        bool       `json:"isDisabled"`
}

// HealthStatus represents the overall health of an indexer.
type HealthStatus string

const (
	HealthStatusHealthy  HealthStatus = "healthy"
	HealthStatusWarning  HealthStatus = "warning"
	HealthStatusDisabled HealthStatus = "disabled"
	HealthStatusUnknown  HealthStatus = "unknown"
)

// IndexerHealth provides a summary of indexer health.
type IndexerHealth struct {
	IndexerID   int64        `json:"indexerId"`
	IndexerName string       `json:"indexerName"`
	Status      HealthStatus `json:"status"`
	Message     string       `json:"message,omitempty"`
	LastSuccess *time.Time   `json:"lastSuccess,omitempty"`
	LastFailure *time.Time   `json:"lastFailure,omitempty"`
	DisabledFor *Duration    `json:"disabledFor,omitempty"`
}

// Duration is a JSON-serializable duration.
type Duration struct {
	time.Duration
}

// MarshalJSON implements json.Marshaler.
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

// BackoffConfig defines the backoff strategy for failed indexers.
type BackoffConfig struct {
	// InitialBackoff is the backoff duration after the first failure
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// Multiplier is the factor by which backoff increases
	Multiplier float64
	// MaxEscalation is the maximum escalation level
	MaxEscalation int
}

// DefaultBackoffConfig returns the default backoff configuration.
func DefaultBackoffConfig() BackoffConfig {
	return BackoffConfig{
		InitialBackoff: 5 * time.Minute,
		MaxBackoff:     3 * time.Hour,
		Multiplier:     2.0,
		MaxEscalation:  5,
	}
}

// BackoffPeriods returns the escalating backoff periods.
func BackoffPeriods() []time.Duration {
	return []time.Duration{
		5 * time.Minute,  // Level 1
		15 * time.Minute, // Level 2
		30 * time.Minute, // Level 3
		1 * time.Hour,    // Level 4
		3 * time.Hour,    // Level 5+
	}
}

// CalculateBackoff returns the backoff duration for the given escalation level.
func CalculateBackoff(escalationLevel int) time.Duration {
	periods := BackoffPeriods()
	if escalationLevel <= 0 {
		return 0
	}
	if escalationLevel > len(periods) {
		return periods[len(periods)-1]
	}
	return periods[escalationLevel-1]
}

// FailureInfo contains details about an indexer failure.
type FailureInfo struct {
	IndexerID int64     `json:"indexerId"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error"`
	Operation string    `json:"operation"` // search, rss, grab, test
}

// StatusStats provides statistics about indexer status.
type StatusStats struct {
	TotalIndexers    int `json:"totalIndexers"`
	HealthyIndexers  int `json:"healthyIndexers"`
	WarningIndexers  int `json:"warningIndexers"`
	DisabledIndexers int `json:"disabledIndexers"`
}

// CookieData contains cached session cookies for an indexer.
type CookieData struct {
	Cookies   string     // Serialized cookie string (name=value; name2=value2)
	ExpiresAt *time.Time // When the cookies expire
}
