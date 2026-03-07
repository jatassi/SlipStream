package movie

import (
	"context"
	"strconv"
	"strings"

	fsmock "github.com/slipstream/slipstream/internal/filesystem/mock"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/module"
)

var _ module.MockFactory = (*Module)(nil)

func (m *Module) CreateMockMetadataProvider() module.MetadataProvider { return nil }

func (m *Module) CreateTestRootFolders(ctx context.Context, mctx *module.MockContext) error {
	folders, err := m.rootFolderSvc.List(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to list root folders for mock movie setup")
		return nil
	}

	for _, f := range folders {
		if f.Path == fsmock.MockMoviesPath {
			m.logger.Info().Msg("Mock movie root folder already exists")
			return nil
		}
	}

	_, err = mctx.RootFolderCreator.Create(ctx, fsmock.MockMoviesPath, "Mock Movies", "movie")
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to create mock movies root folder")
	} else {
		m.logger.Info().Str("path", fsmock.MockMoviesPath).Msg("Created mock movies root folder")
	}
	return nil
}

func (m *Module) CreateSampleLibraryData(ctx context.Context, mctx *module.MockContext) error {
	existingMovies, _ := m.movieService.List(ctx, movies.ListMoviesOptions{})
	if len(existingMovies) > 0 {
		m.logger.Info().Int("count", len(existingMovies)).Msg("Dev database already has movies")
		return nil
	}

	var movieRootID int64
	folders, err := m.rootFolderSvc.List(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to list root folders for mock movies")
		return nil
	}
	for _, f := range folders {
		if f.Path == fsmock.MockMoviesPath {
			movieRootID = f.ID
			break
		}
	}
	if movieRootID == 0 {
		m.logger.Warn().Msg("Mock movie root folder not found, skipping movie population")
		return nil
	}

	qualityProfileID := mctx.DefaultProfileID
	if qualityProfileID == 0 && len(mctx.QualityProfiles) > 0 {
		qualityProfileID = mctx.QualityProfiles[0].ID
	}
	if qualityProfileID == 0 {
		m.logger.Warn().Msg("No quality profiles available for mock movies")
		return nil
	}

	m.populateMockMovies(ctx, movieRootID, qualityProfileID)
	return nil
}

func (m *Module) populateMockMovies(ctx context.Context, rootFolderID, qualityProfileID int64) {
	mockMovieIDs := []struct {
		tmdbID   int
		hasFiles bool
	}{
		{603, true},     // The Matrix
		{27205, true},   // Inception
		{438631, true},  // Dune
		{680, true},     // Pulp Fiction
		{550, true},     // Fight Club
		{693134, false}, // Dune: Part Two
		{872585, false}, // Oppenheimer
		{346698, false}, // Barbie
	}

	for _, mm := range mockMovieIDs {
		movieMeta, err := m.metadataSvc.GetMovie(ctx, mm.tmdbID)
		if err != nil {
			m.logger.Error().Err(err).Int("tmdbID", mm.tmdbID).Msg("Failed to fetch mock movie metadata")
			continue
		}

		path := fsmock.MockMoviesPath + "/" + movieMeta.Title + " (" + strconv.Itoa(movieMeta.Year) + ")"

		input := movies.CreateMovieInput{
			Title:            movieMeta.Title,
			Year:             movieMeta.Year,
			TmdbID:           movieMeta.ID,
			ImdbID:           movieMeta.ImdbID,
			Overview:         movieMeta.Overview,
			Runtime:          movieMeta.Runtime,
			RootFolderID:     rootFolderID,
			QualityProfileID: qualityProfileID,
			Path:             path,
			Monitored:        true,
		}

		movie, err := m.movieService.Create(ctx, &input)
		if err != nil {
			m.logger.Error().Err(err).Str("title", movieMeta.Title).Msg("Failed to create mock movie")
			continue
		}

		if m.artworkDownloader != nil {
			go func(meta *movies.CreateMovieInput) {
				if dlErr := m.artworkDownloader.DownloadMovieArtwork(ctx, movieMeta); dlErr != nil {
					m.logger.Debug().Err(dlErr).Str("title", meta.Title).Msg("Failed to download movie artwork")
				}
			}(&input)
		}

		if mm.hasFiles {
			m.createMockMovieFiles(ctx, movie.ID, path)
		}

		m.logger.Debug().Str("title", movieMeta.Title).Bool("hasFiles", mm.hasFiles).Msg("Created mock movie")
	}

	m.logger.Info().Int("count", len(mockMovieIDs)).Msg("Populated mock movies")
}

func (m *Module) createMockMovieFiles(ctx context.Context, movieID int64, moviePath string) {
	vfs := fsmock.GetInstance()
	files, err := vfs.ListDirectory(moviePath)
	if err != nil {
		return
	}

	for _, f := range files {
		if f.Type != fsmock.FileTypeVideo {
			continue
		}

		qualityName := parseQualityFromFilename(f.Name)
		input := movies.CreateMovieFileInput{
			Path:    f.Path,
			Size:    f.Size,
			Quality: qualityName,
		}
		if q, ok := quality.GetQualityByName(qualityName); ok {
			qid := int64(q.ID)
			input.QualityID = &qid
		}

		_, err := m.movieService.AddFile(ctx, movieID, &input)
		if err != nil {
			m.logger.Debug().Err(err).Str("path", f.Path).Msg("Failed to create movie file")
		}
	}
}

func parseQualityFromFilename(filename string) string {
	filename = strings.ToLower(filename)
	if strings.Contains(filename, "2160p") {
		if strings.Contains(filename, "remux") {
			return "Remux-2160p"
		}
		return "Bluray-2160p"
	}
	if strings.Contains(filename, "1080p") {
		if strings.Contains(filename, "web") {
			return "WEBDL-1080p"
		}
		return "Bluray-1080p"
	}
	if strings.Contains(filename, "720p") {
		if strings.Contains(filename, "web") {
			return "WEBDL-720p"
		}
		return "Bluray-720p"
	}
	return "Unknown"
}
