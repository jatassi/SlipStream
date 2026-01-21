package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/notification/types"
)

// Discord embed colors
const (
	ColorSuccess = 0x2ECC71 // Green
	ColorWarning = 0xF1C40F // Yellow
	ColorDanger  = 0xE74C3C // Red
	ColorInfo    = 0x3498DB // Blue
	ColorDefault = 0x7289DA // Discord blurple
)

// FieldType defines which fields to include in notifications
type FieldType string

const (
	FieldOverview          FieldType = "overview"
	FieldRating            FieldType = "rating"
	FieldGenres            FieldType = "genres"
	FieldQuality           FieldType = "quality"
	FieldReleaseGroup      FieldType = "releaseGroup"
	FieldSize              FieldType = "size"
	FieldLinks             FieldType = "links"
	FieldPoster            FieldType = "poster"
	FieldFanart            FieldType = "fanart"
	FieldIndexer           FieldType = "indexer"
	FieldDownloadClient    FieldType = "downloadClient"
	FieldCustomFormats     FieldType = "customFormats"
	FieldLanguages         FieldType = "languages"
	FieldMediaInfo         FieldType = "mediaInfo"
)

// Settings contains Discord-specific configuration
type Settings struct {
	WebhookURL     string      `json:"webhookUrl"`
	Username       string      `json:"username,omitempty"`
	AvatarURL      string      `json:"avatarUrl,omitempty"`
	Author         string      `json:"author,omitempty"`
	GrabFields     []FieldType `json:"grabFields,omitempty"`
	ImportFields   []FieldType `json:"importFields,omitempty"`
}

// Notifier sends notifications to Discord via webhook
type Notifier struct {
	name       string
	settings   Settings
	httpClient *http.Client
	logger     zerolog.Logger
}

// DefaultGrabFields are shown for grab events if no fields configured
var DefaultGrabFields = []FieldType{FieldQuality, FieldIndexer, FieldDownloadClient, FieldSize, FieldReleaseGroup, FieldLinks, FieldPoster}

// DefaultImportFields are shown for import events if no fields configured
var DefaultImportFields = []FieldType{FieldQuality, FieldReleaseGroup, FieldCustomFormats, FieldLanguages, FieldLinks, FieldPoster}

// New creates a new Discord notifier
func New(name string, settings Settings, httpClient *http.Client, logger zerolog.Logger) *Notifier {
	if len(settings.GrabFields) == 0 {
		settings.GrabFields = DefaultGrabFields
	}
	if len(settings.ImportFields) == 0 {
		settings.ImportFields = DefaultImportFields
	}
	return &Notifier{
		name:       name,
		settings:   settings,
		httpClient: httpClient,
		logger:     logger.With().Str("notifier", "discord").Str("name", name).Logger(),
	}
}

func (n *Notifier) Type() types.NotifierType {
	return types.NotifierDiscord
}

func (n *Notifier) Name() string {
	return n.name
}

func (n *Notifier) Test(ctx context.Context) error {
	payload := WebhookPayload{
		Username:  n.getUsername(),
		AvatarURL: n.settings.AvatarURL,
		Embeds: []Embed{{
			Title:       "SlipStream Test Notification",
			Description: "This is a test notification from SlipStream.",
			Color:       ColorInfo,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}},
	}
	return n.send(ctx, payload)
}

func (n *Notifier) hasField(fields []FieldType, field FieldType) bool {
	for _, f := range fields {
		if f == field {
			return true
		}
	}
	return false
}

