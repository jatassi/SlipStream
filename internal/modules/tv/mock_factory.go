package tv

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	fsmock "github.com/slipstream/slipstream/internal/filesystem/mock"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/status"
	tvlib "github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/module"
)

var _ module.MockFactory = (*Module)(nil)

func (m *Module) CreateMockMetadataProvider() module.MetadataProvider { return nil }

func (m *Module) CreateTestRootFolders(ctx context.Context, mctx *module.MockContext) error {
	folders, err := m.rootFolderSvc.List(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to list root folders for mock TV setup")
		return nil
	}

	for _, f := range folders {
		if f.Path == fsmock.MockTVPath {
			m.logger.Info().Msg("Mock TV root folder already exists")
			return nil
		}
	}

	_, err = mctx.RootFolderCreator.Create(ctx, fsmock.MockTVPath, "Mock TV", "tv")
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to create mock TV root folder")
	} else {
		m.logger.Info().Str("path", fsmock.MockTVPath).Msg("Created mock TV root folder")
	}
	return nil
}

func (m *Module) CreateSampleLibraryData(ctx context.Context, mctx *module.MockContext) error {
	existingSeries, _ := m.tvService.ListSeries(ctx, tvlib.ListSeriesOptions{})
	if len(existingSeries) > 0 {
		m.logger.Info().Int("count", len(existingSeries)).Msg("Dev database already has series")
		return nil
	}

	var tvRootID int64
	folders, err := m.rootFolderSvc.List(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to list root folders for mock series")
		return nil
	}
	for _, f := range folders {
		if f.Path == fsmock.MockTVPath {
			tvRootID = f.ID
			break
		}
	}
	if tvRootID == 0 {
		m.logger.Warn().Msg("Mock TV root folder not found, skipping series population")
		return nil
	}

	qualityProfileID := mctx.DefaultProfileID
	if qualityProfileID == 0 && len(mctx.QualityProfiles) > 0 {
		qualityProfileID = mctx.QualityProfiles[0].ID
	}
	if qualityProfileID == 0 {
		m.logger.Warn().Msg("No quality profiles available for mock series")
		return nil
	}

	m.populateMockSeries(ctx, tvRootID, qualityProfileID)
	return nil
}

func (m *Module) populateMockSeries(ctx context.Context, rootFolderID, qualityProfileID int64) {
	mockSeriesIDs := []struct {
		tvdbID   int
		hasFiles bool
	}{
		{81189, true},   // Breaking Bad
		{121361, true},  // Game of Thrones
		{305288, true},  // Stranger Things
		{361753, true},  // The Mandalorian
		{355567, false}, // The Boys
	}

	for _, s := range mockSeriesIDs {
		m.createOneMockSeries(ctx, s.tvdbID, s.hasFiles, rootFolderID, qualityProfileID)
	}

	m.logger.Info().Int("count", len(mockSeriesIDs)).Msg("Populated mock series")
}

func (m *Module) createOneMockSeries(ctx context.Context, tvdbID int, hasFiles bool, rootFolderID, qualityProfileID int64) {
	seriesMeta, err := m.metadataSvc.GetSeriesByTVDB(ctx, tvdbID)
	if err != nil {
		m.logger.Error().Err(err).Int("tvdbID", tvdbID).Msg("Failed to fetch mock series metadata")
		return
	}

	seasonsMeta, err := m.metadataSvc.GetSeriesSeasons(ctx, seriesMeta.TmdbID, seriesMeta.TvdbID)
	if err != nil {
		m.logger.Warn().Err(err).Str("title", seriesMeta.Title).Msg("Failed to fetch seasons, using empty")
		seasonsMeta = nil
	}

	seasons := convertSeasonsMetadata(seasonsMeta)
	path := fsmock.MockTVPath + "/" + seriesMeta.Title

	input := tvlib.CreateSeriesInput{
		Title:            seriesMeta.Title,
		Year:             seriesMeta.Year,
		TvdbID:           seriesMeta.TvdbID,
		TmdbID:           seriesMeta.TmdbID,
		ImdbID:           seriesMeta.ImdbID,
		Overview:         seriesMeta.Overview,
		Runtime:          seriesMeta.Runtime,
		Network:          seriesMeta.Network,
		NetworkLogoURL:   seriesMeta.NetworkLogoURL,
		RootFolderID:     rootFolderID,
		QualityProfileID: qualityProfileID,
		Path:             path,
		Monitored:        true,
		SeasonFolder:     true,
		Seasons:          seasons,
	}

	series, err := m.tvService.CreateSeries(ctx, &input)
	if err != nil {
		m.logger.Error().Err(err).Str("title", seriesMeta.Title).Msg("Failed to create mock series")
		return
	}

	if m.artworkDownloader != nil {
		go func(meta *metadata.SeriesResult) {
			if dlErr := m.artworkDownloader.DownloadSeriesArtwork(ctx, meta); dlErr != nil {
				m.logger.Debug().Err(dlErr).Str("title", meta.Title).Msg("Failed to download series artwork")
			}
		}(seriesMeta)
	}

	if hasFiles {
		m.createMockEpisodeFiles(ctx, series.ID, path, qualityProfileID)
	}

	m.logger.Debug().Str("title", seriesMeta.Title).Bool("hasFiles", hasFiles).Int("seasons", len(seasons)).Msg("Created mock series")
}

