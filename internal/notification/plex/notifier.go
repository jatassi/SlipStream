package plex

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/notification/types"
)

// Notifier handles Plex library refresh notifications
type Notifier struct {
	name           string
	notificationID int64
	settings       Settings
	client         *Client
	queries        *sqlc.Queries
	logger         zerolog.Logger
	serverURL      string
	sections       []LibrarySection
}

// New creates a new Plex notifier
func New(name string, notificationID int64, settings Settings, client *Client, queries *sqlc.Queries, logger zerolog.Logger) *Notifier {
	return &Notifier{
		name:           name,
		notificationID: notificationID,
		settings:       settings,
		client:         client,
		queries:        queries,
		logger:         logger.With().Str("component", "plex-notifier").Str("name", name).Logger(),
	}
}

func (n *Notifier) Type() types.NotifierType {
	return types.NotifierPlex
}

func (n *Notifier) Name() string {
	return n.name
}

func (n *Notifier) Test(ctx context.Context) error {
	if n.settings.AuthToken == "" {
		return fmt.Errorf("no auth token configured")
	}

	if n.settings.ServerID == "" {
		return fmt.Errorf("no server selected")
	}

	serverURL, err := n.getServerURL(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := n.client.TestConnection(ctx, serverURL, n.settings.AuthToken); err != nil {
		return fmt.Errorf("failed to connect to Plex server: %w", err)
	}

	sections, err := n.client.GetLibrarySections(ctx, serverURL, n.settings.AuthToken)
	if err != nil {
		return fmt.Errorf("failed to get library sections: %w", err)
	}

	if len(n.settings.SectionIDs) == 0 {
		return fmt.Errorf("no library sections configured")
	}

	foundCount := 0
	for _, targetID := range n.settings.SectionIDs {
		for _, section := range sections {
			if section.Key == targetID {
				foundCount++
				break
			}
		}
	}

	if foundCount == 0 {
		return fmt.Errorf("none of the configured library sections were found on the server")
	}

	return nil
}

func (n *Notifier) getServerURL(ctx context.Context) (string, error) {
	if n.serverURL != "" {
		return n.serverURL, nil
	}

	servers, err := n.client.GetResources(ctx, n.settings.AuthToken)
	if err != nil {
		return "", err
	}

	var targetServer *PlexServer
	for i := range servers {
		if servers[i].ClientID == n.settings.ServerID {
			targetServer = &servers[i]
			break
		}
	}

	if targetServer == nil {
		return "", fmt.Errorf("server %s not found", n.settings.ServerID)
	}

	url, err := n.client.FindServerURL(ctx, *targetServer, n.settings.AuthToken)
	if err != nil {
		return "", err
	}

	n.serverURL = url
	return url, nil
}

func (n *Notifier) getSections(ctx context.Context) ([]LibrarySection, error) {
	if len(n.sections) > 0 {
		return n.sections, nil
	}

	serverURL, err := n.getServerURL(ctx)
	if err != nil {
		return nil, err
	}

	sections, err := n.client.GetLibrarySections(ctx, serverURL, n.settings.AuthToken)
	if err != nil {
		return nil, err
	}

	n.sections = sections
	return sections, nil
}

func (n *Notifier) enqueueRefresh(ctx context.Context, path string) error {
	if !n.settings.UpdateLibrary {
		return nil
	}

	mappedPath := MapPath(path, n.settings.PathMappings)
	sections, err := n.getSections(ctx)
	if err != nil {
		n.logger.Warn().Err(err).Msg("Failed to get sections, will queue for all configured sections")
		for _, sectionID := range n.settings.SectionIDs {
			if _, queueErr := n.queries.EnqueuePlexRefresh(ctx, sqlc.EnqueuePlexRefreshParams{
				NotificationID: n.notificationID,
				ServerID:       n.settings.ServerID,
				SectionKey:     int64(sectionID),
			}); queueErr != nil {
				n.logger.Error().Err(queueErr).Int("sectionId", sectionID).Msg("Failed to enqueue refresh")
			}
		}
		return nil
	}

	section := FindMatchingSection(mappedPath, sections, n.settings.SectionIDs)
	if section == nil {
		n.logger.Warn().Str("path", mappedPath).Msg("No matching library section found for path")
		return nil
	}

	var pathVal sql.NullString
	if n.settings.UsePartialRefresh {
		pathVal = sql.NullString{String: filepath.Dir(mappedPath), Valid: true}
	}

	_, err = n.queries.EnqueuePlexRefresh(ctx, sqlc.EnqueuePlexRefreshParams{
		NotificationID: n.notificationID,
		ServerID:       n.settings.ServerID,
		SectionKey:     int64(section.Key),
		Path:           pathVal,
	})
	return err
}

func (n *Notifier) OnGrab(_ context.Context, _ types.GrabEvent) error {
	return nil
}

func (n *Notifier) OnImport(ctx context.Context, event types.ImportEvent) error {
	return n.enqueueRefresh(ctx, event.DestinationPath)
}

func (n *Notifier) OnUpgrade(ctx context.Context, event types.UpgradeEvent) error {
	return n.enqueueRefresh(ctx, event.NewPath)
}

func (n *Notifier) OnMovieAdded(_ context.Context, _ types.MovieAddedEvent) error {
	return nil
}

func (n *Notifier) OnMovieDeleted(_ context.Context, _ types.MovieDeletedEvent) error {
	return nil
}

func (n *Notifier) OnSeriesAdded(_ context.Context, _ types.SeriesAddedEvent) error {
	return nil
}

func (n *Notifier) OnSeriesDeleted(_ context.Context, _ types.SeriesDeletedEvent) error {
	return nil
}

func (n *Notifier) OnHealthIssue(_ context.Context, _ types.HealthEvent) error {
	return nil
}

func (n *Notifier) OnHealthRestored(_ context.Context, _ types.HealthEvent) error {
	return nil
}

func (n *Notifier) OnApplicationUpdate(_ context.Context, _ types.AppUpdateEvent) error {
	return nil
}

func (n *Notifier) SendMessage(_ context.Context, _ types.MessageEvent) error {
	return nil
}
