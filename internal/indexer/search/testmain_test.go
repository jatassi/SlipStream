package search

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
	searchTVPatternSE          = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?[.\s_-]*(.*)$`)
	searchTVPatternX           = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(\d{1,2})[xX](\d{1,2})[.\s_-]*(.*)$`)
	searchTVPatternSeasonPack  = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})(?:[.\s_-]|$)(.*)$`)
	searchTVPatternSeasonRange = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})-s?(\d{1,2})[.\s_-]+(.*)$`)
	searchTVPatternComplete    = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(?:complete[.\s_-]*(?:series)?|the[.\s_-]+complete[.\s_-]+series)[.\s_-]+(.*)$`)
)

// Movie patterns
var (
	searchMoviePatternParen  = regexp.MustCompile(`^(.+?)\s*\((\d{4})\)\s*(.*)$`)
	searchMoviePatternDot    = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})[.\s_-]+(.*)$`)
	searchMoviePatternSimple = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})$`)
)

type searchTVExtra struct {
	Season           int
	EndSeason        int
	Episode          int
	EndEpisode       int
	IsSeasonPack     bool
	IsCompleteSeries bool
}

func (e *searchTVExtra) TVSeason() int            { return e.Season }
func (e *searchTVExtra) TVEndSeason() int         { return e.EndSeason }
func (e *searchTVExtra) TVEpisode() int           { return e.Episode }
func (e *searchTVExtra) TVEndEpisode() int        { return e.EndEpisode }
func (e *searchTVExtra) TVIsSeasonPack() bool     { return e.IsSeasonPack }
func (e *searchTVExtra) TVIsCompleteSeries() bool { return e.IsCompleteSeries }

type searchTVParser struct{}

func (p *searchTVParser) ID() string { return "tv" }

func (p *searchTVParser) TryMatch(filename string) (float64, *scanner.ParseResultData) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	result := searchParseTVName(name)
	if result == nil {
		return 0, nil
	}

	extra := result.Extra.(*searchTVExtra)
	if extra.Season > 0 && extra.Episode > 0 {
		return 0.9, result
	}
	if extra.IsSeasonPack {
		return 0.85, result
	}
	return 0.4, result
}

func searchParseTVName(name string) *scanner.ParseResultData {
	if result := searchTryTVEpisodePatterns(name); result != nil {
		return result
	}
	return searchTryTVPackPatterns(name)
}

func searchTryTVEpisodePatterns(name string) *scanner.ParseResultData {
	if match := searchTVPatternSE.FindStringSubmatch(name); match != nil {
		extra := &searchTVExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		if match[4] != "" {
			extra.EndEpisode, _ = strconv.Atoi(match[4])
		}
		return searchBuildTVResult(match[1], match[5], extra)
	}

	if match := searchTVPatternX.FindStringSubmatch(name); match != nil {
		extra := &searchTVExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		return searchBuildTVResult(match[1], match[4], extra)
	}

	return nil
}

func searchTryTVPackPatterns(name string) *scanner.ParseResultData {
	if match := searchTVPatternSeasonRange.FindStringSubmatch(name); match != nil {
		extra := &searchTVExtra{IsSeasonPack: true, IsCompleteSeries: true}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.EndSeason, _ = strconv.Atoi(match[3])
		return searchBuildTVResult(match[1], match[4], extra)
	}

	if match := searchTVPatternSeasonPack.FindStringSubmatch(name); match != nil {
		extra := &searchTVExtra{IsSeasonPack: true}
		extra.Season, _ = strconv.Atoi(match[2])
		return searchBuildTVResult(match[1], match[3], extra)
	}

	if match := searchTVPatternComplete.FindStringSubmatch(name); match != nil {
		extra := &searchTVExtra{IsSeasonPack: true, IsCompleteSeries: true}
		return searchBuildTVResult(match[1], match[2], extra)
	}

	return nil
}

func searchBuildTVResult(rawTitle, qualityText string, extra *searchTVExtra) *scanner.ParseResultData {
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

type searchMovieParser struct{}

func (p *searchMovieParser) ID() string { return "movie" }

func (p *searchMovieParser) TryMatch(filename string) (float64, *scanner.ParseResultData) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	result := searchParseMovieName(name)
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

func searchParseMovieName(name string) *scanner.ParseResultData {
	if match := searchMoviePatternParen.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		return searchBuildMovieResult(match[1], year, match[3])
	}

	if match := searchMoviePatternDot.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			return searchBuildMovieResult(match[1], year, match[3])
		}
	}

	if match := searchMoviePatternSimple.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			return searchBuildMovieResult(match[1], year, "")
		}
	}

	return nil
}

func searchBuildMovieResult(rawTitle string, year int, qualityText string) *scanner.ParseResultData {
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

type searchParserRegistry struct{}

func (r *searchParserRegistry) AllFileParsers() []scanner.ModuleFileParser {
	return []scanner.ModuleFileParser{
		&searchTVParser{},
		&searchMovieParser{},
	}
}

func TestMain(m *testing.M) {
	scanner.SetGlobalRegistry(&searchParserRegistry{})
	os.Exit(m.Run())
}
