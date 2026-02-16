package importer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/tv"
)

// RenamePreview represents a preview of a rename operation.
type RenamePreview struct {
	ID              int64  `json:"id"`
	MediaType       string `json:"mediaType"` // "movie" or mediaTypeEpisode
	CurrentPath     string `json:"currentPath"`
	CurrentFilename string `json:"currentFilename"`
	NewPath         string `json:"newPath"`
	NewFilename     string `json:"newFilename"`
	NeedsRename     bool   `json:"needsRename"`
	Error           string `json:"error,omitempty"`

	// Additional context for display
	Title        string `json:"title"`
	SeriesTitle  string `json:"seriesTitle,omitempty"`
	SeasonNumber int    `json:"seasonNumber,omitempty"`
	EpisodeNum   int    `json:"episodeNumber,omitempty"`
	Year         int    `json:"year,omitempty"`
}

// MassRenameResult contains results from a mass rename operation.
type MassRenameResult struct {
	Total     int             `json:"total"`
	Succeeded int             `json:"succeeded"`
	Failed    int             `json:"failed"`
	Skipped   int             `json:"skipped"`
	Results   []RenamePreview `json:"results"`
}

// GetRenamePreviewSeries returns a preview of files that would be renamed for series.
func (s *Service) GetRenamePreviewSeries(ctx context.Context, seriesID *int64) ([]RenamePreview, error) {
	var previews []RenamePreview

	// Get all series or a specific one
	var seriesList []*tv.Series
	if seriesID != nil {
		series, err := s.tv.GetSeries(ctx, *seriesID)
		if err != nil {
			return nil, fmt.Errorf("failed to get series: %w", err)
		}
		seriesList = []*tv.Series{series}
	} else {
		var err error
		seriesList, err = s.tv.ListSeries(ctx, tv.ListSeriesOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list series: %w", err)
		}
	}

	for _, series := range seriesList {
		// Get episodes with files
		episodes, err := s.tv.ListEpisodes(ctx, series.ID, nil)
		if err != nil {
			s.logger.Warn().Err(err).Int64("seriesId", series.ID).Msg("Failed to list episodes")
			continue
		}

		// Get root folder for path computation
		rf, err := s.rootfolder.Get(ctx, series.RootFolderID)
		if err != nil {
			s.logger.Warn().Err(err).Int64("seriesId", series.ID).Msg("Failed to get root folder")
			continue
		}

		for _, ep := range episodes {
			if ep.EpisodeFile == nil {
				continue
			}

			preview := s.computeEpisodeRenamePreview(series, &ep, rf.Path)
			previews = append(previews, preview)
		}
	}

	return previews, nil
}

// GetRenamePreviewMovies returns a preview of files that would be renamed for movies.
func (s *Service) GetRenamePreviewMovies(ctx context.Context, movieID *int64) ([]RenamePreview, error) {
	movieList, err := s.getMovieList(ctx, movieID)
	if err != nil {
		return nil, err
	}

	var previews []RenamePreview
	for _, movie := range movieList {
		previews = append(previews, s.collectMovieRenamePreviews(ctx, movie)...)
	}

	return previews, nil
}

func (s *Service) getMovieList(ctx context.Context, movieID *int64) ([]*movies.Movie, error) {
	if movieID != nil {
		movie, err := s.movies.Get(ctx, *movieID)
		if err != nil {
			return nil, fmt.Errorf("failed to get movie: %w", err)
		}
		return []*movies.Movie{movie}, nil
	}

	movieList, err := s.movies.List(ctx, movies.ListMoviesOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list movies: %w", err)
	}
	return movieList, nil
}

func (s *Service) collectMovieRenamePreviews(ctx context.Context, movie *movies.Movie) []RenamePreview {
	s.ensureMovieFilesLoaded(ctx, movie)
	if len(movie.MovieFiles) == 0 {
		return nil
	}

	rf, err := s.rootfolder.Get(ctx, movie.RootFolderID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("movieId", movie.ID).Msg("Failed to get root folder")
		return nil
	}

	var previews []RenamePreview
	for i := range movie.MovieFiles {
		file := &movie.MovieFiles[i]
		previews = append(previews, s.computeMovieRenamePreview(movie, file, rf.Path))
	}
	return previews
}

func (s *Service) ensureMovieFilesLoaded(ctx context.Context, movie *movies.Movie) {
	if len(movie.MovieFiles) > 0 {
		return
	}
	files, err := s.movies.GetFiles(ctx, movie.ID)
	if err != nil {
		return
	}
	movie.MovieFiles = files
}

