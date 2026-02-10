package decisioning

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// BackoffChecker determines whether an item should be skipped due to search backoff.
// Returns true if the item should be skipped.
type BackoffChecker interface {
	ShouldSkip(ctx context.Context, itemType string, itemID int64, searchType string) bool
}

// NoBackoff is a BackoffChecker that never skips items. Used by RSS sync.
type NoBackoff struct{}

func (NoBackoff) ShouldSkip(context.Context, string, int64, string) bool { return false }

// Collector provides database access and configuration for collecting wanted items.
type Collector struct {
	Queries        *sqlc.Queries
	Logger         zerolog.Logger
	BackoffChecker BackoffChecker
}

// CollectWantedItems gathers all missing and upgrade-eligible movies and episodes.
// Unlike the autosearch version, this returns items without priority sorting
// (RSS sync doesn't need release-date ordering since it matches against all items).
func CollectWantedItems(ctx context.Context, c *Collector) ([]SearchableItem, error) {
	var items []SearchableItem

	missing, err := collectMissingMovies(ctx, c)
	if err != nil {
		return nil, err
	}
	items = append(items, missing...)

	upgradeMovies, err := collectUpgradeMovies(ctx, c)
	if err != nil {
		return nil, err
	}
	items = append(items, upgradeMovies...)

	missingEps, err := collectMissingEpisodes(ctx, c)
	if err != nil {
		return nil, err
	}
	items = append(items, missingEps...)

	upgradeEps, err := collectUpgradeEpisodes(ctx, c)
	if err != nil {
		return nil, err
	}
	items = append(items, upgradeEps...)

	return items, nil
}

func collectMissingMovies(ctx context.Context, c *Collector) ([]SearchableItem, error) {
	rows, err := c.Queries.ListMissingMovies(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]SearchableItem, 0, len(rows))
	for _, row := range rows {
		if row.Status == "failed" {
			continue
		}
		if c.BackoffChecker.ShouldSkip(ctx, "movie", row.ID, "missing") {
			continue
		}
		items = append(items, movieToSearchableItem(ctx, c.Queries, c.Logger, row))
	}
	return items, nil
}

func collectUpgradeMovies(ctx context.Context, c *Collector) ([]SearchableItem, error) {
	rows, err := c.Queries.ListMovieUpgradeCandidates(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]SearchableItem, 0, len(rows))
	for _, row := range rows {
		if c.BackoffChecker.ShouldSkip(ctx, "movie", row.ID, "upgrade") {
			continue
		}
		items = append(items, MovieUpgradeCandidateToSearchableItem(row))
	}
	return items, nil
}

func collectMissingEpisodes(ctx context.Context, c *Collector) ([]SearchableItem, error) {
	rows, err := c.Queries.ListMissingEpisodes(ctx)
	if err != nil {
		return nil, err
	}

	type seasonKey struct {
		seriesID     int64
		seasonNumber int64
	}
	seasonEpisodes := make(map[seasonKey][]*sqlc.ListMissingEpisodesRow)

	for _, row := range rows {
		if row.Status == "failed" {
			continue
		}
		key := seasonKey{seriesID: row.SeriesID, seasonNumber: row.SeasonNumber}
		seasonEpisodes[key] = append(seasonEpisodes[key], row)
	}

	var items []SearchableItem

	for key, episodes := range seasonEpisodes {
		if IsSeasonPackEligible(ctx, c.Queries, c.Logger, key.seriesID, int(key.seasonNumber)) {
			if c.BackoffChecker.ShouldSkip(ctx, "series", key.seriesID, "missing") {
				continue
			}
			firstEp := episodes[0]
			item := missingEpisodeRowToSeasonItem(firstEp, int(key.seasonNumber))
			items = append(items, item)
		} else {
			for _, ep := range episodes {
				if c.BackoffChecker.ShouldSkip(ctx, "episode", ep.ID, "missing") {
					continue
				}
				items = append(items, missingEpisodeRowToItem(ep))
			}
		}
	}

	return items, nil
}

