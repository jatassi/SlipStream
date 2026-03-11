package decisioning

import (
	"context"
	"strconv"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/module"
)

const (
	statusUpgradable = "upgradable"
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
	Logger         *zerolog.Logger
	BackoffChecker BackoffChecker
}

// CollectWantedItems gathers all missing and upgrade-eligible movies and episodes.
// Unlike the autosearch version, this returns items without priority sorting
// (RSS sync doesn't need release-date ordering since it matches against all items).
func CollectWantedItems(ctx context.Context, c *Collector) ([]module.SearchableItem, error) {
	var items []module.SearchableItem

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

func collectMissingMovies(ctx context.Context, c *Collector) ([]module.SearchableItem, error) {
	rows, err := c.Queries.ListMissingMovies(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]module.SearchableItem, 0, len(rows))
	for _, row := range rows {
		if row.Status == "failed" {
			continue
		}
		if c.BackoffChecker.ShouldSkip(ctx, "movie", row.ID, "missing") {
			continue
		}
		items = append(items, movieToWantedItem(ctx, c.Queries, row))
	}
	return items, nil
}

func collectUpgradeMovies(ctx context.Context, c *Collector) ([]module.SearchableItem, error) {
	rows, err := c.Queries.ListMovieUpgradeCandidates(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]module.SearchableItem, 0, len(rows))
	for _, row := range rows {
		if c.BackoffChecker.ShouldSkip(ctx, "movie", row.ID, "upgrade") {
			continue
		}
		items = append(items, MovieUpgradeCandidateToWantedItem(row))
	}
	return items, nil
}

func collectMissingEpisodes(ctx context.Context, c *Collector) ([]module.SearchableItem, error) {
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

	var items []module.SearchableItem
	for key, episodes := range seasonEpisodes {
		items = append(items, buildMissingItems(ctx, c, key.seriesID, int(key.seasonNumber), episodes)...)
	}

	return items, nil
}

func buildMissingItems(ctx context.Context, c *Collector, seriesID int64, seasonNumber int, episodes []*sqlc.ListMissingEpisodesRow) []module.SearchableItem {
	if !IsSeasonPackEligible(ctx, c.Queries, c.Logger, seriesID, seasonNumber) {
		return buildIndividualMissingItems(ctx, c, episodes)
	}

	if c.BackoffChecker.ShouldSkip(ctx, "series", seriesID, "missing") {
		return nil
	}

	item := missingEpisodeRowToSeasonWantedItem(episodes[0], seasonNumber)
	return []module.SearchableItem{item}
}

func buildIndividualMissingItems(ctx context.Context, c *Collector, episodes []*sqlc.ListMissingEpisodesRow) []module.SearchableItem {
	var items []module.SearchableItem
	for _, ep := range episodes {
		if c.BackoffChecker.ShouldSkip(ctx, "episode", ep.ID, "missing") {
			continue
		}
		items = append(items, missingEpisodeRowToWantedItem(ep))
	}
	return items
}

func collectUpgradeEpisodes(ctx context.Context, c *Collector) ([]module.SearchableItem, error) {
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

	var items []module.SearchableItem
	for key, episodes := range seasonEpisodes {
		items = append(items, buildUpgradeItems(ctx, c, key.seriesID, int(key.seasonNumber), episodes)...)
	}

	return items, nil
}

func buildUpgradeItems(ctx context.Context, c *Collector, seriesID int64, seasonNumber int, episodes []*sqlc.ListEpisodeUpgradeCandidatesRow) []module.SearchableItem {
	if !IsSeasonPackUpgradeEligible(ctx, c.Queries, c.Logger, seriesID, seasonNumber) {
		return buildIndividualUpgradeItems(ctx, c, episodes)
	}

	if c.BackoffChecker.ShouldSkip(ctx, "series", seriesID, "upgrade") {
		return nil
	}

	maxQualityID := findMaxQualityID(episodes)
	item := upgradeEpisodeRowToSeasonWantedItem(episodes[0], seasonNumber, maxQualityID)
	return []module.SearchableItem{item}
}

func buildIndividualUpgradeItems(ctx context.Context, c *Collector, episodes []*sqlc.ListEpisodeUpgradeCandidatesRow) []module.SearchableItem {
	var items []module.SearchableItem
	for _, ep := range episodes {
		if c.BackoffChecker.ShouldSkip(ctx, "episode", ep.ID, "upgrade") {
			continue
		}
		items = append(items, EpisodeUpgradeCandidateToWantedItem(ep))
	}
	return items
}

