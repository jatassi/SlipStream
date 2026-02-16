package notification

import (
	"encoding/json"
	"time"

	"github.com/slipstream/slipstream/internal/notification/types"
)

// Re-export types from the types sub-package
type (
	NotifierType = types.NotifierType
	Notifier     = types.Notifier

	MediaInfo          = types.MediaInfo
	SeriesInfo         = types.SeriesInfo
	EpisodeInfo        = types.EpisodeInfo
	ReleaseInfo        = types.ReleaseInfo
	DownloadClientInfo = types.DownloadClientInfo
	CustomFormat       = types.CustomFormat
	MediaFileInfo      = types.MediaFileInfo
	SlotInfo           = types.SlotInfo
	GrabEvent          = types.GrabEvent
	ImportEvent        = types.ImportEvent
	UpgradeEvent       = types.UpgradeEvent
	MovieAddedEvent    = types.MovieAddedEvent
	MovieDeletedEvent  = types.MovieDeletedEvent
	SeriesAddedEvent   = types.SeriesAddedEvent
	SeriesDeletedEvent = types.SeriesDeletedEvent
	HealthEvent        = types.HealthEvent
	AppUpdateEvent     = types.AppUpdateEvent
	MessageEvent       = types.MessageEvent
)

// Re-export constants
const (
	NotifierDiscord      = types.NotifierDiscord
	NotifierTelegram     = types.NotifierTelegram
	NotifierWebhook      = types.NotifierWebhook
	NotifierEmail        = types.NotifierEmail
	NotifierSlack        = types.NotifierSlack
	NotifierPushover     = types.NotifierPushover
	NotifierGotify       = types.NotifierGotify
	NotifierNtfy         = types.NotifierNtfy
	NotifierApprise      = types.NotifierApprise
	NotifierPushbullet   = types.NotifierPushbullet
	NotifierJoin         = types.NotifierJoin
	NotifierProwl        = types.NotifierProwl
	NotifierSimplepush   = types.NotifierSimplepush
	NotifierSignal       = types.NotifierSignal
	NotifierCustomScript = types.NotifierCustomScript
	NotifierMock         = types.NotifierMock
	NotifierPlex         = types.NotifierPlex
)

// Config represents a notification configuration stored in the database
type Config struct {
	ID       int64           `json:"id"`
	Name     string          `json:"name"`
	Type     NotifierType    `json:"type"`
	Enabled  bool            `json:"enabled"`
	Settings json.RawMessage `json:"settings"`

	OnGrab           bool `json:"onGrab"`
	OnImport         bool `json:"onImport"`
	OnUpgrade        bool `json:"onUpgrade"`
	OnMovieAdded     bool `json:"onMovieAdded"`
	OnMovieDeleted   bool `json:"onMovieDeleted"`
	OnSeriesAdded    bool `json:"onSeriesAdded"`
	OnSeriesDeleted  bool `json:"onSeriesDeleted"`
	OnHealthIssue    bool `json:"onHealthIssue"`
	OnHealthRestored bool `json:"onHealthRestored"`
	OnAppUpdate      bool `json:"onAppUpdate"`

	IncludeHealthWarnings bool    `json:"includeHealthWarnings"`
	Tags                  []int64 `json:"tags,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateInput is used when creating a new notification
type CreateInput struct {
	Name     string          `json:"name"`
	Type     NotifierType    `json:"type"`
	Enabled  bool            `json:"enabled"`
	Settings json.RawMessage `json:"settings"`

	OnGrab           bool `json:"onGrab"`
	OnImport         bool `json:"onImport"`
	OnUpgrade        bool `json:"onUpgrade"`
	OnMovieAdded     bool `json:"onMovieAdded"`
	OnMovieDeleted   bool `json:"onMovieDeleted"`
	OnSeriesAdded    bool `json:"onSeriesAdded"`
	OnSeriesDeleted  bool `json:"onSeriesDeleted"`
	OnHealthIssue    bool `json:"onHealthIssue"`
	OnHealthRestored bool `json:"onHealthRestored"`
	OnAppUpdate      bool `json:"onAppUpdate"`

	IncludeHealthWarnings bool    `json:"includeHealthWarnings"`
	Tags                  []int64 `json:"tags,omitempty"`
}

// UpdateInput is used when updating an existing notification
type UpdateInput struct {
	Name     *string          `json:"name,omitempty"`
	Type     *NotifierType    `json:"type,omitempty"`
	Enabled  *bool            `json:"enabled,omitempty"`
	Settings *json.RawMessage `json:"settings,omitempty"`

	OnGrab           *bool `json:"onGrab,omitempty"`
	OnImport         *bool `json:"onImport,omitempty"`
	OnUpgrade        *bool `json:"onUpgrade,omitempty"`
	OnMovieAdded     *bool `json:"onMovieAdded,omitempty"`
	OnMovieDeleted   *bool `json:"onMovieDeleted,omitempty"`
	OnSeriesAdded    *bool `json:"onSeriesAdded,omitempty"`
	OnSeriesDeleted  *bool `json:"onSeriesDeleted,omitempty"`
	OnHealthIssue    *bool `json:"onHealthIssue,omitempty"`
	OnHealthRestored *bool `json:"onHealthRestored,omitempty"`
	OnAppUpdate      *bool `json:"onAppUpdate,omitempty"`

	IncludeHealthWarnings *bool    `json:"includeHealthWarnings,omitempty"`
	Tags                  *[]int64 `json:"tags,omitempty"`
}

// Status tracks notification failures for backoff logic
type Status struct {
	NotificationID    int64      `json:"notificationId"`
	InitialFailure    *time.Time `json:"initialFailure,omitempty"`
	MostRecentFailure *time.Time `json:"mostRecentFailure,omitempty"`
	EscalationLevel   int        `json:"escalationLevel"`
	DisabledTill      *time.Time `json:"disabledTill,omitempty"`
}

// TestResult contains the result of testing a notification
type TestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
