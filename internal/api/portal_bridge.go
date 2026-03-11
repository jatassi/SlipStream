package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/portal/autoapprove"
	"github.com/slipstream/slipstream/internal/portal/requests"
	"github.com/slipstream/slipstream/internal/portal/users"
)

type portalAutoApproveAdapter struct {
	svc *autoapprove.Service
}

func (a *portalAutoApproveAdapter) ProcessAutoApprove(req *requests.Request, user *users.User) error {
	ctx := context.Background()
	_, err := a.svc.ProcessAutoApprove(ctx, req)
	return err
}

// portalRequestSearcherAdapter adapts requests.RequestSearcher to admin.RequestSearcher interface.
type portalRequestSearcherAdapter struct {
	searcher *requests.RequestSearcher
}

func (a *portalRequestSearcherAdapter) SearchForRequestAsync(requestID int64) {
	a.searcher.SearchForRequestAsync(context.Background(), requestID)
}

// portalUserQualityProfileAdapter implements requests.UserQualityProfileGetter
type portalUserQualityProfileAdapter struct {
	usersSvc *users.Service
}

func (a *portalUserQualityProfileAdapter) GetQualityProfileID(ctx context.Context, userID int64, moduleType string) (*int64, error) {
	settings, err := a.usersSvc.GetModuleSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, ms := range settings {
		if ms.ModuleType == moduleType && ms.QualityProfileID.Valid {
			return &ms.QualityProfileID.Int64, nil
		}
	}
	return nil, nil //nolint:nilnil // no profile configured is a valid state
}

// portalQueueGetterAdapter adapts downloader.Service to requests.QueueGetter interface.
type portalQueueGetterAdapter struct {
	downloaderSvc *downloader.Service
}

func (a *portalQueueGetterAdapter) GetQueue(ctx context.Context) ([]requests.QueueItem, error) {
	resp, err := a.downloaderSvc.GetQueue(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]requests.QueueItem, len(resp.Items))
	for i := range resp.Items {
		item := &resp.Items[i]
		result[i] = requests.QueueItem{
			ID:             item.ID,
			ClientID:       item.ClientID,
			ClientName:     item.ClientName,
			Title:          item.Title,
			MediaType:      item.MediaType,
			Status:         item.Status,
			Progress:       item.Progress,
			Size:           item.Size,
			DownloadedSize: item.DownloadedSize,
			DownloadSpeed:  item.DownloadSpeed,
			ETA:            item.ETA,
			Season:         item.Season,
			Episode:        item.Episode,
			MovieID:        item.MovieID,
			SeriesID:       item.SeriesID,
			SeasonNumber:   item.SeasonNumber,
			IsSeasonPack:   item.IsSeasonPack,
		}
	}
	return result, nil
}

// portalMediaLookupAdapter adapts sqlc.Queries to requests.MediaLookup interface.
type portalMediaLookupAdapter struct {
	queries *sqlc.Queries
}

func (a *portalMediaLookupAdapter) GetMovieTmdbID(ctx context.Context, movieID int64) (*int64, error) {
	movie, err := a.queries.GetMovie(ctx, movieID)
	if err != nil {
		return nil, err
	}
	if !movie.TmdbID.Valid {
		return nil, sql.ErrNoRows
	}
	return &movie.TmdbID.Int64, nil
}

func (a *portalMediaLookupAdapter) GetSeriesTvdbID(ctx context.Context, seriesID int64) (*int64, error) {
	series, err := a.queries.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if !series.TvdbID.Valid {
		return nil, sql.ErrNoRows
	}
	return &series.TvdbID.Int64, nil
}

// portalEnabledChecker checks if the external requests portal is enabled.
type portalEnabledChecker struct {
	queries *sqlc.Queries
}

func (c *portalEnabledChecker) IsPortalEnabled(ctx context.Context) bool {
	setting, err := c.queries.GetSetting(ctx, "requests_portal_enabled")
	if err != nil {
		return true // Default to enabled if setting not found
	}
	return setting.Value != "0" && setting.Value != "false"
}

// portalUserQualityProfileAdapter is above; this is adminRequestLibraryCheckerAdapter.
// adminRequestLibraryCheckerAdapter adapts sqlc.Queries to admin.RequestLibraryChecker.
type adminRequestLibraryCheckerAdapter struct {
	queries *sqlc.Queries
}

func (a *adminRequestLibraryCheckerAdapter) CheckMovieInLibrary(ctx context.Context, tmdbID int64) (inLibrary bool, mediaID *int64, err error) {
	movie, lookupErr := a.queries.GetMovieByTmdbID(ctx, sql.NullInt64{Int64: tmdbID, Valid: true})
	if lookupErr != nil {
		if errors.Is(lookupErr, sql.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, lookupErr
	}
	return true, &movie.ID, nil
}

func (a *adminRequestLibraryCheckerAdapter) CheckSeriesInLibrary(ctx context.Context, tvdbID int64) (inLibrary bool, mediaID *int64, err error) {
	series, lookupErr := a.queries.GetSeriesByTvdbID(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if lookupErr != nil {
		if errors.Is(lookupErr, sql.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, lookupErr
	}
	return true, &series.ID, nil
}

func (a *adminRequestLibraryCheckerAdapter) SetDB(db *sql.DB) {
	a.queries = sqlc.New(db)
}

// moduleProvisionerAdapter implements requests.ModuleProvisioner by dispatching
// to the correct module's PortalProvisioner via the registry.
type moduleProvisionerAdapter struct {
	registry *module.Registry
}

func (a *moduleProvisionerAdapter) EnsureInLibrary(ctx context.Context, moduleType string, input *module.ProvisionInput) (int64, error) {
	provisioner := a.registry.GetProvisioner(moduleType)
	if provisioner == nil {
		return 0, fmt.Errorf("no provisioner registered for module type %q", moduleType)
	}
	return provisioner.EnsureInLibrary(ctx, input)
}
