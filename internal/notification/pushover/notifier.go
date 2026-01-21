package pushover

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/notification/types"
)

const pushoverAPIURL = "https://api.pushover.net/1/messages.json"

// Priority levels for Pushover notifications
type Priority int

const (
	PrioritySilent    Priority = -2
	PriorityQuiet     Priority = -1
	PriorityNormal    Priority = 0
	PriorityHigh      Priority = 1
	PriorityEmergency Priority = 2
)

// Settings contains Pushover-specific configuration
type Settings struct {
	UserKey  string   `json:"userKey"`
	APIToken string   `json:"apiToken"`
	Devices  string   `json:"devices,omitempty"`
	Priority Priority `json:"priority,omitempty"`
	Retry    int      `json:"retry,omitempty"`
	Expire   int      `json:"expire,omitempty"`
	TTL      int      `json:"ttl,omitempty"`
	Sound    string   `json:"sound,omitempty"`
}

// Notifier sends notifications via Pushover
type Notifier struct {
	name       string
	settings   Settings
	httpClient *http.Client
	logger     zerolog.Logger
}

// New creates a new Pushover notifier
func New(name string, settings Settings, httpClient *http.Client, logger zerolog.Logger) *Notifier {
	if settings.Retry == 0 {
		settings.Retry = 60
	}
	if settings.Expire == 0 {
		settings.Expire = 3600
	}
	if settings.Retry < 30 {
		settings.Retry = 30
	}
	return &Notifier{
		name:       name,
		settings:   settings,
		httpClient: httpClient,
		logger:     logger.With().Str("notifier", "pushover").Str("name", name).Logger(),
	}
}

func (n *Notifier) Type() types.NotifierType {
	return types.NotifierPushover
}

func (n *Notifier) Name() string {
	return n.name
}

func (n *Notifier) Test(ctx context.Context) error {
	return n.sendMessage(ctx, "SlipStream Test", "This is a test notification from SlipStream.", "")
}

func (n *Notifier) OnGrab(ctx context.Context, event types.GrabEvent) error {
	var title, message string

	if event.Movie != nil {
		title = "Movie Grabbed"
		message = fmt.Sprintf("%s", event.Movie.Title)
		if event.Movie.Year > 0 {
			message = fmt.Sprintf("%s (%d)", event.Movie.Title, event.Movie.Year)
		}
	} else if event.Episode != nil {
		title = "Episode Grabbed"
		message = fmt.Sprintf("%s S%02dE%02d", event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber)
	}

	message += fmt.Sprintf("\n\nQuality: %s\nIndexer: %s", event.Release.Quality, event.Release.Indexer)

	return n.sendMessage(ctx, title, message, "")
}

func (n *Notifier) OnDownload(ctx context.Context, event types.DownloadEvent) error {
	var title, message string

	if event.Movie != nil {
		title = "Movie Downloaded"
		message = fmt.Sprintf("%s", event.Movie.Title)
		if event.Movie.Year > 0 {
			message = fmt.Sprintf("%s (%d)", event.Movie.Title, event.Movie.Year)
		}
	} else if event.Episode != nil {
		title = "Episode Downloaded"
		message = fmt.Sprintf("%s S%02dE%02d", event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber)
	}

	message += fmt.Sprintf("\n\nQuality: %s", event.Quality)

	return n.sendMessage(ctx, title, message, "")
}

func (n *Notifier) OnUpgrade(ctx context.Context, event types.UpgradeEvent) error {
	var title, message string

	if event.Movie != nil {
		title = "Movie Upgraded"
		message = fmt.Sprintf("%s", event.Movie.Title)
		if event.Movie.Year > 0 {
			message = fmt.Sprintf("%s (%d)", event.Movie.Title, event.Movie.Year)
		}
	} else if event.Episode != nil {
		title = "Episode Upgraded"
		message = fmt.Sprintf("%s S%02dE%02d", event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber)
	}

	message += fmt.Sprintf("\n\n%s → %s", event.OldQuality, event.NewQuality)

	return n.sendMessage(ctx, title, message, "")
}