func findMaxQualityID(episodes []*sqlc.ListEpisodeUpgradeCandidatesRow) int {
	var maxQualityID int
	for _, ep := range episodes {
		if ep.CurrentQualityID.Valid && int(ep.CurrentQualityID.Int64) > maxQualityID {
			maxQualityID = int(ep.CurrentQualityID.Int64)
		}
	}
	return maxQualityID
}

// IsSeasonPackEligible checks if ALL episodes in a season are released, monitored, and missing.
func IsSeasonPackEligible(ctx context.Context, queries *sqlc.Queries, logger *zerolog.Logger, seriesID int64, seasonNumber int) bool {
	season, err := queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil {
		logger.Debug().Err(err).Int64("seriesId", seriesID).Int("season", seasonNumber).
			Msg("Season pack ineligible: failed to get season")
		return false
	}
	if !season.Monitored {
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
		if !ep.Monitored || ep.Status != "missing" {
			return false
		}
	}

	return true
}

// IsSeasonPackUpgradeEligible checks if ALL monitored episodes in a season are upgradable.
func IsSeasonPackUpgradeEligible(ctx context.Context, queries *sqlc.Queries, logger *zerolog.Logger, seriesID int64, seasonNumber int) bool {
	season, err := queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil || !season.Monitored {
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
		if !ep.Monitored {
			continue
		}
		if ep.Status != statusUpgradable {
			return false
		}
		upgradableCount++
	}

	return upgradableCount > 1
}

// Conversion helpers — return module.SearchableItem (via *module.WantedItem)

func movieToWantedItem(ctx context.Context, queries *sqlc.Queries, movie *sqlc.Movie) module.SearchableItem {
	extIDs := buildMovieExternalIDs(movie)
	extra := map[string]any{}
	if movie.Year.Valid {
		extra["year"] = int(movie.Year.Int64)
	}

	var profileID int64
	if movie.QualityProfileID.Valid {
		profileID = movie.QualityProfileID.Int64
	}

	var currentQID *int64
	if movie.Status == "upgradable" || movie.Status == "available" {
		currentQID = findMovieMaxQuality(ctx, queries, movie.ID)
	}

	return module.NewWantedItem(module.TypeMovie, string(MediaTypeMovie), movie.ID, movie.Title, extIDs, profileID, currentQID, module.SearchParams{Extra: extra})
}

func findMovieMaxQuality(ctx context.Context, queries *sqlc.Queries, movieID int64) *int64 {
	files, err := queries.GetMovieFilesWithImportInfo(ctx, movieID)
	if err != nil || len(files) == 0 {
		qid := int64(0)
		return &qid // HasFile=true but quality unknown
	}

	var maxQID int64
	for _, f := range files {
		if f.QualityID.Valid && f.QualityID.Int64 > maxQID {
			maxQID = f.QualityID.Int64
		}
	}
	return &maxQID
}

func buildMovieExternalIDs(movie *sqlc.Movie) map[string]string {
	extIDs := make(map[string]string)
	if movie.ImdbID.Valid {
		extIDs["imdbId"] = movie.ImdbID.String
	}
	if movie.TmdbID.Valid {
		extIDs["tmdbId"] = strconv.FormatInt(movie.TmdbID.Int64, 10)
	}
	return extIDs
}

// MovieUpgradeCandidateToWantedItem converts an upgrade candidate movie to a module.SearchableItem.
func MovieUpgradeCandidateToWantedItem(movie *sqlc.ListMovieUpgradeCandidatesRow) module.SearchableItem {
	extIDs := make(map[string]string)
	if movie.ImdbID.Valid {
		extIDs["imdbId"] = movie.ImdbID.String
	}
	if movie.TmdbID.Valid {
		extIDs["tmdbId"] = strconv.FormatInt(movie.TmdbID.Int64, 10)
	}

	extra := map[string]any{}
	if movie.Year.Valid {
		extra["year"] = int(movie.Year.Int64)
	}

	var profileID int64
	if movie.QualityProfileID.Valid {
		profileID = movie.QualityProfileID.Int64
	}

	var currentQID *int64
	if movie.CurrentQualityID.Valid {
		qid := movie.CurrentQualityID.Int64
		currentQID = &qid
	} else {
		qid := int64(0)
		currentQID = &qid
	}

	return module.NewWantedItem(module.TypeMovie, string(MediaTypeMovie), movie.ID, movie.Title, extIDs, profileID, currentQID, module.SearchParams{Extra: extra})
}

