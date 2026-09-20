package importer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/organizer"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/mediainfo"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/modules/movie"
	tvmodule "github.com/slipstream/slipstream/internal/modules/tv"
	"github.com/slipstream/slipstream/internal/testutil"
)

type manualImportFixture struct {
	handlers *importer.Handlers
	movies   *movies.Service
	tv       *tv.Service
	movieID  int64
	movieRF  string
	episode  *tv.Episode
	seriesRF string
}

func newManualImportFixture(t *testing.T) *manualImportFixture {
	t.Helper()
	ctx := context.Background()
	tdb := testutil.NewTestDB(t)
	t.Cleanup(tdb.Close)
	logger := testutil.NopLogger()

	qualitySvc := quality.NewService(tdb.Conn, &logger)
	rootFolderSvc := rootfolder.NewService(tdb.Conn, &logger, nil, nil)
	moviesSvc := movies.NewService(tdb.Conn, nil, &logger, qualitySvc, nil)
	tvSvc := tv.NewService(tdb.Conn, nil, &logger, qualitySvc, nil)

	importSvc := importer.NewService(
		tdb.Conn, nil, moviesSvc, tvSvc, rootFolderSvc,
		organizer.NewService(&logger),
		mediainfo.NewService(mediainfo.DefaultConfig(), &logger),
		nil, importer.DefaultConfig(), &logger, nil, nil, qualitySvc, nil, nil,
	)
	registry := module.NewRegistry()
	registry.Register(movie.NewModule(tdb.Conn, nil, moviesSvc, rootFolderSvc, nil, &logger))
	registry.Register(tvmodule.NewModule(tdb.Conn, nil, tvSvc, rootFolderSvc, nil, qualitySvc, &logger))
	importSvc.SetRegistry(registry)

	settings := importer.DefaultImportSettings()
	settings.MinimumFileSizeMB = 1
	if _, err := importSvc.UpdateSettings(ctx, &settings); err != nil {
		t.Fatalf("update settings: %v", err)
	}

	movieProfile := createProfile(t, qualitySvc, "movie")
	tvProfile := createProfile(t, qualitySvc, "tv")

	movieRF := filepath.Join(t.TempDir(), "movies")
	seriesRF := filepath.Join(t.TempDir(), "series")
	for _, dir := range []string{movieRF, seriesRF} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	movieRoot, err := rootFolderSvc.Create(ctx, rootfolder.CreateRootFolderInput{Path: movieRF, MediaType: "movie"})
	if err != nil {
		t.Fatalf("create movie root folder: %v", err)
	}
	seriesRoot, err := rootFolderSvc.Create(ctx, rootfolder.CreateRootFolderInput{Path: seriesRF, MediaType: "tv"})
	if err != nil {
		t.Fatalf("create series root folder: %v", err)
	}

	mv, err := moviesSvc.Create(ctx, &movies.CreateMovieInput{
		Title: "Oppenheimer", Year: 2023, TmdbID: 872585,
		RootFolderID: movieRoot.ID, QualityProfileID: movieProfile.ID, Monitored: true,
	})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}

	series, err := tvSvc.CreateSeries(ctx, &tv.CreateSeriesInput{
		Title: "Breaking Bad", Year: 2008, TvdbID: 81189,
		RootFolderID: seriesRoot.ID, QualityProfileID: tvProfile.ID,
		Monitored: true, SeasonFolder: true, ProductionStatus: "ended",
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	episode, err := tvSvc.CreateEpisode(ctx, series.ID, 1, 1, "Pilot")
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}

	return &manualImportFixture{
		handlers: importer.NewHandlers(importSvc, tdb.Conn),
		movies:   moviesSvc,
		tv:       tvSvc,
		movieID:  mv.ID,
		movieRF:  movieRoot.Path,
		episode:  episode,
		seriesRF: seriesRoot.Path,
	}
}

func createProfile(t *testing.T, qs *quality.Service, moduleType string) *quality.Profile {
	t.Helper()
	profile, err := qs.Create(context.Background(), &quality.CreateProfileInput{
		Name:       "HD-1080p " + moduleType,
		ModuleType: moduleType,
		Cutoff:     11,
		Items:      quality.HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("create %s quality profile: %v", moduleType, err)
	}
	return profile
}

func writeVideoFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, make([]byte, 2*1024*1024), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func (f *manualImportFixture) manualImport(t *testing.T, req importer.ManualImportRequest) importer.ManualImportResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/import/manual", bytes.NewReader(body))
	httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	if err := f.handlers.ManualImport(e.NewContext(httpReq, rec)); err != nil {
		t.Fatalf("ManualImport returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	var resp importer.ManualImportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestManualImportMovie(t *testing.T) {
	f := newManualImportFixture(t)
	source := writeVideoFile(t, "Oppenheimer.2023.1080p.BluRay.x264-GROUP.mkv")

	resp := f.manualImport(t, importer.ManualImportRequest{Path: source, MediaType: "movie", MediaID: f.movieID})

	if !resp.Success {
		t.Fatalf("manual movie import failed: %s", resp.Error)
	}
	if !strings.HasPrefix(resp.DestinationPath, f.movieRF) {
		t.Fatalf("destination %q is not under root folder %q", resp.DestinationPath, f.movieRF)
	}
	if _, err := os.Stat(resp.DestinationPath); err != nil {
		t.Fatalf("destination file missing: %v", err)
	}
	files, err := f.movies.GetFiles(context.Background(), f.movieID)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != resp.DestinationPath {
		t.Fatalf("expected one movie file at %q, got %+v", resp.DestinationPath, files)
	}
}

func TestManualImportEpisode(t *testing.T) {
	f := newManualImportFixture(t)
	source := writeVideoFile(t, "Breaking.Bad.S01E01.1080p.BluRay.x264-GROUP.mkv")

	resp := f.manualImport(t, importer.ManualImportRequest{Path: source, MediaType: "episode", MediaID: f.episode.ID})

	if !resp.Success {
		t.Fatalf("manual episode import failed: %s", resp.Error)
	}
	if !strings.HasPrefix(resp.DestinationPath, f.seriesRF) {
		t.Fatalf("destination %q is not under root folder %q", resp.DestinationPath, f.seriesRF)
	}
	if _, err := os.Stat(resp.DestinationPath); err != nil {
		t.Fatalf("destination file missing: %v", err)
	}
	file, err := f.tv.GetEpisodeFile(context.Background(), f.episode.ID)
	if err != nil {
		t.Fatal(err)
	}
	if file.Path != resp.DestinationPath {
		t.Fatalf("episode file path %q, want %q", file.Path, resp.DestinationPath)
	}
}
