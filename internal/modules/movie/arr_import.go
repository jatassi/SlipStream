package movie

import (
	"context"
	"errors"
	"os"
	"path"
	"slices"
	"time"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/pathutil"
)

// radarrQualityMap maps Radarr quality IDs to SlipStream quality IDs.
// Mirrors arrimport.radarrQualityMap — duplicated here to avoid an import cycle
// (movie -> arrimport -> notification -> movie).
var radarrQualityMap = map[int]int64{
	1:  1,  // SDTV -> SDTV
	2:  2,  // DVD -> DVD
	8:  3,  // WEBDL-480p -> WEBRip-480p (nearest)
	12: 3,  // WEBRip-480p -> WEBRip-480p
	4:  4,  // HDTV-720p -> HDTV-720p
	5:  6,  // WEBDL-720p -> WEBDL-720p
	14: 5,  // WEBRip-720p -> WEBRip-720p
	6:  7,  // Bluray-720p -> Bluray-720p
	9:  8,  // HDTV-1080p -> HDTV-1080p
	3:  10, // WEBDL-1080p -> WEBDL-1080p
	15: 9,  // WEBRip-1080p -> WEBRip-1080p
	7:  11, // Bluray-1080p -> Bluray-1080p
	30: 12, // Remux-1080p -> Remux-1080p
	16: 13, // HDTV-2160p -> HDTV-2160p
	18: 15, // WEBDL-2160p -> WEBDL-2160p
	17: 14, // WEBRip-2160p -> WEBRip-2160p
	19: 16, // Bluray-2160p -> Bluray-2160p
	31: 17, // Remux-2160p -> Remux-2160p
	20: 2,  // Bluray-480p -> DVD (nearest)
	21: 2,  // Bluray-576p -> DVD (nearest)
	23: 2,  // DVD-R -> DVD
}

func (m *Module) ExternalAppName() string { return "Radarr" }

func (m *Module) PreviewMovies(ctx context.Context, reader module.ArrReader) ([]module.ArrImportPreviewItem, error) {
	sourceMovies, err := reader.ReadMovies(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]module.ArrImportPreviewItem, 0, len(sourceMovies))
	for i := range sourceMovies {
		sm := &sourceMovies[i]
		item := module.ArrImportPreviewItem{
			Title:            sm.Title,
			Year:             sm.Year,
			TmdbID:           sm.TmdbID,
			HasFile:          sm.HasFile,
			Monitored:        sm.Monitored,
			QualityProfileID: sm.QualityProfileID,
			PosterURL:        sm.PosterURL,
		}

		if sm.HasFile {
			item.FileCount = 1
		}

		if sm.File != nil {
			item.Quality = sm.File.QualityName
		}

		if sm.TmdbID == 0 {
			item.Status = "skip"
			item.SkipReason = "no TMDB ID"
			items = append(items, item)
			continue
		}

		_, err := m.movieService.GetByTmdbID(ctx, sm.TmdbID)
		if err != nil {
			errMsg := err.Error()
			if errMsg == "movie not found" {
				item.Status = "new"
			} else {
				m.logger.Warn().Int("tmdbId", sm.TmdbID).Err(err).Msg("failed to check movie existence")
				item.Status = "skip"
				item.SkipReason = "error checking existence"
			}
		} else {
			item.Status = "duplicate"
		}

		items = append(items, item)
	}

	return items, nil
}

func (m *Module) ImportMovie(ctx context.Context, movie module.ArrSourceMovie, mappings module.ArrImportMappings) (*module.ImportedEntity, error) { //nolint:gocritic // interface-mandated signature
	if m.shouldSkipMovie(ctx, &movie, &mappings) {
		return nil, nil //nolint:nilnil // nil entity + nil error = skipped per convention
	}

	rootFolderID, ok := mappings.RootFolderMapping[movie.RootFolderPath]
	if !ok {
		return nil, nil //nolint:nilnil // nil entity + nil error = skipped per convention
	}

	qualityProfileID, ok := mappings.QualityProfileMapping[movie.QualityProfileID]
	if !ok {
		return nil, nil //nolint:nilnil // nil entity + nil error = skipped per convention
	}

	input := m.buildArrMovieInput(&movie, rootFolderID, qualityProfileID)

	createdMovie, err := m.movieService.Create(ctx, input)
	if err != nil {
		return nil, err
	}

	m.preserveMovieAddedAt(ctx, movie.Added, createdMovie.ID)
	m.initMovieSlots(ctx, createdMovie.ID)
	m.refreshMovieMetadata(ctx, createdMovie.ID, movie.Title)

	entity := &module.ImportedEntity{
		EntityType: module.EntityMovie,
		EntityID:   createdMovie.ID,
		Title:      movie.Title,
	}

	if movie.HasFile && movie.File != nil {
		if err := m.importArrMovieFile(ctx, createdMovie, movie.File, entity); err != nil {
			entity.Errors = append(entity.Errors, err.Error())
		}
	}

	return entity, nil
}

