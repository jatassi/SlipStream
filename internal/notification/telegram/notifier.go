package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/notification/types"
)

const telegramAPIBase = "https://api.telegram.org/bot"

// MetadataLink defines which metadata links to include
type MetadataLink string

const (
	MetadataLinkTMDb  MetadataLink = "tmdb"
	MetadataLinkIMDb  MetadataLink = "imdb"
	MetadataLinkTVDb  MetadataLink = "tvdb"
	MetadataLinkTrakt MetadataLink = "trakt"
)

// DefaultMetadataLinks are shown if no links are configured
var DefaultMetadataLinks = []MetadataLink{MetadataLinkTMDb, MetadataLinkIMDb}

// Settings contains Telegram-specific configuration
type Settings struct {
	BotToken             string         `json:"botToken"`
	ChatID               string         `json:"chatId"`
	TopicID              int64          `json:"topicId,omitempty"`
	Silent               bool           `json:"silent,omitempty"`
	IncludeLinks         bool           `json:"includeLinks,omitempty"`
	MetadataLinks        []MetadataLink `json:"metadataLinks,omitempty"`
	IncludeAppNameInTitle bool          `json:"includeAppNameInTitle,omitempty"`
}

// Notifier sends notifications via Telegram bot
type Notifier struct {
	name       string
	settings   Settings
	httpClient *http.Client
	logger     zerolog.Logger
}

// New creates a new Telegram notifier
func New(name string, settings Settings, httpClient *http.Client, logger zerolog.Logger) *Notifier {
	if settings.IncludeLinks && len(settings.MetadataLinks) == 0 {
		settings.MetadataLinks = DefaultMetadataLinks
	}
	return &Notifier{
		name:       name,
		settings:   settings,
		httpClient: httpClient,
		logger:     logger.With().Str("notifier", "telegram").Str("name", name).Logger(),
	}
}

func (n *Notifier) Type() types.NotifierType {
	return types.NotifierTelegram
}

func (n *Notifier) Name() string {
	return n.name
}

func (n *Notifier) Test(ctx context.Context) error {
	message := "<b>SlipStream Test Notification</b>\n\nThis is a test notification from SlipStream."
	return n.sendMessage(ctx, message)
}

func (n *Notifier) OnGrab(ctx context.Context, event types.GrabEvent) error {
	var sb strings.Builder
	sb.WriteString("<b>🎬 Release Grabbed</b>\n\n")

	if event.Movie != nil {
		sb.WriteString(fmt.Sprintf("<b>%s</b>", html.EscapeString(event.Movie.Title)))
		if event.Movie.Year > 0 {
			sb.WriteString(fmt.Sprintf(" (%d)", event.Movie.Year))
		}
		sb.WriteString("\n")
		n.writeLinks(&sb, event.Movie.TMDbID, event.Movie.IMDbID, event.Movie.TraktID, "movie")
	} else if event.Episode != nil {
		sb.WriteString(fmt.Sprintf("<b>%s</b> S%02dE%02d\n",
			html.EscapeString(event.Episode.SeriesTitle),
			event.Episode.SeasonNumber,
			event.Episode.EpisodeNumber))
	}

	sb.WriteString(fmt.Sprintf("\n<code>%s</code>\n", html.EscapeString(event.Release.ReleaseName)))
	sb.WriteString(fmt.Sprintf("\n📊 Quality: %s", event.Release.Quality))
	sb.WriteString(fmt.Sprintf("\n🔍 Indexer: %s", event.Release.Indexer))
	sb.WriteString(fmt.Sprintf("\n💾 Client: %s", event.DownloadClient.Name))

	return n.sendMessage(ctx, sb.String())
}

func (n *Notifier) OnDownload(ctx context.Context, event types.DownloadEvent) error {
	var sb strings.Builder
	sb.WriteString("<b>✅ Download Complete</b>\n\n")

	if event.Movie != nil {
		sb.WriteString(fmt.Sprintf("<b>%s</b>", html.EscapeString(event.Movie.Title)))
		if event.Movie.Year > 0 {
			sb.WriteString(fmt.Sprintf(" (%d)", event.Movie.Year))
		}
		sb.WriteString("\n")
		n.writeLinks(&sb, event.Movie.TMDbID, event.Movie.IMDbID, event.Movie.TraktID, "movie")
	} else if event.Episode != nil {
		sb.WriteString(fmt.Sprintf("<b>%s</b> S%02dE%02d\n",
			html.EscapeString(event.Episode.SeriesTitle),
			event.Episode.SeasonNumber,
			event.Episode.EpisodeNumber))
	}

	sb.WriteString(fmt.Sprintf("\n📊 Quality: %s", event.Quality))
	if event.ReleaseGroup != "" {
		sb.WriteString(fmt.Sprintf("\n👥 Group: %s", event.ReleaseGroup))
	}

	return n.sendMessage(ctx, sb.String())
}

