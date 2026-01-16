package notification

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/notification/discord"
	"github.com/slipstream/slipstream/internal/notification/email"
	"github.com/slipstream/slipstream/internal/notification/pushover"
	"github.com/slipstream/slipstream/internal/notification/slack"
	"github.com/slipstream/slipstream/internal/notification/telegram"
	"github.com/slipstream/slipstream/internal/notification/webhook"
)

// Factory creates Notifier instances from Config
type Factory struct {
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewFactory creates a new notification factory
func NewFactory(logger zerolog.Logger) *Factory {
	return &Factory{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With().Str("component", "notification-factory").Logger(),
	}
}

// Create creates a Notifier instance from a Config
func (f *Factory) Create(cfg Config) (Notifier, error) {
	switch cfg.Type {
	case NotifierDiscord:
		return f.createDiscord(cfg)
	case NotifierTelegram:
		return f.createTelegram(cfg)
	case NotifierWebhook:
		return f.createWebhook(cfg)
	case NotifierEmail:
		return f.createEmail(cfg)
	case NotifierSlack:
		return f.createSlack(cfg)
	case NotifierPushover:
		return f.createPushover(cfg)
	default:
		return nil, fmt.Errorf("unsupported notifier type: %s", cfg.Type)
	}
}

func (f *Factory) createDiscord(cfg Config) (Notifier, error) {
	var settings discord.Settings
	if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
		return nil, fmt.Errorf("invalid discord settings: %w", err)
	}
	return discord.New(cfg.Name, settings, f.httpClient, f.logger), nil
}

func (f *Factory) createTelegram(cfg Config) (Notifier, error) {
	var settings telegram.Settings
	if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
		return nil, fmt.Errorf("invalid telegram settings: %w", err)
	}
	return telegram.New(cfg.Name, settings, f.httpClient, f.logger), nil
}

func (f *Factory) createWebhook(cfg Config) (Notifier, error) {
	var settings webhook.Settings
	if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
		return nil, fmt.Errorf("invalid webhook settings: %w", err)
	}
	return webhook.New(cfg.Name, settings, f.httpClient, f.logger), nil
}

func (f *Factory) createEmail(cfg Config) (Notifier, error) {
	var settings email.Settings
	if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
		return nil, fmt.Errorf("invalid email settings: %w", err)
	}
	return email.New(cfg.Name, settings, f.logger), nil
}

func (f *Factory) createSlack(cfg Config) (Notifier, error) {
	var settings slack.Settings
	if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
		return nil, fmt.Errorf("invalid slack settings: %w", err)
	}
	return slack.New(cfg.Name, settings, f.httpClient, f.logger), nil
}

func (f *Factory) createPushover(cfg Config) (Notifier, error) {
	var settings pushover.Settings
	if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
		return nil, fmt.Errorf("invalid pushover settings: %w", err)
	}
	return pushover.New(cfg.Name, settings, f.httpClient, f.logger), nil
}
