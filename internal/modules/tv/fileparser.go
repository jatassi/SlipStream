package tv

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	tvlib "github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/module/parseutil"
)

var _ module.FileParser = (*fileParser)(nil)

// TVParseExtra carries TV-specific parsed data in ParseResult.Extra.
type TVParseExtra struct {
	Season           int
	EndSeason        int
	Episode          int
	EndEpisode       int
	IsSeasonPack     bool
	IsCompleteSeries bool
}

type fileParser struct {
	tvSvc         *tvlib.Service
	rootFolderSvc *rootfolder.Service
	logger        zerolog.Logger
}

func newFileParser(tvSvc *tvlib.Service, rootFolderSvc *rootfolder.Service, logger *zerolog.Logger) *fileParser {
	return &fileParser{
		tvSvc:         tvSvc,
		rootFolderSvc: rootFolderSvc,
		logger:        logger.With().Str("component", "tv-fileparser").Logger(),
	}
}

func (p *fileParser) ParseFilename(filename string) (*module.ParseResult, error) {
	parsed := scanner.ParseFilename(filename)
	if !parsed.IsTV {
		return nil, fmt.Errorf("filename %q not detected as TV", filename)
	}
	return tvParsedMediaToResult(parsed), nil
}

func (p *fileParser) TryMatch(filename string) (confidence float64, match *module.ParseResult) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	parsed := scanner.ParseFilename(filename)
	if !parsed.IsTV {
		return 0, nil
	}

	result := tvParsedMediaToResult(parsed)

	if parsed.Season > 0 && parsed.Episode > 0 {
		return 0.9, result
	}
	if parsed.IsSeasonPack {
		return 0.85, result
	}
	return 0.4, result
}

func (p *fileParser) MatchToEntity(ctx context.Context, parseResult *module.ParseResult) (*module.MatchedEntity, error) {
	tvExtra, ok := parseResult.Extra.(*TVParseExtra)
	if !ok || tvExtra == nil {
		return nil, fmt.Errorf("parse result missing TV extra data")
	}

	series, err := p.tvSvc.FindByTitle(ctx, parseResult.Title, parseResult.Year)
	if err != nil {
		return nil, err
	}
	if series == nil {
		return nil, nil //nolint:nilnil // nil means no match found
	}

	rf, err := p.rootFolderSvc.Get(ctx, series.RootFolderID)
	if err != nil {
		return nil, fmt.Errorf("root folder %d not found: %w", series.RootFolderID, err)
	}

	if tvExtra.IsSeasonPack {
		return p.matchSeasonPack(series, rf.Path, tvExtra), nil
	}

	return p.matchEpisode(ctx, series, rf.Path, tvExtra)
}

func (p *fileParser) matchSeasonPack(series *tvlib.Series, rootFolder string, extra *TVParseExtra) *module.MatchedEntity {
	return &module.MatchedEntity{
		ModuleType:       module.TypeTV,
		EntityType:       module.EntitySeason,
		EntityID:         0,
		Title:            fmt.Sprintf("%s - Season %d", series.Title, extra.Season),
		RootFolder:       rootFolder,
		Confidence:       0.8,
		Source:           "parse",
		QualityProfileID: series.QualityProfileID,
		TokenData: map[string]any{
			"SeriesTitle":  series.Title,
			"SeriesYear":   series.Year,
			"SeasonNumber": extra.Season,
		},
		GroupInfo: &module.GroupMatchInfo{
			ParentEntityType: module.EntitySeason,
			IsSeasonPack:     true,
			IsCompleteSeries: extra.IsCompleteSeries,
		},
	}
}

func (p *fileParser) matchEpisode(ctx context.Context, series *tvlib.Series, rootFolder string, extra *TVParseExtra) (*module.MatchedEntity, error) {
	episode, err := p.tvSvc.GetEpisodeByNumber(ctx, series.ID, extra.Season, extra.Episode)
	if err != nil {
		if errors.Is(err, tvlib.ErrEpisodeNotFound) {
			return nil, nil //nolint:nilnil // nil means no match found
		}
		return nil, err
	}

	entity := &module.MatchedEntity{
		ModuleType:       module.TypeTV,
		EntityType:       module.EntityEpisode,
		EntityID:         episode.ID,
		Title:            fmt.Sprintf("%s - S%02dE%02d - %s", series.Title, extra.Season, extra.Episode, episode.Title),
		RootFolder:       rootFolder,
		Confidence:       0.8,
		Source:           "parse",
		QualityProfileID: series.QualityProfileID,
		TokenData:        buildTVTokenData(series, episode),
	}

	if extra.EndEpisode > extra.Episode {
		entity.EntityIDs = p.collectEpisodeIDs(ctx, series.ID, extra.Season, extra.Episode, extra.EndEpisode)
	}

	return entity, nil
}

func (p *fileParser) collectEpisodeIDs(ctx context.Context, seriesID int64, season, startEp, endEp int) []int64 {
	var ids []int64
	for epNum := startEp; epNum <= endEp; epNum++ {
		ep, err := p.tvSvc.GetEpisodeByNumber(ctx, seriesID, season, epNum)
		if err == nil && ep != nil {
			ids = append(ids, ep.ID)
		}
	}
	return ids
}

func tvParsedMediaToResult(parsed *scanner.ParsedMedia) *module.ParseResult {
	return &module.ParseResult{
		Title:         parsed.Title,
		Year:          parsed.Year,
		Quality:       parsed.Quality,
		Source:        parsed.Source,
		Codec:         parsed.Codec,
		HDRFormats:    parsed.HDRFormats,
		AudioCodecs:   parsed.AudioCodecs,
		AudioChannels: parsed.AudioChannels,
		ReleaseGroup:  parsed.ReleaseGroup,
		Revision:      parsed.Revision,
		Edition:       parsed.Edition,
		Languages:     parsed.Languages,
		Extra: &TVParseExtra{
			Season:           parsed.Season,
			EndSeason:        parsed.EndSeason,
			Episode:          parsed.Episode,
			EndEpisode:       parsed.EndEpisode,
			IsSeasonPack:     parsed.IsSeasonPack,
			IsCompleteSeries: parsed.IsCompleteSeries,
		},
	}
}