func collectUpgradeEpisodes(ctx context.Context, c *Collector) ([]SearchableItem, error) {
	rows, err := c.Queries.ListEpisodeUpgradeCandidates(ctx)
	if err != nil {
		return nil, err
	}

	type seasonKey struct {
		seriesID     int64
		seasonNumber int64
	}
	seasonEpisodes := make(map[seasonKey][]*sqlc.ListEpisodeUpgradeCandidatesRow)

	for _, row := range rows {
		key := seasonKey{seriesID: row.SeriesID, seasonNumber: row.SeasonNumber}
		seasonEpisodes[key] = append(seasonEpisodes[key], row)
	}

	var items []SearchableItem

	for key, episodes := range seasonEpisodes {
		if IsSeasonPackUpgradeEligible(ctx, c.Queries, c.Logger, key.seriesID, int(key.seasonNumber)) {
			if c.BackoffChecker.ShouldSkip(ctx, "series", key.seriesID, "upgrade") {
				continue
			}

			firstEp := episodes[0]
			var maxQualityID int
			for _, ep := range episodes {
				if ep.CurrentQualityID.Valid && int(ep.CurrentQualityID.Int64) > maxQualityID {
					maxQualityID = int(ep.CurrentQualityID.Int64)
				}
			}

			item := upgradeEpisodeRowToSeasonItem(firstEp, int(key.seasonNumber), maxQualityID)
			items = append(items, item)
		} else {
			for _, ep := range episodes {
				if c.BackoffChecker.ShouldSkip(ctx, "episode", ep.ID, "upgrade") {
					continue
				}
				items = append(items, EpisodeUpgradeCandidateToSearchableItem(ep))
			}
		}
	}

	return items, nil
}

