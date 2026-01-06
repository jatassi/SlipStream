package rootfolder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/defaults"
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

// Service provides root folder operations.
type Service struct {
	db       *sql.DB
	queries  *sqlc.Queries
	logger   zerolog.Logger
	defaults *defaults.Service
}

// NewService creates a new root folder service.
func NewService(db *sql.DB, logger zerolog.Logger, defaults *defaults.Service) *Service {
	return &Service{
		db:       db,
		queries:  sqlc.New(db),
		logger:   logger.With().Str("component", "rootfolder").Logger(),
		defaults: defaults,
	}
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
	// Validate media type
	if input.MediaType != "movie" && input.MediaType != "tv" {
		return nil, ErrInvalidMediaType
	}

	// Get absolute path
	absPath, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Verify path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to check path: %w", err)
	}
	if !info.IsDir() {
		return nil, ErrPathNotDirectory
	}

	// Check if path already exists
	_, err = s.queries.GetRootFolderByPath(ctx, absPath)
	if err == nil {
		return nil, ErrPathAlreadyExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing path: %w", err)
	}

	// Get free space
	freeSpace := s.getFreeSpace(absPath)

	// Determine name
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

	return s.rowToRootFolder(row), nil
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
	defer tx.Rollback()

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
	case "movie":
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
	case "movie":
		defaultMediaType = defaults.MediaTypeMovie
	case "tv":
		defaultMediaType = defaults.MediaTypeTV
	default:
		return errors.New("invalid media type")
	}

	return s.defaults.ClearDefault(ctx, defaults.EntityTypeRootFolder, defaultMediaType)
}