func convertSeasonsMetadata(seasonsMeta []metadata.SeasonResult) []tvlib.SeasonInput {
	var seasons []tvlib.SeasonInput
	for _, sm := range seasonsMeta {
		episodes := convertEpisodesMetadata(sm.Episodes)
		seasons = append(seasons, tvlib.SeasonInput{
			SeasonNumber: sm.SeasonNumber,
			Monitored:    true,
			Episodes:     episodes,
		})
	}
	return seasons
}

func convertEpisodesMetadata(episodesMeta []metadata.EpisodeResult) []tvlib.EpisodeInput {
	var episodes []tvlib.EpisodeInput
	for _, ep := range episodesMeta {
		var airDate *time.Time
		if ep.AirDate != "" {
			if t, err := time.Parse("2006-01-02", ep.AirDate); err == nil {
				airDate = &t
			}
		}
		episodes = append(episodes, tvlib.EpisodeInput{
			EpisodeNumber: ep.EpisodeNumber,
			Title:         ep.Title,
			Overview:      ep.Overview,
			AirDate:       airDate,
			Monitored:     true,
		})
	}
	return episodes
}

func (m *Module) createMockEpisodeFiles(ctx context.Context, seriesID int64, seriesPath string, qualityProfileID int64) {
	vfs := fsmock.GetInstance()
	seasonDirs, err := vfs.ListDirectory(seriesPath)
	if err != nil {
		return
	}

	episodes, err := m.tvService.ListEpisodes(ctx, seriesID, nil)
	if err != nil {
		return
	}

	var profile *quality.Profile
	if m.qualitySvc != nil {
		profile, _ = m.qualitySvc.Get(ctx, qualityProfileID)
	}

	queries := sqlc.New(m.db)
	episodeMap := buildEpisodeMap(episodes)

	for _, seasonDir := range seasonDirs {
		processSeasonDirectory(ctx, m.logger, queries, episodeMap, profile, vfs, seasonDir)
	}
}

func buildEpisodeMap(episodes []tvlib.Episode) map[string]int64 {
	episodeMap := make(map[string]int64)
	for _, ep := range episodes {
		key := strconv.Itoa(ep.SeasonNumber) + ":" + strconv.Itoa(ep.EpisodeNumber)
		episodeMap[key] = ep.ID
	}
	return episodeMap
}

func processSeasonDirectory(ctx context.Context, logger *zerolog.Logger, queries *sqlc.Queries, episodeMap map[string]int64, profile *quality.Profile, vfs *fsmock.VirtualFS, seasonDir *fsmock.VirtualFile) {
	if seasonDir.Type != fsmock.FileTypeDirectory {
		return
	}

	seasonNum := parseSeasonNumber(seasonDir.Name)
	if seasonNum == 0 {
		return
	}

	episodeFiles, err := vfs.ListDirectory(seasonDir.Path)
	if err != nil {
		return
	}

	for _, f := range episodeFiles {
		processEpisodeFile(ctx, logger, queries, episodeMap, profile, f, seasonNum)
	}
}

func processEpisodeFile(ctx context.Context, _ *zerolog.Logger, queries *sqlc.Queries, episodeMap map[string]int64, profile *quality.Profile, f *fsmock.VirtualFile, seasonNum int) {
	if f.Type != fsmock.FileTypeVideo {
		return
	}

	epNum := parseEpisodeNumber(f.Name)
	if epNum == 0 {
		return
	}

	key := strconv.Itoa(seasonNum) + ":" + strconv.Itoa(epNum)
	episodeID, ok := episodeMap[key]
	if !ok {
		return
	}

	qualityName := parseQualityFromFilename(f.Name)
	qualityID := sql.NullInt64{}
	if q, ok := quality.GetQualityByName(qualityName); ok {
		qualityID = sql.NullInt64{Int64: int64(q.ID), Valid: true}
	}

	_, _ = queries.CreateEpisodeFile(ctx, sqlc.CreateEpisodeFileParams{
		EpisodeID: episodeID,
		Path:      f.Path,
		Size:      f.Size,
		Quality:   sql.NullString{String: qualityName, Valid: qualityName != ""},
		QualityID: qualityID,
	})

	episodeStatus := status.Available
	if qualityID.Valid && profile != nil {
		episodeStatus = profile.StatusForQuality(int(qualityID.Int64))
	}
	_ = queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
		ID:     episodeID,
		Status: episodeStatus,
	})
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

func parseSeasonNumber(name string) int {
	name = strings.ToLower(name)
	name = strings.TrimPrefix(name, "season ")
	name = strings.TrimPrefix(name, "s")
	name = strings.TrimSpace(name)
	num, _ := strconv.Atoi(name)
	return num
}

func parseEpisodeNumber(filename string) int {
	filename = strings.ToLower(filename)
	for i := 1; i < len(filename)-1; i++ {
		if !isEpisodeMarker(filename, i) {
			continue
		}
		return extractEpisodeDigits(filename, i)
	}
	return 0
}

func isEpisodeMarker(filename string, i int) bool {
	if filename[i] != 'e' {
		return false
	}
	if filename[i-1] < '0' || filename[i-1] > '9' {
		return false
	}
	if i+1 >= len(filename) {
		return false
	}
	return filename[i+1] >= '0' && filename[i+1] <= '9'
}

func extractEpisodeDigits(filename string, startIdx int) int {
	end := startIdx + 1
	for end < len(filename) && end < startIdx+4 && filename[end] >= '0' && filename[end] <= '9' {
		end++
	}
	num, _ := strconv.Atoi(filename[startIdx+1 : end])
	return num
}
