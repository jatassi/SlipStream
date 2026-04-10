package tv

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	tvlib "github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/module/parseutil"
)

var _ module.FileParser = (*fileParser)(nil)

// TV-specific regex patterns for filename parsing.
var (
	tvPatternSE                   = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?[.\s_-]*(.*)$`)
	tvPatternX                    = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(\d{1,2})[xX](\d{1,2})[.\s_-]*(.*)$`)
	tvPatternSeasonEpisodeSpelled = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss]eason[.\s_-]+(\d{1,2})[.\s_-]+[Ee]pisode[.\s_-]+(\d{1,2})[.\s_-]*(.*)$`)
	tvPatternSeasonPack           = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})(?:[.\s_-]|$)(.*)$`)
	tvPatternSeasonSpelled        = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss]eason[.\s_-]+(\d{1,2})(?:[.\s_-]|$)(.*)$`)
	tvPatternSeasonRange          = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})-s?(\d{1,2})[.\s_-]+(.*)$`)
	tvPatternComplete             = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(?:complete[.\s_-]*(?:series)?|the[.\s_-]+complete[.\s_-]+series)[.\s_-]+(.*)$`)
)

// TVParseExtra carries TV-specific parsed data in ParseResult.Extra.
type TVParseExtra struct {
	Season           int
	EndSeason        int
	Episode          int
	EndEpisode       int
	IsSeasonPack     bool
	IsCompleteSeries bool
}

// Compile-time check that TVParseExtra implements module.TVExtraAccessor.
var _ module.TVExtraAccessor = (*TVParseExtra)(nil)

func (e *TVParseExtra) TVSeason() int            { return e.Season }
func (e *TVParseExtra) TVEndSeason() int         { return e.EndSeason }
func (e *TVParseExtra) TVEpisode() int           { return e.Episode }
func (e *TVParseExtra) TVEndEpisode() int        { return e.EndEpisode }
func (e *TVParseExtra) TVIsSeasonPack() bool     { return e.IsSeasonPack }
func (e *TVParseExtra) TVIsCompleteSeries() bool { return e.IsCompleteSeries }

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
	result := parseTVFilename(filename)
	if result == nil {
		return nil, fmt.Errorf("filename %q not detected as TV", filename)
	}
	return result, nil
}

func (p *fileParser) TryMatch(filename string) (confidence float64, match *module.ParseResult) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	result := parseTVName(name)
	if result == nil {
		return 0, nil
	}

	extra, ok := result.Extra.(*TVParseExtra)
	if !ok || extra == nil {
		return 0, nil
	}
	if extra.Season > 0 && extra.Episode > 0 {
		return 0.9, result
	}
	if extra.IsSeasonPack {
		return 0.85, result
	}
	return 0.4, result
}

// parseTVFilename parses a TV filename (with extension) into a ParseResult.
// Returns nil if the filename doesn't match any TV pattern.
func parseTVFilename(filename string) *module.ParseResult {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return parseTVName(name)
}

// parseTVName parses a media name (without extension) trying TV patterns.
// Returns nil if no TV pattern matches.
func parseTVName(name string) *module.ParseResult {
	if result := tryTVEpisodePatterns(name); result != nil {
		return result
	}
	return tryTVPackPatterns(name)
}

func tryTVEpisodePatterns(name string) *module.ParseResult {
	if match := tvPatternSE.FindStringSubmatch(name); match != nil {
		extra := &TVParseExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		if match[4] != "" {
			extra.EndEpisode, _ = strconv.Atoi(match[4])
		}
		return buildTVParseResult(match[1], match[5], extra)
	}

	if match := tvPatternX.FindStringSubmatch(name); match != nil {
		extra := &TVParseExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		return buildTVParseResult(match[1], match[4], extra)
	}

	if match := tvPatternSeasonEpisodeSpelled.FindStringSubmatch(name); match != nil {
		extra := &TVParseExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		return buildTVParseResult(match[1], match[4], extra)
	}

	return nil
}

func tryTVPackPatterns(name string) *module.ParseResult {
	if match := tvPatternSeasonRange.FindStringSubmatch(name); match != nil {
		extra := &TVParseExtra{IsSeasonPack: true, IsCompleteSeries: true}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.EndSeason, _ = strconv.Atoi(match[3])
		return buildTVParseResult(match[1], match[4], extra)
	}

	if match := tvPatternSeasonPack.FindStringSubmatch(name); match != nil {
		extra := &TVParseExtra{IsSeasonPack: true}
		extra.Season, _ = strconv.Atoi(match[2])
		return buildTVParseResult(match[1], match[3], extra)
	}

	if match := tvPatternSeasonSpelled.FindStringSubmatch(name); match != nil {
		extra := &TVParseExtra{IsSeasonPack: true}
		extra.Season, _ = strconv.Atoi(match[2])
		return buildTVParseResult(match[1], match[3], extra)
	}

	if match := tvPatternComplete.FindStringSubmatch(name); match != nil {
		extra := &TVParseExtra{IsSeasonPack: true, IsCompleteSeries: true}
		return buildTVParseResult(match[1], match[2], extra)
	}

	return nil
}

func buildTVParseResult(rawTitle, qualityText string, extra *TVParseExtra) *module.ParseResult {
	attrs := parseutil.DetectQualityAttributes(qualityText)
	codec := attrs.Codec
	if codec != "" {
		codec = quality.NormalizeVideoCodec(codec)
	}

	var audioCodecs []string
	for _, c := range attrs.AudioCodecs {
		audioCodecs = append(audioCodecs, quality.NormalizeAudioCodec(c))
	}
	var audioChannels []string
	for _, ch := range attrs.AudioChannels {
		audioChannels = append(audioChannels, quality.NormalizeAudioChannels(ch))
	}

	hdrFormats := attrs.HDRFormats
	if len(hdrFormats) == 0 {
		hdrFormats = []string{"SDR"}
	}

	title := parseutil.CleanTitle(rawTitle)
	year := 0
	if y, remainder, found := parseutil.ExtractYear(title); found && remainder != "" {
		title = remainder
		year = y
	}

	return &module.ParseResult{
		Title:         title,
		Year:          year,
		Quality:       attrs.Quality,
		Source:        attrs.Source,
		Codec:         codec,
		HDRFormats:    hdrFormats,
		AudioCodecs:   audioCodecs,
		AudioChannels: audioChannels,
		ReleaseGroup:  parseutil.ParseReleaseGroup(qualityText),
		Revision:      parseutil.ParseRevision(qualityText),
		Edition:       parseutil.ParseEdition(qualityText),
		Languages:     parseutil.ParseLanguages(qualityText),
		Extra:         extra,
	}
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
			"SeriesID":     series.ID,
			"SeriesTitle":  series.Title,
			"SeriesYear":   series.Year,
			"SeasonNumber": extra.Season,
		},
		GroupInfo: &module.GroupMatchInfo{},
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
