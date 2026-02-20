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
	FieldOverview       FieldType = "overview"
	FieldRating         FieldType = "rating"
	FieldGenres         FieldType = "genres"
	FieldQuality        FieldType = "quality"
	FieldReleaseGroup   FieldType = "releaseGroup"
	FieldSize           FieldType = "size"
	FieldLinks          FieldType = "links"
	FieldPoster         FieldType = "poster"
	FieldFanart         FieldType = "fanart"
	FieldIndexer        FieldType = "indexer"
	FieldDownloadClient FieldType = "downloadClient"
	FieldCustomFormats  FieldType = "customFormats"
	FieldLanguages      FieldType = "languages"
	FieldMediaInfo      FieldType = "mediaInfo"
)

// Settings contains Discord-specific configuration
type Settings struct {
	WebhookURL   string      `json:"webhookUrl"`
	Username     string      `json:"username,omitempty"`
	AvatarURL    string      `json:"avatarUrl,omitempty"`
	Author       string      `json:"author,omitempty"`
	GrabFields   []FieldType `json:"grabFields,omitempty"`
	ImportFields []FieldType `json:"importFields,omitempty"`
}

// Notifier sends notifications to Discord via webhook
type Notifier struct {
	name       string
	settings   Settings
	httpClient *http.Client
	logger     *zerolog.Logger
}

// DefaultGrabFields are shown for grab events if no fields configured
var DefaultGrabFields = []FieldType{FieldQuality, FieldIndexer, FieldDownloadClient, FieldSize, FieldReleaseGroup, FieldLinks, FieldPoster}

// DefaultImportFields are shown for import events if no fields configured
var DefaultImportFields = []FieldType{FieldQuality, FieldReleaseGroup, FieldCustomFormats, FieldLanguages, FieldLinks, FieldPoster}

