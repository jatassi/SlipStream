package arrimport

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/progress"
)

// Executor handles the actual import of media from a source into SlipStream.
type Executor struct {
	db              *sql.DB
	reader          Reader
	sourceType      SourceType
	movieService    MovieService
	tvService       TVService
	slotsService    SlotsService
	progressManager *progress.Manager
	logger          *zerolog.Logger
}

// NewExecutor creates a new import executor.
func NewExecutor(
	db *sql.DB,
	reader Reader,
	sourceType SourceType,
	movieService MovieService,
	tvService TVService,
	slotsService SlotsService,
	progressManager *progress.Manager,
	logger *zerolog.Logger,
) *Executor {
	return &Executor{
		db:              db,
		reader:          reader,
		sourceType:      sourceType,
		movieService:    movieService,
		tvService:       tvService,
		slotsService:    slotsService,
		progressManager: progressManager,
		logger:          logger,
	}
}

// Run executes the full import process.
func (e *Executor) Run(ctx context.Context, mappings ImportMappings) {
	report := &ImportReport{}
	_ = e.progressManager.StartActivity("arrimport", progress.ActivityTypeImport, "Library Import")

	defer func() {
		if len(report.Errors) > 0 {
			e.progressManager.FailActivity("arrimport", fmt.Sprintf("Completed with %d errors", len(report.Errors)))
		} else {
			e.progressManager.CompleteActivity("arrimport", fmt.Sprintf("Import complete: %d movies, %d series imported", report.MoviesCreated, report.SeriesCreated))
		}
	}()

	if e.sourceType == SourceTypeRadarr {
		e.importMovies(ctx, mappings, report)
	}

	if e.sourceType == SourceTypeSonarr {
		e.importAllSeries(ctx, mappings, report)
	}

	e.logger.Info().
		Int("moviesCreated", report.MoviesCreated).
		Int("moviesSkipped", report.MoviesSkipped).
		Int("seriesCreated", report.SeriesCreated).
		Int("seriesSkipped", report.SeriesSkipped).
		Int("filesImported", report.FilesImported).
		Int("errors", len(report.Errors)).
		Msg("import finished")
}

func (e *Executor) importMovies(ctx context.Context, mappings ImportMappings, report *ImportReport) {
	sourceMovies, err := e.reader.ReadMovies(ctx)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("failed to read movies: %v", err))
		return
	}

	total := len(sourceMovies)
	for i := range sourceMovies {
		pct := 0
		if total > 0 {
			pct = (i * 100) / total
		}
		e.progressManager.UpdateActivity("arrimport", fmt.Sprintf("Importing: %s", sourceMovies[i].Title), pct)
		e.importMovie(ctx, &sourceMovies[i], mappings, report)
	}
}

func (e *Executor) importMovie(ctx context.Context, movie *SourceMovie, mappings ImportMappings, report *ImportReport) {
	if movie.TmdbID == 0 {
		report.MoviesSkipped++
		return
	}

	if e.movieExists(ctx, movie, report) {
		return
	}

	rootFolderID, qualityProfileID, ok := e.resolveMappings(
		mappings, movie.RootFolderPath, movie.QualityProfileID, movie.Title, "movie", report,
	)
	if !ok {
		report.MoviesErrored++
		return
	}

	input := e.buildMovieInput(movie, rootFolderID, qualityProfileID)

	createdMovie, err := e.movieService.Create(ctx, input)
	if err != nil {
		report.MoviesErrored++
		report.Errors = append(report.Errors, fmt.Sprintf("failed to create movie %q: %v", movie.Title, err))
		return
	}

	e.preserveMovieAddedAt(ctx, movie.Added, createdMovie.ID)
	e.initMovieSlots(ctx, createdMovie.ID)

	if movie.HasFile && movie.File != nil {
		e.importMovieFile(ctx, createdMovie, movie.File, report)
	}

	report.MoviesCreated++
}

func (e *Executor) movieExists(ctx context.Context, movie *SourceMovie, report *ImportReport) bool {
	_, err := e.movieService.GetByTmdbID(ctx, movie.TmdbID)
	if err == nil {
		report.MoviesSkipped++
		return true
	}
	if !errors.Is(err, movies.ErrMovieNotFound) {
		e.logger.Warn().Int("tmdbId", movie.TmdbID).Err(err).Msg("error checking movie existence")
		report.MoviesErrored++
		report.Errors = append(report.Errors, fmt.Sprintf("failed to check movie %q (TMDB %d): %v", movie.Title, movie.TmdbID, err))
		return true
	}
	return false
}

