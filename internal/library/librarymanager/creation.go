package librarymanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
)

func (s *Service) createMovieFromParsed(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsed *scanner.ParsedMedia,
	qualityProfileID int64,
	meta *metadata.MovieResult,
) (*movies.Movie, bool, error) {
	input := movies.CreateMovieInput{
		Title:            parsed.Title,
		Year:             parsed.Year,
		RootFolderID:     folder.ID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
	}

	input.Path = movies.GenerateMoviePath(folder.Path, parsed.Title, parsed.Year)

	if meta != nil {
		input.Title = meta.Title
		input.Year = meta.Year
		input.TmdbID = meta.ID
		input.ImdbID = meta.ImdbID
		input.Overview = meta.Overview
		input.Runtime = meta.Runtime
		input.Path = movies.GenerateMoviePath(folder.Path, meta.Title, meta.Year)

		// Fetch release dates from TMDB
		if meta.ID > 0 {
			digital, physical, theatrical, err := s.metadata.GetMovieReleaseDates(ctx, meta.ID)
			if err != nil {
				s.logger.Warn().Err(err).Int("tmdbId", meta.ID).Msg("Failed to fetch release dates during scan")
			} else {
				input.ReleaseDate = digital
				input.PhysicalReleaseDate = physical
				input.TheatricalReleaseDate = theatrical
			}
		}
	}

	movie, err := s.movies.Create(ctx, &input)
	if err != nil {
		if errors.Is(err, movies.ErrDuplicateTmdbID) && meta != nil {
			existing, err := s.movies.GetByTmdbID(ctx, meta.ID)
			if err == nil {
				return existing, false, nil
			}
		}
		return nil, false, err
	}

	return movie, true, nil
}

// addMovieFile adds a file to a movie and assigns it to a slot if multi-version is enabled.
// Req 13.1.2: Auto-assign files to best matching slot based on quality profile matching
// Req 13.1.3: Extra files (more than slot count) queued for user review (slot_id = NULL)
// Callers should check if the file already exists before calling this.
func (s *Service) addMovieFile(ctx context.Context, movieID int64, parsed *scanner.ParsedMedia) error {
	input := movies.CreateMovieFileInput{
		Path:       parsed.FilePath,
		Size:       parsed.FileSize,
		Quality:    parsed.Quality,
		VideoCodec: parsed.Codec,
		Resolution: parsed.Quality,
	}

	file, err := s.movies.AddFile(ctx, movieID, &input)
	if err != nil {
		return err
	}

	// Try to assign to a slot if slots service is available and multi-version is enabled
	if s.slotsSvc != nil && s.slotsSvc.IsMultiVersionEnabled(ctx) {
		assignment, err := s.slotsSvc.DetermineTargetSlot(ctx, parsed, "movie", movieID)
		if err != nil {
			// No matching slot or all slots filled - file will be in review queue (slot_id = NULL)
			s.logger.Debug().
				Err(err).
				Int64("movieId", movieID).
				Int64("fileId", file.ID).
				Str("path", parsed.FilePath).
				Msg("Could not assign file to slot, will be in review queue")
			return nil
		}

		// Assign file to the determined slot
		if err := s.slotsSvc.AssignFileToSlot(ctx, "movie", movieID, assignment.SlotID, file.ID); err != nil {
			s.logger.Warn().
				Err(err).
				Int64("movieId", movieID).
				Int64("fileId", file.ID).
				Int64("slotId", assignment.SlotID).
				Msg("Failed to assign file to slot")
		} else {
			s.logger.Debug().
				Int64("movieId", movieID).
				Int64("fileId", file.ID).
				Int64("slotId", assignment.SlotID).
				Str("slotName", assignment.SlotName).
				Bool("isUpgrade", assignment.IsUpgrade).
				Bool("isNewFill", assignment.IsNewFill).
				Msg("Assigned movie file to slot")
		}
	}

	return nil
}

// createSeriesFromParsed creates a new series from parsed media and optional metadata.
// Also fetches seasons and episodes from metadata providers to ensure complete data.
func (s *Service) createSeriesFromParsed(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsed *scanner.ParsedMedia,
	qualityProfileID int64,
	meta *metadata.SeriesResult,
) (*tv.Series, bool, error) {
	input := tv.CreateSeriesInput{
		Title:            parsed.Title,
		RootFolderID:     folder.ID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
		SeasonFolder:     true,
	}

	input.Path = tv.GenerateSeriesPath(folder.Path, parsed.Title)

	var tmdbID, tvdbID int
	if meta != nil {
		input.Title = meta.Title
		input.Year = meta.Year
		input.TvdbID = meta.TvdbID
		input.TmdbID = meta.TmdbID
		input.ImdbID = meta.ImdbID
		input.Overview = meta.Overview
		input.Runtime = meta.Runtime
		input.ProductionStatus = meta.Status
		input.Path = tv.GenerateSeriesPath(folder.Path, meta.Title)
		tmdbID = meta.TmdbID
		tvdbID = meta.TvdbID
	}

	series, err := s.tv.CreateSeries(ctx, &input)
	if err != nil {
		if errors.Is(err, tv.ErrDuplicateTvdbID) && meta != nil && meta.TvdbID > 0 {
			existing, err := s.tv.GetSeriesByTvdbID(ctx, meta.TvdbID)
			if err == nil {
				return existing, false, nil
			}
		}
		return nil, false, err
	}

	series = s.fetchAndLinkSeasonMetadata(ctx, series, tmdbID, tvdbID)
	return series, true, nil
}

