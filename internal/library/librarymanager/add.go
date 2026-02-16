package librarymanager

import (
	"context"

	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/preferences"
)

type AddMovieInput struct {
	Title                 string `json:"title"`
	Year                  int    `json:"year,omitempty"`
	TmdbID                int    `json:"tmdbId,omitempty"`
	ImdbID                string `json:"imdbId,omitempty"`
	Overview              string `json:"overview,omitempty"`
	Runtime               int    `json:"runtime,omitempty"`
	Path                  string `json:"path,omitempty"`
	RootFolderID          int64  `json:"rootFolderId"`
	QualityProfileID      int64  `json:"qualityProfileId"`
	Monitored             bool   `json:"monitored"`
	PosterURL             string `json:"posterUrl,omitempty"`
	BackdropURL           string `json:"backdropUrl,omitempty"`
	ReleaseDate           string `json:"releaseDate,omitempty"`           // Digital/streaming release date
	PhysicalReleaseDate   string `json:"physicalReleaseDate,omitempty"`   // Bluray release date
	TheatricalReleaseDate string `json:"theatricalReleaseDate,omitempty"` // Theatrical release date
	Studio                string `json:"studio,omitempty"`
	ContentRating         string `json:"contentRating,omitempty"`
	SearchOnAdd           *bool  `json:"searchOnAdd,omitempty"` // Trigger autosearch after add
	AddedBy               *int64 `json:"-"`
}

// AddMovie creates a new movie and downloads artwork in the background.
func (s *Service) AddMovie(ctx context.Context, input *AddMovieInput) (*movies.Movie, error) {
	releaseDate, physicalReleaseDate, theatricalReleaseDate := s.fetchMovieReleaseDates(ctx, input)
	contentRating := s.fetchMovieContentRating(ctx, input)

	movie, err := s.movies.Create(ctx, &movies.CreateMovieInput{
		Title:                 input.Title,
		Year:                  input.Year,
		TmdbID:                input.TmdbID,
		ImdbID:                input.ImdbID,
		Overview:              input.Overview,
		Runtime:               input.Runtime,
		Path:                  input.Path,
		RootFolderID:          input.RootFolderID,
		QualityProfileID:      input.QualityProfileID,
		Monitored:             input.Monitored,
		ReleaseDate:           releaseDate,
		PhysicalReleaseDate:   physicalReleaseDate,
		TheatricalReleaseDate: theatricalReleaseDate,
		Studio:                input.Studio,
		ContentRating:         contentRating,
		AddedBy:               input.AddedBy,
	})
	if err != nil {
		return nil, err
	}

	s.downloadMovieArtworkIfNeeded(ctx, input)
	s.triggerMovieSearchIfNeeded(movie, input.SearchOnAdd)
	s.saveMoviePreferenceIfNeeded(input.SearchOnAdd)

	return movie, nil
}

func (s *Service) fetchMovieReleaseDates(ctx context.Context, input *AddMovieInput) (releaseDate, physicalReleaseDate, theatricalReleaseDate string) {
	releaseDate = input.ReleaseDate
	physicalReleaseDate = input.PhysicalReleaseDate
	theatricalReleaseDate = input.TheatricalReleaseDate

	if input.TmdbID > 0 && releaseDate == "" && physicalReleaseDate == "" && theatricalReleaseDate == "" {
		digital, physical, theatrical, err := s.metadata.GetMovieReleaseDates(ctx, input.TmdbID)
		if err != nil {
			s.logger.Warn().Err(err).Int("tmdbId", input.TmdbID).Msg("Failed to fetch release dates from TMDB")
		} else {
			return digital, physical, theatrical
		}
	}

	return releaseDate, physicalReleaseDate, theatricalReleaseDate
}

func (s *Service) fetchMovieContentRating(ctx context.Context, input *AddMovieInput) string {
	if input.ContentRating != "" || input.TmdbID == 0 {
		return input.ContentRating
	}

	if cr, err := s.metadata.GetMovieContentRating(ctx, input.TmdbID); err == nil && cr != "" {
		return cr
	}

	return input.ContentRating
}