func (e *Executor) resolveMappings(
	mappings ImportMappings, rootFolderPath string, sourceProfileID int64, title, mediaType string, report *ImportReport,
) (rootFolderID, qualityProfileID int64, ok bool) {
	rootFolderID, ok = mappings.RootFolderMapping[rootFolderPath]
	if !ok {
		report.Errors = append(report.Errors, fmt.Sprintf("no root folder mapping for %q (%s: %s)", rootFolderPath, mediaType, title))
		return 0, 0, false
	}

	qualityProfileID, ok = mappings.QualityProfileMapping[sourceProfileID]
	if !ok {
		report.Errors = append(report.Errors, fmt.Sprintf("no quality profile mapping for source profile %d (%s: %s)", sourceProfileID, mediaType, title))
		return 0, 0, false
	}

	return rootFolderID, qualityProfileID, true
}

func (e *Executor) buildMovieInput(movie *SourceMovie, rootFolderID, qualityProfileID int64) *movies.CreateMovieInput {
	return &movies.CreateMovieInput{
		Title:                 movie.Title,
		Year:                  movie.Year,
		TmdbID:                movie.TmdbID,
		ImdbID:                movie.ImdbID,
		Overview:              movie.Overview,
		Runtime:               movie.Runtime,
		Path:                  movie.Path,
		RootFolderID:          rootFolderID,
		QualityProfileID:      qualityProfileID,
		Monitored:             movie.Monitored,
		Studio:                movie.Studio,
		ContentRating:         movie.Certification,
		ReleaseDate:           formatDate(movie.DigitalRelease),
		PhysicalReleaseDate:   formatDate(movie.PhysicalRelease),
		TheatricalReleaseDate: formatDate(movie.InCinemas),
	}
}

func (e *Executor) preserveMovieAddedAt(ctx context.Context, added time.Time, movieID int64) {
	if !added.IsZero() {
		_, _ = e.db.ExecContext(ctx, "UPDATE movies SET added_at = ? WHERE id = ?", added, movieID)
	}
}

func (e *Executor) initMovieSlots(ctx context.Context, movieID int64) {
	if e.slotsService != nil && e.slotsService.IsMultiVersionEnabled(ctx) {
		_ = e.slotsService.InitializeSlotAssignments(ctx, "movie", movieID)
	}
}

func (e *Executor) importMovieFile(ctx context.Context, createdMovie *movies.Movie, file *SourceMovieFile, report *ImportReport) {
	qualityID := MapQualityID(e.sourceType, file.QualityID, file.QualityName)

	var originalFilename string
	if file.OriginalFilePath != "" {
		originalFilename = path.Base(file.OriginalFilePath)
	}

	fileInput := &movies.CreateMovieFileInput{
		Path:             file.Path,
		Size:             file.Size,
		QualityID:        qualityID,
		VideoCodec:       file.VideoCodec,
		AudioCodec:       file.AudioCodec,
		AudioChannels:    file.AudioChannels,
		DynamicRange:     file.DynamicRange,
		Resolution:       file.Resolution,
		OriginalPath:     file.OriginalFilePath,
		OriginalFilename: originalFilename,
	}

	createdFile, err := e.movieService.AddFile(ctx, createdMovie.ID, fileInput)
	if err != nil {
		e.logger.Warn().Int64("movieId", createdMovie.ID).Str("path", file.Path).Err(err).Msg("failed to add movie file")
		return
	}
	report.FilesImported++

	e.assignSlot(ctx, file.Path, "movie", createdMovie.ID, createdFile.ID)
}

func (e *Executor) assignSlot(ctx context.Context, filePath, mediaType string, mediaID, fileID int64) {
	if e.slotsService == nil || !e.slotsService.IsMultiVersionEnabled(ctx) {
		return
	}
	parsed := scanner.ParsePath(filePath)
	slot, err := e.slotsService.DetermineTargetSlot(ctx, parsed, mediaType, mediaID)
	if err == nil && slot != nil {
		_ = e.slotsService.AssignFileToSlot(ctx, mediaType, mediaID, slot.SlotID, fileID)
	}
}

