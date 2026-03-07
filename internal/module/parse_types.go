package module

// ParseResult is the result of parsing a filename.
// Common fields are populated by all modules. Module-specific data goes in Extra.
type ParseResult struct {
	Title         string   // Cleaned media title
	Year          int      // Release year (0 if not detected)
	Quality       string   // "2160p", "1080p", "720p", "480p", ""
	Source        string   // "BluRay", "WEBDL", "WEBRip", "HDTV", "Remux", etc.
	Codec         string   // "x265", "x264", "AV1", etc.
	HDRFormats    []string // "DV", "HDR10+", "HDR10", "HDR", "HLG", "SDR"
	AudioCodecs   []string // Normalized: "TrueHD", "DTS-HD MA", "DDP", "DD", "AAC", etc.
	AudioChannels []string // "7.1", "5.1", "2.0"
	ReleaseGroup  string   // e.g., "SPARKS", "NTb"
	Revision      string   // "Proper", "REPACK", "REAL", "RERIP"
	Edition       string   // "Director's Cut", "Extended", etc.
	Languages     []string // Non-English languages detected

	// Extra carries module-specific parsed data.
	// Movie module: nil (all data in common fields).
	// TV module: *TVParseExtra (defined in internal/modules/tv/).
	Extra any
}

// MonitoringPreset defines a monitoring strategy.
type MonitoringPreset struct {
	ID          string
	Label       string
	Description string
	HasOptions  bool
}