func (s *Service) downloadMovieArtworkIfNeeded(ctx context.Context, input *AddMovieInput) {
	if s.artwork == nil || input.TmdbID == 0 {
		return
	}

	logoURL := s.fetchMovieLogoURL(ctx, input.TmdbID)
	studioLogoURL := s.fetchStudioLogoURL(ctx, input.TmdbID)

	if input.PosterURL == "" && input.BackdropURL == "" && logoURL == "" && studioLogoURL == "" {
		return
	}

	go func() {
		movieResult := &metadata.MovieResult{
			ID:            input.TmdbID,
			Title:         input.Title,
			PosterURL:     input.PosterURL,
			BackdropURL:   input.BackdropURL,
			LogoURL:       logoURL,
			StudioLogoURL: studioLogoURL,
		}
		if err := s.artwork.DownloadMovieArtwork(context.Background(), movieResult); err != nil {
			s.logger.Warn().Err(err).Int("tmdbId", input.TmdbID).Msg("Failed to download movie artwork")
		} else {
			s.logger.Info().Int("tmdbId", input.TmdbID).Msg("Movie artwork downloaded")
		}
	}()
}

func (s *Service) fetchMovieLogoURL(ctx context.Context, tmdbID int) string {
	if url, err := s.metadata.GetMovieLogoURL(ctx, tmdbID); err == nil && url != "" {
		return url
	}
	return ""
}

func (s *Service) fetchStudioLogoURL(ctx context.Context, tmdbID int) string {
	if details, err := s.metadata.GetMovie(ctx, tmdbID); err == nil {
		return details.StudioLogoURL
	}
	return ""
}

func (s *Service) triggerMovieSearchIfNeeded(movie *movies.Movie, searchOnAdd *bool) {
	if searchOnAdd == nil || !*searchOnAdd || s.autosearchSvc == nil || movie.Status == "unreleased" {
		return
	}

	go func() {
		s.logger.Info().Int64("movieId", movie.ID).Str("title", movie.Title).Msg("Triggering search-on-add for movie")
		if _, err := s.autosearchSvc.SearchMovie(context.Background(), movie.ID, autosearch.SearchSourceAdd); err != nil {
			s.logger.Warn().Err(err).Int64("movieId", movie.ID).Msg("Search-on-add failed for movie")
		}
	}()
}

func (s *Service) saveMoviePreferenceIfNeeded(searchOnAdd *bool) {
	if searchOnAdd == nil || s.preferencesSvc == nil {
		return
	}

	go func() {
		if err := s.preferencesSvc.SetMovieSearchOnAdd(context.Background(), *searchOnAdd); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to save movie search-on-add preference")
		}
	}()
}

// AddSeriesInput contains fields for adding a series with artwork.
type AddSeriesInput struct {
	Title            string           `json:"title"`
	Year             int              `json:"year,omitempty"`
	TvdbID           int              `json:"tvdbId,omitempty"`
	TmdbID           int              `json:"tmdbId,omitempty"`
	ImdbID           string           `json:"imdbId,omitempty"`
	Overview         string           `json:"overview,omitempty"`
	Runtime          int              `json:"runtime,omitempty"`
	ProductionStatus string           `json:"productionStatus,omitempty"` // "continuing", "ended", "upcoming"
	Path             string           `json:"path,omitempty"`
	RootFolderID     int64            `json:"rootFolderId"`
	QualityProfileID int64            `json:"qualityProfileId"`
	Monitored        bool             `json:"monitored"`
	SeasonFolder     bool             `json:"seasonFolder"`
	Seasons          []tv.SeasonInput `json:"seasons,omitempty"`
	Network          string           `json:"network,omitempty"`
	NetworkLogoURL   string           `json:"networkLogoUrl,omitempty"`
	PosterURL        string           `json:"posterUrl,omitempty"`
	BackdropURL      string           `json:"backdropUrl,omitempty"`

	// Search and monitoring options for add flow
	SearchOnAdd     *string `json:"searchOnAdd,omitempty"`     // "no", "first_episode", "first_season", "latest_season", "all"
	MonitorOnAdd    *string `json:"monitorOnAdd,omitempty"`    // "none", "first_season", "latest_season", "future", "all"
	IncludeSpecials *bool   `json:"includeSpecials,omitempty"` // Whether to include specials in monitoring/search

	AddedBy *int64 `json:"-"`
}

// applyMonitoringOnAdd applies the monitoring-on-add settings to a newly added series
func (s *Service) applyMonitoringOnAdd(ctx context.Context, seriesID int64, monitorOnAdd string, includeSpecials bool) error {
	monitorType := preferences.SeriesMonitorOnAdd(monitorOnAdd)
	if !preferences.ValidSeriesMonitorOnAdd(monitorOnAdd) {
		monitorType = preferences.SeriesMonitorOnAddFuture
	}

	if err := s.applyMonitoringType(ctx, seriesID, monitorType); err != nil {
		return err
	}

	if !includeSpecials {
		return s.unmonitorSpecials(ctx, seriesID)
	}

	return nil
}

