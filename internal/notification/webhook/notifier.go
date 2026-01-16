package webhook

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/notification/types"
)

// Settings contains webhook-specific configuration
type Settings struct {
	URL            string            `json:"url"`
	Method         string            `json:"method,omitempty"`
	Username       string            `json:"username,omitempty"`
	Password       string            `json:"password,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	ApplicationURL string            `json:"applicationUrl,omitempty"`
}

// Notifier sends notifications to a custom webhook endpoint
type Notifier struct {
	name       string
	settings   Settings
	httpClient *http.Client
	logger     zerolog.Logger
}

// New creates a new webhook notifier
func New(name string, settings Settings, httpClient *http.Client, logger zerolog.Logger) *Notifier {
	if settings.Method == "" {
		settings.Method = "POST"
	}
	return &Notifier{
		name:       name,
		settings:   settings,
		httpClient: httpClient,
		logger:     logger.With().Str("notifier", "webhook").Str("name", name).Logger(),
	}
}

func (n *Notifier) Type() types.NotifierType {
	return types.NotifierWebhook
}

func (n *Notifier) Name() string {
	return n.name
}

func (n *Notifier) Test(ctx context.Context) error {
	payload := Payload{
		EventType:      "test",
		InstanceName:   "SlipStream",
		ApplicationURL: n.settings.ApplicationURL,
		Message:        "Test notification from SlipStream",
		Timestamp:      time.Now().UTC(),
	}
	return n.send(ctx, payload)
}

func (n *Notifier) OnGrab(ctx context.Context, event types.GrabEvent) error {
	payload := Payload{
		EventType:      "grab",
		InstanceName:   "SlipStream",
		ApplicationURL: n.settings.ApplicationURL,
		Timestamp:      event.GrabbedAt,
		Movie:          n.mapMovie(event.Movie),
		Episode:        n.mapEpisode(event.Episode),
		Release:        n.mapRelease(event.Release),
		DownloadClient: &PayloadDownloadClient{
			ID:   event.DownloadClient.ID,
			Name: event.DownloadClient.Name,
			Type: event.DownloadClient.Type,
		},
		DownloadID:    event.DownloadID,
		CustomFormats: n.mapCustomFormats(event.Release.CustomFormats),
		Languages:     event.Release.Languages,
	}
	return n.send(ctx, payload)
}

func (n *Notifier) OnDownload(ctx context.Context, event types.DownloadEvent) error {
	payload := Payload{
		EventType:       "download",
		InstanceName:    "SlipStream",
		ApplicationURL:  n.settings.ApplicationURL,
		Timestamp:       event.ImportedAt,
		Movie:           n.mapMovie(event.Movie),
		Episode:         n.mapEpisode(event.Episode),
		Quality:         event.Quality,
		SourcePath:      event.SourcePath,
		DestinationPath: event.DestinationPath,
		IsUpgrade:       false,
		DownloadID:      event.DownloadID,
		CustomFormats:   n.mapCustomFormats(event.CustomFormats),
		MediaInfo:       n.mapMediaFileInfo(event.MediaInfo),
		Languages:       event.Languages,
	}
	return n.send(ctx, payload)
}

func (n *Notifier) OnUpgrade(ctx context.Context, event types.UpgradeEvent) error {
	payload := Payload{
		EventType:      "upgrade",
		InstanceName:   "SlipStream",
		ApplicationURL: n.settings.ApplicationURL,
		Timestamp:      event.UpgradedAt,
		Movie:          n.mapMovie(event.Movie),
		Episode:        n.mapEpisode(event.Episode),
		OldQuality:     event.OldQuality,
		Quality:        event.NewQuality,
		IsUpgrade:      true,
		CustomFormats:  n.mapCustomFormats(event.CustomFormats),
		MediaInfo:      n.mapMediaFileInfo(event.MediaInfo),
		Languages:      event.Languages,
	}
	return n.send(ctx, payload)
}

func (n *Notifier) OnMovieAdded(ctx context.Context, event types.MovieAddedEvent) error {
	payload := Payload{
		EventType:      "movieAdded",
		InstanceName:   "SlipStream",
		ApplicationURL: n.settings.ApplicationURL,
		Timestamp:      event.AddedAt,
		Movie:          n.mapMediaInfo(&event.Movie),
	}
	return n.send(ctx, payload)
}

func (n *Notifier) OnMovieDeleted(ctx context.Context, event types.MovieDeletedEvent) error {
	payload := Payload{
		EventType:      "movieDeleted",
		InstanceName:   "SlipStream",
		ApplicationURL: n.settings.ApplicationURL,
		Timestamp:      event.DeletedAt,
		Movie:          n.mapMediaInfo(&event.Movie),
		DeletedFiles:   event.DeletedFiles,
	}
	return n.send(ctx, payload)
}

func (n *Notifier) OnSeriesAdded(ctx context.Context, event types.SeriesAddedEvent) error {
	payload := Payload{
		EventType:      "seriesAdded",
		InstanceName:   "SlipStream",
		ApplicationURL: n.settings.ApplicationURL,
		Timestamp:      event.AddedAt,
		Series:         n.mapSeriesInfo(&event.Series),
	}
	return n.send(ctx, payload)
}

func (n *Notifier) OnSeriesDeleted(ctx context.Context, event types.SeriesDeletedEvent) error {
	payload := Payload{
		EventType:      "seriesDeleted",
		InstanceName:   "SlipStream",
		ApplicationURL: n.settings.ApplicationURL,
		Timestamp:      event.DeletedAt,
		Series:         n.mapSeriesInfo(&event.Series),
		DeletedFiles:   event.DeletedFiles,
	}
	return n.send(ctx, payload)
}

func (n *Notifier) OnHealthIssue(ctx context.Context, event types.HealthEvent) error {
	payload := Payload{
		EventType:      "healthIssue",
		InstanceName:   "SlipStream",
		ApplicationURL: n.settings.ApplicationURL,
		Timestamp:      event.OccuredAt,
		Health: &PayloadHealth{
			Source:  event.Source,
			Type:    event.Type,
			Message: event.Message,
			WikiURL: event.WikiURL,
		},
	}
	return n.send(ctx, payload)
}

func (n *Notifier) OnHealthRestored(ctx context.Context, event types.HealthEvent) error {
	payload := Payload{
		EventType:      "healthRestored",
		InstanceName:   "SlipStream",
		ApplicationURL: n.settings.ApplicationURL,
		Timestamp:      event.OccuredAt,
		Health: &PayloadHealth{
			Source:  event.Source,
			Type:    event.Type,
			Message: event.Message,
		},
	}
	return n.send(ctx, payload)
}

func (n *Notifier) OnApplicationUpdate(ctx context.Context, event types.AppUpdateEvent) error {
	payload := Payload{
		EventType:       "applicationUpdate",
		InstanceName:    "SlipStream",
		ApplicationURL:  n.settings.ApplicationURL,
		Timestamp:       event.UpdatedAt,
		PreviousVersion: event.PreviousVersion,
		NewVersion:      event.NewVersion,
	}
	return n.send(ctx, payload)
}

func (n *Notifier) send(ctx context.Context, payload Payload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, n.settings.Method, n.settings.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add basic auth if configured
	if n.settings.Username != "" && n.settings.Password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(n.settings.Username + ":" + n.settings.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	// Add custom headers
	for key, value := range n.settings.Headers {
		req.Header.Set(key, value)
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (n *Notifier) mapMovie(m *types.MediaInfo) *PayloadMovie {
	if m == nil {
		return nil
	}
	return n.mapMediaInfo(m)
}

func (n *Notifier) mapMediaInfo(m *types.MediaInfo) *PayloadMovie {
	var images []PayloadImage
	if m.PosterURL != "" {
		images = append(images, PayloadImage{CoverType: "poster", URL: m.PosterURL})
	}
	if m.FanartURL != "" {
		images = append(images, PayloadImage{CoverType: "fanart", URL: m.FanartURL})
	}
	return &PayloadMovie{
		ID:         m.ID,
		Title:      m.Title,
		Year:       m.Year,
		TMDbID:     m.TMDbID,
		IMDbID:     m.IMDbID,
		TraktID:    m.TraktID,
		Overview:   m.Overview,
		Genres:     m.Genres,
		Tags:       m.Tags,
		Rating:     m.Rating,
		Images:     images,
		TrailerURL: m.TrailerURL,
		WebsiteURL: m.WebsiteURL,
	}
}

func (n *Notifier) mapSeriesInfo(s *types.SeriesInfo) *PayloadSeries {
	var images []PayloadImage
	if s.PosterURL != "" {
		images = append(images, PayloadImage{CoverType: "poster", URL: s.PosterURL})
	}
	if s.FanartURL != "" {
		images = append(images, PayloadImage{CoverType: "fanart", URL: s.FanartURL})
	}
	return &PayloadSeries{
		ID:         s.ID,
		Title:      s.Title,
		Year:       s.Year,
		TMDbID:     s.TMDbID,
		TVDbID:     s.TVDbID,
		IMDbID:     s.IMDbID,
		TraktID:    s.TraktID,
		Overview:   s.Overview,
		Genres:     s.Genres,
		Tags:       s.Tags,
		Rating:     s.Rating,
		Images:     images,
		TrailerURL: s.TrailerURL,
	}
}

func (n *Notifier) mapEpisode(e *types.EpisodeInfo) *PayloadEpisode {
	if e == nil {
		return nil
	}
	return &PayloadEpisode{
		SeriesID:      e.SeriesID,
		SeriesTitle:   e.SeriesTitle,
		SeasonNumber:  e.SeasonNumber,
		EpisodeNumber: e.EpisodeNumber,
		EpisodeTitle:  e.EpisodeTitle,
		AirDate:       e.AirDate,
	}
}

func (n *Notifier) mapRelease(r types.ReleaseInfo) *PayloadRelease {
	return &PayloadRelease{
		ReleaseName:  r.ReleaseName,
		Quality:      r.Quality,
		Size:         r.Size,
		Indexer:      r.Indexer,
		ReleaseGroup: r.ReleaseGroup,
	}
}

func (n *Notifier) mapCustomFormats(formats []types.CustomFormat) []PayloadCustomFormat {
	if len(formats) == 0 {
		return nil
	}
	result := make([]PayloadCustomFormat, len(formats))
	for i, f := range formats {
		result[i] = PayloadCustomFormat{ID: f.ID, Name: f.Name}
	}
	return result
}

func (n *Notifier) mapMediaFileInfo(mi *types.MediaFileInfo) *PayloadMediaInfo {
	if mi == nil {
		return nil
	}
	return &PayloadMediaInfo{
		VideoCodec:        mi.VideoCodec,
		VideoBitrate:      mi.VideoBitrate,
		VideoResolution:   mi.VideoResolution,
		VideoDynamicRange: mi.VideoDynamicRange,
		AudioCodec:        mi.AudioCodec,
		AudioBitrate:      mi.AudioBitrate,
		AudioChannels:     mi.AudioChannels,
		AudioLanguages:    mi.AudioLanguages,
		Subtitles:         mi.Subtitles,
		Runtime:           mi.Runtime,
		ScanType:          mi.ScanType,
	}
}

// Payload is the webhook request body
type Payload struct {
	EventType       string                 `json:"eventType"`
	InstanceName    string                 `json:"instanceName"`
	ApplicationURL  string                 `json:"applicationUrl,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
	Message         string                 `json:"message,omitempty"`
	Movie           *PayloadMovie          `json:"movie,omitempty"`
	Series          *PayloadSeries         `json:"series,omitempty"`
	Episode         *PayloadEpisode        `json:"episode,omitempty"`
	Release         *PayloadRelease        `json:"release,omitempty"`
	DownloadClient  *PayloadDownloadClient `json:"downloadClient,omitempty"`
	DownloadID      string                 `json:"downloadId,omitempty"`
	Health          *PayloadHealth         `json:"health,omitempty"`
	Quality         string                 `json:"quality,omitempty"`
	OldQuality      string                 `json:"oldQuality,omitempty"`
	SourcePath      string                 `json:"sourcePath,omitempty"`
	DestinationPath string                 `json:"destinationPath,omitempty"`
	IsUpgrade       bool                   `json:"isUpgrade,omitempty"`
	DeletedFiles    bool                   `json:"deletedFiles,omitempty"`
	PreviousVersion string                 `json:"previousVersion,omitempty"`
	NewVersion      string                 `json:"newVersion,omitempty"`
	CustomFormats   []PayloadCustomFormat  `json:"customFormats,omitempty"`
	MediaInfo       *PayloadMediaInfo      `json:"mediaInfo,omitempty"`
	Languages       []string               `json:"languages,omitempty"`
}

