package arrimport

import "encoding/json"

// Source types for config import — internal transfer types read by both SQLite and API readers.

type SourceDownloadClient struct {
	ID                       int64           `json:"id"`
	Name                     string          `json:"name"`
	Implementation           string          `json:"implementation"`
	Settings                 json.RawMessage `json:"settings"`
	Enabled                  bool            `json:"enabled"`
	Priority                 int             `json:"priority"`
	RemoveCompletedDownloads bool            `json:"removeCompletedDownloads"`
	RemoveFailedDownloads    bool            `json:"removeFailedDownloads"`
}

type SourceIndexer struct {
	ID                      int64           `json:"id"`
	Name                    string          `json:"name"`
	Implementation          string          `json:"implementation"`
	Settings                json.RawMessage `json:"settings"`
	EnableRss               bool            `json:"enableRss"`
	EnableAutomaticSearch   bool            `json:"enableAutomaticSearch"`
	EnableInteractiveSearch bool            `json:"enableInteractiveSearch"`
	Priority                int             `json:"priority"`
}

type SourceNotification struct {
	ID                    int64           `json:"id"`
	Name                  string          `json:"name"`
	Implementation        string          `json:"implementation"`
	Settings              json.RawMessage `json:"settings"`
	OnGrab                bool            `json:"onGrab"`
	OnDownload            bool            `json:"onDownload"`
	OnUpgrade             bool            `json:"onUpgrade"`
	OnHealthIssue         bool            `json:"onHealthIssue"`
	IncludeHealthWarnings bool            `json:"includeHealthWarnings"`
	OnHealthRestored      bool            `json:"onHealthRestored"`
	OnApplicationUpdate   bool            `json:"onApplicationUpdate"`
	// Sonarr-specific (zero for Radarr)
	OnSeriesAdd    bool `json:"onSeriesAdd"`
	OnSeriesDelete bool `json:"onSeriesDelete"`
	// Radarr-specific (zero for Sonarr)
	OnMovieAdded  bool `json:"onMovieAdded"`
	OnMovieDelete bool `json:"onMovieDelete"`
}

type SourceNamingConfig struct {
	// Shared
	ReplaceIllegalCharacters bool `json:"replaceIllegalCharacters"`
	ColonReplacementFormat   int  `json:"colonReplacementFormat"`
	// Sonarr-only
	RenameEpisodes        bool   `json:"renameEpisodes"`
	MultiEpisodeStyle     int    `json:"multiEpisodeStyle"`
	StandardEpisodeFormat string `json:"standardEpisodeFormat"`
	DailyEpisodeFormat    string `json:"dailyEpisodeFormat"`
	AnimeEpisodeFormat    string `json:"animeEpisodeFormat"`
	SeriesFolderFormat    string `json:"seriesFolderFormat"`
	SeasonFolderFormat    string `json:"seasonFolderFormat"`
	SpecialsFolderFormat  string `json:"specialsFolderFormat"`
	// Radarr-only
	RenameMovies        bool   `json:"renameMovies"`
	StandardMovieFormat string `json:"standardMovieFormat"`
	MovieFolderFormat   string `json:"movieFolderFormat"`
}

type SourceQualityProfileFull struct {
	ID             int64           `json:"id"`
	Name           string          `json:"name"`
	Cutoff         int             `json:"cutoff"`
	UpgradeAllowed bool            `json:"upgradeAllowed"`
	Items          json.RawMessage `json:"items"`
}

// Preview/report types for config import

type ConfigPreview struct {
	DownloadClients []ConfigPreviewItem  `json:"downloadClients"`
	Indexers        []ConfigPreviewItem  `json:"indexers"`
	Notifications   []ConfigPreviewItem  `json:"notifications"`
	QualityProfiles []ConfigPreviewItem  `json:"qualityProfiles"`
	NamingConfig    *NamingConfigPreview `json:"namingConfig,omitempty"`
	Warnings        []string             `json:"warnings"`
}

type ConfigPreviewItem struct {
	SourceID     int64  `json:"sourceId"`
	SourceName   string `json:"sourceName"`
	SourceType   string `json:"sourceType"`
	MappedType   string `json:"mappedType"`
	Status       string `json:"status"`
	StatusReason string `json:"statusReason,omitempty"`
}

type NamingConfigPreview struct {
	Source SourceNamingConfig `json:"source"`
	Status string             `json:"status"` // "different", "same"
}

type ConfigImportSelections struct {
	DownloadClientIDs  []int64 `json:"downloadClientIds"`
	IndexerIDs         []int64 `json:"indexerIds"`
	NotificationIDs    []int64 `json:"notificationIds"`
	QualityProfileIDs  []int64 `json:"qualityProfileIds"`
	ImportNamingConfig bool    `json:"importNamingConfig"`
}

type ConfigImportReport struct {
	DownloadClientsCreated int      `json:"downloadClientsCreated"`
	DownloadClientsSkipped int      `json:"downloadClientsSkipped"`
	IndexersCreated        int      `json:"indexersCreated"`
	IndexersSkipped        int      `json:"indexersSkipped"`
	NotificationsCreated   int      `json:"notificationsCreated"`
	NotificationsSkipped   int      `json:"notificationsSkipped"`
	QualityProfilesCreated int      `json:"qualityProfilesCreated"`
	QualityProfilesSkipped int      `json:"qualityProfilesSkipped"`
	NamingConfigImported   bool     `json:"namingConfigImported"`
	Warnings               []string `json:"warnings"`
	Errors                 []string `json:"errors"`
}

func newConfigImportReport() *ConfigImportReport {
	return &ConfigImportReport{Warnings: []string{}, Errors: []string{}}
}
