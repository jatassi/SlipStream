package movie

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/module/parseutil"
)

var _ module.FileParser = (*fileParser)(nil)

// Movie-specific regex patterns for filename parsing.
var (
	moviePatternParen  = regexp.MustCompile(`^(.+?)\s*\((\d{4})\)\s*(.*)$`)
	moviePatternDot    = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})[.\s_-]+(.*)$`)
	moviePatternSimple = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})$`)
)

type fileParser struct {
	movieSvc      *movies.Service
	rootFolderSvc *rootfolder.Service
	logger        zerolog.Logger
}

func newFileParser(movieSvc *movies.Service, rootFolderSvc *rootfolder.Service, logger *zerolog.Logger) *fileParser {
	return &fileParser{
		movieSvc:      movieSvc,
		rootFolderSvc: rootFolderSvc,
		logger:        logger.With().Str("component", "movie-fileparser").Logger(),
	}
}

func (p *fileParser) ParseFilename(filename string) (*module.ParseResult, error) {
	result := parseMovieFilename(filename)
	if result == nil {
		return nil, fmt.Errorf("filename %q not detected as movie", filename)
	}
	return result, nil
}

func (p *fileParser) TryMatch(filename string) (confidence float64, match *module.ParseResult) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	result := parseMovieName(name)
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

func (p *fileParser) MatchToEntity(ctx context.Context, parseResult *module.ParseResult) (*module.MatchedEntity, error) {
	movie, err := p.movieSvc.FindByTitleAndYear(ctx, parseResult.Title, parseResult.Year)
	if err != nil {
		return nil, err
	}
	if movie == nil {
		return nil, nil //nolint:nilnil // nil entity with no error means "no match found"
	}

	rf, err := p.rootFolderSvc.Get(ctx, movie.RootFolderID)
	if err != nil {
		return nil, fmt.Errorf("root folder %d not found: %w", movie.RootFolderID, err)
	}

	return &module.MatchedEntity{
		ModuleType:       module.TypeMovie,
		EntityType:       module.EntityMovie,
		EntityID:         movie.ID,
		Title:            movie.Title,
		RootFolder:       rf.Path,
		Confidence:       0.8,
		Source:           "parse",
		QualityProfileID: movie.QualityProfileID,
		TokenData: map[string]any{
			"MovieTitle": movie.Title,
			"MovieYear":  movie.Year,
		},
	}, nil
}

// parseMovieFilename parses a movie filename (with extension) into a ParseResult.
// Returns nil if the filename doesn't match any movie pattern.
func parseMovieFilename(filename string) *module.ParseResult {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return parseMovieName(name)
}

// parseMovieName parses a media name (without extension) trying movie patterns.
// Returns nil if no movie pattern matches.
func parseMovieName(name string) *module.ParseResult {
	if match := moviePatternParen.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		return buildMovieParseResult(match[1], year, match[3])
	}

	if match := moviePatternDot.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			return buildMovieParseResult(match[1], year, match[3])
		}
	}

	if match := moviePatternSimple.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			return buildMovieParseResult(match[1], year, "")
		}
	}

	return nil
}

func buildMovieParseResult(rawTitle string, year int, qualityText string) *module.ParseResult {
	result := &module.ParseResult{
		Title: parseutil.CleanTitle(rawTitle),
		Year:  year,
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