func (e *Executor) importAllSeries(ctx context.Context, mappings ImportMappings, report *ImportReport) {
	seriesList, err := e.reader.ReadSeries(ctx)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("failed to read series: %v", err))
		return
	}

	total := len(seriesList)
	for i := range seriesList {
		pct := 0
		if total > 0 {
			pct = (i * 100) / total
		}
		e.progressManager.UpdateActivity("arrimport", fmt.Sprintf("Importing: %s", seriesList[i].Title), pct)
		e.importSeries(ctx, &seriesList[i], mappings, report)
	}
}

func (e *Executor) importSeries(ctx context.Context, series *SourceSeries, mappings ImportMappings, report *ImportReport) {
	if series.TvdbID == 0 {
		report.SeriesSkipped++
		return
	}

	if e.seriesExists(ctx, series, report) {
		return
	}

	rootFolderID, qualityProfileID, ok := e.resolveMappings(
		mappings, series.RootFolderPath, series.QualityProfileID, series.Title, "series", report,
	)
	if !ok {
		report.SeriesErrored++
		return
	}

	episodes, files := e.readSeriesData(ctx, series.ID)
	fileIDToEpisode := buildFileEpisodeLookup(episodes)
	seasonInputs := buildSeasonInputs(series.Seasons, episodes)

	input := e.buildSeriesInput(series, rootFolderID, qualityProfileID, seasonInputs)

	createdSeries, err := e.tvService.CreateSeries(ctx, input)
	if err != nil {
		report.SeriesErrored++
		report.Errors = append(report.Errors, fmt.Sprintf("failed to create series %q: %v", series.Title, err))
		return
	}

	e.preserveSeriesAddedAt(ctx, series.Added, createdSeries.ID)
	e.importEpisodeFiles(ctx, createdSeries.ID, series.Path, files, fileIDToEpisode, report)

	report.SeriesCreated++
}

func (e *Executor) seriesExists(ctx context.Context, series *SourceSeries, report *ImportReport) bool {
	_, err := e.tvService.GetSeriesByTvdbID(ctx, series.TvdbID)
	if err == nil {
		report.SeriesSkipped++
		return true
	}
	if !errors.Is(err, tv.ErrSeriesNotFound) {
		e.logger.Warn().Int("tvdbId", series.TvdbID).Err(err).Msg("error checking series existence")
		report.SeriesErrored++
		report.Errors = append(report.Errors, fmt.Sprintf("failed to check series %q (TVDB %d): %v", series.Title, series.TvdbID, err))
		return true
	}
	return false
}

func (e *Executor) readSeriesData(ctx context.Context, seriesID int64) ([]SourceEpisode, []SourceEpisodeFile) {
	episodes, err := e.reader.ReadEpisodes(ctx, seriesID)
	if err != nil {
		e.logger.Warn().Int64("seriesId", seriesID).Err(err).Msg("failed to read episodes")
		episodes = []SourceEpisode{}
	}

	files, err := e.reader.ReadEpisodeFiles(ctx, seriesID)
	if err != nil {
		e.logger.Warn().Int64("seriesId", seriesID).Err(err).Msg("failed to read episode files")
		files = []SourceEpisodeFile{}
	}

	return episodes, files
}

func buildFileEpisodeLookup(episodes []SourceEpisode) map[int64]SourceEpisode {
	lookup := make(map[int64]SourceEpisode)
	for i := range episodes {
		ep := &episodes[i]
		if ep.HasFile && ep.EpisodeFileID > 0 {
			lookup[ep.EpisodeFileID] = *ep
		}
	}
	return lookup
}

func buildSeasonInputs(seasons []SourceSeason, episodes []SourceEpisode) []tv.SeasonInput {
	episodesBySeason := make(map[int][]SourceEpisode)
	for i := range episodes {
		ep := &episodes[i]
		episodesBySeason[ep.SeasonNumber] = append(episodesBySeason[ep.SeasonNumber], *ep)
	}

	seasonInputs := make([]tv.SeasonInput, 0, len(seasons))
	for i := range seasons {
		season := &seasons[i]
		si := tv.SeasonInput{
			SeasonNumber: season.SeasonNumber,
			Monitored:    season.Monitored,
		}

		seasonEps := episodesBySeason[season.SeasonNumber]
		epInputs := make([]tv.EpisodeInput, 0, len(seasonEps))
		for j := range seasonEps {
			ep := &seasonEps[j]
			epInputs = append(epInputs, tv.EpisodeInput{
				EpisodeNumber: ep.EpisodeNumber,
				Title:         ep.Title,
				Overview:      ep.Overview,
				AirDate:       parseAirDate(ep.AirDateUtc),
				Monitored:     ep.Monitored,
			})
		}
		si.Episodes = epInputs
		seasonInputs = append(seasonInputs, si)
	}

	return seasonInputs
}