func (m *Module) shouldSkipMovie(ctx context.Context, movie *module.ArrSourceMovie, mappings *module.ArrImportMappings) bool {
	if movie.TmdbID == 0 {
		return true
	}
	if len(mappings.SelectedIDs) > 0 && !slices.Contains(mappings.SelectedIDs, movie.TmdbID) {
		return true
	}
	return m.movieExists(ctx, movie.TmdbID)
}

func (m *Module) movieExists(ctx context.Context, tmdbID int) bool {
	_, err := m.movieService.GetByTmdbID(ctx, tmdbID)
	if err == nil {
		return true
	}
	if !errors.Is(err, movies.ErrMovieNotFound) {
		m.logger.Warn().Int("tmdbId", tmdbID).Err(err).Msg("error checking movie existence")
		return true
	}
	return false
}

func (m *Module) buildArrMovieInput(movie *module.ArrSourceMovie, rootFolderID, qualityProfileID int64) *movies.CreateMovieInput {
	return &movies.CreateMovieInput{
		Title:                 movie.Title,
		Year:                  movie.Year,
		TmdbID:                movie.TmdbID,
		ImdbID:                movie.ImdbID,
		Overview:              movie.Overview,
		Runtime:               movie.Runtime,
		Path:                  pathutil.NormalizePath(movie.Path),
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

func (m *Module) preserveMovieAddedAt(ctx context.Context, added time.Time, movieID int64) {
	if !added.IsZero() {
		_, _ = m.db.ExecContext(ctx, "UPDATE movies SET added_at = ? WHERE id = ?", added, movieID)
	}
}

func (m *Module) initMovieSlots(ctx context.Context, movieID int64) {
	if m.slotsService != nil && m.slotsService.IsMultiVersionEnabled(ctx) {
		_ = m.slotsService.InitializeSlotAssignments(ctx, "movie", movieID)
	}
}

func (m *Module) refreshMovieMetadata(ctx context.Context, movieID int64, title string) {
	if _, err := m.metadataProvider.RefreshMetadata(ctx, movieID); err != nil {
		m.logger.Warn().Err(err).Int64("movieId", movieID).Str("title", title).Msg("failed to refresh movie metadata")
	}
}

func (m *Module) importArrMovieFile(ctx context.Context, createdMovie *movies.Movie, file *module.ArrSourceMovieFile, entity *module.ImportedEntity) error {
	if _, err := os.Stat(file.Path); err != nil {
		return err
	}

	qualityID := mapRadarrQualityID(file.QualityID, file.QualityName)

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

	createdFile, err := m.movieService.AddFile(ctx, createdMovie.ID, fileInput)
	if err != nil {
		return err
	}
	entity.FilesImported++

	m.assignSlot(ctx, file.Path, "movie", createdMovie.ID, createdFile.ID)
	return nil
}

func (m *Module) assignSlot(ctx context.Context, filePath, mediaType string, mediaID, fileID int64) {
	if m.slotsService == nil || !m.slotsService.IsMultiVersionEnabled(ctx) {
		return
	}
	parsed := scanner.ParsePath(filePath)
	slot, err := m.slotsService.DetermineTargetSlot(ctx, parsed, mediaType, mediaID)
	if err == nil && slot != nil {
		_ = m.slotsService.AssignFileToSlot(ctx, mediaType, mediaID, slot.SlotID, fileID)
	}
}

// mapRadarrQualityID maps a Radarr quality ID to a SlipStream quality ID.
// Falls back to matching by quality name if the static map has no entry.
func mapRadarrQualityID(sourceQualityID int, sourceQualityName string) *int64 {
	if ssID, ok := radarrQualityMap[sourceQualityID]; ok {
		return &ssID
	}
	if q, ok := quality.GetQualityByName(sourceQualityName); ok {
		id := int64(q.ID)
		return &id
	}
	return nil
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}