func (n *Notifier) buildLinks(movie *types.MediaInfo) string {
	if movie == nil {
		return ""
	}
	var links []string
	if movie.TMDbID > 0 {
		links = append(links, fmt.Sprintf("[TMDb](https://www.themoviedb.org/movie/%d)", movie.TMDbID))
	}
	if movie.IMDbID != "" {
		links = append(links, fmt.Sprintf("[IMDb](https://www.imdb.com/title/%s)", movie.IMDbID))
	}
	if movie.TraktID > 0 {
		links = append(links, fmt.Sprintf("[Trakt](https://trakt.tv/movies/%d)", movie.TraktID))
	}
	if movie.TrailerURL != "" {
		links = append(links, fmt.Sprintf("[Trailer](%s)", movie.TrailerURL))
	}
	if movie.WebsiteURL != "" {
		links = append(links, fmt.Sprintf("[Website](%s)", movie.WebsiteURL))
	}
	return joinStrings(links, " | ")
}

func (n *Notifier) buildSeriesLinks(series *types.SeriesInfo) string {
	if series == nil {
		return ""
	}
	var links []string
	if series.TMDbID > 0 {
		links = append(links, fmt.Sprintf("[TMDb](https://www.themoviedb.org/tv/%d)", series.TMDbID))
	}
	if series.IMDbID != "" {
		links = append(links, fmt.Sprintf("[IMDb](https://www.imdb.com/title/%s)", series.IMDbID))
	}
	if series.TVDbID > 0 {
		links = append(links, fmt.Sprintf("[TVDb](https://thetvdb.com/series/%d)", series.TVDbID))
	}
	if series.TraktID > 0 {
		links = append(links, fmt.Sprintf("[Trakt](https://trakt.tv/shows/%d)", series.TraktID))
	}
	if series.TrailerURL != "" {
		links = append(links, fmt.Sprintf("[Trailer](%s)", series.TrailerURL))
	}
	return joinStrings(links, " | ")
}

func (n *Notifier) formatCustomFormats(formats []types.CustomFormat) string {
	if len(formats) == 0 {
		return ""
	}
	names := make([]string, len(formats))
	for i, f := range formats {
		names[i] = f.Name
	}
	return joinStrings(names, ", ")
}