// computeEpisodeRenamePreview computes the rename preview for a single episode.
func (s *Service) computeEpisodeRenamePreview(series *tv.Series, ep *tv.Episode, rootPath string) RenamePreview {
	preview := RenamePreview{
		ID:              ep.EpisodeFile.ID,
		MediaType:       mediaTypeEpisode,
		CurrentPath:     ep.EpisodeFile.Path,
		CurrentFilename: filepath.Base(ep.EpisodeFile.Path),
		Title:           ep.Title,
		SeriesTitle:     series.Title,
		SeasonNumber:    ep.SeasonNumber,
		EpisodeNum:      ep.EpisodeNumber,
		Year:            series.Year,
	}

	// Build token context
	tokenCtx := s.buildEpisodeTokenContext(series, ep)

	// Compute new filename
	ext := filepath.Ext(ep.EpisodeFile.Path)
	newFilename, err := s.renamer.ResolveEpisodeFilename(tokenCtx, ext)
	if err != nil {
		preview.Error = err.Error()
		return preview
	}

	// Compute new path
	seriesFolder, err := s.renamer.ResolveSeriesFolderName(tokenCtx)
	if err != nil {
		preview.Error = err.Error()
		return preview
	}
	seasonFolder := s.renamer.ResolveSeasonFolderName(ep.SeasonNumber)

	newPath := filepath.Join(rootPath, seriesFolder, seasonFolder, newFilename)
	preview.NewPath = newPath
	preview.NewFilename = newFilename
	preview.NeedsRename = preview.CurrentPath != newPath

	return preview
}

// computeMovieRenamePreview computes the rename preview for a single movie.
func (s *Service) computeMovieRenamePreview(movie *movies.Movie, file *movies.MovieFile, rootPath string) RenamePreview {
	preview := RenamePreview{
		ID:              file.ID,
		MediaType:       "movie",
		CurrentPath:     file.Path,
		CurrentFilename: filepath.Base(file.Path),
		Title:           movie.Title,
		Year:            movie.Year,
	}

	// Build token context
	tokenCtx := s.buildMovieTokenContext(movie, file)

	// Compute new filename
	ext := filepath.Ext(file.Path)
	newFilename, err := s.renamer.ResolveMovieFilename(tokenCtx, ext)
	if err != nil {
		preview.Error = err.Error()
		return preview
	}

	// Compute new folder
	folderName, err := s.renamer.ResolveMovieFolderName(tokenCtx)
	if err != nil {
		preview.Error = err.Error()
		return preview
	}

	newPath := filepath.Join(rootPath, folderName, newFilename)
	preview.NewPath = newPath
	preview.NewFilename = newFilename
	preview.NeedsRename = preview.CurrentPath != newPath

	return preview
}

// buildEpisodeTokenContext builds a token context from series and episode data.
func (s *Service) buildEpisodeTokenContext(series *tv.Series, ep *tv.Episode) *renamer.TokenContext {
	ctx := &renamer.TokenContext{
		SeriesTitle:   series.Title,
		SeriesYear:    series.Year,
		SeriesType:    series.FormatType,
		SeasonNumber:  ep.SeasonNumber,
		EpisodeNumber: ep.EpisodeNumber,
		EpisodeTitle:  ep.Title,
		OriginalFile:  filepath.Base(ep.EpisodeFile.Path),
	}

	// Default to standard if not set
	if ctx.SeriesType == "" {
		ctx.SeriesType = "standard"
	}

	// Set air date if available
	if ep.AirDate != nil {
		ctx.AirDate = *ep.AirDate
	}

	// Extract quality info from file if available
	if ep.EpisodeFile != nil {
		ctx.Quality = ep.EpisodeFile.Resolution
		ctx.VideoCodec = ep.EpisodeFile.VideoCodec
		ctx.AudioCodec = ep.EpisodeFile.AudioCodec
	}

	return ctx
}

// buildMovieTokenContext builds a token context from movie data.
func (s *Service) buildMovieTokenContext(movie *movies.Movie, file *movies.MovieFile) *renamer.TokenContext {
	ctx := &renamer.TokenContext{
		MovieTitle: movie.Title,
		MovieYear:  movie.Year,
	}

	// Extract quality info from file if available
	if file != nil {
		ctx.OriginalFile = filepath.Base(file.Path)
		ctx.Quality = file.Resolution
		ctx.VideoCodec = file.VideoCodec
		ctx.AudioCodec = file.AudioCodec
	}

	return ctx
}

// ExecuteMassRename performs mass rename operations for the specified items.
func (s *Service) ExecuteMassRename(ctx context.Context, mediaType string, fileIDs []int64) (*MassRenameResult, error) {
	result := &MassRenameResult{
		Total:   len(fileIDs),
		Results: make([]RenamePreview, 0, len(fileIDs)),
	}

	switch mediaType {
	case mediaTypeEpisode, "series":
		return s.executeMassRenameEpisodes(ctx, fileIDs, result)
	case mediaTypeMovie:
		return s.executeMassRenameMovies(ctx, fileIDs, result)
	}

	return nil, fmt.Errorf("invalid media type: %s", mediaType)
}

