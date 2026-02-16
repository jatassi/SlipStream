package rootfolder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/defaults"
	"github.com/slipstream/slipstream/internal/filesystem/mock"
)

const (
	mediaTypeMovie = "movie"
)

var (
	ErrRootFolderNotFound = errors.New("root folder not found")
	ErrPathNotFound       = errors.New("path does not exist")
	ErrPathNotDirectory   = errors.New("path is not a directory")
	ErrPathAlreadyExists  = errors.New("root folder path already exists")
	ErrInvalidMediaType   = errors.New("invalid media type (must be 'movie' or 'tv')")
)

// RootFolder represents a root folder for media storage.
type RootFolder struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	MediaType string    `json:"mediaType"` // "movie" or "tv"
	FreeSpace int64     `json:"freeSpace"`
	CreatedAt time.Time `json:"createdAt"`
	IsDefault bool      `json:"isDefault"`
}

// CreateRootFolderInput contains fields for creating a root folder.
type CreateRootFolderInput struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	MediaType string `json:"mediaType"`
}

// HealthService is the interface for central health tracking.
type HealthService interface {
	RegisterItemStr(category, id, name string)
	UnregisterItemStr(category, id string)
	SetErrorStr(category, id, message string)
	ClearStatusStr(category, id string)
}

// Service provides root folder operations.
type Service struct {
	db            *sql.DB
	queries       *sqlc.Queries
	logger        *zerolog.Logger
	defaults      *defaults.Service
	healthService HealthService
}

// NewService creates a new root folder service.
func NewService(db *sql.DB, logger *zerolog.Logger, defaultsSvc *defaults.Service) *Service {
	subLogger := logger.With().Str("component", "rootfolder").Logger()
	return &Service{
		db:       db,
		queries:  sqlc.New(db),
		logger:   &subLogger,
		defaults: defaultsSvc,
	}
}

// SetHealthService sets the central health service for registration tracking.
func (s *Service) SetHealthService(hs HealthService) {
	s.healthService = hs
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// RegisterExistingRootFolders registers all existing root folders with the health service.
func (s *Service) RegisterExistingRootFolders(ctx context.Context) error {
	if s.healthService == nil {
		return nil
	}

	folders, err := s.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list root folders for health registration: %w", err)
	}

	for _, folder := range folders {
		s.healthService.RegisterItemStr("rootFolders", fmt.Sprintf("%d", folder.ID), folder.Path)
	}

	s.logger.Info().Int("count", len(folders)).Msg("Registered existing root folders with health service")
	return nil
}

// Get retrieves a root folder by ID.
func (s *Service) Get(ctx context.Context, id int64) (*RootFolder, error) {
	row, err := s.queries.GetRootFolder(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRootFolderNotFound
		}
		return nil, fmt.Errorf("failed to get root folder: %w", err)
	}
	return s.rowToRootFolder(row), nil
}

// GetByPath retrieves a root folder by path.
func (s *Service) GetByPath(ctx context.Context, path string) (*RootFolder, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	row, err := s.queries.GetRootFolderByPath(ctx, absPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRootFolderNotFound
		}
		return nil, fmt.Errorf("failed to get root folder: %w", err)
	}
	return s.rowToRootFolder(row), nil
}

// List returns all root folders.
func (s *Service) List(ctx context.Context) ([]*RootFolder, error) {
	rows, err := s.queries.ListRootFolders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list root folders: %w", err)
	}

	folders := make([]*RootFolder, len(rows))
	for i, row := range rows {
		folders[i] = s.rowToRootFolderWithDefaults(ctx, row)
	}
	return folders, nil
}

// ListByType returns root folders of a specific media type.
func (s *Service) ListByType(ctx context.Context, mediaType string) ([]*RootFolder, error) {
	rows, err := s.queries.ListRootFoldersByType(ctx, mediaType)
	if err != nil {
		return nil, fmt.Errorf("failed to list root folders: %w", err)
	}

	folders := make([]*RootFolder, len(rows))
	for i, row := range rows {
		folders[i] = s.rowToRootFolderWithDefaults(ctx, row)
	}
	return folders, nil
}

