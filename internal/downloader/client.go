package downloader

// Client defines the interface for download clients.
type Client interface {
	// Name returns the client name.
	Name() string

	// Test verifies the client connection.
	Test() error

	// Add adds a download to the client.
	Add(download Download) error

	// List returns all downloads.
	List() ([]Download, error)

	// Remove removes a download.
	Remove(id string) error
}

// Download represents a download in progress or completed.
type Download struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Status   string  `json:"status"` // downloading, paused, completed, failed
	Progress float64 `json:"progress"`
	Size     int64   `json:"size"`
	Path     string  `json:"path,omitempty"`
}

// Type represents the type of download client.
type Type string

const (
	TypeTorrent Type = "torrent"
	TypeUsenet  Type = "usenet"
)
