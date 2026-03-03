package api

import (
	"context"
	"database/sql"
	"errors"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/tv"
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

func (a *portalUserQualityProfileAdapter) GetQualityProfileID(ctx context.Context, userID int64, mediaType string) (*int64, error) {
	user, err := a.usersSvc.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user.QualityProfileIDFor(mediaType), nil
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

type statusTrackerMovieLookup struct {
	movieSvc *movies.Service
}

func (l *statusTrackerMovieLookup) GetTmdbIDByMovieID(ctx context.Context, movieID int64) (int64, error) {
	movie, err := l.movieSvc.Get(ctx, movieID)
	if err != nil {
		return 0, err
	}
	return int64(movie.TmdbID), nil
}

// statusTrackerEpisodeLookup implements requests.EpisodeLookup
type statusTrackerEpisodeLookup struct {
	tvSvc *tv.Service
}

func (l *statusTrackerEpisodeLookup) GetEpisodeInfo(ctx context.Context, episodeID int64) (tvdbID int64, seasonNum, episodeNum int, err error) {
	episode, err := l.tvSvc.GetEpisode(ctx, episodeID)
	if err != nil {
		return 0, 0, 0, err
	}

	series, err := l.tvSvc.GetSeries(ctx, episode.SeriesID)
	if err != nil {
		return 0, 0, 0, err
	}

	return int64(series.TvdbID), episode.SeasonNumber, episode.EpisodeNumber, nil
}