func (n *Notifier) OnUpgrade(ctx context.Context, event types.UpgradeEvent) error {
	var sb strings.Builder
	sb.WriteString("<b>⬆️ Quality Upgraded</b>\n\n")

	if event.Movie != nil {
		sb.WriteString(fmt.Sprintf("<b>%s</b>", html.EscapeString(event.Movie.Title)))
		if event.Movie.Year > 0 {
			sb.WriteString(fmt.Sprintf(" (%d)", event.Movie.Year))
		}
		sb.WriteString("\n")
		n.writeLinks(&sb, event.Movie.TMDbID, event.Movie.IMDbID, event.Movie.TraktID, "movie")
	} else if event.Episode != nil {
		sb.WriteString(fmt.Sprintf("<b>%s</b> S%02dE%02d\n",
			html.EscapeString(event.Episode.SeriesTitle),
			event.Episode.SeasonNumber,
			event.Episode.EpisodeNumber))
	}

	sb.WriteString(fmt.Sprintf("\n📊 %s → %s", event.OldQuality, event.NewQuality))

	return n.sendMessage(ctx, sb.String())
}

func (n *Notifier) OnMovieAdded(ctx context.Context, event types.MovieAddedEvent) error {
	var sb strings.Builder
	sb.WriteString("<b>➕ Movie Added</b>\n\n")

	sb.WriteString(fmt.Sprintf("<b>%s</b>", html.EscapeString(event.Movie.Title)))
	if event.Movie.Year > 0 {
		sb.WriteString(fmt.Sprintf(" (%d)", event.Movie.Year))
	}
	sb.WriteString("\n")

	n.writeLinks(&sb, event.Movie.TMDbID, event.Movie.IMDbID, event.Movie.TraktID, "movie")

	if event.Movie.Overview != "" {
		overview := event.Movie.Overview
		if len(overview) > 200 {
			overview = overview[:197] + "..."
		}
		sb.WriteString(fmt.Sprintf("\n%s", html.EscapeString(overview)))
	}

	return n.sendMessage(ctx, sb.String())
}

func (n *Notifier) OnMovieDeleted(ctx context.Context, event types.MovieDeletedEvent) error {
	var sb strings.Builder
	sb.WriteString("<b>🗑️ Movie Deleted</b>\n\n")

	sb.WriteString(fmt.Sprintf("<b>%s</b>", html.EscapeString(event.Movie.Title)))
	if event.Movie.Year > 0 {
		sb.WriteString(fmt.Sprintf(" (%d)", event.Movie.Year))
	}
	sb.WriteString("\n")

	if event.DeletedFiles {
		sb.WriteString("\n⚠️ Files were also deleted")
	}

	return n.sendMessage(ctx, sb.String())
}

func (n *Notifier) OnSeriesAdded(ctx context.Context, event types.SeriesAddedEvent) error {
	var sb strings.Builder
	sb.WriteString("<b>➕ Series Added</b>\n\n")

	sb.WriteString(fmt.Sprintf("<b>%s</b>", html.EscapeString(event.Series.Title)))
	if event.Series.Year > 0 {
		sb.WriteString(fmt.Sprintf(" (%d)", event.Series.Year))
	}
	sb.WriteString("\n")

	n.writeSeriesLinks(&sb, event.Series.TMDbID, event.Series.IMDbID, event.Series.TVDbID, event.Series.TraktID)

	if event.Series.Overview != "" {
		overview := event.Series.Overview
		if len(overview) > 200 {
			overview = overview[:197] + "..."
		}
		sb.WriteString(fmt.Sprintf("\n%s", html.EscapeString(overview)))
	}

	return n.sendMessage(ctx, sb.String())
}

func (n *Notifier) OnSeriesDeleted(ctx context.Context, event types.SeriesDeletedEvent) error {
	var sb strings.Builder
	sb.WriteString("<b>🗑️ Series Deleted</b>\n\n")

	sb.WriteString(fmt.Sprintf("<b>%s</b>", html.EscapeString(event.Series.Title)))
	sb.WriteString("\n")

	if event.DeletedFiles {
		sb.WriteString("\n⚠️ Files were also deleted")
	}

	return n.sendMessage(ctx, sb.String())
}