func (e *Executor) buildSeriesInput(series *SourceSeries, rootFolderID, qualityProfileID int64, seasonInputs []tv.SeasonInput) *tv.CreateSeriesInput {
	return &tv.CreateSeriesInput{
		Title:            series.Title,
		Year:             series.Year,
		TvdbID:           series.TvdbID,
		TmdbID:           series.TmdbID,
		ImdbID:           series.ImdbID,
		Overview:         series.Overview,
		Runtime:          series.Runtime,
		Path:             series.Path,
		RootFolderID:     rootFolderID,
		QualityProfileID: qualityProfileID,
		Monitored:        series.Monitored,
		SeasonFolder:     series.SeasonFolder,
		Network:          series.Network,
		FormatType:       series.SeriesType,
		ProductionStatus: series.Status,
		Seasons:          seasonInputs,
	}
}

func (e *Executor) preserveSeriesAddedAt(ctx context.Context, added time.Time, seriesID int64) {
	if !added.IsZero() {
		_, _ = e.db.ExecContext(ctx, "UPDATE series SET added_at = ? WHERE id = ?", added, seriesID)
	}
}

func (e *Executor) importEpisodeFiles(
	ctx context.Context, seriesID int64, seriesPath string,
	files []SourceEpisodeFile, fileIDToEpisode map[int64]SourceEpisode, report *ImportReport,
) {
	for i := range files {
		file := &files[i]
		sourceEp, found := fileIDToEpisode[file.ID]
		if !found {
			continue
		}

		slipEpisode, err := e.tvService.GetEpisodeByNumber(ctx, seriesID, sourceEp.SeasonNumber, sourceEp.EpisodeNumber)
		if err != nil {
			e.logger.Warn().Int64("seriesId", seriesID).Int("season", sourceEp.SeasonNumber).Int("episode", sourceEp.EpisodeNumber).Err(err).Msg("episode not found for file")
			continue
		}

		e.importSingleEpisodeFile(ctx, slipEpisode, file, seriesPath, report)
	}
}

func (e *Executor) importSingleEpisodeFile(
	ctx context.Context, episode *tv.Episode, file *SourceEpisodeFile, seriesPath string, report *ImportReport,
) {
	filePath := resolveEpisodeFilePath(seriesPath, file.RelativePath)
	qualityID := MapQualityID(e.sourceType, file.QualityID, file.QualityName)

	var originalFilename string
	if file.OriginalFilePath != "" {
		originalFilename = path.Base(file.OriginalFilePath)
	}

	fileInput := &tv.CreateEpisodeFileInput{
		Path:             filePath,
		Size:             file.Size,
		QualityID:        qualityID,
		VideoCodec:       file.VideoCodec,
		AudioCodec:       file.AudioCodec,
		AudioChannels:    file.AudioChannels,
		DynamicRange:     file.DynamicRange,
		Resolution:       file.Resolution,
		OriginalPath:     file.OriginalFilePath,
		OriginalFilename: originalFilename,
	}

	createdFile, err := e.tvService.AddEpisodeFile(ctx, episode.ID, fileInput)
	if err != nil {
		e.logger.Warn().Int64("episodeId", episode.ID).Str("path", filePath).Err(err).Msg("failed to add episode file")
		return
	}
	report.FilesImported++

	if e.slotsService != nil && e.slotsService.IsMultiVersionEnabled(ctx) {
		_ = e.slotsService.InitializeSlotAssignments(ctx, "episode", episode.ID)
	}
	e.assignSlot(ctx, filePath, "episode", episode.ID, createdFile.ID)
}

// resolveEpisodeFilePath constructs the full file path for an episode file.
// For SQLite sources, RelativePath is relative to the series folder.
// For API sources, RelativePath may already be an absolute path.
func resolveEpisodeFilePath(seriesPath, relativePath string) string {
	if strings.HasPrefix(relativePath, "/") {
		return relativePath
	}
	return seriesPath + "/" + relativePath
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func parseAirDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
