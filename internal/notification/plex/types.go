package plex

import "time"

// Settings contains the configuration for a Plex notification
type Settings struct {
	AuthToken         string        `json:"authToken"`
	ServerID          string        `json:"serverId"`
	SectionIDs        []int         `json:"sectionIds"`
	PathMappings      []PathMapping `json:"pathMappings,omitempty"`
	UpdateLibrary     bool          `json:"updateLibrary"`
	UsePartialRefresh bool          `json:"usePartialRefresh"`
}

// PathMapping maps a local path to a Plex server path
type PathMapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// PlexServer represents a Plex Media Server
type PlexServer struct {
	Name            string       `json:"name"`
	ClientID        string       `json:"clientIdentifier"`
	AccessToken     string       `json:"accessToken,omitempty"`
	Connections     []Connection `json:"connections"`
	Owned           bool         `json:"owned"`
	Home            bool         `json:"home"`
	SourceTitle     string       `json:"sourceTitle,omitempty"`
	PublicAddress   string       `json:"publicAddress,omitempty"`
	Product         string       `json:"product,omitempty"`
	ProductVersion  string       `json:"productVersion,omitempty"`
	Platform        string       `json:"platform,omitempty"`
	PlatformVersion string       `json:"platformVersion,omitempty"`
	Provides        string       `json:"provides,omitempty"`
}

// Connection represents a server connection endpoint
type Connection struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	URI      string `json:"uri"`
	Local    bool   `json:"local"`
	Relay    bool   `json:"relay"`
}

// LibrarySection represents a Plex library section
type LibrarySection struct {
	Key       int    `json:"key"`
	Title     string `json:"title"`
	Type      string `json:"type"` // "movie", "show", "artist", etc.
	Agent     string `json:"agent,omitempty"`
	Scanner   string `json:"scanner,omitempty"`
	Language  string `json:"language,omitempty"`
	Locations []struct {
		ID   int    `json:"id"`
		Path string `json:"path"`
	} `json:"locations,omitempty"`
}

// PINResponse represents the response from creating a PIN
type PINResponse struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	ExpiresIn int       `json:"expiresIn"`
	ExpiresAt time.Time `json:"expiresAt"`
	AuthToken string    `json:"authToken,omitempty"`
	Trusted   bool      `json:"trusted"`
}

// PINStatus represents the status of a PIN authentication
type PINStatus struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	AuthToken string    `json:"authToken,omitempty"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// RefreshQueueItem represents an item in the Plex refresh queue
type RefreshQueueItem struct {
	ID             int64     `json:"id"`
	NotificationID int64     `json:"notificationId"`
	ServerID       string    `json:"serverId"`
	SectionKey     int       `json:"sectionKey"`
	Path           string    `json:"path,omitempty"`
	QueuedAt       time.Time `json:"queuedAt"`
}