func (n *Notifier) OnMovieAdded(ctx context.Context, event types.MovieAddedEvent) error {
	title := "Movie Added"
	message := fmt.Sprintf("%s", event.Movie.Title)
	if event.Movie.Year > 0 {
		message = fmt.Sprintf("%s (%d)", event.Movie.Title, event.Movie.Year)
	}

	tmdbURL := ""
	if event.Movie.TMDbID > 0 {
		tmdbURL = fmt.Sprintf("https://www.themoviedb.org/movie/%d", event.Movie.TMDbID)
	}

	return n.sendMessage(ctx, title, message, tmdbURL)
}

func (n *Notifier) OnMovieDeleted(ctx context.Context, event types.MovieDeletedEvent) error {
	title := "Movie Deleted"
	message := fmt.Sprintf("%s", event.Movie.Title)
	if event.Movie.Year > 0 {
		message = fmt.Sprintf("%s (%d)", event.Movie.Title, event.Movie.Year)
	}

	if event.DeletedFiles {
		message += "\n\nFiles were also deleted"
	}

	return n.sendMessage(ctx, title, message, "")
}

func (n *Notifier) OnSeriesAdded(ctx context.Context, event types.SeriesAddedEvent) error {
	title := "Series Added"
	message := fmt.Sprintf("%s", event.Series.Title)
	if event.Series.Year > 0 {
		message = fmt.Sprintf("%s (%d)", event.Series.Title, event.Series.Year)
	}

	tmdbURL := ""
	if event.Series.TMDbID > 0 {
		tmdbURL = fmt.Sprintf("https://www.themoviedb.org/tv/%d", event.Series.TMDbID)
	}

	return n.sendMessage(ctx, title, message, tmdbURL)
}

func (n *Notifier) OnSeriesDeleted(ctx context.Context, event types.SeriesDeletedEvent) error {
	title := "Series Deleted"
	message := event.Series.Title

	if event.DeletedFiles {
		message += "\n\nFiles were also deleted"
	}

	return n.sendMessage(ctx, title, message, "")
}

func (n *Notifier) OnHealthIssue(ctx context.Context, event types.HealthEvent) error {
	title := "Health Issue"
	message := fmt.Sprintf("[%s] %s", event.Source, event.Message)

	return n.sendMessage(ctx, title, message, event.WikiURL)
}

func (n *Notifier) OnHealthRestored(ctx context.Context, event types.HealthEvent) error {
	title := "Health Issue Resolved"
	message := fmt.Sprintf("[%s] %s", event.Source, event.Message)

	return n.sendMessage(ctx, title, message, "")
}

func (n *Notifier) OnApplicationUpdate(ctx context.Context, event types.AppUpdateEvent) error {
	title := "Application Updated"
	message := fmt.Sprintf("SlipStream has been updated from %s to %s", event.PreviousVersion, event.NewVersion)

	return n.sendMessage(ctx, title, message, "")
}

func (n *Notifier) SendMessage(ctx context.Context, event types.MessageEvent) error {
	return n.sendMessage(ctx, event.Title, event.Message, "")
}

func (n *Notifier) sendMessage(ctx context.Context, title, message, urlStr string) error {
	form := url.Values{}
	form.Set("token", n.settings.APIToken)
	form.Set("user", n.settings.UserKey)
	form.Set("title", title)
	form.Set("message", message)
	form.Set("priority", strconv.Itoa(int(n.settings.Priority)))

	if n.settings.Priority == PriorityEmergency {
		form.Set("retry", strconv.Itoa(n.settings.Retry))
		form.Set("expire", strconv.Itoa(n.settings.Expire))
	}

	if n.settings.TTL > 0 {
		form.Set("ttl", strconv.Itoa(n.settings.TTL))
	}

	if n.settings.Devices != "" {
		form.Set("device", n.settings.Devices)
	}

	if n.settings.Sound != "" {
		form.Set("sound", n.settings.Sound)
	}

	if urlStr != "" {
		form.Set("url", urlStr)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushoverAPIURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pushover returned status %d", resp.StatusCode)
	}

	return nil
}