func (n *Notifier) OnHealthIssue(ctx context.Context, event types.HealthEvent) error {
	emoji := "⚠️"
	if event.Type == "error" {
		emoji = "❌"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>%s Health Issue</b>\n\n", emoji))
	sb.WriteString(fmt.Sprintf("Source: %s\n", event.Source))
	sb.WriteString(fmt.Sprintf("Message: %s", html.EscapeString(event.Message)))

	return n.sendMessage(ctx, sb.String())
}

func (n *Notifier) OnHealthRestored(ctx context.Context, event types.HealthEvent) error {
	var sb strings.Builder
	sb.WriteString("<b>✅ Health Issue Resolved</b>\n\n")
	sb.WriteString(fmt.Sprintf("Source: %s\n", event.Source))
	sb.WriteString(fmt.Sprintf("Message: %s", html.EscapeString(event.Message)))

	return n.sendMessage(ctx, sb.String())
}

func (n *Notifier) OnApplicationUpdate(ctx context.Context, event types.AppUpdateEvent) error {
	var sb strings.Builder
	sb.WriteString("<b>🔄 Application Updated</b>\n\n")
	sb.WriteString(fmt.Sprintf("Version: %s → %s", event.PreviousVersion, event.NewVersion))

	return n.sendMessage(ctx, sb.String())
}

func (n *Notifier) SendMessage(ctx context.Context, event types.MessageEvent) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>%s</b>\n\n", html.EscapeString(event.Title)))
	sb.WriteString(html.EscapeString(event.Message))

	return n.sendMessage(ctx, sb.String())
}

func (n *Notifier) hasLink(link MetadataLink) bool {
	for _, l := range n.settings.MetadataLinks {
		if l == link {
			return true
		}
	}
	return false
}

func (n *Notifier) writeLinks(sb *strings.Builder, tmdbID int64, imdbID string, traktID int64, mediaType string) {
	if !n.settings.IncludeLinks {
		return
	}

	var links []string
	if n.hasLink(MetadataLinkTMDb) && tmdbID > 0 {
		links = append(links, fmt.Sprintf("<a href=\"https://www.themoviedb.org/%s/%d\">TMDb</a>", mediaType, tmdbID))
	}
	if n.hasLink(MetadataLinkIMDb) && imdbID != "" {
		links = append(links, fmt.Sprintf("<a href=\"https://www.imdb.com/title/%s\">IMDb</a>", imdbID))
	}
	if n.hasLink(MetadataLinkTrakt) && traktID > 0 {
		traktType := "movies"
		if mediaType == "tv" {
			traktType = "shows"
		}
		links = append(links, fmt.Sprintf("<a href=\"https://trakt.tv/%s/%d\">Trakt</a>", traktType, traktID))
	}

	if len(links) > 0 {
		sb.WriteString(strings.Join(links, " | "))
		sb.WriteString("\n")
	}
}

func (n *Notifier) writeSeriesLinks(sb *strings.Builder, tmdbID int64, imdbID string, tvdbID int64, traktID int64) {
	if !n.settings.IncludeLinks {
		return
	}

	var links []string
	if n.hasLink(MetadataLinkTMDb) && tmdbID > 0 {
		links = append(links, fmt.Sprintf("<a href=\"https://www.themoviedb.org/tv/%d\">TMDb</a>", tmdbID))
	}
	if n.hasLink(MetadataLinkIMDb) && imdbID != "" {
		links = append(links, fmt.Sprintf("<a href=\"https://www.imdb.com/title/%s\">IMDb</a>", imdbID))
	}
	if n.hasLink(MetadataLinkTVDb) && tvdbID > 0 {
		links = append(links, fmt.Sprintf("<a href=\"https://thetvdb.com/series/%d\">TVDb</a>", tvdbID))
	}
	if n.hasLink(MetadataLinkTrakt) && traktID > 0 {
		links = append(links, fmt.Sprintf("<a href=\"https://trakt.tv/shows/%d\">Trakt</a>", traktID))
	}

	if len(links) > 0 {
		sb.WriteString(strings.Join(links, " | "))
		sb.WriteString("\n")
	}
}

func (n *Notifier) sendMessage(ctx context.Context, text string) error {
	url := fmt.Sprintf("%s%s/sendMessage", telegramAPIBase, n.settings.BotToken)

	payload := map[string]any{
		"chat_id":    n.settings.ChatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	if n.settings.Silent {
		payload["disable_notification"] = true
	}

	if n.settings.TopicID > 0 {
		payload["message_thread_id"] = n.settings.TopicID
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
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
		var result struct {
			OK          bool   `json:"ok"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.Description != "" {
			return fmt.Errorf("telegram error: %s", result.Description)
		}
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}

	return nil
}
