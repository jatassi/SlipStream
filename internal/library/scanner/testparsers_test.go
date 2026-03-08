package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/module/parseutil"
)

// Test-only module file parsers that replicate the patterns from the module packages.
// These are used to set up the global registry for scanner unit tests.

// TV patterns (same as internal/modules/tv/fileparser.go)
var (
	testTVPatternSE                   = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?[.\s_-]*(.*)$`)
	testTVPatternX                    = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(\d{1,2})[xX](\d{1,2})[.\s_-]*(.*)$`)
	testTVPatternSeasonEpisodeSpelled = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss]eason[.\s_-]+(\d{1,2})[.\s_-]+[Ee]pisode[.\s_-]+(\d{1,2})[.\s_-]*(.*)$`)
	testTVPatternSeasonPack           = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})(?:[.\s_-]|$)(.*)$`)
	testTVPatternSeasonSpelled        = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss]eason[.\s_-]+(\d{1,2})(?:[.\s_-]|$)(.*)$`)
	testTVPatternSeasonRange          = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})-s?(\d{1,2})[.\s_-]+(.*)$`)
	testTVPatternComplete             = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(?:complete[.\s_-]*(?:series)?|the[.\s_-]+complete[.\s_-]+series)[.\s_-]+(.*)$`)
)

// Movie patterns (same as internal/modules/movie/fileparser.go)
var (
	testMoviePatternParen  = regexp.MustCompile(`^(.+?)\s*\((\d{4})\)\s*(.*)$`)
	testMoviePatternDot    = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})[.\s_-]+(.*)$`)
	testMoviePatternSimple = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})$`)
)

// testTVExtra implements tvExtraAccessor for test TV parse results.
type testTVExtra struct {
	Season           int
	EndSeason        int
	Episode          int
	EndEpisode       int
	IsSeasonPack     bool
	IsCompleteSeries bool
}

func (e *testTVExtra) TVSeason() int            { return e.Season }
func (e *testTVExtra) TVEndSeason() int         { return e.EndSeason }
func (e *testTVExtra) TVEpisode() int           { return e.Episode }
func (e *testTVExtra) TVEndEpisode() int        { return e.EndEpisode }
func (e *testTVExtra) TVIsSeasonPack() bool     { return e.IsSeasonPack }
func (e *testTVExtra) TVIsCompleteSeries() bool { return e.IsCompleteSeries }

// testTVParser implements ModuleFileParser for TV parsing in tests.
type testTVParser struct{}

func (p *testTVParser) ID() string { return "tv" }

func (p *testTVParser) TryMatch(filename string) (float64, *ParseResultData) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	result := testParseTVName(name)
	if result == nil {
		return 0, nil
	}

	extra := result.Extra.(*testTVExtra)
	if extra.Season > 0 && extra.Episode > 0 {
		return 0.9, result
	}
	if extra.IsSeasonPack {
		return 0.85, result
	}
	return 0.4, result
}

func testParseTVName(name string) *ParseResultData {
	if result := testTryTVEpisodePatterns(name); result != nil {
		return result
	}
	return testTryTVPackPatterns(name)
}

func testTryTVEpisodePatterns(name string) *ParseResultData {
	if match := testTVPatternSE.FindStringSubmatch(name); match != nil {
		extra := &testTVExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		if match[4] != "" {
			extra.EndEpisode, _ = strconv.Atoi(match[4])
		}
		return testBuildTVResult(match[1], match[5], extra)
	}

	if match := testTVPatternX.FindStringSubmatch(name); match != nil {
		extra := &testTVExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		return testBuildTVResult(match[1], match[4], extra)
	}

	if match := testTVPatternSeasonEpisodeSpelled.FindStringSubmatch(name); match != nil {
		extra := &testTVExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		return testBuildTVResult(match[1], match[4], extra)
	}

	return nil
}

func testTryTVPackPatterns(name string) *ParseResultData {
	if match := testTVPatternSeasonRange.FindStringSubmatch(name); match != nil {
		extra := &testTVExtra{IsSeasonPack: true, IsCompleteSeries: true}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.EndSeason, _ = strconv.Atoi(match[3])
		return testBuildTVResult(match[1], match[4], extra)
	}

	if match := testTVPatternSeasonPack.FindStringSubmatch(name); match != nil {
		extra := &testTVExtra{IsSeasonPack: true}
		extra.Season, _ = strconv.Atoi(match[2])
		return testBuildTVResult(match[1], match[3], extra)
	}

	if match := testTVPatternSeasonSpelled.FindStringSubmatch(name); match != nil {
		extra := &testTVExtra{IsSeasonPack: true}
		extra.Season, _ = strconv.Atoi(match[2])
		return testBuildTVResult(match[1], match[3], extra)
	}

	if match := testTVPatternComplete.FindStringSubmatch(name); match != nil {
		extra := &testTVExtra{IsSeasonPack: true, IsCompleteSeries: true}
		return testBuildTVResult(match[1], match[2], extra)
	}

	return nil
}

func testBuildTVResult(rawTitle, qualityText string, extra *testTVExtra) *ParseResultData {
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

	return &ParseResultData{
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

// testMovieParser implements ModuleFileParser for movie parsing in tests.
type testMovieParser struct{}

func (p *testMovieParser) ID() string { return "movie" }

func (p *testMovieParser) TryMatch(filename string) (float64, *ParseResultData) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	result := testParseMovieName(name)
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

func testParseMovieName(name string) *ParseResultData {
	if match := testMoviePatternParen.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		return testBuildMovieResult(match[1], year, match[3])
	}

	if match := testMoviePatternDot.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			return testBuildMovieResult(match[1], year, match[3])
		}
	}

	if match := testMoviePatternSimple.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			return testBuildMovieResult(match[1], year, "")
		}
	}

	return nil
}

func testBuildMovieResult(rawTitle string, year int, qualityText string) *ParseResultData {
	result := &ParseResultData{
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

// testParserRegistry implements ModuleParserRegistry for tests.
type testParserRegistry struct{}

func (r *testParserRegistry) AllFileParsers() []ModuleFileParser {
	return []ModuleFileParser{
		&testTVParser{},
		&testMovieParser{},
	}
}

func TestMain(m *testing.M) {
	SetGlobalRegistry(&testParserRegistry{})
	os.Exit(m.Run())
}
