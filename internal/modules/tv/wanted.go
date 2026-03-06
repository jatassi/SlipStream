package tv

import (
	"context"
	"fmt"

	"github.com/slipstream/slipstream/internal/module"
)

func (m *Module) CollectMissing(ctx context.Context) ([]module.SearchableItem, error) {
	rows, err := m.queries.ListMissingEpisodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("tv: collect missing: %w", err)
	}

	items := make([]module.SearchableItem, 0, len(rows))
	for _, row := range rows {
		externalIDs := make(map[string]string)
		if row.SeriesTvdbID.Valid {
			externalIDs["tvdb"] = fmt.Sprintf("%d", row.SeriesTvdbID.Int64)
		}
		if row.SeriesTmdbID.Valid {
			externalIDs["tmdb"] = fmt.Sprintf("%d", row.SeriesTmdbID.Int64)
		}
		if row.SeriesImdbID.Valid && row.SeriesImdbID.String != "" {
			externalIDs["imdb"] = row.SeriesImdbID.String
		}

		var profileID int64
		if row.SeriesQualityProfileID.Valid {
			profileID = row.SeriesQualityProfileID.Int64
		}

		items = append(items, module.NewWantedItem(
			module.TypeTV, "episode", row.ID, row.SeriesTitle,
			externalIDs, profileID, nil,
			module.SearchParams{
				Extra: map[string]any{
					"seriesId":      row.SeriesID,
					"seasonNumber":  int(row.SeasonNumber),
					"episodeNumber": int(row.EpisodeNumber),
				},
			},
		))
	}

	return items, nil
}

func (m *Module) CollectUpgradable(ctx context.Context) ([]module.SearchableItem, error) {
	rows, err := m.queries.ListUpgradableEpisodesWithQuality(ctx)
	if err != nil {
		return nil, fmt.Errorf("tv: collect upgradable: %w", err)
	}

	items := make([]module.SearchableItem, 0, len(rows))
	for _, row := range rows {
		externalIDs := make(map[string]string)
		if row.SeriesTvdbID.Valid {
			externalIDs["tvdb"] = fmt.Sprintf("%d", row.SeriesTvdbID.Int64)
		}
		if row.SeriesTmdbID.Valid {
			externalIDs["tmdb"] = fmt.Sprintf("%d", row.SeriesTmdbID.Int64)
		}
		if row.SeriesImdbID.Valid && row.SeriesImdbID.String != "" {
			externalIDs["imdb"] = row.SeriesImdbID.String
		}

		var profileID int64
		if row.SeriesQualityProfileID.Valid {
			profileID = row.SeriesQualityProfileID.Int64
		}

		var currentQIDPtr *int64
		if row.CurrentQualityID.Valid {
			v := row.CurrentQualityID.Int64
			currentQIDPtr = &v
		}

		items = append(items, module.NewWantedItem(
			module.TypeTV, "episode", row.ID, row.SeriesTitle,
			externalIDs, profileID, currentQIDPtr,
			module.SearchParams{
				Extra: map[string]any{
					"seriesId":      row.SeriesID,
					"seasonNumber":  int(row.SeasonNumber),
					"episodeNumber": int(row.EpisodeNumber),
				},
			},
		))
	}

	return items, nil
}