func (s *Service) fetchAndLinkSeasonMetadata(ctx context.Context, series *tv.Series, tmdbID, tvdbID int) *tv.Series {
	if tmdbID == 0 && tvdbID == 0 {
		return series
	}

	seasonResults, err := s.metadata.GetSeriesSeasons(ctx, tmdbID, tvdbID)
	if err != nil {
		s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Int("tvdbId", tvdbID).Msg("Failed to fetch season metadata during scan")
		return series
	}

	seasonMeta := s.convertSeasonResults(seasonResults)

	if err := s.tv.UpdateSeasonsFromMetadata(ctx, series.ID, seasonMeta); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", series.ID).Msg("Failed to update seasons from metadata during scan")
		return series
	}

	totalEpisodes := s.countTotalEpisodes(seasonMeta)
	s.logger.Info().
		Int64("seriesId", series.ID).
		Int("seasons", len(seasonMeta)).
		Int("episodes", totalEpisodes).
		Msg("Updated seasons and episodes from metadata during scan")

	if updatedSeries, err := s.tv.GetSeries(ctx, series.ID); err == nil {
		return updatedSeries
	}
	return series
}

// addEpisodeFile adds a file to an episode, creating the season/episode if needed.
// Req 13.1.2: Auto-assign files to best matching slot based on quality profile matching
// Req 13.1.3: Extra files (more than slot count) queued for user review (slot_id = NULL)
// Callers should check if the file already exists before calling this.
func (s *Service) addEpisodeFile(ctx context.Context, seriesID int64, parsed *scanner.ParsedMedia) error {
	episode, err := s.getOrCreateEpisode(ctx, seriesID, parsed.Season, parsed.Episode)
	if err != nil {
		return err
	}

	input := tv.CreateEpisodeFileInput{
		Path:       parsed.FilePath,
		Size:       parsed.FileSize,
		Quality:    parsed.Quality,
		VideoCodec: parsed.Codec,
		Resolution: parsed.Quality,
	}

	file, err := s.tv.AddEpisodeFile(ctx, episode.ID, &input)
	if err != nil {
		return err
	}

	// Try to assign to a slot if slots service is available and multi-version is enabled
	if s.slotsSvc != nil && s.slotsSvc.IsMultiVersionEnabled(ctx) {
		assignment, err := s.slotsSvc.DetermineTargetSlot(ctx, parsed, "episode", episode.ID)
		if err != nil {
			// No matching slot or all slots filled - file will be in review queue (slot_id = NULL)
			s.logger.Debug().
				Err(err).
				Int64("episodeId", episode.ID).
				Int64("fileId", file.ID).
				Str("path", parsed.FilePath).
				Msg("Could not assign file to slot, will be in review queue")
			return nil
		}

		// Assign file to the determined slot
		if err := s.slotsSvc.AssignFileToSlot(ctx, "episode", episode.ID, assignment.SlotID, file.ID); err != nil {
			s.logger.Warn().
				Err(err).
				Int64("episodeId", episode.ID).
				Int64("fileId", file.ID).
				Int64("slotId", assignment.SlotID).
				Msg("Failed to assign file to slot")
		} else {
			s.logger.Debug().
				Int64("episodeId", episode.ID).
				Int64("fileId", file.ID).
				Int64("slotId", assignment.SlotID).
				Str("slotName", assignment.SlotName).
				Bool("isUpgrade", assignment.IsUpgrade).
				Bool("isNewFill", assignment.IsNewFill).
				Msg("Assigned episode file to slot")
		}
	}

	return nil
}

// getOrCreateEpisode gets an existing episode or creates one.
func (s *Service) getOrCreateEpisode(ctx context.Context, seriesID int64, seasonNum, episodeNum int) (*tv.Episode, error) {
	episodes, err := s.tv.ListEpisodes(ctx, seriesID, &seasonNum)
	if err == nil {
		for i := range episodes {
			if episodes[i].EpisodeNumber == episodeNum {
				return &episodes[i], nil
			}
		}
	}

	// Ensure season exists
	if err := s.ensureSeasonExists(ctx, seriesID, seasonNum); err != nil {
		return nil, err
	}

	// Create episode
	_, err = s.queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      seriesID,
		SeasonNumber:  int64(seasonNum),
		EpisodeNumber: int64(episodeNum),
		Title:         sql.NullString{String: fmt.Sprintf("Episode %d", episodeNum), Valid: true},
		Monitored:     1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create episode: %w", err)
	}

	// Fetch the created episode
	episodes, err = s.tv.ListEpisodes(ctx, seriesID, &seasonNum)
	if err != nil {
		return nil, err
	}
	for i := range episodes {
		if episodes[i].EpisodeNumber == episodeNum {
			return &episodes[i], nil
		}
	}

	return nil, fmt.Errorf("failed to find created episode")
}

// ensureSeasonExists ensures a season exists for a series.
func (s *Service) ensureSeasonExists(ctx context.Context, seriesID int64, seasonNum int) error {
	_, err := s.queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNum),
	})
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	_, err = s.queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNum),
		Monitored:    1,
	})
	return err
}