// EpisodeUpgradeCandidateToWantedItem converts an upgrade candidate row to a module.SearchableItem.
func EpisodeUpgradeCandidateToWantedItem(row *sqlc.ListEpisodeUpgradeCandidatesRow) module.SearchableItem {
	extIDs := buildSeriesExternalIDsFromUpgradeRow(row)
	extra := map[string]any{
		"seriesId":      row.SeriesID,
		"seasonNumber":  int(row.SeasonNumber),
		"episodeNumber": int(row.EpisodeNumber),
	}
	if row.SeriesYear.Valid {
		extra["year"] = int(row.SeriesYear.Int64)
	}

	var profileID int64
	if row.SeriesQualityProfileID.Valid {
		profileID = row.SeriesQualityProfileID.Int64
	}

	var currentQID *int64
	if row.CurrentQualityID.Valid {
		qid := row.CurrentQualityID.Int64
		currentQID = &qid
	} else {
		qid := int64(0)
		currentQID = &qid
	}

	return module.NewWantedItem(module.TypeTV, string(MediaTypeEpisode), row.ID, row.SeriesTitle, extIDs, profileID, currentQID, module.SearchParams{Extra: extra})
}

func buildSeriesExternalIDsFromUpgradeRow(row *sqlc.ListEpisodeUpgradeCandidatesRow) map[string]string {
	extIDs := make(map[string]string)
	if row.SeriesTvdbID.Valid {
		extIDs["tvdbId"] = strconv.FormatInt(row.SeriesTvdbID.Int64, 10)
	}
	if row.SeriesTmdbID.Valid {
		extIDs["tmdbId"] = strconv.FormatInt(row.SeriesTmdbID.Int64, 10)
	}
	if row.SeriesImdbID.Valid {
		extIDs["imdbId"] = row.SeriesImdbID.String
	}
	return extIDs
}

func missingEpisodeRowToWantedItem(ep *sqlc.ListMissingEpisodesRow) module.SearchableItem {
	extIDs := buildSeriesExternalIDsFromMissingRow(ep)
	extra := map[string]any{
		"seriesId":      ep.SeriesID,
		"seasonNumber":  int(ep.SeasonNumber),
		"episodeNumber": int(ep.EpisodeNumber),
	}
	if ep.SeriesYear.Valid {
		extra["year"] = int(ep.SeriesYear.Int64)
	}

	var profileID int64
	if ep.SeriesQualityProfileID.Valid {
		profileID = ep.SeriesQualityProfileID.Int64
	}

	return module.NewWantedItem(module.TypeTV, string(MediaTypeEpisode), ep.ID, ep.SeriesTitle, extIDs, profileID, nil, module.SearchParams{Extra: extra})
}

func missingEpisodeRowToSeasonWantedItem(firstEp *sqlc.ListMissingEpisodesRow, seasonNumber int) module.SearchableItem {
	extIDs := buildSeriesExternalIDsFromMissingRow(firstEp)
	extra := map[string]any{
		"seriesId":     firstEp.SeriesID,
		"seasonNumber": seasonNumber,
	}
	if firstEp.SeriesYear.Valid {
		extra["year"] = int(firstEp.SeriesYear.Int64)
	}

	var profileID int64
	if firstEp.SeriesQualityProfileID.Valid {
		profileID = firstEp.SeriesQualityProfileID.Int64
	}

	return module.NewWantedItem(module.TypeTV, string(MediaTypeSeason), firstEp.SeriesID, firstEp.SeriesTitle, extIDs, profileID, nil, module.SearchParams{Extra: extra})
}

func buildSeriesExternalIDsFromMissingRow(ep *sqlc.ListMissingEpisodesRow) map[string]string {
	extIDs := make(map[string]string)
	if ep.SeriesTvdbID.Valid {
		extIDs["tvdbId"] = strconv.FormatInt(ep.SeriesTvdbID.Int64, 10)
	}
	if ep.SeriesTmdbID.Valid {
		extIDs["tmdbId"] = strconv.FormatInt(ep.SeriesTmdbID.Int64, 10)
	}
	if ep.SeriesImdbID.Valid {
		extIDs["imdbId"] = ep.SeriesImdbID.String
	}
	return extIDs
}

