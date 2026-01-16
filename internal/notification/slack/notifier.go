package slack

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

// Colors for Slack attachments
const (
	ColorGood    = "good"    // Green
	ColorWarning = "warning" // Yellow
	ColorDanger  = "danger"  // Red
)

// Settings contains Slack-specific configuration
type Settings struct {
	WebhookURL string `json:"webhookUrl"`
	Username   string `json:"username,omitempty"`
	IconEmoji  string `json:"iconEmoji,omitempty"`
	IconURL    string `json:"iconUrl,omitempty"`
	Channel    string `json:"channel,omitempty"`
}

// Notifier sends notifications via Slack webhook
type Notifier struct {
	name       string
	settings   Settings
	httpClient *http.Client
	logger     zerolog.Logger
}

// New creates a new Slack notifier
func New(name string, settings Settings, httpClient *http.Client, logger zerolog.Logger) *Notifier {
	return &Notifier{
		name:       name,
		settings:   settings,
		httpClient: httpClient,
		logger:     logger.With().Str("notifier", "slack").Str("name", name).Logger(),
	}
}

func (n *Notifier) Type() types.NotifierType {
	return types.NotifierSlack
}

func (n *Notifier) Name() string {
	return n.name
}

func (n *Notifier) Test(ctx context.Context) error {
	payload := Payload{
		Username:  n.getUsername(),
		IconEmoji: n.settings.IconEmoji,
		Channel:   n.settings.Channel,
		Attachments: []Attachment{{
			Color:   ColorGood,
			Title:   "SlipStream Test Notification",
			Text:    "This is a test notification from SlipStream.",
			Footer:  "SlipStream",
			Ts:      time.Now().Unix(),
		}},
	}
	return n.send(ctx, payload)
}

