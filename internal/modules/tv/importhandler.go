package tv

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	tvlib "github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/module"
)

var _ module.ImportHandler = (*importHandler)(nil)

type importHandler struct {
	tvSvc         *tvlib.Service
	rootFolderSvc *rootfolder.Service
	logger        zerolog.Logger
}

func newImportHandler(tvSvc *tvlib.Service, rootFolderSvc *rootfolder.Service, logger *zerolog.Logger) *importHandler {
	return &importHandler{
		tvSvc:         tvSvc,
		rootFolderSvc: rootFolderSvc,
		logger:        logger.With().Str("component", "tv-import").Logger(),
	}
}

func (h *importHandler) MatchDownload(ctx context.Context, download *module.CompletedDownload) ([]module.MatchedEntity, error) {
	if download.IsGroupDownload {
		return h.matchGroupDownload(ctx, download)
	}
	return h.matchSingleEpisode(ctx, download)
}

func (h *importHandler) matchGroupDownload(ctx context.Context, download *module.CompletedDownload) ([]module.MatchedEntity, error) {
	series, err := h.tvSvc.GetSeries(ctx, download.EntityID)
	if err != nil {
		return nil, fmt.Errorf("series %d not found: %w", download.EntityID, err)
	}

	rf, err := h.rootFolderSvc.Get(ctx, series.RootFolderID)
	if err != nil {
		return nil, fmt.Errorf("root folder %d not found: %w", series.RootFolderID, err)
	}

	entity := module.MatchedEntity{
		ModuleType:       module.TypeTV,
		EntityType:       module.EntitySeries,
		EntityID:         series.ID,
		Title:            series.Title,
		RootFolder:       rf.Path,
		Confidence:       1.0,
		Source:           "queue",
		QualityProfileID: series.QualityProfileID,
		TokenData:        buildSeriesTokenData(series),
		GroupInfo:        &module.GroupMatchInfo{},
	}

	if download.SeasonNumber != nil {
		entity.TokenData["SeasonNumber"] = *download.SeasonNumber
	}

	return []module.MatchedEntity{entity}, nil
}

func (h *importHandler) matchSingleEpisode(ctx context.Context, download *module.CompletedDownload) ([]module.MatchedEntity, error) {
	episode, err := h.tvSvc.GetEpisode(ctx, download.EntityID)
	if err != nil {
		return nil, fmt.Errorf("episode %d not found: %w", download.EntityID, err)
	}

	series, err := h.tvSvc.GetSeries(ctx, episode.SeriesID)
	if err != nil {
		return nil, err
	}

	rf, err := h.rootFolderSvc.Get(ctx, series.RootFolderID)
	if err != nil {
		return nil, fmt.Errorf("root folder %d not found: %w", series.RootFolderID, err)
	}

	return []module.MatchedEntity{{
		ModuleType:       module.TypeTV,
		EntityType:       module.EntityEpisode,
		EntityID:         episode.ID,
		Title:            fmt.Sprintf("%s - S%02dE%02d", series.Title, episode.SeasonNumber, episode.EpisodeNumber),
		RootFolder:       rf.Path,
		Confidence:       1.0,
		Source:           "queue",
		QualityProfileID: series.QualityProfileID,
		TokenData:        buildTVTokenData(series, episode),
	}}, nil
}

func (h *importHandler) MatchIndividualFile(ctx context.Context, filePath string, parentEntity *module.MatchedEntity) (*module.MatchedEntity, error) {
	filename := filepath.Base(filePath)
	parsed := scanner.ParseFilename(filename)

	if !parsed.IsTV || parsed.Episode == 0 {
		return nil, fmt.Errorf("cannot parse TV episode from %q", filename)
	}

	seasonNum := resolveSeasonNumber(parsed.Season, parentEntity)

	series, err := h.tvSvc.GetSeries(ctx, parentEntity.EntityID)
	if err != nil {
		return nil, err
	}

	episode, err := h.tvSvc.GetEpisodeByNumber(ctx, series.ID, seasonNum, parsed.Episode)
	if err != nil {
		return nil, fmt.Errorf("episode S%02dE%02d not found for series %d: %w", seasonNum, parsed.Episode, series.ID, err)
	}

	rf, err := h.rootFolderSvc.Get(ctx, series.RootFolderID)
	if err != nil {
		return nil, fmt.Errorf("root folder %d not found: %w", series.RootFolderID, err)
	}

	entity := &module.MatchedEntity{
		ModuleType:       module.TypeTV,
		EntityType:       module.EntityEpisode,
		EntityID:         episode.ID,
		Title:            fmt.Sprintf("%s - S%02dE%02d", series.Title, seasonNum, parsed.Episode),
		RootFolder:       rf.Path,
		Confidence:       0.9,
		Source:           "parse",
		QualityProfileID: series.QualityProfileID,
		TokenData:        buildTVTokenData(series, episode),
	}

	if parsed.EndEpisode > parsed.Episode {
		entity.EntityIDs = h.collectMultiEpisodeIDs(ctx, series.ID, seasonNum, parsed.Episode, parsed.EndEpisode)
	}

	return entity, nil
}

