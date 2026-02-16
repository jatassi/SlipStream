package librarymanager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/preferences"
	"github.com/slipstream/slipstream/internal/progress"

	"database/sql"
)

const (
	mediaTypeMovie = "movie"
)

// HealthService defines the interface for health tracking.
type HealthService interface {
	SetWarningStr(category, id, message string)
	ClearStatusStr(category, id string)
}

var (
	ErrNoMetadataProvider = errors.New("no metadata provider configured")
	ErrNoQualityProfile   = errors.New("no quality profile available")
	ErrScanInProgress     = errors.New("scan already in progress for this folder")
)

// ScanResult represents the final result of a scan operation.
type ScanResult struct {
	RootFolderID    int64    `json:"rootFolderId"`
	TotalFiles      int      `json:"totalFiles"`
	MoviesAdded     int      `json:"moviesAdded"`
	SeriesAdded     int      `json:"seriesAdded"`
	FilesLinked     int      `json:"filesLinked"`
	MetadataMatched int      `json:"metadataMatched"`
	ArtworksFetched int      `json:"artworksFetched"`
	Errors          []string `json:"errors,omitempty"`
}

// pendingArtwork tracks items that need artwork downloaded.
type pendingArtwork struct {
	movieMeta  []*metadata.MovieResult
	seriesMeta []*metadata.SeriesResult
}

// Service orchestrates library scanning, file matching, and metadata lookup.
type Service struct {
	db              *sql.DB
	queries         *sqlc.Queries
	scanner         *scanner.Service
	movies          *movies.Service
	tv              *tv.Service
	metadata        *metadata.Service
	artwork         *metadata.ArtworkDownloader
	rootfolders     *rootfolder.Service
	qualityProfiles *quality.Service
	progress        *progress.Manager
	logger          *zerolog.Logger

	// Optional services for search-on-add
	autosearchSvc  *autosearch.Service
	preferencesSvc *preferences.Service

	// Optional slots service for multi-version support
	slotsSvc *slots.Service

	// Optional health service for file verification alerts
	healthSvc HealthService

	// Track active scans by root folder ID
	activeScans map[int64]string // maps folderID -> activityID
	scanMu      sync.RWMutex
}

// NewService creates a new library manager service.
func NewService(
	db *sql.DB,
	scannerSvc *scanner.Service,
	moviesSvc *movies.Service,
	tvSvc *tv.Service,
	metadataSvc *metadata.Service,
	artworkSvc *metadata.ArtworkDownloader,
	rootfolderSvc *rootfolder.Service,
	qualityProfileSvc *quality.Service,
	progressMgr *progress.Manager,
	logger *zerolog.Logger,
) *Service {
	subLogger := logger.With().Str("component", "librarymanager").Logger()
	return &Service{
		db:              db,
		queries:         sqlc.New(db),
		scanner:         scannerSvc,
		movies:          moviesSvc,
		tv:              tvSvc,
		metadata:        metadataSvc,
		artwork:         artworkSvc,
		rootfolders:     rootfolderSvc,
		qualityProfiles: qualityProfileSvc,
		progress:        progressMgr,
		logger:          &subLogger,
		activeScans:     make(map[int64]string),
	}
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// SetAutosearchService sets the optional autosearch service for search-on-add functionality
func (s *Service) SetAutosearchService(svc *autosearch.Service) {
	s.autosearchSvc = svc
}

// SetPreferencesService sets the optional preferences service for add-flow defaults
func (s *Service) SetPreferencesService(svc *preferences.Service) {
	s.preferencesSvc = svc
}

// SetSlotsService sets the optional slots service for multi-version slot assignment.
// Req 13.1.2: Auto-assign files to best matching slot based on quality profile matching
func (s *Service) SetSlotsService(svc *slots.Service) {
	s.slotsSvc = svc
}

// SetHealthService sets the optional health service for file verification alerts.
func (s *Service) SetHealthService(svc HealthService) {
	s.healthSvc = svc
}

// getDefaultQualityProfile returns the first available quality profile.
func (s *Service) getDefaultQualityProfile(ctx context.Context) (*quality.Profile, error) {
	profiles, err := s.qualityProfiles.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, ErrNoQualityProfile
	}
	return profiles[0], nil
}

// IsScanActive returns true if a scan is active for the given folder.
func (s *Service) IsScanActive(rootFolderID int64) bool {
	return s.isScanActive(rootFolderID)
}

// GetActiveScanActivity returns the activity ID for an active scan, or empty string if none.
func (s *Service) GetActiveScanActivity(rootFolderID int64) string {
	s.scanMu.RLock()
	defer s.scanMu.RUnlock()
	return s.activeScans[rootFolderID]
}

// CancelScan cancels an active scan.
func (s *Service) CancelScan(rootFolderID int64) bool {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	activityID, exists := s.activeScans[rootFolderID]
	if !exists {
		return false
	}

	if s.progress != nil {
		s.progress.CancelActivity(activityID)
	}
	delete(s.activeScans, rootFolderID)
	return true
}

func (s *Service) isScanActive(rootFolderID int64) bool {
	s.scanMu.RLock()
	defer s.scanMu.RUnlock()
	_, exists := s.activeScans[rootFolderID]
	return exists
}

func (s *Service) setScanActive(rootFolderID int64, activityID string) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	s.activeScans[rootFolderID] = activityID
}

func (s *Service) clearScanActive(rootFolderID int64) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	delete(s.activeScans, rootFolderID)
}

// buildScanSummary creates a human-readable summary of scan results.
func (s *Service) buildScanSummary(result *ScanResult) string {
	var parts []string

	if result.MoviesAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d movies added", result.MoviesAdded))
	}
	if result.SeriesAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d series added", result.SeriesAdded))
	}
	if result.MetadataMatched > 0 {
		parts = append(parts, fmt.Sprintf("%d matched", result.MetadataMatched))
	}
	if result.ArtworksFetched > 0 {
		parts = append(parts, fmt.Sprintf("%d artworks", result.ArtworksFetched))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Found %d files", result.TotalFiles)
	}

	return strings.Join(parts, ", ")
}

// normalizeTitle removes punctuation and extra whitespace from a title for comparison.
// This helps match "Top Gun: Maverick" to "Top Gun Maverick".
func normalizeTitle(title string) string {
	// Convert to lowercase
	result := strings.ToLower(title)

	// Replace common punctuation with space
	replacer := strings.NewReplacer(
		":", " ",
		"-", " ",
		"'", "",
		"\u2019", "",
		",", "",
		".", "",
		"!", "",
		"?", "",
		"&", "and",
	)
	result = replacer.Replace(result)

	// Collapse multiple spaces to single space
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	// Trim whitespace
	result = strings.TrimSpace(result)

	return result
}
