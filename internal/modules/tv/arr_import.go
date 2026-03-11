package tv

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
	tvlib "github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/pathutil"
)

// sonarrQualityMap maps Sonarr quality IDs to SlipStream quality IDs.
// Mirrors arrimport.sonarrQualityMap — duplicated here to avoid an import cycle.
var sonarrQualityMap = map[int]int64{
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
	23: 2,  // DVD-R -> DVD
	13: 2,  // Bluray-480p -> DVD (nearest) [Sonarr-only]
	22: 2,  // Bluray-576p -> DVD (nearest) [Sonarr-only]
	20: 12, // Sonarr: Bluray-1080p Remux -> Remux-1080p
	21: 17, // Sonarr: Bluray-2160p Remux -> Remux-2160p
}

func (m *Module) ExternalAppName() string { return "Sonarr" }

func (m *Module) PreviewSeries(ctx context.Context, reader module.ArrReader) ([]module.ArrImportPreviewItem, error) {
	seriesList, err := reader.ReadSeries(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]module.ArrImportPreviewItem, 0, len(seriesList))
	for i := range seriesList {
		s := &seriesList[i]
		item := module.ArrImportPreviewItem{
			Title:            s.Title,
			Year:             s.Year,
			TmdbID:           s.TmdbID,
			TvdbID:           s.TvdbID,
			Monitored:        s.Monitored,
			QualityProfileID: s.QualityProfileID,
			PosterURL:        s.PosterURL,
		}

		if s.TvdbID == 0 {
			item.Status = "skip"
			item.SkipReason = "no TVDB ID"
			items = append(items, item)
			continue
		}

		episodes, err := reader.ReadEpisodes(ctx, s.ID)
		if err != nil {
			m.logger.Warn().Int64("seriesId", s.ID).Err(err).Msg("failed to read episodes")
			episodes = []module.ArrSourceEpisode{}
		}
		item.EpisodeCount = len(episodes)

		files, err := reader.ReadEpisodeFiles(ctx, s.ID)
		if err != nil {
			m.logger.Warn().Int64("seriesId", s.ID).Err(err).Msg("failed to read episode files")
			files = []module.ArrSourceEpisodeFile{}
		}
		item.FileCount = len(files)

		_, err = m.tvService.GetSeriesByTvdbID(ctx, s.TvdbID)
		if err != nil {
			errMsg := err.Error()
			if errMsg == "series not found" {
				item.Status = "new"
			} else {
				m.logger.Warn().Int("tvdbId", s.TvdbID).Err(err).Msg("failed to check series existence")
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

func (m *Module) ImportSeries(ctx context.Context, series module.ArrSourceSeries, reader module.ArrReader, mappings module.ArrImportMappings) (*module.ImportedEntity, error) { //nolint:gocritic // interface-mandated signature
	if m.shouldSkipSeries(ctx, &series, &mappings) {
		return nil, nil //nolint:nilnil // nil entity + nil error = skipped per convention
	}

	rootFolderID, ok := mappings.RootFolderMapping[series.RootFolderPath]
	if !ok {
		return nil, nil //nolint:nilnil // nil entity + nil error = skipped per convention
	}

	qualityProfileID, ok := mappings.QualityProfileMapping[series.QualityProfileID]
	if !ok {
		return nil, nil //nolint:nilnil // nil entity + nil error = skipped per convention
	}

	episodes, episodeFiles := m.readSeriesData(ctx, &series, reader)
	fileIDToEpisode := buildFileEpisodeLookup(episodes)
	seasonInputs := buildSeasonInputs(series.Seasons, episodes)

	input := m.buildSeriesInput(&series, rootFolderID, qualityProfileID, seasonInputs)

	createdSeries, err := m.tvService.CreateSeries(ctx, input)
	if err != nil {
		return nil, err
	}

	m.preserveSeriesAddedAt(ctx, series.Added, createdSeries.ID)
	m.refreshSeriesMetadata(ctx, createdSeries.ID, series.Title)

	entity := &module.ImportedEntity{
		EntityType: module.EntitySeries,
		EntityID:   createdSeries.ID,
		Title:      series.Title,
	}

	m.importEpisodeFiles(ctx, createdSeries.ID, series.Path, episodeFiles, fileIDToEpisode, entity)

	return entity, nil
}

func (m *Module) shouldSkipSeries(ctx context.Context, series *module.ArrSourceSeries, mappings *module.ArrImportMappings) bool {
	if series.TvdbID == 0 {
		return true
	}
	if len(mappings.SelectedIDs) > 0 && !slices.Contains(mappings.SelectedIDs, series.TvdbID) {
		return true
	}
	return m.seriesExists(ctx, series.TvdbID)
}

func (m *Module) seriesExists(ctx context.Context, tvdbID int) bool {
	_, err := m.tvService.GetSeriesByTvdbID(ctx, tvdbID)
	if err == nil {
		return true
	}
	if !errors.Is(err, tvlib.ErrSeriesNotFound) {
		m.logger.Warn().Int("tvdbId", tvdbID).Err(err).Msg("error checking series existence")
		return true
	}
	return false
}

func (m *Module) readSeriesData(ctx context.Context, series *module.ArrSourceSeries, reader module.ArrReader) ([]module.ArrSourceEpisode, []module.ArrSourceEpisodeFile) {
	episodes, err := reader.ReadEpisodes(ctx, series.ID)
	if err != nil {
		m.logger.Warn().Int64("seriesId", series.ID).Err(err).Msg("failed to read episodes")
		episodes = []module.ArrSourceEpisode{}
	}

	files, err := reader.ReadEpisodeFiles(ctx, series.ID)
	if err != nil {
		m.logger.Warn().Int64("seriesId", series.ID).Err(err).Msg("failed to read episode files")
		files = []module.ArrSourceEpisodeFile{}
	}

	return episodes, files
}

func buildFileEpisodeLookup(episodes []module.ArrSourceEpisode) map[int64]module.ArrSourceEpisode {
	lookup := make(map[int64]module.ArrSourceEpisode)
	for i := range episodes {
		ep := &episodes[i]
		if ep.HasFile && ep.EpisodeFileID > 0 {
			lookup[ep.EpisodeFileID] = *ep
		}
	}
	return lookup
}

func buildSeasonInputs(seasons []module.ArrSourceSeason, episodes []module.ArrSourceEpisode) []tvlib.SeasonInput {
	episodesBySeason := make(map[int][]module.ArrSourceEpisode)
	for i := range episodes {
		ep := &episodes[i]
		episodesBySeason[ep.SeasonNumber] = append(episodesBySeason[ep.SeasonNumber], *ep)
	}

	seasonInputs := make([]tvlib.SeasonInput, 0, len(seasons))
	for i := range seasons {
		season := &seasons[i]
		si := tvlib.SeasonInput{
			SeasonNumber: season.SeasonNumber,
			Monitored:    season.Monitored,
		}

		seasonEps := episodesBySeason[season.SeasonNumber]
		epInputs := make([]tvlib.EpisodeInput, 0, len(seasonEps))
		for j := range seasonEps {
			ep := &seasonEps[j]
			epInputs = append(epInputs, tvlib.EpisodeInput{
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

func (m *Module) buildSeriesInput(series *module.ArrSourceSeries, rootFolderID, qualityProfileID int64, seasonInputs []tvlib.SeasonInput) *tvlib.CreateSeriesInput {
	return &tvlib.CreateSeriesInput{
		Title:            series.Title,
		Year:             series.Year,
		TvdbID:           series.TvdbID,
		TmdbID:           series.TmdbID,
		ImdbID:           series.ImdbID,
		Overview:         series.Overview,
		Runtime:          series.Runtime,
		Path:             pathutil.NormalizePath(series.Path),
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

func (m *Module) preserveSeriesAddedAt(ctx context.Context, added time.Time, seriesID int64) {
	if !added.IsZero() {
		_, _ = m.db.ExecContext(ctx, "UPDATE series SET added_at = ? WHERE id = ?", added, seriesID)
	}
}

func (m *Module) refreshSeriesMetadata(ctx context.Context, seriesID int64, title string) {
	if _, err := m.metadataProvider.RefreshMetadata(ctx, seriesID); err != nil {
		m.logger.Warn().Err(err).Int64("seriesId", seriesID).Str("title", title).Msg("failed to refresh series metadata")
	}
}

func (m *Module) importEpisodeFiles(
	ctx context.Context, seriesID int64, seriesPath string,
	files []module.ArrSourceEpisodeFile, fileIDToEpisode map[int64]module.ArrSourceEpisode, entity *module.ImportedEntity,
) {
	for i := range files {
		file := &files[i]
		sourceEp, found := fileIDToEpisode[file.ID]
		if !found {
			continue
		}

		slipEpisode, err := m.tvService.GetEpisodeByNumber(ctx, seriesID, sourceEp.SeasonNumber, sourceEp.EpisodeNumber)
		if err != nil {
			entity.Errors = append(entity.Errors, fmt.Sprintf("episode not found for file S%02dE%02d: %v", sourceEp.SeasonNumber, sourceEp.EpisodeNumber, err))
			continue
		}

		if err := m.importSingleEpisodeFile(ctx, slipEpisode, file, seriesPath, entity); err != nil {
			entity.Errors = append(entity.Errors, err.Error())
		}
	}
}

func (m *Module) importSingleEpisodeFile(
	ctx context.Context, episode *tvlib.Episode, file *module.ArrSourceEpisodeFile, seriesPath string, entity *module.ImportedEntity,
) error {
	filePath := resolveFilePath(seriesPath, file.RelativePath)

	if _, err := os.Stat(filePath); err != nil {
		return err
	}

	qualityID := mapSonarrQualityID(file.QualityID, file.QualityName)

	var originalFilename string
	if file.OriginalFilePath != "" {
		originalFilename = path.Base(file.OriginalFilePath)
	}

	fileInput := &tvlib.CreateEpisodeFileInput{
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

	createdFile, err := m.tvService.AddEpisodeFile(ctx, episode.ID, fileInput)
	if err != nil {
		return err
	}
	entity.FilesImported++

	if m.slotsService != nil && m.slotsService.IsMultiVersionEnabled(ctx) {
		_ = m.slotsService.InitializeSlotAssignments(ctx, "episode", episode.ID)
	}
	m.assignSlot(ctx, filePath, "episode", episode.ID, createdFile.ID)
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

// mapSonarrQualityID maps a Sonarr quality ID to a SlipStream quality ID.
// Falls back to matching by quality name if the static map has no entry.
func mapSonarrQualityID(sourceQualityID int, sourceQualityName string) *int64 {
	if ssID, ok := sonarrQualityMap[sourceQualityID]; ok {
		return &ssID
	}
	if q, ok := quality.GetQualityByName(sourceQualityName); ok {
		id := int64(q.ID)
		return &id
	}
	return nil
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

func resolveFilePath(parentPath, relativePath string) string {
	if strings.HasPrefix(relativePath, "/") || (len(relativePath) >= 2 && relativePath[1] == ':') {
		return pathutil.NormalizePath(relativePath)
	}
	sep := "/"
	if strings.Contains(parentPath, "\\") {
		sep = "\\"
	}
	return pathutil.NormalizePath(strings.TrimRight(parentPath, "/\\") + sep + relativePath)
}
