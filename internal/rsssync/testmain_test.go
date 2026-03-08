package rsssync

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/module/parseutil"
)

// TV patterns (replicated from internal/modules/tv/fileparser.go for test use)
var (
	rssTVPatternSE          = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?[.\s_-]*(.*)$`)
	rssTVPatternX           = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(\d{1,2})[xX](\d{1,2})[.\s_-]*(.*)$`)
	rssTVPatternSeasonPack  = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})(?:[.\s_-]|$)(.*)$`)
	rssTVPatternSeasonRange = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})-s?(\d{1,2})[.\s_-]+(.*)$`)
	rssTVPatternComplete    = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(?:complete[.\s_-]*(?:series)?|the[.\s_-]+complete[.\s_-]+series)[.\s_-]+(.*)$`)
)

// Movie patterns
var (
	rssMoviePatternParen  = regexp.MustCompile(`^(.+?)\s*\((\d{4})\)\s*(.*)$`)
	rssMoviePatternDot    = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})[.\s_-]+(.*)$`)
	rssMoviePatternSimple = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})$`)
)

// rssTVExtra implements scanner.tvExtraAccessor for TV parse results.
type rssTVExtra struct {
	Season           int
	EndSeason        int
	Episode          int
	EndEpisode       int
	IsSeasonPack     bool
	IsCompleteSeries bool
}

func (e *rssTVExtra) TVSeason() int            { return e.Season }
func (e *rssTVExtra) TVEndSeason() int         { return e.EndSeason }
func (e *rssTVExtra) TVEpisode() int           { return e.Episode }
func (e *rssTVExtra) TVEndEpisode() int        { return e.EndEpisode }
func (e *rssTVExtra) TVIsSeasonPack() bool     { return e.IsSeasonPack }
func (e *rssTVExtra) TVIsCompleteSeries() bool { return e.IsCompleteSeries }

// rssTVParser implements scanner.ModuleFileParser for TV.
type rssTVParser struct{}

func (p *rssTVParser) ID() string { return "tv" }

func (p *rssTVParser) TryMatch(filename string) (float64, *scanner.ParseResultData) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	result := rssParseTVName(name)
	if result == nil {
		return 0, nil
	}

	extra := result.Extra.(*rssTVExtra)
	if extra.Season > 0 && extra.Episode > 0 {
		return 0.9, result
	}
	if extra.IsSeasonPack {
		return 0.85, result
	}
	return 0.4, result
}

func rssParseTVName(name string) *scanner.ParseResultData {
	if result := rssTryTVEpisodePatterns(name); result != nil {
		return result
	}
	return rssTryTVPackPatterns(name)
}

func rssTryTVEpisodePatterns(name string) *scanner.ParseResultData {
	if match := rssTVPatternSE.FindStringSubmatch(name); match != nil {
		extra := &rssTVExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		if match[4] != "" {
			extra.EndEpisode, _ = strconv.Atoi(match[4])
		}
		return rssBuildTVResult(match[1], match[5], extra)
	}

	if match := rssTVPatternX.FindStringSubmatch(name); match != nil {
		extra := &rssTVExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		return rssBuildTVResult(match[1], match[4], extra)
	}

	return nil
}

func rssTryTVPackPatterns(name string) *scanner.ParseResultData {
	if match := rssTVPatternSeasonRange.FindStringSubmatch(name); match != nil {
		extra := &rssTVExtra{IsSeasonPack: true, IsCompleteSeries: true}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.EndSeason, _ = strconv.Atoi(match[3])
		return rssBuildTVResult(match[1], match[4], extra)
	}

	if match := rssTVPatternSeasonPack.FindStringSubmatch(name); match != nil {
		extra := &rssTVExtra{IsSeasonPack: true}
		extra.Season, _ = strconv.Atoi(match[2])
		return rssBuildTVResult(match[1], match[3], extra)
	}

	if match := rssTVPatternComplete.FindStringSubmatch(name); match != nil {
		extra := &rssTVExtra{IsSeasonPack: true, IsCompleteSeries: true}
		return rssBuildTVResult(match[1], match[2], extra)
	}

	return nil
}

func rssBuildTVResult(rawTitle, qualityText string, extra *rssTVExtra) *scanner.ParseResultData {
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

	return &scanner.ParseResultData{
		Title:         parseutil.CleanTitle(rawTitle),
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
		IsTV:          true,
		Extra:         extra,
	}
}

// rssMovieParser implements scanner.ModuleFileParser for movies.
type rssMovieParser struct{}

func (p *rssMovieParser) ID() string { return "movie" }

func (p *rssMovieParser) TryMatch(filename string) (float64, *scanner.ParseResultData) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	result := rssParseMovieName(name)
	if result == nil {
		return 0, nil
	}

	if result.Title != "" && result.Year > 0 {
		return 0.8, result
	}
	if result.Title != "" {
		return 0.3, result
	}
	return 0, nil
}

func rssParseMovieName(name string) *scanner.ParseResultData {
	if match := rssMoviePatternParen.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		return rssBuildMovieResult(match[1], year, match[3])
	}

	if match := rssMoviePatternDot.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			return rssBuildMovieResult(match[1], year, match[3])
		}
	}

	if match := rssMoviePatternSimple.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			return rssBuildMovieResult(match[1], year, "")
		}
	}

	return nil
}

func rssBuildMovieResult(rawTitle string, year int, qualityText string) *scanner.ParseResultData {
	result := &scanner.ParseResultData{
		Title: parseutil.CleanTitle(rawTitle),
		Year:  year,
		IsTV:  false,
	}

	if qualityText != "" {
		attrs := parseutil.DetectQualityAttributes(qualityText)
		result.Quality = attrs.Quality
		result.Source = attrs.Source
		if attrs.Codec != "" {
			result.Codec = quality.NormalizeVideoCodec(attrs.Codec)
		}

		for _, c := range attrs.AudioCodecs {
			result.AudioCodecs = append(result.AudioCodecs, quality.NormalizeAudioCodec(c))
		}
		for _, ch := range attrs.AudioChannels {
			result.AudioChannels = append(result.AudioChannels, quality.NormalizeAudioChannels(ch))
		}

		hdrFormats := attrs.HDRFormats
		if len(hdrFormats) == 0 {
			hdrFormats = []string{"SDR"}
		}
		result.HDRFormats = hdrFormats

		result.ReleaseGroup = parseutil.ParseReleaseGroup(qualityText)
		result.Revision = parseutil.ParseRevision(qualityText)
		result.Edition = parseutil.ParseEdition(qualityText)
		result.Languages = parseutil.ParseLanguages(qualityText)
	}

	return result
}

// rssParserRegistry implements scanner.ModuleParserRegistry.
type rssParserRegistry struct{}

func (r *rssParserRegistry) AllFileParsers() []scanner.ModuleFileParser {
	return []scanner.ModuleFileParser{
		&rssTVParser{},
		&rssMovieParser{},
	}
}

func TestMain(m *testing.M) {
	scanner.SetGlobalRegistry(&rssParserRegistry{})
	os.Exit(m.Run())
}
