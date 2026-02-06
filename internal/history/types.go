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
	// Req 17.1.1: Slot-related event types for multi-version tracking
	EventTypeSlotAssigned   EventType = "slot_assigned"
	EventTypeSlotReassigned EventType = "slot_reassigned"
	EventTypeSlotUnassigned EventType = "slot_unassigned"
	// Status consolidation: transitions not covered by existing events
	EventTypeStatusChanged EventType = "status_changed"
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
	// Req 17.1.2: Slot information in history entries
	SlotID   *int64 `json:"slotId,omitempty"`
	SlotName string `json:"slotName,omitempty"`
}

// AutoSearchUpgradeData contains data for autosearch upgrade events.
type AutoSearchUpgradeData struct {
	ReleaseName string `json:"releaseName,omitempty"`
	Indexer     string `json:"indexer,omitempty"`
	ClientName  string `json:"clientName,omitempty"`
	DownloadID  string `json:"downloadId,omitempty"`
	OldQuality  string `json:"oldQuality,omitempty"`
	NewQuality  string `json:"newQuality,omitempty"`
	Source      string `json:"source,omitempty"`
	// Req 17.1.2: Slot information in history entries
	SlotID   *int64 `json:"slotId,omitempty"`
	SlotName string `json:"slotName,omitempty"`
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
	// Req 17.1.2: Slot information in history entries
	SlotID   *int64 `json:"slotId,omitempty"`
	SlotName string `json:"slotName,omitempty"`
}

// SlotEventData contains data for slot-specific history events (assign/reassign/unassign).
// Req 17.1.1: Log all slot-related events
type SlotEventData struct {
	SlotID       int64  `json:"slotId"`
	SlotName     string `json:"slotName"`
	FileID       int64  `json:"fileId,omitempty"`
	FilePath     string `json:"filePath,omitempty"`
	PreviousSlot *int64 `json:"previousSlotId,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// StatusChangedData contains data for status transition events not covered by existing event types.
type StatusChangedData struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Reason string `json:"reason"`
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