type PayloadMovie struct {
	ID         int64          `json:"id"`
	Title      string         `json:"title"`
	Year       int            `json:"year,omitempty"`
	TMDbID     int64          `json:"tmdbId,omitempty"`
	IMDbID     string         `json:"imdbId,omitempty"`
	TraktID    int64          `json:"traktId,omitempty"`
	Overview   string         `json:"overview,omitempty"`
	Genres     []string       `json:"genres,omitempty"`
	Tags       []int64        `json:"tags,omitempty"`
	Rating     float64        `json:"rating,omitempty"`
	Images     []PayloadImage `json:"images,omitempty"`
	TrailerURL string         `json:"trailerUrl,omitempty"`
	WebsiteURL string         `json:"websiteUrl,omitempty"`
}

type PayloadSeries struct {
	ID         int64          `json:"id"`
	Title      string         `json:"title"`
	Year       int            `json:"year,omitempty"`
	TMDbID     int64          `json:"tmdbId,omitempty"`
	TVDbID     int64          `json:"tvdbId,omitempty"`
	IMDbID     string         `json:"imdbId,omitempty"`
	TraktID    int64          `json:"traktId,omitempty"`
	Overview   string         `json:"overview,omitempty"`
	Genres     []string       `json:"genres,omitempty"`
	Tags       []int64        `json:"tags,omitempty"`
	Rating     float64        `json:"rating,omitempty"`
	Images     []PayloadImage `json:"images,omitempty"`
	TrailerURL string         `json:"trailerUrl,omitempty"`
}

