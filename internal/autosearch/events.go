package autosearch

// WebSocket event types for automatic search operations.
const (
	// Per-item search events
	EventAutoSearchStarted   = "autosearch:started"
	EventAutoSearchCompleted = "autosearch:completed"
	EventAutoSearchFailed    = "autosearch:failed"

	// Scheduled task events
	EventAutoSearchTaskStarted   = "autosearch:task:started"
	EventAutoSearchTaskProgress  = "autosearch:task:progress"
	EventAutoSearchTaskCompleted = "autosearch:task:completed"
)

// AutoSearchStartedPayload is sent when an automatic search begins.
type AutoSearchStartedPayload struct {
	MediaType MediaType    `json:"mediaType"`
	MediaID   int64        `json:"mediaId"`
	Title     string       `json:"title"`
	Source    SearchSource `json:"source"`
}

// AutoSearchCompletedPayload is sent when an automatic search finishes successfully.
type AutoSearchCompletedPayload struct {
	MediaType   MediaType `json:"mediaType"`
	MediaID     int64     `json:"mediaId"`
	Title       string    `json:"title"`
	Found       bool      `json:"found"`
	Downloaded  bool      `json:"downloaded"`
	Upgraded    bool      `json:"upgraded"`
	ReleaseName string    `json:"releaseName,omitempty"`
	ClientName  string    `json:"clientName,omitempty"`
}

// AutoSearchFailedPayload is sent when an automatic search fails.
type AutoSearchFailedPayload struct {
	MediaType MediaType `json:"mediaType"`
	MediaID   int64     `json:"mediaId"`
	Title     string    `json:"title"`
	Error     string    `json:"error"`
}

// AutoSearchTaskStartedPayload is sent when the scheduled task begins.
type AutoSearchTaskStartedPayload struct {
	TotalItems int `json:"totalItems"`
}

// AutoSearchTaskProgressPayload is sent during scheduled task execution.
type AutoSearchTaskProgressPayload struct {
	CurrentItem  int    `json:"currentItem"`
	TotalItems   int    `json:"totalItems"`
	CurrentTitle string `json:"currentTitle"`
}

// AutoSearchTaskCompletedPayload is sent when the scheduled task finishes.
type AutoSearchTaskCompletedPayload struct {
	TotalSearched int   `json:"totalSearched"`
	Found         int   `json:"found"`
	Downloaded    int   `json:"downloaded"`
	Failed        int   `json:"failed"`
	ElapsedMs     int64 `json:"elapsedMs"`
}
