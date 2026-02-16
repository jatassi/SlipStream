package indexer

// WebSocket event types for indexer operations.
const (
	EventSearchStarted   = "search:started"
	EventSearchCompleted = "search:completed"
	EventGrabStarted     = "grab:started"
	EventGrabCompleted   = "grab:completed"
	EventIndexerStatus   = "indexer:status"
)

// SearchStartedPayload is sent when a search begins.
type SearchStartedPayload struct {
	Query      string  `json:"query,omitempty"`
	Type       string  `json:"type"`
	IndexerIDs []int64 `json:"indexerIds,omitempty"`
}

// SearchCompletedPayload is sent when a search finishes.
type SearchCompletedPayload struct {
	Query        string   `json:"query,omitempty"`
	Type         string   `json:"type"`
	TotalResults int      `json:"totalResults"`
	IndexersUsed int      `json:"indexersUsed"`
	Errors       []string `json:"errors,omitempty"`
	ElapsedMs    int64    `json:"elapsedMs"`
}

// GrabStartedPayload is sent when a grab begins.
type GrabStartedPayload struct {
	Title     string `json:"title"`
	IndexerID int64  `json:"indexerId"`
	Protocol  string `json:"protocol"`
}

// GrabCompletedPayload is sent when a grab finishes.
type GrabCompletedPayload struct {
	Title      string `json:"title"`
	IndexerID  int64  `json:"indexerId"`
	Success    bool   `json:"success"`
	DownloadID string `json:"downloadId,omitempty"`
	ClientName string `json:"clientName,omitempty"`
	Error      string `json:"error,omitempty"`
}

// IndexerStatusPayload is sent when indexer status changes.
type IndexerStatusPayload struct {
	IndexerID   int64  `json:"indexerId"`
	IndexerName string `json:"indexerName"`
	Status      string `json:"status"` // healthy, warning, disabled
	Message     string `json:"message,omitempty"`
}
