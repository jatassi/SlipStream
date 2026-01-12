package history

import "encoding/json"

// EventType represents the type of history event.
type EventType string

const (
	EventTypeGrabbed            EventType = "grabbed"
	EventTypeImported           EventType = "imported"
	EventTypeDeleted            EventType = "deleted"
	EventTypeFailed             EventType = "failed"
	EventTypeRenamed            EventType = "renamed"
	EventTypeAutoSearchDownload EventType = "autosearch_download"
	EventTypeAutoSearchUpgrade  EventType = "autosearch_upgrade"
	EventTypeAutoSearchFailed   EventType = "autosearch_failed"
	EventTypeImportStarted      EventType = "import_started"
	EventTypeImportCompleted    EventType = "import_completed"
	EventTypeImportFailed       EventType = "import_failed"
	EventTypeImportUpgrade      EventType = "import_upgrade"
)

// MediaType represents the type of media.
type MediaType string

const (
	MediaTypeMovie   MediaType = "movie"
	MediaTypeEpisode MediaType = "episode"
)

// Entry represents a history entry.
type Entry struct {
	ID         int64             `json:"id"`
	EventType  EventType         `json:"eventType"`
	MediaType  MediaType         `json:"mediaType"`
	MediaID    int64             `json:"mediaId"`
	Source     string            `json:"source,omitempty"`
	Quality    string            `json:"quality,omitempty"`
	Data       map[string]any    `json:"data,omitempty"`
	CreatedAt  string            `json:"createdAt"`
	MediaTitle string            `json:"mediaTitle,omitempty"`
}

// CreateInput contains fields for creating a history entry.
type CreateInput struct {
	EventType EventType
	MediaType MediaType
	MediaID   int64
	Source    string
	Quality   string
	Data      map[string]any
}

// ListOptions contains options for listing history.
type ListOptions struct {
	EventType string
	MediaType string
	Page      int
	PageSize  int
}

// ListResponse contains paginated history results.
type ListResponse struct {
	Items      []*Entry `json:"items"`
	Page       int      `json:"page"`
	PageSize   int      `json:"pageSize"`
	TotalCount int64    `json:"totalCount"`
	TotalPages int      `json:"totalPages"`
}

// AutoSearchDownloadData contains data for autosearch download events.
type AutoSearchDownloadData struct {
	ReleaseName string `json:"releaseName,omitempty"`
	Indexer     string `json:"indexer,omitempty"`
	ClientName  string `json:"clientName,omitempty"`
	DownloadID  string `json:"downloadId,omitempty"`
	Source      string `json:"source,omitempty"` // "manual", "scheduled", "add"
}

// AutoSearchUpgradeData contains data for autosearch upgrade events.
type AutoSearchUpgradeData struct {
	ReleaseName    string `json:"releaseName,omitempty"`
	Indexer        string `json:"indexer,omitempty"`
	ClientName     string `json:"clientName,omitempty"`
	DownloadID     string `json:"downloadId,omitempty"`
	OldQuality     string `json:"oldQuality,omitempty"`
	NewQuality     string `json:"newQuality,omitempty"`
	Source         string `json:"source,omitempty"`
}

// AutoSearchFailedData contains data for autosearch failure events.
type AutoSearchFailedData struct {
	Error   string `json:"error,omitempty"`
	Indexer string `json:"indexer,omitempty"`
	Source  string `json:"source,omitempty"`
}

// ImportEventData contains data for import-related history events.
type ImportEventData struct {
	SourcePath       string `json:"sourcePath,omitempty"`
	DestinationPath  string `json:"destinationPath,omitempty"`
	OriginalFilename string `json:"originalFilename,omitempty"`
	FinalFilename    string `json:"finalFilename,omitempty"`
	Quality          string `json:"quality,omitempty"`
	Source           string `json:"source,omitempty"` // "queue", "manual", "scan"
	Codec            string `json:"codec,omitempty"`
	Size             int64  `json:"size,omitempty"`
	Error            string `json:"error,omitempty"`
	IsUpgrade        bool   `json:"isUpgrade,omitempty"`
	PreviousFile     string `json:"previousFile,omitempty"`
	ClientName       string `json:"clientName,omitempty"`
}

// ToJSON converts a data struct to a JSON map.
func ToJSON(v any) (map[string]any, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(bytes, &result); err != nil {
		return nil, err
	}
	return result, nil
}