func resolveSeasonNumber(parsedSeason int, parentEntity *module.MatchedEntity) int {
	if parsedSeason != 0 {
		return parsedSeason
	}
	if parentEntity.GroupInfo != nil {
		if sn, ok := parentEntity.TokenData["SeasonNumber"].(int); ok {
			return sn
		}
	}
	return 0
}

func (h *importHandler) collectMultiEpisodeIDs(ctx context.Context, seriesID int64, seasonNum, startEp, endEp int) []int64 {
	var ids []int64
	for epNum := startEp; epNum <= endEp; epNum++ {
		ep, err := h.tvSvc.GetEpisodeByNumber(ctx, seriesID, seasonNum, epNum)
		if err == nil {
			ids = append(ids, ep.ID)
		}
	}
	return ids
}

func (h *importHandler) IsGroupImportReady(_ context.Context, _ *module.MatchedEntity, matchedFiles []module.MatchedEntity) bool {
	return len(matchedFiles) > 0
}

func (h *importHandler) ImportFile(ctx context.Context, filePath string, entity *module.MatchedEntity, qi *module.QualityInfo) (*module.ImportResult, error) {
	qid := int64(qi.QualityID)
	originalPath, _ := entity.TokenData["OriginalPath"].(string)
	epFile, err := h.tvSvc.AddEpisodeFile(ctx, entity.EntityID, &tvlib.CreateEpisodeFileInput{
		Path:         filePath,
		QualityID:    &qid,
		Quality:      qi.Quality,
		OriginalPath: originalPath,
	})
	if err != nil {
		return nil, fmt.Errorf("create episode file record: %w", err)
	}

	if len(entity.EntityIDs) > 1 {
		for _, epID := range entity.EntityIDs[1:] {
			_, err := h.tvSvc.AddEpisodeFile(ctx, epID, &tvlib.CreateEpisodeFileInput{
				Path:      filePath,
				QualityID: &qid,
				Quality:   qi.Quality,
			})
			if err != nil {
				h.logger.Warn().Err(err).Int64("episodeID", epID).Msg("Failed to create multi-episode file record")
			}
		}
	}

	return &module.ImportResult{
		FileID:          epFile.ID,
		DestinationPath: filePath,
		QualityID:       qi.QualityID,
	}, nil
}

func defaultFormatType(ft string) string {
	if ft == "" {
		return "standard"
	}
	return ft
}

func (h *importHandler) SupportsMultiFileDownload() bool { return true }

func (h *importHandler) MediaInfoFields() []module.MediaInfoFieldDecl {
	return []module.MediaInfoFieldDecl{
		{Name: "video_codec", Required: false},
		{Name: "audio_codec", Required: false},
		{Name: "audio_channels", Required: false},
		{Name: "resolution", Required: false},
		{Name: "dynamic_range", Required: false},
	}
}

func buildSeriesTokenData(series *tvlib.Series) map[string]any {
	return map[string]any{
		"SeriesTitle": series.Title,
		"SeriesYear":  series.Year,
		"SeriesType":  defaultFormatType(series.FormatType),
	}
}

func buildTVTokenData(series *tvlib.Series, episode *tvlib.Episode) map[string]any {
	data := map[string]any{
		"SeriesID":      series.ID,
		"SeriesTitle":   series.Title,
		"SeriesYear":    series.Year,
		"SeriesType":    defaultFormatType(series.FormatType),
		"SeasonNumber":  episode.SeasonNumber,
		"EpisodeNumber": episode.EpisodeNumber,
		"EpisodeTitle":  episode.Title,
		"SeasonFolder":  series.SeasonFolder,
		"IsSpecial":     episode.SeasonNumber == 0,
	}
	if episode.AirDate != nil {
		data["AirDate"] = *episode.AirDate
	}
	return data
}