// executeMassRenameEpisodes renames episode files.
func (s *Service) executeMassRenameEpisodes(ctx context.Context, fileIDs []int64, result *MassRenameResult) (*MassRenameResult, error) {
	for _, fileID := range fileIDs {
		// Get file info - need to find which episode this file belongs to
		file, episode, series, err := s.getEpisodeFileInfo(ctx, fileID)
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, RenamePreview{
				ID:        fileID,
				MediaType: mediaTypeEpisode,
				Error:     err.Error(),
			})
			continue
		}

		// Get root folder
		rf, err := s.rootfolder.Get(ctx, series.RootFolderID)
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, RenamePreview{
				ID:          fileID,
				MediaType:   mediaTypeEpisode,
				CurrentPath: file.Path,
				Error:       fmt.Sprintf("failed to get root folder: %v", err),
			})
			continue
		}

		// Compute new path
		preview := s.computeEpisodeRenamePreview(series, episode, rf.Path)
		preview.ID = fileID

		if !preview.NeedsRename {
			result.Skipped++
			result.Results = append(result.Results, preview)
			continue
		}

		if preview.Error != "" {
			result.Failed++
			result.Results = append(result.Results, preview)
			continue
		}

		// Perform the rename
		if err := s.renameFile(preview.CurrentPath, preview.NewPath); err != nil {
			result.Failed++
			preview.Error = err.Error()
			result.Results = append(result.Results, preview)
			continue
		}

		// Update database with new path
		if err := s.tv.UpdateEpisodeFilePath(ctx, fileID, preview.NewPath); err != nil {
			s.logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to update file path in database")
		}

		result.Succeeded++
		result.Results = append(result.Results, preview)

		// Log to history
		s.logRenameToHistory(ctx, mediaTypeEpisode, episode.ID, &preview)
	}

	return result, nil
}

// executeMassRenameMovies renames movie files.
func (s *Service) executeMassRenameMovies(ctx context.Context, fileIDs []int64, result *MassRenameResult) (*MassRenameResult, error) {
	for _, fileID := range fileIDs {
		// Get file info
		file, movie, err := s.getMovieFileInfo(ctx, fileID)
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, RenamePreview{
				ID:        fileID,
				MediaType: "movie",
				Error:     err.Error(),
			})
			continue
		}

		// Get root folder
		rf, err := s.rootfolder.Get(ctx, movie.RootFolderID)
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, RenamePreview{
				ID:          fileID,
				MediaType:   "movie",
				CurrentPath: file.Path,
				Error:       fmt.Sprintf("failed to get root folder: %v", err),
			})
			continue
		}

		// Compute new path
		preview := s.computeMovieRenamePreview(movie, file, rf.Path)
		preview.ID = fileID

		if !preview.NeedsRename {
			result.Skipped++
			result.Results = append(result.Results, preview)
			continue
		}

		if preview.Error != "" {
			result.Failed++
			result.Results = append(result.Results, preview)
			continue
		}

		// Perform the rename
		if err := s.renameFile(preview.CurrentPath, preview.NewPath); err != nil {
			result.Failed++
			preview.Error = err.Error()
			result.Results = append(result.Results, preview)
			continue
		}

		// Update database with new path
		if err := s.movies.UpdateMovieFilePath(ctx, fileID, preview.NewPath); err != nil {
			s.logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to update file path in database")
		}

		result.Succeeded++
		result.Results = append(result.Results, preview)

		// Log to history
		s.logRenameToHistory(ctx, mediaTypeMovie, movie.ID, &preview)
	}

	return result, nil
}

// getEpisodeFileInfo retrieves file, episode, and series info for a file ID.
func (s *Service) getEpisodeFileInfo(ctx context.Context, fileID int64) (*tv.EpisodeFile, *tv.Episode, *tv.Series, error) {
	// Get the file
	file, err := s.tv.GetEpisodeFileByID(ctx, fileID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get episode file: %w", err)
	}

	// Get the episode
	episode, err := s.tv.GetEpisode(ctx, file.EpisodeID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get episode: %w", err)
	}

	// Make sure EpisodeFile is set
	episode.EpisodeFile = file

	// Get the series
	series, err := s.tv.GetSeries(ctx, episode.SeriesID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get series: %w", err)
	}

	return file, episode, series, nil
}

// getMovieFileInfo retrieves file and movie info for a file ID.
func (s *Service) getMovieFileInfo(ctx context.Context, fileID int64) (*movies.MovieFile, *movies.Movie, error) {
	// Get the file
	file, err := s.movies.GetFileByID(ctx, fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get movie file: %w", err)
	}

	// Get the movie
	movie, err := s.movies.Get(ctx, file.MovieID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get movie: %w", err)
	}

	return file, movie, nil
}

// renameFile moves a file from old path to new path.
func (s *Service) renameFile(oldPath, newPath string) error {
	// Check if source exists
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", oldPath)
	}

	// Create destination directory
	destDir := filepath.Dir(newPath)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Rename/move the file
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	s.logger.Info().
		Str("from", oldPath).
		Str("to", newPath).
		Msg("Renamed file")

	return nil
}

// logRenameToHistory logs a rename operation to history.
func (s *Service) logRenameToHistory(ctx context.Context, mediaType string, mediaID int64, preview *RenamePreview) {
	if s.history == nil {
		return
	}

	err := s.history.Create(ctx, &HistoryInput{
		EventType: "file_renamed",
		MediaType: mediaType,
		MediaID:   mediaID,
		Data: map[string]any{
			"source_path":      preview.CurrentPath,
			"destination_path": preview.NewPath,
			"old_filename":     preview.CurrentFilename,
			"new_filename":     preview.NewFilename,
		},
	})
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to log rename to history")
	}
}
