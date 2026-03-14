package importer

import "context"

const (
	mediaTypeMovie   = "movie"
	mediaTypeEpisode = "episode"
	mediaSeason      = "season"
	mediaSeries      = "series"
	fileStatusFailed = "failed"
)

// matchToLibrary attempts to match a file to a library item using the module system.
func (s *Service) matchToLibrary(ctx context.Context, path string, mapping *DownloadMapping) (*LibraryMatch, error) {
	// When we have a download mapping with module info, use the module's ImportHandler
	if mapping != nil && mapping.ModuleType != "" {
		match, err := s.matchToLibraryViaModule(ctx, path, mapping)
		if err == nil {
			return match, nil
		}
		s.logger.Debug().Err(err).Str("path", path).Msg("Module matching via download mapping failed")
	}

	// Parse-based matching: try to identify the file via registered module file parsers
	match := s.matchFromParse(ctx, path)
	if match != nil {
		return match, nil
	}

	return nil, ErrNoMatch
}

// matchFromParse attempts to match a file by parsing its filename via module file parsers.
func (s *Service) matchFromParse(ctx context.Context, path string) *LibraryMatch {
	entity := s.matchOrphanViaModules(ctx, path)
	if entity != nil {
		return moduleEntityToLibraryMatch(entity)
	}
	return nil
}