func (n *Notifier) OnGrab(ctx context.Context, event types.GrabEvent) error {
	var title, description string
	var thumbnail, image *EmbedImage
	fields := n.settings.GrabFields

	if event.Movie != nil {
		title = fmt.Sprintf("Movie Grabbed - %s", event.Movie.Title)
		if event.Movie.Year > 0 {
			title = fmt.Sprintf("Movie Grabbed - %s (%d)", event.Movie.Title, event.Movie.Year)
		}
		description = fmt.Sprintf("`%s`", event.Release.ReleaseName)
		if n.hasField(fields, FieldPoster) && event.Movie.PosterURL != "" {
			thumbnail = &EmbedImage{URL: event.Movie.PosterURL}
		}
		if n.hasField(fields, FieldFanart) && event.Movie.FanartURL != "" {
			image = &EmbedImage{URL: event.Movie.FanartURL}
		}
	} else if event.Episode != nil {
		title = fmt.Sprintf("Episode Grabbed - %s S%02dE%02d", event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber)
		description = fmt.Sprintf("`%s`", event.Release.ReleaseName)
	}

	var embedFields []EmbedField

	if n.hasField(fields, FieldQuality) {
		embedFields = append(embedFields, EmbedField{Name: "Quality", Value: event.Release.Quality, Inline: true})
	}
	if n.hasField(fields, FieldIndexer) && event.Release.Indexer != "" {
		embedFields = append(embedFields, EmbedField{Name: "Indexer", Value: event.Release.Indexer, Inline: true})
	}
	if n.hasField(fields, FieldDownloadClient) {
		embedFields = append(embedFields, EmbedField{Name: "Download Client", Value: event.DownloadClient.Name, Inline: true})
	}
	if n.hasField(fields, FieldSize) && event.Release.Size > 0 {
		embedFields = append(embedFields, EmbedField{Name: "Size", Value: formatSize(event.Release.Size), Inline: true})
	}
	if n.hasField(fields, FieldReleaseGroup) && event.Release.ReleaseGroup != "" {
		embedFields = append(embedFields, EmbedField{Name: "Release Group", Value: event.Release.ReleaseGroup, Inline: true})
	}
	if n.hasField(fields, FieldCustomFormats) && len(event.Release.CustomFormats) > 0 {
		cfStr := n.formatCustomFormats(event.Release.CustomFormats)
		if event.Release.CustomFormatScore != 0 {
			cfStr = fmt.Sprintf("%s (%d)", cfStr, event.Release.CustomFormatScore)
		}
		embedFields = append(embedFields, EmbedField{Name: "Custom Formats", Value: cfStr, Inline: true})
	}
	if n.hasField(fields, FieldLanguages) && len(event.Release.Languages) > 0 {
		embedFields = append(embedFields, EmbedField{Name: "Languages", Value: joinStrings(event.Release.Languages, ", "), Inline: true})
	}
	if n.hasField(fields, FieldLinks) {
		if event.Movie != nil {
			if links := n.buildLinks(event.Movie); links != "" {
				embedFields = append(embedFields, EmbedField{Name: "Links", Value: links, Inline: false})
			}
		}
	}

	payload := n.buildPayload(Embed{
		Title:       title,
		Description: description,
		Color:       ColorDefault,
		Fields:      embedFields,
		Thumbnail:   thumbnail,
		Image:       image,
		Timestamp:   event.GrabbedAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) formatMediaInfo(mi *types.MediaFileInfo) string {
	if mi == nil {
		return ""
	}
	var parts []string
	if mi.VideoCodec != "" {
		video := mi.VideoCodec
		if mi.VideoResolution != "" {
			video = fmt.Sprintf("%s %s", mi.VideoResolution, video)
		}
		if mi.VideoDynamicRange != "" {
			video = fmt.Sprintf("%s %s", video, mi.VideoDynamicRange)
		}
		parts = append(parts, video)
	}
	if mi.AudioCodec != "" {
		audio := mi.AudioCodec
		if mi.AudioChannels != "" {
			audio = fmt.Sprintf("%s %s", audio, mi.AudioChannels)
		}
		parts = append(parts, audio)
	}
	return joinStrings(parts, " / ")
}

func (n *Notifier) OnDownload(ctx context.Context, event types.DownloadEvent) error {
	var title string
	var thumbnail, image *EmbedImage
	fields := n.settings.ImportFields

	if event.Movie != nil {
		title = fmt.Sprintf("Movie Downloaded - %s", event.Movie.Title)
		if event.Movie.Year > 0 {
			title = fmt.Sprintf("Movie Downloaded - %s (%d)", event.Movie.Title, event.Movie.Year)
		}
		if n.hasField(fields, FieldPoster) && event.Movie.PosterURL != "" {
			thumbnail = &EmbedImage{URL: event.Movie.PosterURL}
		}
		if n.hasField(fields, FieldFanart) && event.Movie.FanartURL != "" {
			image = &EmbedImage{URL: event.Movie.FanartURL}
		}
	} else if event.Episode != nil {
		title = fmt.Sprintf("Episode Downloaded - %s S%02dE%02d", event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber)
	}

	var embedFields []EmbedField

	if n.hasField(fields, FieldQuality) {
		embedFields = append(embedFields, EmbedField{Name: "Quality", Value: event.Quality, Inline: true})
	}
	if n.hasField(fields, FieldReleaseGroup) && event.ReleaseGroup != "" {
		embedFields = append(embedFields, EmbedField{Name: "Release Group", Value: event.ReleaseGroup, Inline: true})
	}
	if n.hasField(fields, FieldCustomFormats) && len(event.CustomFormats) > 0 {
		cfStr := n.formatCustomFormats(event.CustomFormats)
		if event.CustomFormatScore != 0 {
			cfStr = fmt.Sprintf("%s (%d)", cfStr, event.CustomFormatScore)
		}
		embedFields = append(embedFields, EmbedField{Name: "Custom Formats", Value: cfStr, Inline: true})
	}
	if n.hasField(fields, FieldLanguages) && len(event.Languages) > 0 {
		embedFields = append(embedFields, EmbedField{Name: "Languages", Value: joinStrings(event.Languages, ", "), Inline: true})
	}
	if n.hasField(fields, FieldMediaInfo) && event.MediaInfo != nil {
		if mi := n.formatMediaInfo(event.MediaInfo); mi != "" {
			embedFields = append(embedFields, EmbedField{Name: "Media Info", Value: mi, Inline: false})
		}
	}
	if n.hasField(fields, FieldLinks) {
		if event.Movie != nil {
			if links := n.buildLinks(event.Movie); links != "" {
				embedFields = append(embedFields, EmbedField{Name: "Links", Value: links, Inline: false})
			}
		}
	}

	payload := n.buildPayload(Embed{
		Title:     title,
		Color:     ColorSuccess,
		Fields:    embedFields,
		Thumbnail: thumbnail,
		Image:     image,
		Timestamp: event.ImportedAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnUpgrade(ctx context.Context, event types.UpgradeEvent) error {
	var title string
	var thumbnail, image *EmbedImage
	fields := n.settings.ImportFields

	if event.Movie != nil {
		title = fmt.Sprintf("Movie Upgraded - %s", event.Movie.Title)
		if event.Movie.Year > 0 {
			title = fmt.Sprintf("Movie Upgraded - %s (%d)", event.Movie.Title, event.Movie.Year)
		}
		if n.hasField(fields, FieldPoster) && event.Movie.PosterURL != "" {
			thumbnail = &EmbedImage{URL: event.Movie.PosterURL}
		}
		if n.hasField(fields, FieldFanart) && event.Movie.FanartURL != "" {
			image = &EmbedImage{URL: event.Movie.FanartURL}
		}
	} else if event.Episode != nil {
		title = fmt.Sprintf("Episode Upgraded - %s S%02dE%02d", event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber)
	}

	embedFields := []EmbedField{
		{Name: "Old Quality", Value: event.OldQuality, Inline: true},
		{Name: "New Quality", Value: event.NewQuality, Inline: true},
	}

	if n.hasField(fields, FieldReleaseGroup) && event.ReleaseGroup != "" {
		embedFields = append(embedFields, EmbedField{Name: "Release Group", Value: event.ReleaseGroup, Inline: true})
	}
	if n.hasField(fields, FieldCustomFormats) && len(event.CustomFormats) > 0 {
		cfStr := n.formatCustomFormats(event.CustomFormats)
		if event.CustomFormatScore != 0 {
			cfStr = fmt.Sprintf("%s (%d)", cfStr, event.CustomFormatScore)
		}
		embedFields = append(embedFields, EmbedField{Name: "Custom Formats", Value: cfStr, Inline: true})
	}
	if n.hasField(fields, FieldLanguages) && len(event.Languages) > 0 {
		embedFields = append(embedFields, EmbedField{Name: "Languages", Value: joinStrings(event.Languages, ", "), Inline: true})
	}
	if n.hasField(fields, FieldMediaInfo) && event.MediaInfo != nil {
		if mi := n.formatMediaInfo(event.MediaInfo); mi != "" {
			embedFields = append(embedFields, EmbedField{Name: "Media Info", Value: mi, Inline: false})
		}
	}
	if n.hasField(fields, FieldLinks) {
		if event.Movie != nil {
			if links := n.buildLinks(event.Movie); links != "" {
				embedFields = append(embedFields, EmbedField{Name: "Links", Value: links, Inline: false})
			}
		}
	}

	payload := n.buildPayload(Embed{
		Title:     title,
		Color:     ColorInfo,
		Fields:    embedFields,
		Thumbnail: thumbnail,
		Image:     image,
		Timestamp: event.UpgradedAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnMovieAdded(ctx context.Context, event types.MovieAddedEvent) error {
	title := fmt.Sprintf("Movie Added - %s", event.Movie.Title)
	if event.Movie.Year > 0 {
		title = fmt.Sprintf("Movie Added - %s (%d)", event.Movie.Title, event.Movie.Year)
	}

	var thumbnail, image *EmbedImage
	fields := n.settings.ImportFields
	if n.hasField(fields, FieldPoster) && event.Movie.PosterURL != "" {
		thumbnail = &EmbedImage{URL: event.Movie.PosterURL}
	}
	if n.hasField(fields, FieldFanart) && event.Movie.FanartURL != "" {
		image = &EmbedImage{URL: event.Movie.FanartURL}
	}

	var description string
	if n.hasField(fields, FieldOverview) && event.Movie.Overview != "" {
		description = truncate(event.Movie.Overview, 300)
	}

	var embedFields []EmbedField
	if n.hasField(fields, FieldRating) && event.Movie.Rating > 0 {
		embedFields = append(embedFields, EmbedField{Name: "Rating", Value: fmt.Sprintf("%.1f", event.Movie.Rating), Inline: true})
	}
	if n.hasField(fields, FieldGenres) && len(event.Movie.Genres) > 0 {
		embedFields = append(embedFields, EmbedField{Name: "Genres", Value: joinStrings(event.Movie.Genres, ", "), Inline: true})
	}
	if n.hasField(fields, FieldLinks) {
		if links := n.buildLinks(&event.Movie); links != "" {
			embedFields = append(embedFields, EmbedField{Name: "Links", Value: links, Inline: false})
		}
	}

	payload := n.buildPayload(Embed{
		Title:       title,
		Description: description,
		Color:       ColorSuccess,
		Fields:      embedFields,
		Thumbnail:   thumbnail,
		Image:       image,
		Timestamp:   event.AddedAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnMovieDeleted(ctx context.Context, event types.MovieDeletedEvent) error {
	title := fmt.Sprintf("Movie Deleted - %s", event.Movie.Title)
	if event.Movie.Year > 0 {
		title = fmt.Sprintf("Movie Deleted - %s (%d)", event.Movie.Title, event.Movie.Year)
	}

	description := "Movie removed from library"
	if event.DeletedFiles {
		description = "Movie removed from library and files deleted"
	}

	payload := n.buildPayload(Embed{
		Title:       title,
		Description: description,
		Color:       ColorDanger,
		Timestamp:   event.DeletedAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnSeriesAdded(ctx context.Context, event types.SeriesAddedEvent) error {
	title := fmt.Sprintf("Series Added - %s", event.Series.Title)
	if event.Series.Year > 0 {
		title = fmt.Sprintf("Series Added - %s (%d)", event.Series.Title, event.Series.Year)
	}

	var thumbnail, image *EmbedImage
	fields := n.settings.ImportFields
	if n.hasField(fields, FieldPoster) && event.Series.PosterURL != "" {
		thumbnail = &EmbedImage{URL: event.Series.PosterURL}
	}
	if n.hasField(fields, FieldFanart) && event.Series.FanartURL != "" {
		image = &EmbedImage{URL: event.Series.FanartURL}
	}

	var description string
	if n.hasField(fields, FieldOverview) && event.Series.Overview != "" {
		description = truncate(event.Series.Overview, 300)
	}

	var embedFields []EmbedField
	if n.hasField(fields, FieldRating) && event.Series.Rating > 0 {
		embedFields = append(embedFields, EmbedField{Name: "Rating", Value: fmt.Sprintf("%.1f", event.Series.Rating), Inline: true})
	}
	if n.hasField(fields, FieldGenres) && len(event.Series.Genres) > 0 {
		embedFields = append(embedFields, EmbedField{Name: "Genres", Value: joinStrings(event.Series.Genres, ", "), Inline: true})
	}
	if n.hasField(fields, FieldLinks) {
		if links := n.buildSeriesLinks(&event.Series); links != "" {
			embedFields = append(embedFields, EmbedField{Name: "Links", Value: links, Inline: false})
		}
	}

	payload := n.buildPayload(Embed{
		Title:       title,
		Description: description,
		Color:       ColorSuccess,
		Fields:      embedFields,
		Thumbnail:   thumbnail,
		Image:       image,
		Timestamp:   event.AddedAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnSeriesDeleted(ctx context.Context, event types.SeriesDeletedEvent) error {
	title := fmt.Sprintf("Series Deleted - %s", event.Series.Title)

	description := "Series removed from library"
	if event.DeletedFiles {
		description = "Series removed from library and files deleted"
	}

	payload := n.buildPayload(Embed{
		Title:       title,
		Description: description,
		Color:       ColorDanger,
		Timestamp:   event.DeletedAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnHealthIssue(ctx context.Context, event types.HealthEvent) error {
	color := ColorWarning
	if event.Type == "error" {
		color = ColorDanger
	}

	fields := []EmbedField{
		{Name: "Source", Value: event.Source, Inline: true},
		{Name: "Type", Value: event.Type, Inline: true},
	}

	payload := n.buildPayload(Embed{
		Title:     "Health Issue",
		Description: event.Message,
		Color:     color,
		Fields:    fields,
		Timestamp: event.OccuredAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnHealthRestored(ctx context.Context, event types.HealthEvent) error {
	fields := []EmbedField{
		{Name: "Source", Value: event.Source, Inline: true},
	}

	payload := n.buildPayload(Embed{
		Title:       "Health Issue Resolved",
		Description: event.Message,
		Color:       ColorSuccess,
		Fields:      fields,
		Timestamp:   event.OccuredAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnApplicationUpdate(ctx context.Context, event types.AppUpdateEvent) error {
	payload := n.buildPayload(Embed{
		Title: "Application Updated",
		Fields: []EmbedField{
			{Name: "Previous Version", Value: event.PreviousVersion, Inline: true},
			{Name: "New Version", Value: event.NewVersion, Inline: true},
		},
		Color:     ColorInfo,
		Timestamp: event.UpdatedAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) SendMessage(ctx context.Context, event types.MessageEvent) error {
	payload := n.buildPayload(Embed{
		Title:       event.Title,
		Description: event.Message,
		Color:       ColorInfo,
		Timestamp:   event.SentAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) buildPayload(embed Embed) WebhookPayload {
	authorName := "SlipStream"
	if n.settings.Author != "" {
		authorName = n.settings.Author
	}
	embed.Author = &EmbedAuthor{
		Name:    authorName,
		IconURL: "https://raw.githubusercontent.com/slipstream/slipstream/main/web/public/logo.png",
	}

	return WebhookPayload{
		Username:  n.getUsername(),
		AvatarURL: n.settings.AvatarURL,
		Embeds:    []Embed{embed},
	}
}

func (n *Notifier) getUsername() string {
	if n.settings.Username != "" {
		return n.settings.Username
	}
	return "SlipStream"
}

func (n *Notifier) send(ctx context.Context, payload WebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.settings.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord returned status %d", resp.StatusCode)
	}

	return nil
}

// WebhookPayload is the Discord webhook request body
type WebhookPayload struct {
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Content   string  `json:"content,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

// Embed is a Discord embed object
type Embed struct {
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`
	URL         string        `json:"url,omitempty"`
	Color       int           `json:"color,omitempty"`
	Timestamp   string        `json:"timestamp,omitempty"`
	Author      *EmbedAuthor  `json:"author,omitempty"`
	Thumbnail   *EmbedImage   `json:"thumbnail,omitempty"`
	Image       *EmbedImage   `json:"image,omitempty"`
	Fields      []EmbedField  `json:"fields,omitempty"`
	Footer      *EmbedFooter  `json:"footer,omitempty"`
}

// EmbedAuthor is the author section of an embed
type EmbedAuthor struct {
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// EmbedImage is an image in an embed
type EmbedImage struct {
	URL string `json:"url,omitempty"`
}

// EmbedField is a field in an embed
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// EmbedFooter is the footer section of an embed
type EmbedFooter struct {
	Text    string `json:"text,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
