package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/notification"
	"github.com/slipstream/slipstream/internal/notification/plex"
	"github.com/slipstream/slipstream/internal/scheduler"
)

const (
	maxPathsBeforeFullRefresh = 10
	refreshDelayBetweenCalls  = 1 * time.Second
)

// PlexRefreshTask processes the Plex library refresh queue
type PlexRefreshTask struct {
	queries             *sqlc.Queries
	notificationService *notification.Service
	client              *plex.Client
	logger              *zerolog.Logger
}

// NewPlexRefreshTask creates a new Plex refresh task
func NewPlexRefreshTask(
	queries *sqlc.Queries,
	notificationService *notification.Service,
	client *plex.Client,
	logger *zerolog.Logger,
) *PlexRefreshTask {
	subLogger := logger.With().Str("task", "plex-refresh").Logger()
	return &PlexRefreshTask{
		queries:             queries,
		notificationService: notificationService,
		client:              client,
		logger:              &subLogger,
	}
}

// Run processes the Plex refresh queue
func (t *PlexRefreshTask) Run(ctx context.Context) error {
	counts, err := t.queries.CountPendingPlexRefreshesPerSection(ctx)
	if err != nil {
		return err
	}

	if len(counts) == 0 {
		return nil
	}

	t.logger.Info().Int("sections", len(counts)).Msg("Processing Plex refresh queue")

	for _, count := range counts {
		if err := t.processSection(ctx, count); err != nil {
			t.logger.Error().Err(err).
				Int64("notificationId", count.NotificationID).
				Str("serverId", count.ServerID).
				Int64("sectionKey", count.SectionKey).
				Msg("Failed to process section refresh")
		}
	}

	return nil
}

func (t *PlexRefreshTask) processSection(ctx context.Context, count *sqlc.CountPendingPlexRefreshesPerSectionRow) error {
	cfg, err := t.notificationService.Get(ctx, count.NotificationID)
	if err != nil {
		return err
	}

	var settings plex.Settings
	if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
		return err
	}

	serverURL, err := t.getServerURL(ctx, &settings)
	if err != nil {
		return err
	}

	t.refreshSection(ctx, count, serverURL, &settings)

	if err := t.queries.ClearPlexRefreshesBySection(ctx, sqlc.ClearPlexRefreshesBySectionParams{
		NotificationID: count.NotificationID,
		ServerID:       count.ServerID,
		SectionKey:     count.SectionKey,
	}); err != nil {
		t.logger.Error().Err(err).Msg("Failed to clear processed refresh items")
	}

	time.Sleep(refreshDelayBetweenCalls)
	return nil
}

func (t *PlexRefreshTask) refreshSection(ctx context.Context, count *sqlc.CountPendingPlexRefreshesPerSectionRow, serverURL string, settings *plex.Settings) {
	useFullRefresh := count.Count > maxPathsBeforeFullRefresh || !settings.UsePartialRefresh

	if useFullRefresh {
		if err := t.doFullRefresh(ctx, serverURL, int(count.SectionKey), settings.AuthToken); err != nil {
			t.logger.Error().Err(err).Int64("sectionKey", count.SectionKey).Msg("Full section refresh failed")
		}
		return
	}

	err := t.doPartialRefreshes(ctx, count, serverURL, settings.AuthToken)
	if err == nil {
		return
	}
	t.logger.Warn().Err(err).Int64("sectionKey", count.SectionKey).Msg("Partial refresh failed, attempting full refresh")
	if err := t.doFullRefresh(ctx, serverURL, int(count.SectionKey), settings.AuthToken); err != nil {
		t.logger.Error().Err(err).Int64("sectionKey", count.SectionKey).Msg("Full section refresh also failed")
	}
}

func (t *PlexRefreshTask) getServerURL(ctx context.Context, settings *plex.Settings) (string, error) {
	servers, err := t.client.GetResources(ctx, settings.AuthToken)
	if err != nil {
		return "", err
	}

	var targetServer *plex.PlexServer
	for i := range servers {
		if servers[i].ClientID == settings.ServerID {
			targetServer = &servers[i]
			break
		}
	}

	if targetServer == nil {
		return "", sql.ErrNoRows
	}

	return t.client.FindServerURL(ctx, targetServer, settings.AuthToken)
}

func (t *PlexRefreshTask) doFullRefresh(ctx context.Context, serverURL string, sectionKey int, token string) error {
	t.logger.Info().Int("sectionKey", sectionKey).Msg("Performing full section refresh")
	return t.client.RefreshSection(ctx, serverURL, sectionKey, token)
}

func (t *PlexRefreshTask) doPartialRefreshes(ctx context.Context, count *sqlc.CountPendingPlexRefreshesPerSectionRow, serverURL, token string) error {
	items, err := t.queries.GetPendingPlexRefreshesBySection(ctx, sqlc.GetPendingPlexRefreshesBySectionParams{
		NotificationID: count.NotificationID,
		ServerID:       count.ServerID,
		SectionKey:     count.SectionKey,
	})
	if err != nil {
		return err
	}

	for _, item := range items {
		if !item.Path.Valid || item.Path.String == "" {
			continue
		}

		t.logger.Debug().
			Int64("sectionKey", count.SectionKey).
			Str("path", item.Path.String).
			Msg("Performing partial path refresh")

		if err := t.client.RefreshPath(ctx, serverURL, int(count.SectionKey), item.Path.String, token); err != nil {
			return err
		}

		time.Sleep(refreshDelayBetweenCalls)
	}

	return nil
}

// RegisterPlexRefreshTask registers the Plex refresh task with the scheduler
func RegisterPlexRefreshTask(
	sched *scheduler.Scheduler,
	queries *sqlc.Queries,
	notificationService *notification.Service,
	client *plex.Client,
	logger *zerolog.Logger,
) error {
	task := NewPlexRefreshTask(queries, notificationService, client, logger)

	return sched.RegisterTask(&scheduler.TaskConfig{
		ID:          "plex-refresh",
		Name:        "Plex Library Refresh",
		Description: "Processes queued Plex library refresh requests",
		Cron:        "*/5 * * * *", // Every 5 minutes
		RunOnStart:  false,
		Func:        task.Run,
	})
}
