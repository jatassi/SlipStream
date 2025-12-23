package watcher

import (
	"context"
	"sync"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/rootfolder"
)

// FileProcessor is called to process detected file events.
type FileProcessor func(ctx context.Context, filePath string) error

// Service manages file watching for all root folders.
type Service struct {
	watcher        *Watcher
	rootfolders    *rootfolder.Service
	fileProcessor  FileProcessor
	logger         zerolog.Logger

	// Track which root folders are being watched
	watchedFolders map[int64]string
	mu             sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

// NewService creates a new watcher service.
func NewService(
	rootfolderSvc *rootfolder.Service,
	logger zerolog.Logger,
) (*Service, error) {
	config := DefaultConfig()
	watcher, err := New(config, logger)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Service{
		watcher:        watcher,
		rootfolders:    rootfolderSvc,
		logger:         logger.With().Str("component", "watcher-service").Logger(),
		watchedFolders: make(map[int64]string),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Set up event handler
	watcher.SetHandler(s.handleEvents)

	return s, nil
}

// SetFileProcessor sets the function that processes file events.
func (s *Service) SetFileProcessor(processor FileProcessor) {
	s.fileProcessor = processor
}

// Start begins watching all configured root folders.
func (s *Service) Start(ctx context.Context) error {
	// Get all root folders
	folders, err := s.rootfolders.List(ctx)
	if err != nil {
		return err
	}

	// Add watches for each folder
	for _, folder := range folders {
		if err := s.WatchFolder(folder.ID, folder.Path); err != nil {
			s.logger.Warn().Err(err).Int64("folderId", folder.ID).Str("path", folder.Path).Msg("Failed to watch folder")
		}
	}

	// Start the watcher
	s.watcher.Start()

	s.logger.Info().Int("folderCount", len(folders)).Msg("Watcher service started")
	return nil
}

// Stop stops the watcher service.
func (s *Service) Stop() error {
	s.cancel()
	return s.watcher.Stop()
}

// WatchFolder adds a root folder to the watch list.
func (s *Service) WatchFolder(folderID int64, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.watchedFolders[folderID]; exists {
		return nil // Already watching
	}

	if err := s.watcher.AddPath(path); err != nil {
		return err
	}

	s.watchedFolders[folderID] = path
	s.logger.Info().Int64("folderId", folderID).Str("path", path).Msg("Started watching folder")
	return nil
}

// UnwatchFolder removes a root folder from the watch list.
func (s *Service) UnwatchFolder(folderID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, exists := s.watchedFolders[folderID]
	if !exists {
		return nil // Not watching
	}

	if err := s.watcher.RemovePath(path); err != nil {
		return err
	}

	delete(s.watchedFolders, folderID)
	s.logger.Info().Int64("folderId", folderID).Str("path", path).Msg("Stopped watching folder")
	return nil
}

// IsWatching returns true if a folder is being watched.
func (s *Service) IsWatching(folderID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.watchedFolders[folderID]
	return exists
}

// WatchedFolderIDs returns the IDs of all watched folders.
func (s *Service) WatchedFolderIDs() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]int64, 0, len(s.watchedFolders))
	for id := range s.watchedFolders {
		ids = append(ids, id)
	}
	return ids
}

// handleEvents processes batched file events.
func (s *Service) handleEvents(events []FileEvent) {
	if s.fileProcessor == nil {
		s.logger.Warn().Int("count", len(events)).Msg("No file processor configured, ignoring events")
		return
	}

	for _, event := range events {
		// Only process create and write events (new or modified files)
		if event.Op != "create" && event.Op != "write" {
			continue
		}

		s.logger.Debug().
			Str("path", event.Path).
			Str("op", event.Op).
			Msg("Processing file event")

		if err := s.fileProcessor(s.ctx, event.Path); err != nil {
			s.logger.Warn().Err(err).Str("path", event.Path).Msg("Failed to process file event")
		}
	}
}

// RefreshWatches reloads the watch list from root folders.
func (s *Service) RefreshWatches(ctx context.Context) error {
	folders, err := s.rootfolders.List(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Build set of current folder IDs
	currentIDs := make(map[int64]bool)
	for _, folder := range folders {
		currentIDs[folder.ID] = true
	}

	// Remove watches for deleted folders
	for id, path := range s.watchedFolders {
		if !currentIDs[id] {
			s.watcher.RemovePath(path)
			delete(s.watchedFolders, id)
			s.logger.Info().Int64("folderId", id).Msg("Removed watch for deleted folder")
		}
	}

	// Add watches for new folders
	for _, folder := range folders {
		if _, exists := s.watchedFolders[folder.ID]; !exists {
			if err := s.watcher.AddPath(folder.Path); err != nil {
				s.logger.Warn().Err(err).Int64("folderId", folder.ID).Msg("Failed to add watch")
				continue
			}
			s.watchedFolders[folder.ID] = folder.Path
			s.logger.Info().Int64("folderId", folder.ID).Msg("Added watch for new folder")
		}
	}

	return nil
}
