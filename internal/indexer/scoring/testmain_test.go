package scoring

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
	scoringTVPatternSE          = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?[.\s_-]*(.*)$`)
	scoringTVPatternX           = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(\d{1,2})[xX](\d{1,2})[.\s_-]*(.*)$`)
	scoringTVPatternSeasonPack  = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})(?:[.\s_-]|$)(.*)$`)
	scoringTVPatternSeasonRange = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})-s?(\d{1,2})[.\s_-]+(.*)$`)
	scoringTVPatternComplete    = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(?:complete[.\s_-]*(?:series)?|the[.\s_-]+complete[.\s_-]+series)[.\s_-]+(.*)$`)
)

// Movie patterns
var (
	scoringMoviePatternParen  = regexp.MustCompile(`^(.+?)\s*\((\d{4})\)\s*(.*)$`)
	scoringMoviePatternDot    = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})[.\s_-]+(.*)$`)
	scoringMoviePatternSimple = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})$`)
)

type scoringTVExtra struct {
	Season           int
	EndSeason        int
	Episode          int
	EndEpisode       int
	IsSeasonPack     bool
	IsCompleteSeries bool
}

func (e *scoringTVExtra) TVSeason() int            { return e.Season }
func (e *scoringTVExtra) TVEndSeason() int         { return e.EndSeason }
func (e *scoringTVExtra) TVEpisode() int           { return e.Episode }
func (e *scoringTVExtra) TVEndEpisode() int        { return e.EndEpisode }
func (e *scoringTVExtra) TVIsSeasonPack() bool     { return e.IsSeasonPack }
func (e *scoringTVExtra) TVIsCompleteSeries() bool { return e.IsCompleteSeries }

type scoringTVParser struct{}

func (p *scoringTVParser) ID() string { return "tv" }

func (p *scoringTVParser) TryMatch(filename string) (float64, *scanner.ParseResultData) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	result := scoringParseTVName(name)
	if result == nil {
		return 0, nil
	}

	extra := result.Extra.(*scoringTVExtra)
	if extra.Season > 0 && extra.Episode > 0 {
		return 0.9, result
	}
	if extra.IsSeasonPack {
		return 0.85, result
	}
	return 0.4, result
}

func scoringParseTVName(name string) *scanner.ParseResultData {
	if result := scoringTryTVEpisodePatterns(name); result != nil {
		return result
	}
	return scoringTryTVPackPatterns(name)
}

func scoringTryTVEpisodePatterns(name string) *scanner.ParseResultData {
	if match := scoringTVPatternSE.FindStringSubmatch(name); match != nil {
		extra := &scoringTVExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		if match[4] != "" {
			extra.EndEpisode, _ = strconv.Atoi(match[4])
		}
		return scoringBuildTVResult(match[1], match[5], extra)
	}

	if match := scoringTVPatternX.FindStringSubmatch(name); match != nil {
		extra := &scoringTVExtra{}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.Episode, _ = strconv.Atoi(match[3])
		return scoringBuildTVResult(match[1], match[4], extra)
	}

	return nil
}

func scoringTryTVPackPatterns(name string) *scanner.ParseResultData {
	if match := scoringTVPatternSeasonRange.FindStringSubmatch(name); match != nil {
		extra := &scoringTVExtra{IsSeasonPack: true, IsCompleteSeries: true}
		extra.Season, _ = strconv.Atoi(match[2])
		extra.EndSeason, _ = strconv.Atoi(match[3])
		return scoringBuildTVResult(match[1], match[4], extra)
	}

	if match := scoringTVPatternSeasonPack.FindStringSubmatch(name); match != nil {
		extra := &scoringTVExtra{IsSeasonPack: true}
		extra.Season, _ = strconv.Atoi(match[2])
		return scoringBuildTVResult(match[1], match[3], extra)
	}

	if match := scoringTVPatternComplete.FindStringSubmatch(name); match != nil {
		extra := &scoringTVExtra{IsSeasonPack: true, IsCompleteSeries: true}
		return scoringBuildTVResult(match[1], match[2], extra)
	}

	return nil
}

func scoringBuildTVResult(rawTitle, qualityText string, extra *scoringTVExtra) *scanner.ParseResultData {
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

type scoringMovieParser struct{}

func (p *scoringMovieParser) ID() string { return "movie" }

func (p *scoringMovieParser) TryMatch(filename string) (float64, *scanner.ParseResultData) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	result := scoringParseMovieName(name)
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

func scoringParseMovieName(name string) *scanner.ParseResultData {
	if match := scoringMoviePatternParen.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		return scoringBuildMovieResult(match[1], year, match[3])
	}

	if match := scoringMoviePatternDot.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			return scoringBuildMovieResult(match[1], year, match[3])
		}
	}

	if match := scoringMoviePatternSimple.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			return scoringBuildMovieResult(match[1], year, "")
		}
	}

	return nil
}

func scoringBuildMovieResult(rawTitle string, year int, qualityText string) *scanner.ParseResultData {
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

type scoringParserRegistry struct{}

func (r *scoringParserRegistry) AllFileParsers() []scanner.ModuleFileParser {
	return []scanner.ModuleFileParser{
		&scoringTVParser{},
		&scoringMovieParser{},
	}
}

func TestMain(m *testing.M) {
	scanner.SetGlobalRegistry(&scoringParserRegistry{})
	os.Exit(m.Run())
}
