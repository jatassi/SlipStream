package mediainfo

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Config holds MediaInfo service configuration.
type Config struct {
	MediaInfoPath string        // Path to mediainfo binary (empty = search PATH)
	FFprobePath   string        // Path to ffprobe binary (empty = search PATH)
	CacheEnabled  bool          // Enable caching of probe results
	CacheTTL      time.Duration // How long to keep cached results
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
	}
}

// cacheEntry holds a cached MediaInfo result.
type cacheEntry struct {
	info      *MediaInfo
	timestamp time.Time
	size      int64
	modTime   time.Time
}

// Service provides media file information extraction.
type Service struct {
	config Config
	logger *zerolog.Logger
	cache  map[string]*cacheEntry
	mu     sync.RWMutex

	// Probe methods in priority order
	probeFunc func(ctx context.Context, path string) (*MediaInfo, error)
}

// NewService creates a new MediaInfo service.
func NewService(config Config, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "mediainfo").Logger()
	s := &Service{
		config: config,
		logger: &subLogger,
		cache:  make(map[string]*cacheEntry),
	}

	// Determine which probe method to use
	s.probeFunc = s.selectProbeMethod()

	return s
}

// selectProbeMethod determines the best available probe method.
func (s *Service) selectProbeMethod() func(context.Context, string) (*MediaInfo, error) {
	// Try mediainfo first
	if path := findExecutable("mediainfo", s.config.MediaInfoPath); path != "" {
		s.logger.Info().Str("path", path).Msg("Using mediainfo CLI")
		return func(ctx context.Context, p string) (*MediaInfo, error) {
			return s.probeWithMediaInfo(ctx, p, path)
		}
	}

	// Try ffprobe as fallback
	if path := findExecutable("ffprobe", s.config.FFprobePath); path != "" {
		s.logger.Info().Str("path", path).Msg("Using ffprobe CLI")
		return func(ctx context.Context, p string) (*MediaInfo, error) {
			return s.probeWithFFprobe(ctx, p, path)
		}
	}

	s.logger.Warn().Msg("No media probe tool found (mediainfo or ffprobe)")
	return nil
}

// Probe extracts media information from a file.
func (s *Service) Probe(ctx context.Context, path string) (*MediaInfo, error) {
	s.logger.Debug().Str("path", path).Msg("Probing media file")

	// Check cache first
	if s.config.CacheEnabled {
		if info := s.getCached(path); info != nil {
			s.logger.Debug().Str("path", path).Msg("Cache hit")
			return info, nil
		}
	}

	// No probe tool available
	if s.probeFunc == nil {
		s.logger.Debug().Str("path", path).Msg("No probe tool available, returning empty")
		return &MediaInfo{}, nil
	}

	// Probe the file
	info, err := s.probeFunc(ctx, path)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.config.CacheEnabled {
		s.setCache(path, info)
	}

	return info, nil
}

// ProbeWithFallback probes a file, falling back to parsed info on error.
func (s *Service) ProbeWithFallback(ctx context.Context, path string, fallback *MediaInfo) *MediaInfo {
	info, err := s.Probe(ctx, path)
	if err != nil {
		s.logger.Warn().Err(err).Str("path", path).Msg("Probe failed, using fallback")
		if fallback != nil {
			return fallback
		}
		return &MediaInfo{}
	}

	// Merge with fallback for missing fields
	if fallback != nil {
		mergeMediaInfo(info, fallback)
	}

	return info
}

// IsAvailable returns true if a probe tool is available.
func (s *Service) IsAvailable() bool {
	return s.probeFunc != nil
}

// getCached retrieves a cached result if valid.
func (s *Service) getCached(path string) *MediaInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.cache[path]
	if !ok {
		return nil
	}

	// Check TTL
	if time.Since(entry.timestamp) > s.config.CacheTTL {
		return nil
	}

	// Verify file hasn't changed
	stat, err := os.Stat(path)
	if err != nil {
		return nil
	}

	if stat.Size() != entry.size || stat.ModTime() != entry.modTime {
		return nil
	}

	return entry.info
}

// setCache stores a result in the cache.
func (s *Service) setCache(path string, info *MediaInfo) {
	stat, err := os.Stat(path)
	if err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[path] = &cacheEntry{
		info:      info,
		timestamp: time.Now(),
		size:      stat.Size(),
		modTime:   stat.ModTime(),
	}
}

// ClearCache clears all cached entries.
func (s *Service) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]*cacheEntry)
}

// mergeMediaInfo merges fallback values into info for empty fields.
func mergeMediaInfo(info, fallback *MediaInfo) {
	if info.VideoCodec == "" {
		info.VideoCodec = fallback.VideoCodec
	}
	if info.VideoBitDepth == 0 {
		info.VideoBitDepth = fallback.VideoBitDepth
	}
	if info.VideoResolution == "" {
		info.VideoResolution = fallback.VideoResolution
	}
	if info.DynamicRange == "" {
		info.DynamicRange = fallback.DynamicRange
	}
	if info.DynamicRangeType == "" {
		info.DynamicRangeType = fallback.DynamicRangeType
	}
	if info.AudioCodec == "" {
		info.AudioCodec = fallback.AudioCodec
	}
	if info.AudioChannels == "" {
		info.AudioChannels = fallback.AudioChannels
	}
	if len(info.AudioLanguages) == 0 {
		info.AudioLanguages = fallback.AudioLanguages
	}
	if len(info.SubtitleLanguages) == 0 {
		info.SubtitleLanguages = fallback.SubtitleLanguages
	}
}