func (s *Service) applyMonitoringType(ctx context.Context, seriesID int64, monitorType preferences.SeriesMonitorOnAdd) error {
	switch monitorType {
	case preferences.SeriesMonitorOnAddNone:
		return s.applyMonitorNone(ctx, seriesID)
	case preferences.SeriesMonitorOnAddFirstSeason:
		return s.applyMonitorFirstSeason(ctx, seriesID)
	case preferences.SeriesMonitorOnAddLatestSeason:
		return s.applyMonitorLatestSeason(ctx, seriesID)
	case preferences.SeriesMonitorOnAddFuture:
		return s.applyMonitorFuture(ctx, seriesID)
	case preferences.SeriesMonitorOnAddAll:
		return nil
	}
	return nil
}

func (s *Service) applyMonitorNone(ctx context.Context, seriesID int64) error {
	if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return err
	}
	if err := s.queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return err
	}
	_, err := s.tv.UpdateSeries(ctx, seriesID, &tv.UpdateSeriesInput{Monitored: boolPtr(false)})
	return err
}

func (s *Service) applyMonitorFirstSeason(ctx context.Context, seriesID int64) error {
	if err := s.unmonitorAll(ctx, seriesID); err != nil {
		return err
	}
	return s.monitorSeason(ctx, seriesID, 1)
}

func (s *Service) applyMonitorLatestSeason(ctx context.Context, seriesID int64) error {
	latestSeasonVal, err := s.queries.GetLatestSeasonNumber(ctx, seriesID)
	if err != nil {
		return err
	}

	if err := s.unmonitorAll(ctx, seriesID); err != nil {
		return err
	}

	latestSeason := s.extractSeasonNumber(latestSeasonVal)
	if latestSeason > 0 {
		return s.monitorSeason(ctx, seriesID, latestSeason)
	}
	return nil
}

func (s *Service) applyMonitorFuture(ctx context.Context, seriesID int64) error {
	if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return err
	}
	return s.queries.UpdateFutureEpisodesMonitored(ctx, sqlc.UpdateFutureEpisodesMonitoredParams{
		Monitored: 1,
		SeriesID:  seriesID,
	})
}

func (s *Service) unmonitorAll(ctx context.Context, seriesID int64) error {
	if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return err
	}
	return s.queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	})
}

func (s *Service) monitorSeason(ctx context.Context, seriesID, seasonNumber int64) error {
	if err := s.queries.UpdateEpisodesMonitoredBySeason(ctx, sqlc.UpdateEpisodesMonitoredBySeasonParams{
		Monitored:    1,
		SeriesID:     seriesID,
		SeasonNumber: seasonNumber,
	}); err != nil {
		return err
	}
	return s.queries.UpdateSeasonMonitoredByNumber(ctx, sqlc.UpdateSeasonMonitoredByNumberParams{
		Monitored:    1,
		SeriesID:     seriesID,
		SeasonNumber: seasonNumber,
	})
}

func (s *Service) extractSeasonNumber(latestSeasonVal any) int64 {
	if latestSeasonVal == nil {
		return 0
	}
	switch v := latestSeasonVal.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

func (s *Service) unmonitorSpecials(ctx context.Context, seriesID int64) error {
	if err := s.queries.UpdateEpisodesMonitoredBySeason(ctx, sqlc.UpdateEpisodesMonitoredBySeasonParams{
		Monitored:    0,
		SeriesID:     seriesID,
		SeasonNumber: 0,
	}); err != nil {
		return err
	}
	return s.queries.UpdateSeasonMonitoredByNumber(ctx, sqlc.UpdateSeasonMonitoredByNumberParams{
		Monitored:    0,
		SeriesID:     seriesID,
		SeasonNumber: 0,
	})
}

func boolPtr(b bool) *bool {
	return &b
}

// AddSeries creates a new series, fetches metadata, and downloads artwork in the background.
func (s *Service) AddSeries(ctx context.Context, input *AddSeriesInput) (*tv.Series, error) {
	series, err := s.tv.CreateSeries(ctx, &tv.CreateSeriesInput{
		Title:            input.Title,
		Year:             input.Year,
		TvdbID:           input.TvdbID,
		TmdbID:           input.TmdbID,
		ImdbID:           input.ImdbID,
		Overview:         input.Overview,
		Runtime:          input.Runtime,
		ProductionStatus: input.ProductionStatus,
		Network:          input.Network,
		NetworkLogoURL:   input.NetworkLogoURL,
		Path:             input.Path,
		RootFolderID:     input.RootFolderID,
		QualityProfileID: input.QualityProfileID,
		Monitored:        input.Monitored,
		SeasonFolder:     input.SeasonFolder,
		Seasons:          input.Seasons,
		AddedBy:          input.AddedBy,
	})
	if err != nil {
		return nil, err
	}

	s.fetchAndUpdateSeasonMetadata(ctx, series.ID, input.TmdbID, input.TvdbID)
	s.downloadSeriesArtworkAsync(ctx, input)
	s.applyMonitoringSettings(ctx, series.ID, input.MonitorOnAdd, input.IncludeSpecials)
	s.saveSeriesPreferences(input.SearchOnAdd, input.MonitorOnAdd, input.IncludeSpecials)
	s.triggerSeriesSearch(series.ID, input.SearchOnAdd)

	return s.tv.GetSeries(ctx, series.ID)
}

func (s *Service) fetchAndUpdateSeasonMetadata(ctx context.Context, seriesID int64, tmdbID, tvdbID int) {
	if tmdbID == 0 && tvdbID == 0 {
		return
	}

	seasonResults, err := s.metadata.GetSeriesSeasons(ctx, tmdbID, tvdbID)
	if err != nil {
		s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Int("tvdbId", tvdbID).Msg("Failed to fetch season metadata for new series")
		return
	}

	seasonMeta := s.convertSeasonResults(seasonResults)
	if err := s.tv.UpdateSeasonsFromMetadata(ctx, seriesID, seasonMeta); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Failed to update seasons from metadata for new series")
		return
	}

	totalEpisodes := s.countTotalEpisodes(seasonMeta)
	s.logger.Info().
		Int64("seriesId", seriesID).
		Int("seasons", len(seasonMeta)).
		Int("episodes", totalEpisodes).
		Msg("Updated seasons and episodes for new series")
}