func (n *Notifier) OnGrab(ctx context.Context, event types.GrabEvent) error {
	var title string
	if event.Movie != nil {
		title = fmt.Sprintf("Movie Grabbed - %s", event.Movie.Title)
		if event.Movie.Year > 0 {
			title = fmt.Sprintf("Movie Grabbed - %s (%d)", event.Movie.Title, event.Movie.Year)
		}
	} else if event.Episode != nil {
		title = fmt.Sprintf("Episode Grabbed - %s S%02dE%02d", event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber)
	}

	fields := []Field{
		{Title: "Quality", Value: event.Release.Quality, Short: true},
		{Title: "Indexer", Value: event.Release.Indexer, Short: true},
		{Title: "Client", Value: event.DownloadClient.Name, Short: true},
	}

	if event.Release.ReleaseGroup != "" {
		fields = append(fields, Field{Title: "Group", Value: event.Release.ReleaseGroup, Short: true})
	}

	payload := n.buildPayload(Attachment{
		Color:  "#7289DA",
		Title:  title,
		Text:   fmt.Sprintf("`%s`", event.Release.ReleaseName),
		Fields: fields,
		Ts:     event.GrabbedAt.Unix(),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnDownload(ctx context.Context, event types.DownloadEvent) error {
	var title string
	if event.Movie != nil {
		title = fmt.Sprintf("Movie Downloaded - %s", event.Movie.Title)
		if event.Movie.Year > 0 {
			title = fmt.Sprintf("Movie Downloaded - %s (%d)", event.Movie.Title, event.Movie.Year)
		}
	} else if event.Episode != nil {
		title = fmt.Sprintf("Episode Downloaded - %s S%02dE%02d", event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber)
	}

	fields := []Field{
		{Title: "Quality", Value: event.Quality, Short: true},
	}

	if event.ReleaseGroup != "" {
		fields = append(fields, Field{Title: "Group", Value: event.ReleaseGroup, Short: true})
	}

	payload := n.buildPayload(Attachment{
		Color:  ColorGood,
		Title:  title,
		Fields: fields,
		Ts:     event.ImportedAt.Unix(),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnUpgrade(ctx context.Context, event types.UpgradeEvent) error {
	var title string
	if event.Movie != nil {
		title = fmt.Sprintf("Movie Upgraded - %s", event.Movie.Title)
		if event.Movie.Year > 0 {
			title = fmt.Sprintf("Movie Upgraded - %s (%d)", event.Movie.Title, event.Movie.Year)
		}
	} else if event.Episode != nil {
		title = fmt.Sprintf("Episode Upgraded - %s S%02dE%02d", event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber)
	}

	fields := []Field{
		{Title: "Old Quality", Value: event.OldQuality, Short: true},
		{Title: "New Quality", Value: event.NewQuality, Short: true},
	}

	payload := n.buildPayload(Attachment{
		Color:  ColorGood,
		Title:  title,
		Fields: fields,
		Ts:     event.UpgradedAt.Unix(),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnMovieAdded(ctx context.Context, event types.MovieAddedEvent) error {
	title := fmt.Sprintf("Movie Added - %s", event.Movie.Title)
	if event.Movie.Year > 0 {
		title = fmt.Sprintf("Movie Added - %s (%d)", event.Movie.Title, event.Movie.Year)
	}

	text := ""
	if event.Movie.Overview != "" {
		text = truncate(event.Movie.Overview, 200)
	}

	payload := n.buildPayload(Attachment{
		Color:    ColorGood,
		Title:    title,
		Text:     text,
		ThumbURL: event.Movie.PosterURL,
		Ts:       event.AddedAt.Unix(),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnMovieDeleted(ctx context.Context, event types.MovieDeletedEvent) error {
	title := fmt.Sprintf("Movie Deleted - %s", event.Movie.Title)
	if event.Movie.Year > 0 {
		title = fmt.Sprintf("Movie Deleted - %s (%d)", event.Movie.Title, event.Movie.Year)
	}

	text := "Movie removed from library"
	if event.DeletedFiles {
		text = "Movie removed from library and files deleted"
	}

	payload := n.buildPayload(Attachment{
		Color: ColorDanger,
		Title: title,
		Text:  text,
		Ts:    event.DeletedAt.Unix(),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnSeriesAdded(ctx context.Context, event types.SeriesAddedEvent) error {
	title := fmt.Sprintf("Series Added - %s", event.Series.Title)
	if event.Series.Year > 0 {
		title = fmt.Sprintf("Series Added - %s (%d)", event.Series.Title, event.Series.Year)
	}

	text := ""
	if event.Series.Overview != "" {
		text = truncate(event.Series.Overview, 200)
	}

	payload := n.buildPayload(Attachment{
		Color:    ColorGood,
		Title:    title,
		Text:     text,
		ThumbURL: event.Series.PosterURL,
		Ts:       event.AddedAt.Unix(),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnSeriesDeleted(ctx context.Context, event types.SeriesDeletedEvent) error {
	title := fmt.Sprintf("Series Deleted - %s", event.Series.Title)

	text := "Series removed from library"
	if event.DeletedFiles {
		text = "Series removed from library and files deleted"
	}

	payload := n.buildPayload(Attachment{
		Color: ColorDanger,
		Title: title,
		Text:  text,
		Ts:    event.DeletedAt.Unix(),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnHealthIssue(ctx context.Context, event types.HealthEvent) error {
	color := ColorWarning
	if event.Type == "error" {
		color = ColorDanger
	}

	fields := []Field{
		{Title: "Source", Value: event.Source, Short: true},
		{Title: "Type", Value: event.Type, Short: true},
	}

	payload := n.buildPayload(Attachment{
		Color:  color,
		Title:  "Health Issue",
		Text:   event.Message,
		Fields: fields,
		Ts:     event.OccuredAt.Unix(),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnHealthRestored(ctx context.Context, event types.HealthEvent) error {
	fields := []Field{
		{Title: "Source", Value: event.Source, Short: true},
	}

	payload := n.buildPayload(Attachment{
		Color:  ColorGood,
		Title:  "Health Issue Resolved",
		Text:   event.Message,
		Fields: fields,
		Ts:     event.OccuredAt.Unix(),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) OnApplicationUpdate(ctx context.Context, event types.AppUpdateEvent) error {
	fields := []Field{
		{Title: "Previous Version", Value: event.PreviousVersion, Short: true},
		{Title: "New Version", Value: event.NewVersion, Short: true},
	}

	payload := n.buildPayload(Attachment{
		Color:  ColorGood,
		Title:  "Application Updated",
		Fields: fields,
		Ts:     event.UpdatedAt.Unix(),
	})

	return n.send(ctx, payload)
}

func (n *Notifier) buildPayload(attachment Attachment) Payload {
	attachment.Footer = "SlipStream"
	attachment.FooterIcon = "https://raw.githubusercontent.com/slipstream/slipstream/main/web/public/logo.png"

	if attachment.Fallback == "" {
		attachment.Fallback = attachment.Title
		if attachment.Text != "" {
			attachment.Fallback = attachment.Title + " - " + attachment.Text
		}
	}

	payload := Payload{
		Username:    n.getUsername(),
		Channel:     n.settings.Channel,
		Attachments: []Attachment{attachment},
	}

	if n.settings.IconURL != "" {
		payload.IconURL = n.settings.IconURL
	} else if n.settings.IconEmoji != "" {
		payload.IconEmoji = n.settings.IconEmoji
	}

	return payload
}

func (n *Notifier) getUsername() string {
	if n.settings.Username != "" {
		return n.settings.Username
	}
	return "SlipStream"
}

func (n *Notifier) send(ctx context.Context, payload Payload) error {
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

// Payload is the Slack webhook request body
type Payload struct {
	Username    string       `json:"username,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	IconURL     string       `json:"icon_url,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment is a Slack message attachment
type Attachment struct {
	Fallback   string  `json:"fallback,omitempty"`
	Color      string  `json:"color,omitempty"`
	Title      string  `json:"title,omitempty"`
	TitleLink  string  `json:"title_link,omitempty"`
	Text       string  `json:"text,omitempty"`
	Fields     []Field `json:"fields,omitempty"`
	ThumbURL   string  `json:"thumb_url,omitempty"`
	Footer     string  `json:"footer,omitempty"`
	FooterIcon string  `json:"footer_icon,omitempty"`
	Ts         int64   `json:"ts,omitempty"`
}

// Field is a field in a Slack attachment
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
