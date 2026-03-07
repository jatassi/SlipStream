package movie

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/module/parseutil"
)

var _ module.FileParser = (*fileParser)(nil)

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
	parsed := scanner.ParseFilename(filename)

	if parsed.IsTV {
		return nil, fmt.Errorf("filename %q detected as TV, not movie", filename)
	}

	return parsedMediaToResult(parsed), nil
}

func (p *fileParser) TryMatch(filename string) (confidence float64, match *module.ParseResult) {
	if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
		return 0, nil
	}

	parsed := scanner.ParseFilename(filename)

	if parsed.IsTV {
		return 0, nil
	}

	if parsed.Title != "" && parsed.Year > 0 {
		return 0.8, parsedMediaToResult(parsed)
	}

	if parsed.Title != "" {
		return 0.3, parsedMediaToResult(parsed)
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

func parsedMediaToResult(parsed *scanner.ParsedMedia) *module.ParseResult {
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
	}
}