func (s *Service) convertSeasonResults(seasonResults []metadata.SeasonResult) []tv.SeasonMetadata {
	seasonMeta := make([]tv.SeasonMetadata, len(seasonResults))
	for i, sr := range seasonResults {
		episodes := make([]tv.EpisodeMetadata, len(sr.Episodes))
		for j, ep := range sr.Episodes {
			episodes[j] = tv.EpisodeMetadata{
				EpisodeNumber: ep.EpisodeNumber,
				SeasonNumber:  ep.SeasonNumber,
				Title:         ep.Title,
				Overview:      ep.Overview,
				AirDate:       ep.AirDate,
				Runtime:       ep.Runtime,
			}
		}
		seasonMeta[i] = tv.SeasonMetadata{
			SeasonNumber: sr.SeasonNumber,
			Name:         sr.Name,
			Overview:     sr.Overview,
			PosterURL:    sr.PosterURL,
			AirDate:      sr.AirDate,
			Episodes:     episodes,
		}
	}
	return seasonMeta
}

func (s *Service) countTotalEpisodes(seasonMeta []tv.SeasonMetadata) int {
	total := 0
	for _, sm := range seasonMeta {
		total += len(sm.Episodes)
	}
	return total
}

func (s *Service) downloadSeriesArtworkAsync(ctx context.Context, input *AddSeriesInput) {
	if s.artwork == nil {
		return
	}

	artworkID := input.TmdbID
	if artworkID == 0 {
		artworkID = input.TvdbID
	}
	if artworkID == 0 {
		return
	}

	logoURL := s.fetchSeriesLogoURL(ctx, input.TmdbID)
	if input.PosterURL == "" && input.BackdropURL == "" && logoURL == "" {
		return
	}

	go func() {
		seriesResult := &metadata.SeriesResult{
			ID:          artworkID,
			TmdbID:      input.TmdbID,
			TvdbID:      input.TvdbID,
			Title:       input.Title,
			PosterURL:   input.PosterURL,
			BackdropURL: input.BackdropURL,
			LogoURL:     logoURL,
		}
		if err := s.artwork.DownloadSeriesArtwork(context.Background(), seriesResult); err != nil {
			s.logger.Warn().Err(err).Int("tmdbId", input.TmdbID).Int("tvdbId", input.TvdbID).Msg("Failed to download series artwork")
		} else {
			s.logger.Info().Int("tmdbId", input.TmdbID).Int("tvdbId", input.TvdbID).Msg("Series artwork downloaded")
		}
	}()
}

func (s *Service) fetchSeriesLogoURL(ctx context.Context, tmdbID int) string {
	if tmdbID == 0 {
		return ""
	}
	url, err := s.metadata.GetSeriesLogoURL(ctx, tmdbID)
	if err != nil || url == "" {
		return ""
	}
	return url
}