// New creates a new Discord notifier
func New(name string, settings *Settings, httpClient *http.Client, logger *zerolog.Logger) *Notifier {
	if len(settings.GrabFields) == 0 {
		settings.GrabFields = DefaultGrabFields
	}
	if len(settings.ImportFields) == 0 {
		settings.ImportFields = DefaultImportFields
	}
	subLogger := logger.With().Str("notifier", "discord").Str("name", name).Logger()
	return &Notifier{
		name:       name,
		settings:   *settings,
		httpClient: httpClient,
		logger:     &subLogger,
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

func (n *Notifier) OnGrab(ctx context.Context, event *types.GrabEvent) error {
	embed := n.buildGrabEmbed(event)
	return n.send(ctx, n.buildPayload(embed))
}

func (n *Notifier) buildGrabEmbed(event *types.GrabEvent) *Embed {
	title, description := n.buildGrabTitleAndDesc(event)
	thumbnail, image := n.extractGrabImages(event)
	fields := n.buildGrabFields(event)

	return &Embed{
		Title:       title,
		Description: description,
		Color:       ColorDefault,
		Fields:      fields,
		Thumbnail:   thumbnail,
		Image:       image,
		Timestamp:   event.GrabbedAt.UTC().Format(time.RFC3339),
	}
}

func (n *Notifier) buildGrabTitleAndDesc(event *types.GrabEvent) (title, description string) {
	if event.Movie != nil {
		title = formatMovieTitle("Movie Grabbed", event.Movie.Title, event.Movie.Year)
		description = fmt.Sprintf("`%s`", event.Release.ReleaseName)
	} else if event.Episode != nil {
		title = fmt.Sprintf("%s - %s", event.Episode.FormatEventLabel("Grabbed"), event.Episode.FormatTitle())
		description = fmt.Sprintf("`%s`", event.Release.ReleaseName)
	}
	return title, description
}

func (n *Notifier) extractGrabImages(event *types.GrabEvent) (thumbnail, image *EmbedImage) {
	if event.Movie == nil {
		return nil, nil
	}
	fields := n.settings.GrabFields
	if n.hasField(fields, FieldPoster) && event.Movie.PosterURL != "" {
		thumbnail = &EmbedImage{URL: event.Movie.PosterURL}
	}
	if n.hasField(fields, FieldFanart) && event.Movie.FanartURL != "" {
		image = &EmbedImage{URL: event.Movie.FanartURL}
	}
	return thumbnail, image
}

func (n *Notifier) buildGrabFields(event *types.GrabEvent) []EmbedField {
	fields := n.settings.GrabFields
	var result []EmbedField

	result = n.appendBasicGrabFields(result, fields, event)
	result = n.appendCustomFormatsField(result, fields, event.Release.CustomFormats, event.Release.CustomFormatScore)
	result = n.appendLanguagesField(result, fields, event.Release.Languages)
	result = n.appendGrabLinksField(result, fields, event)

	return result
}

func (n *Notifier) appendBasicGrabFields(result []EmbedField, fields []FieldType, event *types.GrabEvent) []EmbedField {
	if n.hasField(fields, FieldQuality) {
		result = append(result, EmbedField{Name: "Quality", Value: event.Release.Quality, Inline: true})
	}
	if n.hasField(fields, FieldIndexer) && event.Release.Indexer != "" {
		result = append(result, EmbedField{Name: "Indexer", Value: event.Release.Indexer, Inline: true})
	}
	if n.hasField(fields, FieldDownloadClient) {
		result = append(result, EmbedField{Name: "Download Client", Value: event.DownloadClient.Name, Inline: true})
	}
	if n.hasField(fields, FieldSize) && event.Release.Size > 0 {
		result = append(result, EmbedField{Name: "Size", Value: formatSize(event.Release.Size), Inline: true})
	}
	if n.hasField(fields, FieldReleaseGroup) && event.Release.ReleaseGroup != "" {
		result = append(result, EmbedField{Name: "Release Group", Value: event.Release.ReleaseGroup, Inline: true})
	}
	return result
}

func (n *Notifier) appendGrabLinksField(result []EmbedField, fields []FieldType, event *types.GrabEvent) []EmbedField {
	if !n.hasField(fields, FieldLinks) || event.Movie == nil {
		return result
	}
	if links := n.buildLinks(event.Movie); links != "" {
		result = append(result, EmbedField{Name: "Links", Value: links, Inline: false})
	}
	return result
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

func (n *Notifier) OnImport(ctx context.Context, event *types.ImportEvent) error {
	embed := n.buildImportEmbed(event)
	return n.send(ctx, n.buildPayload(embed))
}

func (n *Notifier) buildImportEmbed(event *types.ImportEvent) *Embed {
	title := n.buildImportTitle(event)
	thumbnail, image := n.extractImportImages(event)
	fields := n.buildImportFields(event)

	return &Embed{
		Title:     title,
		Color:     ColorSuccess,
		Fields:    fields,
		Thumbnail: thumbnail,
		Image:     image,
		Timestamp: event.ImportedAt.UTC().Format(time.RFC3339),
	}
}

func (n *Notifier) buildImportTitle(event *types.ImportEvent) string {
	if event.Movie != nil {
		return formatMovieTitle("Movie Downloaded", event.Movie.Title, event.Movie.Year)
	}
	if event.Episode != nil {
		return fmt.Sprintf("%s - %s", event.Episode.FormatEventLabel("Downloaded"), event.Episode.FormatTitle())
	}
	return ""
}

func (n *Notifier) extractImportImages(event *types.ImportEvent) (thumbnail, image *EmbedImage) {
	if event.Movie == nil {
		return nil, nil
	}
	fields := n.settings.ImportFields
	if n.hasField(fields, FieldPoster) && event.Movie.PosterURL != "" {
		thumbnail = &EmbedImage{URL: event.Movie.PosterURL}
	}
	if n.hasField(fields, FieldFanart) && event.Movie.FanartURL != "" {
		image = &EmbedImage{URL: event.Movie.FanartURL}
	}
	return thumbnail, image
}

func (n *Notifier) buildImportFields(event *types.ImportEvent) []EmbedField {
	fields := n.settings.ImportFields
	var result []EmbedField

	if n.hasField(fields, FieldQuality) {
		result = append(result, EmbedField{Name: "Quality", Value: event.Quality, Inline: true})
	}
	if n.hasField(fields, FieldReleaseGroup) && event.ReleaseGroup != "" {
		result = append(result, EmbedField{Name: "Release Group", Value: event.ReleaseGroup, Inline: true})
	}

	result = n.appendCustomFormatsField(result, fields, event.CustomFormats, event.CustomFormatScore)
	result = n.appendLanguagesField(result, fields, event.Languages)

	if n.hasField(fields, FieldMediaInfo) && event.MediaInfo != nil {
		if mi := n.formatMediaInfo(event.MediaInfo); mi != "" {
			result = append(result, EmbedField{Name: "Media Info", Value: mi, Inline: false})
		}
	}
	if n.hasField(fields, FieldLinks) && event.Movie != nil {
		if links := n.buildLinks(event.Movie); links != "" {
			result = append(result, EmbedField{Name: "Links", Value: links, Inline: false})
		}
	}

	return result
}

func (n *Notifier) OnUpgrade(ctx context.Context, event *types.UpgradeEvent) error {
	embed := n.buildUpgradeEmbed(event)
	return n.send(ctx, n.buildPayload(embed))
}

func (n *Notifier) buildUpgradeEmbed(event *types.UpgradeEvent) *Embed {
	title := n.buildUpgradeTitle(event)
	thumbnail, image := n.extractUpgradeImages(event)
	fields := n.buildUpgradeFields(event)

	return &Embed{
		Title:     title,
		Color:     ColorInfo,
		Fields:    fields,
		Thumbnail: thumbnail,
		Image:     image,
		Timestamp: event.UpgradedAt.UTC().Format(time.RFC3339),
	}
}

func (n *Notifier) buildUpgradeTitle(event *types.UpgradeEvent) string {
	if event.Movie != nil {
		return formatMovieTitle("Movie Upgraded", event.Movie.Title, event.Movie.Year)
	}
	if event.Episode != nil {
		return fmt.Sprintf("%s - %s", event.Episode.FormatEventLabel("Upgraded"), event.Episode.FormatTitle())
	}
	return ""
}

func (n *Notifier) extractUpgradeImages(event *types.UpgradeEvent) (thumbnail, image *EmbedImage) {
	if event.Movie == nil {
		return nil, nil
	}
	fields := n.settings.ImportFields
	if n.hasField(fields, FieldPoster) && event.Movie.PosterURL != "" {
		thumbnail = &EmbedImage{URL: event.Movie.PosterURL}
	}
	if n.hasField(fields, FieldFanart) && event.Movie.FanartURL != "" {
		image = &EmbedImage{URL: event.Movie.FanartURL}
	}
	return thumbnail, image
}

func (n *Notifier) buildUpgradeFields(event *types.UpgradeEvent) []EmbedField {
	fields := n.settings.ImportFields
	result := []EmbedField{
		{Name: "Old Quality", Value: event.OldQuality, Inline: true},
		{Name: "New Quality", Value: event.NewQuality, Inline: true},
	}

	if n.hasField(fields, FieldReleaseGroup) && event.ReleaseGroup != "" {
		result = append(result, EmbedField{Name: "Release Group", Value: event.ReleaseGroup, Inline: true})
	}

	result = n.appendCustomFormatsField(result, fields, event.CustomFormats, event.CustomFormatScore)
	result = n.appendLanguagesField(result, fields, event.Languages)

	if n.hasField(fields, FieldMediaInfo) && event.MediaInfo != nil {
		if mi := n.formatMediaInfo(event.MediaInfo); mi != "" {
			result = append(result, EmbedField{Name: "Media Info", Value: mi, Inline: false})
		}
	}
	if n.hasField(fields, FieldLinks) && event.Movie != nil {
		if links := n.buildLinks(event.Movie); links != "" {
			result = append(result, EmbedField{Name: "Links", Value: links, Inline: false})
		}
	}

	return result
}

func (n *Notifier) appendCustomFormatsField(result []EmbedField, fields []FieldType, formats []types.CustomFormat, score int) []EmbedField {
	if !n.hasField(fields, FieldCustomFormats) || len(formats) == 0 {
		return result
	}
	cfStr := n.formatCustomFormats(formats)
	if score != 0 {
		cfStr = fmt.Sprintf("%s (%d)", cfStr, score)
	}
	return append(result, EmbedField{Name: "Custom Formats", Value: cfStr, Inline: true})
}

func (n *Notifier) appendLanguagesField(result []EmbedField, fields []FieldType, languages []string) []EmbedField {
	if !n.hasField(fields, FieldLanguages) || len(languages) == 0 {
		return result
	}
	return append(result, EmbedField{Name: "Languages", Value: joinStrings(languages, ", "), Inline: true})
}

func (n *Notifier) OnMovieAdded(ctx context.Context, event *types.MovieAddedEvent) error {
	embed := n.buildMovieAddedEmbed(event)
	return n.send(ctx, n.buildPayload(embed))
}

func (n *Notifier) buildMovieAddedEmbed(event *types.MovieAddedEvent) *Embed {
	title := formatMovieTitle("Movie Added", event.Movie.Title, event.Movie.Year)
	thumbnail, image := n.extractMovieImages(&event.Movie)
	description := n.buildDescription(event.Movie.Overview)
	fields := n.buildMediaAddedFields(event.Movie.Rating, event.Movie.Genres, n.buildLinks(&event.Movie))

	return &Embed{
		Title:       title,
		Description: description,
		Color:       ColorSuccess,
		Fields:      fields,
		Thumbnail:   thumbnail,
		Image:       image,
		Timestamp:   event.AddedAt.UTC().Format(time.RFC3339),
	}
}

func (n *Notifier) extractMovieImages(movie *types.MediaInfo) (thumbnail, image *EmbedImage) {
	fields := n.settings.ImportFields
	if n.hasField(fields, FieldPoster) && movie.PosterURL != "" {
		thumbnail = &EmbedImage{URL: movie.PosterURL}
	}
	if n.hasField(fields, FieldFanart) && movie.FanartURL != "" {
		image = &EmbedImage{URL: movie.FanartURL}
	}
	return thumbnail, image
}

func (n *Notifier) buildDescription(overview string) string {
	if !n.hasField(n.settings.ImportFields, FieldOverview) || overview == "" {
		return ""
	}
	return truncate(overview, 300)
}

func (n *Notifier) buildMediaAddedFields(rating float64, genres []string, links string) []EmbedField {
	fields := n.settings.ImportFields
	var result []EmbedField

	if n.hasField(fields, FieldRating) && rating > 0 {
		result = append(result, EmbedField{Name: "Rating", Value: fmt.Sprintf("%.1f", rating), Inline: true})
	}
	if n.hasField(fields, FieldGenres) && len(genres) > 0 {
		result = append(result, EmbedField{Name: "Genres", Value: joinStrings(genres, ", "), Inline: true})
	}
	if n.hasField(fields, FieldLinks) && links != "" {
		result = append(result, EmbedField{Name: "Links", Value: links, Inline: false})
	}

	return result
}

func (n *Notifier) OnMovieDeleted(ctx context.Context, event *types.MovieDeletedEvent) error {
	title := formatMovieTitle("Movie Deleted", event.Movie.Title, event.Movie.Year)
	description := "Movie removed from library"
	if event.DeletedFiles {
		description = "Movie removed from library and files deleted"
	}

	payload := n.buildPayload(&Embed{
		Title:       title,
		Description: description,
		Color:       ColorDanger,
		Timestamp:   event.DeletedAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnSeriesAdded(ctx context.Context, event *types.SeriesAddedEvent) error {
	embed := n.buildSeriesAddedEmbed(event)
	return n.send(ctx, n.buildPayload(embed))
}

func (n *Notifier) buildSeriesAddedEmbed(event *types.SeriesAddedEvent) *Embed {
	title := formatSeriesTitle("Series Added", event.Series.Title, event.Series.Year)
	thumbnail, image := n.extractSeriesImages(&event.Series)
	description := n.buildDescription(event.Series.Overview)
	fields := n.buildMediaAddedFields(event.Series.Rating, event.Series.Genres, n.buildSeriesLinks(&event.Series))

	return &Embed{
		Title:       title,
		Description: description,
		Color:       ColorSuccess,
		Fields:      fields,
		Thumbnail:   thumbnail,
		Image:       image,
		Timestamp:   event.AddedAt.UTC().Format(time.RFC3339),
	}
}

func (n *Notifier) extractSeriesImages(series *types.SeriesInfo) (thumbnail, image *EmbedImage) {
	fields := n.settings.ImportFields
	if n.hasField(fields, FieldPoster) && series.PosterURL != "" {
		thumbnail = &EmbedImage{URL: series.PosterURL}
	}
	if n.hasField(fields, FieldFanart) && series.FanartURL != "" {
		image = &EmbedImage{URL: series.FanartURL}
	}
	return thumbnail, image
}

func (n *Notifier) OnSeriesDeleted(ctx context.Context, event *types.SeriesDeletedEvent) error {
	title := fmt.Sprintf("Series Deleted - %s", event.Series.Title)

	description := "Series removed from library"
	if event.DeletedFiles {
		description = "Series removed from library and files deleted"
	}

	payload := n.buildPayload(&Embed{
		Title:       title,
		Description: description,
		Color:       ColorDanger,
		Timestamp:   event.DeletedAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnHealthIssue(ctx context.Context, event *types.HealthEvent) error {
	color := ColorWarning
	if event.Type == "error" {
		color = ColorDanger
	}

	fields := []EmbedField{
		{Name: "Source", Value: event.Source, Inline: true},
		{Name: "Type", Value: event.Type, Inline: true},
	}

	payload := n.buildPayload(&Embed{
		Title:       "Health Issue",
		Description: event.Message,
		Color:       color,
		Fields:      fields,
		Timestamp:   event.OccuredAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnHealthRestored(ctx context.Context, event *types.HealthEvent) error {
	fields := []EmbedField{
		{Name: "Source", Value: event.Source, Inline: true},
	}

	payload := n.buildPayload(&Embed{
		Title:       "Health Issue Resolved",
		Description: event.Message,
		Color:       ColorSuccess,
		Fields:      fields,
		Timestamp:   event.OccuredAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnApplicationUpdate(ctx context.Context, event *types.AppUpdateEvent) error {
	payload := n.buildPayload(&Embed{
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

func (n *Notifier) SendMessage(ctx context.Context, event *types.MessageEvent) error {
	payload := n.buildPayload(&Embed{
		Title:       event.Title,
		Description: event.Message,
		Color:       ColorInfo,
		Timestamp:   event.SentAt.UTC().Format(time.RFC3339),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) buildPayload(embed *Embed) WebhookPayload {
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
		Embeds:    []Embed{*embed},
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
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	URL         string       `json:"url,omitempty"`
	Color       int          `json:"color,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
	Author      *EmbedAuthor `json:"author,omitempty"`
	Thumbnail   *EmbedImage  `json:"thumbnail,omitempty"`
	Image       *EmbedImage  `json:"image,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
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

func formatMovieTitle(prefix, title string, year int) string {
	if year > 0 {
		return fmt.Sprintf("%s - %s (%d)", prefix, title, year)
	}
	return fmt.Sprintf("%s - %s", prefix, title)
}

func formatSeriesTitle(prefix, title string, year int) string {
	if year > 0 {
		return fmt.Sprintf("%s - %s (%d)", prefix, title, year)
	}
	return fmt.Sprintf("%s - %s", prefix, title)
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

func formatSize(sizeBytes int64) string {
	const unit = 1024
	if sizeBytes < unit {
		return fmt.Sprintf("%d B", sizeBytes)
	}
	div, exp := int64(unit), 0
	for n := sizeBytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(sizeBytes)/float64(div), "KMGTPE"[exp])
}