// Create creates a new root folder.
func (s *Service) Create(ctx context.Context, input CreateRootFolderInput) (*RootFolder, error) {
	if input.MediaType != mediaTypeMovie && input.MediaType != "tv" {
		return nil, ErrInvalidMediaType
	}

	absPath, freeSpace, err := s.resolveAndValidatePath(input.Path)
	if err != nil {
		return nil, err
	}

	if err := s.checkPathNotDuplicate(ctx, absPath); err != nil {
		return nil, err
	}

	name := input.Name
	if name == "" {
		name = filepath.Base(absPath)
	}

	row, err := s.queries.CreateRootFolder(ctx, sqlc.CreateRootFolderParams{
		Path:      absPath,
		Name:      name,
		MediaType: input.MediaType,
		FreeSpace: sql.NullInt64{Int64: freeSpace, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create root folder: %w", err)
	}

	s.logger.Info().
		Int64("id", row.ID).
		Str("path", absPath).
		Str("mediaType", input.MediaType).
		Msg("Created root folder")

	if s.healthService != nil {
		s.healthService.RegisterItemStr("rootFolders", fmt.Sprintf("%d", row.ID), absPath)
	}

	return s.rowToRootFolder(row), nil
}

func (s *Service) resolveAndValidatePath(inputPath string) (absPath string, freeSpace int64, err error) {
	if mock.IsMockPath(inputPath) {
		return s.resolveMockPath(inputPath)
	}
	return s.resolveRealPath(inputPath)
}

func (s *Service) resolveMockPath(inputPath string) (absPath string, freeSpace int64, err error) {
	absPath = strings.TrimSuffix(inputPath, "/")
	vfs := mock.GetInstance()
	if !vfs.IsDirectory(absPath) {
		return "", 0, ErrPathNotFound
	}
	return absPath, 1000 * 1024 * 1024 * 1024, nil
}

func (s *Service) resolveRealPath(inputPath string) (absPath string, freeSpace int64, err error) {
	absPath, err = filepath.Abs(inputPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0, ErrPathNotFound
		}
		return "", 0, fmt.Errorf("failed to check path: %w", err)
	}
	if !info.IsDir() {
		return "", 0, ErrPathNotDirectory
	}

	return absPath, s.getFreeSpace(absPath), nil
}

func (s *Service) checkPathNotDuplicate(ctx context.Context, absPath string) error {
	_, err := s.queries.GetRootFolderByPath(ctx, absPath)
	if err == nil {
		return ErrPathAlreadyExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existing path: %w", err)
	}
	return nil
}

// Delete deletes a root folder and all associated movies/series.
func (s *Service) Delete(ctx context.Context, id int64) error {
	// Get folder first for logging
	folder, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Use a transaction to ensure atomic deletion
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	queries := s.queries.WithTx(tx)

	// Delete all movies in this root folder (cascade: files are deleted by FK constraint)
	if err := queries.DeleteMoviesByRootFolder(ctx, sql.NullInt64{Int64: id, Valid: true}); err != nil {
		return fmt.Errorf("failed to delete movies: %w", err)
	}

	// Delete all series in this root folder (cascade: seasons, episodes, files are deleted by FK constraint)
	if err := queries.DeleteSeriesByRootFolder(ctx, sql.NullInt64{Int64: id, Valid: true}); err != nil {
		return fmt.Errorf("failed to delete series: %w", err)
	}

	// Delete the root folder itself
	if err := queries.DeleteRootFolder(ctx, id); err != nil {
		return fmt.Errorf("failed to delete root folder: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info().
		Int64("id", id).
		Str("path", folder.Path).
		Msg("Deleted root folder and associated media")

	// Unregister from health service
	if s.healthService != nil {
		s.healthService.UnregisterItemStr("rootFolders", fmt.Sprintf("%d", id))
	}

	return nil
}

// UpdateFreeSpace updates the free space for a root folder.
func (s *Service) UpdateFreeSpace(ctx context.Context, id int64) error {
	folder, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	freeSpace := s.getFreeSpace(folder.Path)
	return s.queries.UpdateRootFolderFreeSpace(ctx, sqlc.UpdateRootFolderFreeSpaceParams{
		ID:        id,
		FreeSpace: sql.NullInt64{Int64: freeSpace, Valid: true},
	})
}

// RefreshAllFreeSpace updates free space for all root folders.
func (s *Service) RefreshAllFreeSpace(ctx context.Context) error {
	folders, err := s.List(ctx)
	if err != nil {
		return err
	}

	for _, folder := range folders {
		if err := s.UpdateFreeSpace(ctx, folder.ID); err != nil {
			s.logger.Warn().Err(err).Int64("id", folder.ID).Msg("Failed to update free space")
		}
	}
	return nil
}

// getFreeSpace returns the free space in bytes for a path.
func (s *Service) getFreeSpace(path string) int64 {
	// This is a simplified implementation
	// On Windows, we'd use syscall.GetDiskFreeSpaceEx
	// On Unix, we'd use syscall.Statfs
	// For now, return 0 to indicate unknown
	return 0
}

// rowToRootFolder converts a database row to a RootFolder.
func (s *Service) rowToRootFolder(row *sqlc.RootFolder) *RootFolder {
	rf := &RootFolder{
		ID:        row.ID,
		Path:      row.Path,
		Name:      row.Name,
		MediaType: row.MediaType,
	}
	if row.FreeSpace.Valid {
		rf.FreeSpace = row.FreeSpace.Int64
	}
	if row.CreatedAt.Valid {
		rf.CreatedAt = row.CreatedAt.Time
	}

	return rf
}

// rowToRootFolderWithDefaults converts a database row to a RootFolder with default info.
func (s *Service) rowToRootFolderWithDefaults(ctx context.Context, row *sqlc.RootFolder) *RootFolder {
	rf := s.rowToRootFolder(row)

	// Check if this is default for its media type
	if s.defaults != nil {
		mediaType := defaults.MediaType(row.MediaType)
		defaultEntry, err := s.defaults.GetDefault(ctx, defaults.EntityTypeRootFolder, mediaType)
		if err == nil && defaultEntry != nil && defaultEntry.EntityID == row.ID {
			rf.IsDefault = true
		}
	}

	return rf
}

// SetDefault sets a root folder as default for its media type
func (s *Service) SetDefault(ctx context.Context, id int64) error {
	// Get root folder to determine its media type
	folder, err := s.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get root folder: %w", err)
	}

	// Set default using defaults service
	var mediaType defaults.MediaType
	switch folder.MediaType {
	case mediaTypeMovie:
		mediaType = defaults.MediaTypeMovie
	case "tv":
		mediaType = defaults.MediaTypeTV
	default:
		return errors.New("invalid media type for default")
	}

	return s.defaults.SetDefault(ctx, defaults.EntityTypeRootFolder, mediaType, id)
}

// ClearDefault clears the default for the specified media type
func (s *Service) ClearDefault(ctx context.Context, mediaType string) error {
	var defaultMediaType defaults.MediaType
	switch mediaType {
	case mediaTypeMovie:
		defaultMediaType = defaults.MediaTypeMovie
	case "tv":
		defaultMediaType = defaults.MediaTypeTV
	default:
		return errors.New("invalid media type")
	}

	return s.defaults.ClearDefault(ctx, defaults.EntityTypeRootFolder, defaultMediaType)
}