type PayloadImage struct {
	CoverType string `json:"coverType"`
	URL       string `json:"url"`
}

type PayloadCustomFormat struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type PayloadMediaInfo struct {
	VideoCodec        string   `json:"videoCodec,omitempty"`
	VideoBitrate      int64    `json:"videoBitrate,omitempty"`
	VideoResolution   string   `json:"videoResolution,omitempty"`
	VideoDynamicRange string   `json:"videoDynamicRange,omitempty"`
	AudioCodec        string   `json:"audioCodec,omitempty"`
	AudioBitrate      int64    `json:"audioBitrate,omitempty"`
	AudioChannels     string   `json:"audioChannels,omitempty"`
	AudioLanguages    []string `json:"audioLanguages,omitempty"`
	Subtitles         []string `json:"subtitles,omitempty"`
	Runtime           int      `json:"runtime,omitempty"`
	ScanType          string   `json:"scanType,omitempty"`
}

type PayloadEpisode struct {
	SeriesID      int64  `json:"seriesId"`
	SeriesTitle   string `json:"seriesTitle"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	EpisodeTitle  string `json:"episodeTitle,omitempty"`
	AirDate       string `json:"airDate,omitempty"`
}

type PayloadRelease struct {
	ReleaseName  string `json:"releaseName"`
	Quality      string `json:"quality"`
	Size         int64  `json:"size,omitempty"`
	Indexer      string `json:"indexer,omitempty"`
	ReleaseGroup string `json:"releaseGroup,omitempty"`
}

type PayloadDownloadClient struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type PayloadHealth struct {
	Source  string `json:"source"`
	Type    string `json:"type"`
	Message string `json:"message"`
	WikiURL string `json:"wikiUrl,omitempty"`
}
