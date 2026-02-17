package downloader

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader/transmission"
	"github.com/slipstream/slipstream/internal/downloader/types"
)

var (
	ErrClientNotFound    = errors.New("download client not found")
	ErrInvalidClient     = errors.New("invalid download client")
	ErrUnsupportedClient = errors.New("unsupported client type")
)

// DownloadClient represents a download client configuration.
type DownloadClient struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Type               string    `json:"type"`
	Host               string    `json:"host"`
	Port               int       `json:"port"`
	Username           string    `json:"username,omitempty"`
	Password           string    `json:"password,omitempty"`
	UseSSL             bool      `json:"useSsl"`
	APIKey             string    `json:"apiKey,omitempty"`
	Category           string    `json:"category,omitempty"`
	URLBase            string    `json:"urlBase,omitempty"`
	Priority           int       `json:"priority"`
	Enabled            bool      `json:"enabled"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	ImportDelaySeconds int       `json:"importDelaySeconds"`
	CleanupMode        string    `json:"cleanupMode"` // "leave", "delete_after_import", "delete_after_seed_ratio"
	SeedRatioTarget    *float64  `json:"seedRatioTarget,omitempty"`
}

// CreateClientInput represents the input for creating a download client.
type CreateClientInput struct {
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	Host               string   `json:"host"`
	Port               int      `json:"port"`
	Username           string   `json:"username,omitempty"`
	Password           string   `json:"password,omitempty"`
	UseSSL             bool     `json:"useSsl"`
	APIKey             string   `json:"apiKey,omitempty"`
	Category           string   `json:"category,omitempty"`
	URLBase            string   `json:"urlBase,omitempty"`
	Priority           int      `json:"priority"`
	Enabled            bool     `json:"enabled"`
	ImportDelaySeconds int      `json:"importDelaySeconds"`
	CleanupMode        string   `json:"cleanupMode"` // "leave", "delete_after_import", "delete_after_seed_ratio"
	SeedRatioTarget    *float64 `json:"seedRatioTarget,omitempty"`
}

// UpdateClientInput represents the input for updating a download client.
type UpdateClientInput struct {
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	Host               string   `json:"host"`
	Port               int      `json:"port"`
	Username           string   `json:"username,omitempty"`
	Password           string   `json:"password,omitempty"`
	UseSSL             bool     `json:"useSsl"`
	APIKey             string   `json:"apiKey,omitempty"`
	Category           string   `json:"category,omitempty"`
	URLBase            string   `json:"urlBase,omitempty"`
	Priority           int      `json:"priority"`
	Enabled            bool     `json:"enabled"`
	ImportDelaySeconds int      `json:"importDelaySeconds"`
	CleanupMode        string   `json:"cleanupMode"` // "leave", "delete_after_import", "delete_after_seed_ratio"
	SeedRatioTarget    *float64 `json:"seedRatioTarget,omitempty"`
}

// TestResult represents the result of testing a download client connection.
type TestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// HealthService defines the interface for health tracking.
// Uses the string-based wrapper methods to avoid importing health types.
type HealthService interface {
	RegisterItemStr(category, id, name string)
	UnregisterItemStr(category, id string)
	SetErrorStr(category, id, message string)
	ClearStatusStr(category, id string)
}

// StatusChangeLogger logs status transition history events.
type StatusChangeLogger interface {
	LogStatusChanged(ctx context.Context, mediaType string, mediaID int64, from, to, reason string) error
}

// PortalStatusTracker tracks download status for portal request mirroring.
type PortalStatusTracker interface {
	OnDownloadFailed(ctx context.Context, mediaType string, mediaID int64) error
}

// Service provides download client operations.
type Service struct {
	queries             *sqlc.Queries
	logger              *zerolog.Logger
	healthService       HealthService
	broadcaster         Broadcaster
	statusChangeLogger  StatusChangeLogger
	portalStatusTracker PortalStatusTracker

	queueCacheMu sync.RWMutex
	queueCache   map[int64][]QueueItem

	clientPoolMu sync.RWMutex
	clientPool   map[int64]Client
}

// NewService creates a new download client service.
func NewService(db *sql.DB, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "downloader").Logger()
	return &Service{
		queries:    sqlc.New(db),
		logger:     &subLogger,
		queueCache: make(map[int64][]QueueItem),
		clientPool: make(map[int64]Client),
	}
}

// SetBroadcaster sets the WebSocket broadcaster for real-time events.
func (s *Service) SetBroadcaster(broadcaster Broadcaster) {
	s.broadcaster = broadcaster
}

// SetStatusChangeLogger sets the logger for status transition history events.
func (s *Service) SetStatusChangeLogger(logger StatusChangeLogger) {
	s.statusChangeLogger = logger
}

// SetPortalStatusTracker sets the portal status tracker for request mirroring.
func (s *Service) SetPortalStatusTracker(tracker PortalStatusTracker) {
	s.portalStatusTracker = tracker
}

// SetHealthService sets the health service for tracking client health.
func (s *Service) SetHealthService(hs HealthService) {
	s.healthService = hs
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)

	s.clientPoolMu.Lock()
	s.clientPool = make(map[int64]Client)
	s.clientPoolMu.Unlock()
}

// RegisterExistingClients registers all existing download clients with the health service.
// This should be called once during startup after setting the health service.
func (s *Service) RegisterExistingClients(ctx context.Context) error {
	if s.healthService == nil {
		return nil
	}

	clients, err := s.List(ctx)
	if err != nil {
		return err
	}

	for _, client := range clients {
		s.healthService.RegisterItemStr("downloadClients", fmt.Sprintf("%d", client.ID), client.Name)
	}

	s.logger.Info().Int("count", len(clients)).Msg("Registered existing download clients with health service")
	return nil
}

// Get retrieves a download client by ID.
func (s *Service) Get(ctx context.Context, id int64) (*DownloadClient, error) {
	row, err := s.queries.GetDownloadClient(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("failed to get download client: %w", err)
	}
	return s.rowToClient(row), nil
}

// List returns all download clients.
func (s *Service) List(ctx context.Context) ([]*DownloadClient, error) {
	rows, err := s.queries.ListDownloadClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list download clients: %w", err)
	}

	clients := make([]*DownloadClient, 0, len(rows))
	for _, row := range rows {
		clients = append(clients, s.rowToClient(row))
	}
	return clients, nil
}

var validClientTypes = map[string]bool{
	"qbittorrent": true, "transmission": true, "deluge": true, "rtorrent": true,
	"vuze": true, "aria2": true, "flood": true, "utorrent": true,
	"hadouken": true, "downloadstation": true, "freeboxdownload": true,
	"rqbit": true, "tribler": true, "mock": true,
}

func validateCreateInput(input *CreateClientInput) error {
	if input.Name == "" || input.Host == "" || input.Type == "" {
		return ErrInvalidClient
	}
	if !validClientTypes[input.Type] {
		return ErrUnsupportedClient
	}
	if input.Priority == 0 {
		input.Priority = 50
	}
	return nil
}

// Create creates a new download client.
func (s *Service) Create(ctx context.Context, input *CreateClientInput) (*DownloadClient, error) {
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}

	cleanupMode := input.CleanupMode
	if cleanupMode == "" {
		cleanupMode = "leave"
	}

	row, err := s.queries.CreateDownloadClient(ctx, sqlc.CreateDownloadClientParams{
		Name:               input.Name,
		Type:               input.Type,
		Host:               input.Host,
		Port:               int64(input.Port),
		Username:           toNullString(input.Username),
		Password:           toNullString(input.Password),
		UseSsl:             boolToInt64(input.UseSSL),
		ApiKey:             toNullString(input.APIKey),
		Category:           toNullString(input.Category),
		UrlBase:            input.URLBase,
		Priority:           int64(input.Priority),
		Enabled:            boolToInt64(input.Enabled),
		ImportDelaySeconds: int64(input.ImportDelaySeconds),
		CleanupMode:        cleanupMode,
		SeedRatioTarget:    toNullFloat64(input.SeedRatioTarget),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create download client: %w", err)
	}

	s.logger.Info().Int64("id", row.ID).Str("name", input.Name).Str("type", input.Type).Msg("Created download client")

	// Register with health service
	if s.healthService != nil {
		s.healthService.RegisterItemStr("downloadClients", fmt.Sprintf("%d", row.ID), input.Name)
	}

	return s.rowToClient(row), nil
}

// Update updates an existing download client.
func (s *Service) Update(ctx context.Context, id int64, input *UpdateClientInput) (*DownloadClient, error) {
	if input.Name == "" {
		return nil, ErrInvalidClient
	}

	cleanupMode := input.CleanupMode
	if cleanupMode == "" {
		cleanupMode = "leave"
	}

	row, err := s.queries.UpdateDownloadClient(ctx, sqlc.UpdateDownloadClientParams{
		ID:                 id,
		Name:               input.Name,
		Type:               input.Type,
		Host:               input.Host,
		Port:               int64(input.Port),
		Username:           toNullString(input.Username),
		Password:           toNullString(input.Password),
		UseSsl:             boolToInt64(input.UseSSL),
		ApiKey:             toNullString(input.APIKey),
		Category:           toNullString(input.Category),
		UrlBase:            input.URLBase,
		Priority:           int64(input.Priority),
		Enabled:            boolToInt64(input.Enabled),
		ImportDelaySeconds: int64(input.ImportDelaySeconds),
		CleanupMode:        cleanupMode,
		SeedRatioTarget:    toNullFloat64(input.SeedRatioTarget),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("failed to update download client: %w", err)
	}

	s.clientPoolMu.Lock()
	delete(s.clientPool, id)
	s.clientPoolMu.Unlock()

	s.logger.Info().Int64("id", id).Str("name", input.Name).Msg("Updated download client")
	return s.rowToClient(row), nil
}

// Delete deletes a download client.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteDownloadClient(ctx, id); err != nil {
		return fmt.Errorf("failed to delete download client: %w", err)
	}

	s.clientPoolMu.Lock()
	delete(s.clientPool, id)
	s.clientPoolMu.Unlock()

	// Unregister from health service
	if s.healthService != nil {
		s.healthService.UnregisterItemStr("downloadClients", fmt.Sprintf("%d", id))
	}

	s.logger.Info().Int64("id", id).Msg("Deleted download client")
	return nil
}

// Test tests a download client connection by ID.
func (s *Service) Test(ctx context.Context, id int64) (*TestResult, error) {
	client, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.TestConfig(ctx, &CreateClientInput{
		Name:     client.Name,
		Type:     client.Type,
		Host:     client.Host,
		Port:     client.Port,
		Username: client.Username,
		Password: client.Password,
		UseSSL:   client.UseSSL,
		Category: client.Category,
	})
}

// TestConfig tests a download client connection using provided configuration.
func (s *Service) TestConfig(ctx context.Context, input *CreateClientInput) (*TestResult, error) {
	config := &types.ClientConfig{
		Host:     input.Host,
		Port:     input.Port,
		Username: input.Username,
		Password: input.Password,
		UseSSL:   input.UseSSL,
		APIKey:   input.APIKey,
		Category: input.Category,
		URLBase:  input.URLBase,
	}

	// Check if this client type is implemented
	if !IsClientTypeImplemented(input.Type) {
		if IsClientTypeSupported(input.Type) {
			return &TestResult{
				Success: false,
				Message: fmt.Sprintf("%s client is not yet implemented", input.Type),
			}, nil
		}
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Unknown client type: %s", input.Type),
		}, nil
	}

	// Create client using factory
	client, err := NewClient(ClientType(input.Type), config)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create client: %s", err.Error()),
		}, nil
	}

	// Test the connection
	if err := client.Test(ctx); err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %s", err.Error()),
		}, nil
	}

	return &TestResult{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to %s", input.Type),
	}, nil
}

// GetTransmissionClient returns a Transmission client for the given download client ID.
//
// Deprecated: Use GetClient or GetTorrentClient instead for better abstraction.
func (s *Service) GetTransmissionClient(ctx context.Context, id int64) (*transmission.Client, error) {
	cfg, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if cfg.Type != "transmission" {
		return nil, ErrUnsupportedClient
	}

	return transmission.New(&transmission.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Username: cfg.Username,
		Password: cfg.Password,
		UseSSL:   cfg.UseSSL,
	}), nil
}

// GetClient returns a Client interface for the given download client ID.
// Clients are cached in a pool to preserve auth sessions across poll cycles.
func (s *Service) GetClient(ctx context.Context, id int64) (Client, error) {
	s.clientPoolMu.RLock()
	if c, ok := s.clientPool[id]; ok {
		s.clientPoolMu.RUnlock()
		return c, nil
	}
	s.clientPoolMu.RUnlock()

	cfg, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	c, err := ClientFromDownloadClient(cfg)
	if err != nil {
		return nil, err
	}

	s.clientPoolMu.Lock()
	s.clientPool[id] = c
	s.clientPoolMu.Unlock()

	return c, nil
}

// EvictClient removes a client from the instance pool, forcing a fresh
// instance to be created on the next GetClient call.
func (s *Service) EvictClient(id int64) {
	s.clientPoolMu.Lock()
	delete(s.clientPool, id)
	s.clientPoolMu.Unlock()
}

// GetTorrentClient returns a TorrentClient interface for the given download client ID.
// Returns an error if the client is not a torrent client.
func (s *Service) GetTorrentClient(ctx context.Context, id int64) (TorrentClient, error) {
	cfg, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return TorrentClientFromDownloadClient(cfg)
}

// AddTorrent adds a torrent from a URL to a specific client.
// mediaType should be "movie" or "series" to determine the download subdirectory.
// name is an optional display name (used by mock client; real clients get name from torrent file).
func (s *Service) AddTorrent(ctx context.Context, clientID int64, url, mediaType, name string) (string, error) {
	cfg, err := s.Get(ctx, clientID)
	if err != nil {
		return "", err
	}

	// Determine subdirectory based on media type
	var subDir string
	switch mediaType {
	case mediaTypeMovie:
		subDir = "SlipStream/Movies"
	case mediaTypeSeries, "season", "episode":
		subDir = "SlipStream/Series"
	default:
		subDir = "SlipStream"
	}

	// Get the client using the factory
	client, err := ClientFromDownloadClient(cfg)
	if err != nil {
		return "", err
	}

	// Get default download dir and construct full path
	defaultDir, err := client.GetDownloadDir(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Could not get default download dir, using relative path")
		defaultDir = ""
	}

	var downloadDir string
	if defaultDir != "" {
		downloadDir = fmt.Sprintf("%s/%s", defaultDir, subDir)
	}

	// Add the torrent
	torrentID, err := client.Add(ctx, &types.AddOptions{
		URL:         url,
		Name:        name,
		DownloadDir: downloadDir,
	})
	if err != nil {
		return "", fmt.Errorf("failed to add torrent: %w", err)
	}

	// Start the torrent
	if err := client.Resume(ctx, torrentID); err != nil {
		s.logger.Warn().Err(err).Str("id", torrentID).Msg("Failed to start torrent")
	}

	s.logger.Info().Str("url", url).Str("torrentId", torrentID).Str("mediaType", mediaType).Str("subDir", subDir).Msg("Added torrent")
	return torrentID, nil
}

// AddTorrentWithContent adds a torrent from raw file content to a specific client.
// This is used when the torrent file has been pre-downloaded (e.g., from a private tracker
// that requires authentication cookies to download the torrent file).
// mediaType should be "movie" or "series" to determine the download subdirectory.
// name is an optional display name (used by mock client; real clients get name from torrent file).
func (s *Service) AddTorrentWithContent(ctx context.Context, clientID int64, content []byte, mediaType, name string) (string, error) {
	cfg, err := s.Get(ctx, clientID)
	if err != nil {
		return "", err
	}

	// Determine subdirectory based on media type
	var subDir string
	switch mediaType {
	case mediaTypeMovie:
		subDir = "SlipStream/Movies"
	case mediaTypeSeries, "season", "episode":
		subDir = "SlipStream/Series"
	default:
		subDir = "SlipStream"
	}

	// Get the client using the factory
	client, err := ClientFromDownloadClient(cfg)
	if err != nil {
		return "", err
	}

	// Get default download dir and construct full path
	defaultDir, err := client.GetDownloadDir(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Could not get default download dir, using relative path")
		defaultDir = ""
	}

	var downloadDir string
	if defaultDir != "" {
		downloadDir = fmt.Sprintf("%s/%s", defaultDir, subDir)
	}

	// Add the torrent using file content
	torrentID, err := client.Add(ctx, &types.AddOptions{
		FileContent: content,
		Name:        name,
		DownloadDir: downloadDir,
	})
	if err != nil {
		return "", fmt.Errorf("failed to add torrent: %w", err)
	}

	// Start the torrent
	if err := client.Resume(ctx, torrentID); err != nil {
		s.logger.Warn().Err(err).Str("id", torrentID).Msg("Failed to start torrent")
	}

	s.logger.Info().Int("contentSize", len(content)).Str("torrentId", torrentID).Str("mediaType", mediaType).Str("subDir", subDir).Msg("Added torrent from content")
	return torrentID, nil
}

// rowToClient converts a database row to a DownloadClient.
func (s *Service) rowToClient(row *sqlc.DownloadClient) *DownloadClient {
	client := &DownloadClient{
		ID:                 row.ID,
		Name:               row.Name,
		Type:               row.Type,
		Host:               row.Host,
		Port:               int(row.Port),
		UseSSL:             row.UseSsl == 1,
		Priority:           int(row.Priority),
		Enabled:            row.Enabled == 1,
		ImportDelaySeconds: int(row.ImportDelaySeconds),
		CleanupMode:        row.CleanupMode,
	}

	if row.Username.Valid {
		client.Username = row.Username.String
	}
	if row.Password.Valid {
		client.Password = row.Password.String
	}
	if row.ApiKey.Valid {
		client.APIKey = row.ApiKey.String
	}
	if row.Category.Valid {
		client.Category = row.Category.String
	}
	client.URLBase = row.UrlBase
	client.CreatedAt = row.CreatedAt
	client.UpdatedAt = row.UpdatedAt
	if row.SeedRatioTarget.Valid {
		client.SeedRatioTarget = &row.SeedRatioTarget.Float64
	}

	return client
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func toNullFloat64(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}

func (s *Service) getCachedQueue(clientID int64) []QueueItem {
	s.queueCacheMu.RLock()
	defer s.queueCacheMu.RUnlock()
	items := s.queueCache[clientID]
	copied := make([]QueueItem, len(items))
	copy(copied, items)
	return copied
}

func (s *Service) setCachedQueue(clientID int64, items []QueueItem) {
	s.queueCacheMu.Lock()
	defer s.queueCacheMu.Unlock()
	copied := make([]QueueItem, len(items))
	copy(copied, items)
	s.queueCache[clientID] = copied
}
