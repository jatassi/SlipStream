package rsssync

// WebSocket event types for RSS sync.
const (
	EventStarted   = "rss-sync:started"
	EventProgress  = "rss-sync:progress"
	EventCompleted = "rss-sync:completed"
	EventFailed    = "rss-sync:failed"
)

type StartedEvent struct {
	IndexerCount int `json:"indexerCount"`
}

type ProgressEvent struct {
	Indexer       string `json:"indexer"`
	ReleasesFound int    `json:"releasesFound"`
	Matched       int    `json:"matched"`
}

type CompletedEvent struct {
	TotalReleases int `json:"totalReleases"`
	Matched       int `json:"matched"`
	Grabbed       int `json:"grabbed"`
	ElapsedMs     int `json:"elapsed"`
}

type FailedEvent struct {
	Error string `json:"error"`
}
