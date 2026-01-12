package health

import (
	"encoding/json"
	"time"
)

// HealthStatus represents the health state of an item.
type HealthStatus string

const (
	StatusOK      HealthStatus = "ok"
	StatusWarning HealthStatus = "warning"
	StatusError   HealthStatus = "error"
)

// HealthCategory represents the category of health items.
type HealthCategory string

const (
	CategoryDownloadClients HealthCategory = "downloadClients"
	CategoryIndexers        HealthCategory = "indexers"
	CategoryRootFolders     HealthCategory = "rootFolders"
	CategoryMetadata        HealthCategory = "metadata"
	CategoryStorage         HealthCategory = "storage"
	CategoryImport          HealthCategory = "import"
)

// AllCategories returns all health categories in display order.
func AllCategories() []HealthCategory {
	return []HealthCategory{
		CategoryDownloadClients,
		CategoryIndexers,
		CategoryRootFolders,
		CategoryMetadata,
		CategoryStorage,
		CategoryImport,
	}
}

// HealthItem represents a single health-tracked item.
type HealthItem struct {
	ID        string         `json:"id"`
	Category  HealthCategory `json:"category"`
	Name      string         `json:"name"`
	Status    HealthStatus   `json:"status"`
	Message   string         `json:"message,omitempty"`
	Timestamp *time.Time     `json:"timestamp,omitempty"`
}

// MarshalJSON customizes JSON output to omit timestamp for OK status.
func (h HealthItem) MarshalJSON() ([]byte, error) {
	type Alias HealthItem
	alias := Alias(h)

	// Only include timestamp for non-OK statuses
	if h.Status == StatusOK {
		alias.Timestamp = nil
		alias.Message = ""
	}

	return json.Marshal(alias)
}

// CategorySummary provides counts for a health category.
type CategorySummary struct {
	Category HealthCategory `json:"category"`
	OK       int            `json:"ok"`
	Warning  int            `json:"warning"`
	Error    int            `json:"error"`
}

// Total returns the total number of items in the category.
func (c CategorySummary) Total() int {
	return c.OK + c.Warning + c.Error
}

// HasIssues returns true if there are any warning or error items.
func (c CategorySummary) HasIssues() bool {
	return c.Warning > 0 || c.Error > 0
}

// HealthResponse contains all health items grouped by category.
type HealthResponse struct {
	DownloadClients []HealthItem `json:"downloadClients"`
	Indexers        []HealthItem `json:"indexers"`
	RootFolders     []HealthItem `json:"rootFolders"`
	Metadata        []HealthItem `json:"metadata"`
	Storage         []HealthItem `json:"storage"`
	Import          []HealthItem `json:"import"`
}

// HealthSummary provides an overview of system health.
type HealthSummary struct {
	Categories []CategorySummary `json:"categories"`
	HasIssues  bool              `json:"hasIssues"`
}

// HealthUpdatePayload is the WebSocket payload for health updates.
type HealthUpdatePayload struct {
	Category  HealthCategory `json:"category"`
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Status    HealthStatus   `json:"status"`
	Message   string         `json:"message,omitempty"`
	Timestamp *time.Time     `json:"timestamp,omitempty"`
}

// IsBinaryCategory returns true if the category only supports OK/Error (no Warning).
// Download clients and root folders are binary health categories.
func IsBinaryCategory(category HealthCategory) bool {
	return category == CategoryDownloadClients || category == CategoryRootFolders
}
