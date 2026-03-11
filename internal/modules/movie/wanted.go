package movie

import (
	"context"
	"fmt"

	"github.com/slipstream/slipstream/internal/module"
)

func (m *Module) CollectMissing(ctx context.Context) ([]module.SearchableItem, error) {
	rows, err := m.queries.ListMissingMovies(ctx)
	if err != nil {
		return nil, fmt.Errorf("movie: collect missing: %w", err)
	}

	items := make([]module.SearchableItem, 0, len(rows))
	for _, row := range rows {
		externalIDs := make(map[string]string)
		if row.TmdbID.Valid {
			externalIDs["tmdbId"] = fmt.Sprintf("%d", row.TmdbID.Int64)
		}
		if row.ImdbID.Valid && row.ImdbID.String != "" {
			externalIDs["imdbId"] = row.ImdbID.String
		}
		if row.TvdbID.Valid {
			externalIDs["tvdbId"] = fmt.Sprintf("%d", row.TvdbID.Int64)
		}

		var profileID int64
		if row.QualityProfileID.Valid {
			profileID = row.QualityProfileID.Int64
		}

		params := module.SearchParams{}
		if row.Year.Valid {
			params.Extra = map[string]any{"year": int(row.Year.Int64)}
		}

		items = append(items, module.NewWantedItem(
			module.TypeMovie, "movie", row.ID, row.Title,
			externalIDs, profileID, nil, params,
		))
	}

	return items, nil
}

func (m *Module) CollectUpgradable(ctx context.Context) ([]module.SearchableItem, error) {
	rows, err := m.queries.ListUpgradableMoviesWithQuality(ctx)
	if err != nil {
		return nil, fmt.Errorf("movie: collect upgradable: %w", err)
	}

	items := make([]module.SearchableItem, 0, len(rows))
	for _, row := range rows {
		externalIDs := make(map[string]string)
		if row.TmdbID.Valid {
			externalIDs["tmdbId"] = fmt.Sprintf("%d", row.TmdbID.Int64)
		}
		if row.ImdbID.Valid && row.ImdbID.String != "" {
			externalIDs["imdbId"] = row.ImdbID.String
		}
		if row.TvdbID.Valid {
			externalIDs["tvdbId"] = fmt.Sprintf("%d", row.TvdbID.Int64)
		}

		var profileID int64
		if row.QualityProfileID.Valid {
			profileID = row.QualityProfileID.Int64
		}

		var currentQIDPtr *int64
		if row.CurrentQualityID.Valid {
			v := row.CurrentQualityID.Int64
			currentQIDPtr = &v
		}

		params := module.SearchParams{}
		if row.Year.Valid {
			params.Extra = map[string]any{"year": int(row.Year.Int64)}
		}

		items = append(items, module.NewWantedItem(
			module.TypeMovie, "movie", row.ID, row.Title,
			externalIDs, profileID, currentQIDPtr, params,
		))
	}

	return items, nil
}