func (s *Service) applyMonitoringSettings(ctx context.Context, seriesID int64, monitorOnAdd *string, includeSpecials *bool) {
	if monitorOnAdd == nil {
		return
	}

	includeSpec := false
	if includeSpecials != nil {
		includeSpec = *includeSpecials
	}

	if err := s.applyMonitoringOnAdd(ctx, seriesID, *monitorOnAdd, includeSpec); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Failed to apply monitoring-on-add settings")
	}
}

func (s *Service) saveSeriesPreferences(searchOnAdd, monitorOnAdd *string, includeSpecials *bool) {
	if s.preferencesSvc == nil {
		return
	}

	go func() {
		s.persistSeriesPreferences(searchOnAdd, monitorOnAdd, includeSpecials)
	}()
}

func (s *Service) persistSeriesPreferences(searchOnAdd, monitorOnAdd *string, includeSpecials *bool) {
	ctx := context.Background()
	if searchOnAdd != nil {
		if err := s.preferencesSvc.SetSeriesSearchOnAdd(ctx, preferences.SeriesSearchOnAdd(*searchOnAdd)); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to save series search-on-add preference")
		}
	}
	if monitorOnAdd != nil {
		if err := s.preferencesSvc.SetSeriesMonitorOnAdd(ctx, preferences.SeriesMonitorOnAdd(*monitorOnAdd)); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to save series monitor-on-add preference")
		}
	}
	if includeSpecials != nil {
		if err := s.preferencesSvc.SetSeriesIncludeSpecials(ctx, *includeSpecials); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to save series include-specials preference")
		}
	}
}

func (s *Service) triggerSeriesSearch(seriesID int64, searchOnAdd *string) {
	if searchOnAdd == nil || *searchOnAdd == "no" || s.autosearchSvc == nil {
		return
	}

	go func() {
		s.triggerSeriesSearchOnAdd(seriesID, *searchOnAdd)
	}()
}

// triggerSeriesSearchOnAdd triggers autosearch based on the search-on-add option
func (s *Service) triggerSeriesSearchOnAdd(seriesID int64, searchOnAdd string) {
	ctx := context.Background()
	searchType := preferences.SeriesSearchOnAdd(searchOnAdd)

	series, err := s.tv.GetSeries(ctx, seriesID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Failed to get series for search-on-add")
		return
	}

	s.logger.Info().Int64("seriesId", seriesID).Str("title", series.Title).Str("searchType", searchOnAdd).Msg("Triggering search-on-add for series")

	switch searchType {
	case preferences.SeriesSearchOnAddFirstEpisode:
		s.searchFirstEpisode(ctx, seriesID)
	case preferences.SeriesSearchOnAddFirstSeason:
		s.searchFirstSeason(ctx, seriesID)
	case preferences.SeriesSearchOnAddLatestSeason:
		s.searchLatestSeason(ctx, seriesID, series)
	case preferences.SeriesSearchOnAddAll:
		s.searchAllSeries(ctx, seriesID)
	}
}

func (s *Service) searchFirstEpisode(ctx context.Context, seriesID int64) {
	seasonNum := 1
	episodes, err := s.tv.ListEpisodes(ctx, seriesID, &seasonNum)
	if err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Failed to get season 1 episodes")
		return
	}

	for _, ep := range episodes {
		if ep.EpisodeNumber == 1 && ep.Status != "unreleased" {
			if _, err := s.autosearchSvc.SearchEpisode(ctx, ep.ID, autosearch.SearchSourceAdd); err != nil {
				s.logger.Warn().Err(err).Int64("episodeId", ep.ID).Msg("Search-on-add failed for episode")
			}
			return
		}
	}
}

func (s *Service) searchFirstSeason(ctx context.Context, seriesID int64) {
	if _, err := s.autosearchSvc.SearchSeason(ctx, seriesID, 1, autosearch.SearchSourceAdd); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Search-on-add failed for first season")
	}
}

func (s *Service) searchLatestSeason(ctx context.Context, seriesID int64, series *tv.Series) {
	var latestSeason int
	for i := range series.Seasons {
		season := &series.Seasons[i]
		if season.SeasonNumber > latestSeason && season.SeasonNumber > 0 {
			latestSeason = season.SeasonNumber
		}
	}

	if latestSeason > 0 {
		if _, err := s.autosearchSvc.SearchSeason(ctx, seriesID, latestSeason, autosearch.SearchSourceAdd); err != nil {
			s.logger.Warn().Err(err).Int64("seriesId", seriesID).Int("season", latestSeason).Msg("Search-on-add failed for latest season")
		}
	}
}

func (s *Service) searchAllSeries(ctx context.Context, seriesID int64) {
	if _, err := s.autosearchSvc.SearchSeries(ctx, seriesID, autosearch.SearchSourceAdd); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Search-on-add failed for series")
	}
}
