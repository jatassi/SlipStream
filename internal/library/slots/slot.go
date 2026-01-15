package slots

import (
	"time"
)

// Slot represents a version slot for multi-version support.
// Req 1.1.1: Maximum of 3 version slots supported
// Req 1.1.2: Slots are globally defined and shared between Movies and TV Series
type Slot struct {
	ID               int64   `json:"id"`
	SlotNumber       int     `json:"slotNumber"`       // 1, 2, or 3
	Name             string  `json:"name"`             // Req 1.1.3: User-defined custom name
	Enabled          bool    `json:"enabled"`          // Req 1.1.4: Enable/disable toggle
	QualityProfileID *int64  `json:"qualityProfileId"` // Req 1.1.5: Assigned profile ID
	DisplayOrder     int     `json:"displayOrder"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`

	// Root folder assignments (for multi-version mode)
	// Req 22.1.1-22.1.2: Each slot can have a dedicated root folder per media type
	MovieRootFolderID *int64 `json:"movieRootFolderId"`
	TVRootFolderID    *int64 `json:"tvRootFolderId"`

	// Populated when loading with profile
	QualityProfile *SlotProfile `json:"qualityProfile,omitempty"`

	// Populated when loading with root folder info
	MovieRootFolder *SlotRootFolder `json:"movieRootFolder,omitempty"`
	TVRootFolder    *SlotRootFolder `json:"tvRootFolder,omitempty"`

	// Populated when checking file counts
	FileCount int64 `json:"fileCount,omitempty"`
}

// SlotProfile is a simplified profile for slot display.
type SlotProfile struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Cutoff int    `json:"cutoff"`
}

// SlotRootFolder is a simplified root folder for slot display.
type SlotRootFolder struct {
	ID   int64  `json:"id"`
	Path string `json:"path"`
	Name string `json:"name"`
}

// MultiVersionSettings contains the global multi-version feature settings.
// Req 1.2.1: Global master toggle
// Req 1.2.2: When disabled, system behaves as single-version
type MultiVersionSettings struct {
	Enabled         bool       `json:"enabled"`
	DryRunCompleted bool       `json:"dryRunCompleted"`
	LastMigrationAt *time.Time `json:"lastMigrationAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// UpdateSlotInput is the input for updating a slot.
type UpdateSlotInput struct {
	Name              string `json:"name"`
	Enabled           bool   `json:"enabled"`
	QualityProfileID  *int64 `json:"qualityProfileId"`
	DisplayOrder      int    `json:"displayOrder"`
	MovieRootFolderID *int64 `json:"movieRootFolderId"`
	TVRootFolderID    *int64 `json:"tvRootFolderId"`
}

// UpdateMultiVersionSettingsInput is the input for updating multi-version settings.
type UpdateMultiVersionSettingsInput struct {
	Enabled bool `json:"enabled"`
}

// SlotWithProfile extends Slot with full profile information for matching operations.
type SlotWithProfile struct {
	Slot
	ProfileItems                string `json:"-"`
	ProfileHDRSettings          string `json:"-"`
	ProfileVideoCodecSettings   string `json:"-"`
	ProfileAudioCodecSettings   string `json:"-"`
	ProfileAudioChannelSettings string `json:"-"`
}