func upgradeEpisodeRowToSeasonWantedItem(firstEp *sqlc.ListEpisodeUpgradeCandidatesRow, seasonNumber, maxQualityID int) module.SearchableItem {
	extIDs := buildSeriesExternalIDsFromUpgradeRow(firstEp)
	extra := map[string]any{
		"seriesId":     firstEp.SeriesID,
		"seasonNumber": seasonNumber,
	}
	if firstEp.SeriesYear.Valid {
		extra["year"] = int(firstEp.SeriesYear.Int64)
	}

	var profileID int64
	if firstEp.SeriesQualityProfileID.Valid {
		profileID = firstEp.SeriesQualityProfileID.Int64
	}

	qid := int64(maxQualityID)
	return module.NewWantedItem(module.TypeTV, string(MediaTypeSeason), firstEp.SeriesID, firstEp.SeriesTitle, extIDs, profileID, &qid, module.SearchParams{Extra: extra})
}

// MovieToWantedItem converts a movie to a module.SearchableItem.
// Exported for use by both autosearch and RSS sync.
func MovieToWantedItem(ctx context.Context, queries *sqlc.Queries, logger *zerolog.Logger, movie *sqlc.Movie) module.SearchableItem {
	return movieToWantedItem(ctx, queries, movie)
}

// EpisodeToWantedItem converts an episode and series to a module.SearchableItem.
// Exported for use by both autosearch and RSS sync when they need ad-hoc conversion.
func EpisodeToWantedItem(ctx context.Context, queries *sqlc.Queries, logger *zerolog.Logger, episode *sqlc.Episode, series *sqlc.Series) module.SearchableItem {
	extIDs := buildSeriesExternalIDsFromSeries(series)
	extra := map[string]any{
		"seriesId":      series.ID,
		"seasonNumber":  int(episode.SeasonNumber),
		"episodeNumber": int(episode.EpisodeNumber),
	}
	if series.Year.Valid {
		extra["year"] = int(series.Year.Int64)
	}

	var profileID int64
	if series.QualityProfileID.Valid {
		profileID = series.QualityProfileID.Int64
	}

	var currentQID *int64
	if episode.Status == "upgradable" || episode.Status == "available" {
		currentQID = findEpisodeMaxQuality(ctx, queries, episode.ID)
	}

	return module.NewWantedItem(module.TypeTV, string(MediaTypeEpisode), episode.ID, series.Title, extIDs, profileID, currentQID, module.SearchParams{Extra: extra})
}

func findEpisodeMaxQuality(ctx context.Context, queries *sqlc.Queries, episodeID int64) *int64 {
	files, err := queries.ListEpisodeFilesByEpisode(ctx, episodeID)
	if err != nil || len(files) == 0 {
		qid := int64(0)
		return &qid
	}

	var maxQID int64
	for _, f := range files {
		if f.QualityID.Valid && f.QualityID.Int64 > maxQID {
			maxQID = f.QualityID.Int64
		}
	}
	return &maxQID
}

func buildSeriesExternalIDsFromSeries(series *sqlc.Series) map[string]string {
	extIDs := make(map[string]string)
	if series.TvdbID.Valid {
		extIDs["tvdbId"] = strconv.FormatInt(series.TvdbID.Int64, 10)
	}
	if series.TmdbID.Valid {
		extIDs["tmdbId"] = strconv.FormatInt(series.TmdbID.Int64, 10)
	}
	if series.ImdbID.Valid {
		extIDs["imdbId"] = series.ImdbID.String
	}
	return extIDs
}

// SeriesToSeasonPackWantedItem converts a series and season number to a season pack module.SearchableItem.
func SeriesToSeasonPackWantedItem(series *sqlc.Series, seasonNumber int) module.SearchableItem {
	extIDs := buildSeriesExternalIDsFromSeries(series)
	extra := map[string]any{
		"seriesId":     series.ID,
		"seasonNumber": seasonNumber,
	}
	if series.Year.Valid {
		extra["year"] = int(series.Year.Int64)
	}

	var profileID int64
	if series.QualityProfileID.Valid {
		profileID = series.QualityProfileID.Int64
	}

	return module.NewWantedItem(module.TypeTV, string(MediaTypeSeason), series.ID, series.Title, extIDs, profileID, nil, module.SearchParams{Extra: extra})
}