// IsSeasonPackEligible checks if ALL episodes in a season are released, monitored, and missing.
func IsSeasonPackEligible(ctx context.Context, queries *sqlc.Queries, logger zerolog.Logger, seriesID int64, seasonNumber int) bool {
	season, err := queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil {
		logger.Debug().Err(err).Int64("seriesId", seriesID).Int("season", seasonNumber).
			Msg("Season pack ineligible: failed to get season")
		return false
	}
	if season.Monitored != 1 {
		return false
	}

	episodes, err := queries.ListEpisodesBySeason(ctx, sqlc.ListEpisodesBySeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil || len(episodes) <= 1 {
		return false
	}

	for _, ep := range episodes {
		if ep.Monitored != 1 || ep.Status != "missing" {
			return false
		}
	}

	return true
}

// IsSeasonPackUpgradeEligible checks if ALL monitored episodes in a season are upgradable.
func IsSeasonPackUpgradeEligible(ctx context.Context, queries *sqlc.Queries, logger zerolog.Logger, seriesID int64, seasonNumber int) bool {
	season, err := queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil || season.Monitored != 1 {
		return false
	}

	episodes, err := queries.ListEpisodesBySeason(ctx, sqlc.ListEpisodesBySeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil || len(episodes) == 0 {
		return false
	}

	upgradableCount := 0
	for _, ep := range episodes {
		if ep.Monitored != 1 {
			continue
		}
		if ep.Status != "upgradable" {
			return false
		}
		upgradableCount++
	}

	return upgradableCount > 1
}

// Conversion helpers

func movieToSearchableItem(ctx context.Context, queries *sqlc.Queries, logger zerolog.Logger, movie *sqlc.Movie) SearchableItem {
	item := SearchableItem{
		MediaType: MediaTypeMovie,
		MediaID:   movie.ID,
		Title:     movie.Title,
	}

	if movie.Status == "upgradable" || movie.Status == "available" {
		item.HasFile = true
		files, err := queries.GetMovieFilesWithImportInfo(ctx, movie.ID)
		if err == nil && len(files) > 0 {
			for _, f := range files {
				if f.QualityID.Valid && int(f.QualityID.Int64) > item.CurrentQualityID {
					item.CurrentQualityID = int(f.QualityID.Int64)
				}
			}
		}
	}

	if movie.Year.Valid {
		item.Year = int(movie.Year.Int64)
	}
	if movie.ImdbID.Valid {
		item.ImdbID = movie.ImdbID.String
	}
	if movie.TmdbID.Valid {
		item.TmdbID = int(movie.TmdbID.Int64)
	}
	if movie.QualityProfileID.Valid {
		item.QualityProfileID = movie.QualityProfileID.Int64
	}
	return item
}

// MovieUpgradeCandidateToSearchableItem converts an upgrade candidate movie to a SearchableItem.
func MovieUpgradeCandidateToSearchableItem(movie *sqlc.ListMovieUpgradeCandidatesRow) SearchableItem {
	item := SearchableItem{
		MediaType: MediaTypeMovie,
		MediaID:   movie.ID,
		Title:     movie.Title,
		HasFile:   true,
	}
	if movie.Year.Valid {
		item.Year = int(movie.Year.Int64)
	}
	if movie.ImdbID.Valid {
		item.ImdbID = movie.ImdbID.String
	}
	if movie.TmdbID.Valid {
		item.TmdbID = int(movie.TmdbID.Int64)
	}
	if movie.QualityProfileID.Valid {
		item.QualityProfileID = movie.QualityProfileID.Int64
	}
	if movie.CurrentQualityID.Valid {
		item.CurrentQualityID = int(movie.CurrentQualityID.Int64)
	}
	return item
}

// EpisodeUpgradeCandidateToSearchableItem converts an upgrade candidate row to a SearchableItem.
func EpisodeUpgradeCandidateToSearchableItem(row *sqlc.ListEpisodeUpgradeCandidatesRow) SearchableItem {
	item := SearchableItem{
		MediaType:     MediaTypeEpisode,
		MediaID:       row.ID,
		SeriesID:      row.SeriesID,
		Title:         row.SeriesTitle,
		SeasonNumber:  int(row.SeasonNumber),
		EpisodeNumber: int(row.EpisodeNumber),
		HasFile:       true,
	}
	if row.SeriesYear.Valid {
		item.Year = int(row.SeriesYear.Int64)
	}
	if row.SeriesTvdbID.Valid {
		item.TvdbID = int(row.SeriesTvdbID.Int64)
	}
	if row.SeriesTmdbID.Valid {
		item.TmdbID = int(row.SeriesTmdbID.Int64)
	}
	if row.SeriesImdbID.Valid {
		item.ImdbID = row.SeriesImdbID.String
	}
	if row.SeriesQualityProfileID.Valid {
		item.QualityProfileID = row.SeriesQualityProfileID.Int64
	}
	if row.CurrentQualityID.Valid {
		item.CurrentQualityID = int(row.CurrentQualityID.Int64)
	}
	return item
}

func missingEpisodeRowToItem(ep *sqlc.ListMissingEpisodesRow) SearchableItem {
	item := SearchableItem{
		MediaType:     MediaTypeEpisode,
		MediaID:       ep.ID,
		Title:         ep.SeriesTitle,
		SeasonNumber:  int(ep.SeasonNumber),
		EpisodeNumber: int(ep.EpisodeNumber),
	}
	if ep.SeriesYear.Valid {
		item.Year = int(ep.SeriesYear.Int64)
	}
	if ep.SeriesTvdbID.Valid {
		item.TvdbID = int(ep.SeriesTvdbID.Int64)
	}
	if ep.SeriesTmdbID.Valid {
		item.TmdbID = int(ep.SeriesTmdbID.Int64)
	}
	if ep.SeriesImdbID.Valid {
		item.ImdbID = ep.SeriesImdbID.String
	}
	if ep.SeriesQualityProfileID.Valid {
		item.QualityProfileID = ep.SeriesQualityProfileID.Int64
	}
	return item
}

func missingEpisodeRowToSeasonItem(firstEp *sqlc.ListMissingEpisodesRow, seasonNumber int) SearchableItem {
	item := SearchableItem{
		MediaType:    MediaTypeSeason,
		MediaID:      firstEp.SeriesID,
		Title:        firstEp.SeriesTitle,
		SeasonNumber: seasonNumber,
	}
	if firstEp.SeriesYear.Valid {
		item.Year = int(firstEp.SeriesYear.Int64)
	}
	if firstEp.SeriesTvdbID.Valid {
		item.TvdbID = int(firstEp.SeriesTvdbID.Int64)
	}
	if firstEp.SeriesTmdbID.Valid {
		item.TmdbID = int(firstEp.SeriesTmdbID.Int64)
	}
	if firstEp.SeriesImdbID.Valid {
		item.ImdbID = firstEp.SeriesImdbID.String
	}
	if firstEp.SeriesQualityProfileID.Valid {
		item.QualityProfileID = firstEp.SeriesQualityProfileID.Int64
	}
	return item
}

func upgradeEpisodeRowToSeasonItem(firstEp *sqlc.ListEpisodeUpgradeCandidatesRow, seasonNumber int, maxQualityID int) SearchableItem {
	item := SearchableItem{
		MediaType:        MediaTypeSeason,
		MediaID:          firstEp.SeriesID,
		SeriesID:         firstEp.SeriesID,
		Title:            firstEp.SeriesTitle,
		SeasonNumber:     seasonNumber,
		HasFile:          true,
		CurrentQualityID: maxQualityID,
	}
	if firstEp.SeriesYear.Valid {
		item.Year = int(firstEp.SeriesYear.Int64)
	}
	if firstEp.SeriesTvdbID.Valid {
		item.TvdbID = int(firstEp.SeriesTvdbID.Int64)
	}
	if firstEp.SeriesTmdbID.Valid {
		item.TmdbID = int(firstEp.SeriesTmdbID.Int64)
	}
	if firstEp.SeriesImdbID.Valid {
		item.ImdbID = firstEp.SeriesImdbID.String
	}
	if firstEp.SeriesQualityProfileID.Valid {
		item.QualityProfileID = firstEp.SeriesQualityProfileID.Int64
	}
	return item
}

// EpisodeToSearchableItem converts an episode and series to a SearchableItem.
// Exported for use by both autosearch and RSS sync when they need ad-hoc conversion.
func EpisodeToSearchableItem(ctx context.Context, queries *sqlc.Queries, logger zerolog.Logger, episode *sqlc.Episode, series *sqlc.Series) SearchableItem {
	item := SearchableItem{
		MediaType:     MediaTypeEpisode,
		MediaID:       episode.ID,
		SeriesID:      series.ID,
		SeasonNumber:  int(episode.SeasonNumber),
		EpisodeNumber: int(episode.EpisodeNumber),
		Title:         series.Title,
	}

	if episode.Status == "upgradable" || episode.Status == "available" {
		item.HasFile = true
		files, err := queries.ListEpisodeFilesByEpisode(ctx, episode.ID)
		if err == nil && len(files) > 0 {
			for _, f := range files {
				if f.QualityID.Valid && int(f.QualityID.Int64) > item.CurrentQualityID {
					item.CurrentQualityID = int(f.QualityID.Int64)
				}
			}
		}
	}

	if series.Year.Valid {
		item.Year = int(series.Year.Int64)
	}
	if series.TvdbID.Valid {
		item.TvdbID = int(series.TvdbID.Int64)
	}
	if series.TmdbID.Valid {
		item.TmdbID = int(series.TmdbID.Int64)
	}
	if series.ImdbID.Valid {
		item.ImdbID = series.ImdbID.String
	}
	if series.QualityProfileID.Valid {
		item.QualityProfileID = series.QualityProfileID.Int64
	}
	return item
}

// MovieToSearchableItem converts a movie to a SearchableItem.
// Exported for use by both autosearch and RSS sync.
func MovieToSearchableItem(ctx context.Context, queries *sqlc.Queries, logger zerolog.Logger, movie *sqlc.Movie) SearchableItem {
	return movieToSearchableItem(ctx, queries, logger, movie)
}

// SeriesToSeasonPackItem converts a series and season number to a season pack SearchableItem.
func SeriesToSeasonPackItem(series *sqlc.Series, seasonNumber int) SearchableItem {
	item := SearchableItem{
		MediaType:    MediaTypeSeason,
		MediaID:      series.ID,
		SeriesID:     series.ID,
		Title:        series.Title,
		SeasonNumber: seasonNumber,
	}
	if series.Year.Valid {
		item.Year = int(series.Year.Int64)
	}
	if series.TvdbID.Valid {
		item.TvdbID = int(series.TvdbID.Int64)
	}
	if series.TmdbID.Valid {
		item.TmdbID = int(series.TmdbID.Int64)
	}
	if series.ImdbID.Valid {
		item.ImdbID = series.ImdbID.String
	}
	if series.QualityProfileID.Valid {
		item.QualityProfileID = series.QualityProfileID.Int64
	}
	return item
}
