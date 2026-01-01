package downloader

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader/transmission"
)

var (
	ErrClientNotFound    = errors.New("download client not found")
	ErrInvalidClient     = errors.New("invalid download client")
	ErrUnsupportedClient = errors.New("unsupported client type")
)

// DownloadClient represents a download client configuration.
type DownloadClient struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Username  string    `json:"username,omitempty"`
	Password  string    `json:"password,omitempty"`
	UseSSL    bool      `json:"useSsl"`
	Category  string    `json:"category,omitempty"`
	Priority  int       `json:"priority"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateClientInput represents the input for creating a download client.
type CreateClientInput struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	UseSSL   bool   `json:"useSsl"`
	Category string `json:"category,omitempty"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

// UpdateClientInput represents the input for updating a download client.
type UpdateClientInput struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	UseSSL   bool   `json:"useSsl"`
	Category string `json:"category,omitempty"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

// TestResult represents the result of testing a download client connection.
type TestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Service provides download client operations.
type Service struct {
	queries       *sqlc.Queries
	logger        zerolog.Logger
	developerMode bool
	mockQueue     []MockQueueItem
}

// MockQueueItem represents a mock queue item for developer mode.
type MockQueueItem struct {
	ID         string
	ClientID   int64
	ClientName string
	Title      string
	MediaType  string
	Quality    string
	Source     string
	Codec      string
	Attributes []string
	Season     int
	Episode    int
	Size       int64
	Progress   float64
	Status     string
}

// NewService creates a new download client service.
func NewService(db *sql.DB, logger zerolog.Logger) *Service {
	return &Service{
		queries:   sqlc.New(db),
		logger:    logger.With().Str("component", "downloader").Logger(),
		mockQueue: make([]MockQueueItem, 0),
	}
}

// SetDeveloperMode sets the developer mode flag.
func (s *Service) SetDeveloperMode(enabled bool) {
	s.developerMode = enabled
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

// Create creates a new download client.
func (s *Service) Create(ctx context.Context, input CreateClientInput) (*DownloadClient, error) {
	if input.Name == "" {
		return nil, ErrInvalidClient
	}
	if input.Host == "" {
		return nil, ErrInvalidClient
	}
	if input.Type == "" {
		return nil, ErrInvalidClient
	}

	// Validate client type
	validTypes := []string{"qbittorrent", "transmission", "deluge", "rtorrent", "sabnzbd", "nzbget"}
	isValid := false
	for _, t := range validTypes {
		if input.Type == t {
			isValid = true
			break
		}
	}
	if !isValid {
		return nil, ErrUnsupportedClient
	}

	// Default priority
	if input.Priority == 0 {
		input.Priority = 50
	}

	row, err := s.queries.CreateDownloadClient(ctx, sqlc.CreateDownloadClientParams{
		Name:     input.Name,
		Type:     input.Type,
		Host:     input.Host,
		Port:     int64(input.Port),
		Username: toNullString(input.Username),
		Password: toNullString(input.Password),
		UseSsl:   boolToInt64(input.UseSSL),
		Category: toNullString(input.Category),
		Priority: int64(input.Priority),
		Enabled:  boolToInt64(input.Enabled),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create download client: %w", err)
	}

	s.logger.Info().Int64("id", row.ID).Str("name", input.Name).Str("type", input.Type).Msg("Created download client")
	return s.rowToClient(row), nil
}

// Update updates an existing download client.
func (s *Service) Update(ctx context.Context, id int64, input UpdateClientInput) (*DownloadClient, error) {
	if input.Name == "" {
		return nil, ErrInvalidClient
	}

	row, err := s.queries.UpdateDownloadClient(ctx, sqlc.UpdateDownloadClientParams{
		ID:       id,
		Name:     input.Name,
		Type:     input.Type,
		Host:     input.Host,
		Port:     int64(input.Port),
		Username: toNullString(input.Username),
		Password: toNullString(input.Password),
		UseSsl:   boolToInt64(input.UseSSL),
		Category: toNullString(input.Category),
		Priority: int64(input.Priority),
		Enabled:  boolToInt64(input.Enabled),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("failed to update download client: %w", err)
	}

	s.logger.Info().Int64("id", id).Str("name", input.Name).Msg("Updated download client")
	return s.rowToClient(row), nil
}

// Delete deletes a download client.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteDownloadClient(ctx, id); err != nil {
		return fmt.Errorf("failed to delete download client: %w", err)
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

	return s.TestConfig(ctx, CreateClientInput{
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
func (s *Service) TestConfig(ctx context.Context, input CreateClientInput) (*TestResult, error) {
	switch input.Type {
	case "transmission":
		client := transmission.New(transmission.Config{
			Host:     input.Host,
			Port:     input.Port,
			Username: input.Username,
			Password: input.Password,
			UseSSL:   input.UseSSL,
		})

		if err := client.Test(); err != nil {
			return &TestResult{
				Success: false,
				Message: fmt.Sprintf("Connection failed: %s", err.Error()),
			}, nil
		}

		return &TestResult{
			Success: true,
			Message: "Successfully connected to Transmission",
		}, nil

	case "qbittorrent", "deluge", "rtorrent", "sabnzbd", "nzbget":
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("%s client is not yet implemented", input.Type),
		}, nil

	default:
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Unknown client type: %s", input.Type),
		}, nil
	}
}

// GetTransmissionClient returns a Transmission client for the given download client ID.
func (s *Service) GetTransmissionClient(ctx context.Context, id int64) (*transmission.Client, error) {
	cfg, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if cfg.Type != "transmission" {
		return nil, ErrUnsupportedClient
	}

	return transmission.New(transmission.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Username: cfg.Username,
		Password: cfg.Password,
		UseSSL:   cfg.UseSSL,
	}), nil
}

// AddTorrent adds a torrent from a URL to a specific client.
// mediaType should be "movie" or "series" to determine the download subdirectory.
func (s *Service) AddTorrent(ctx context.Context, clientID int64, url string, mediaType string) (string, error) {
	cfg, err := s.Get(ctx, clientID)
	if err != nil {
		return "", err
	}

	// Determine subdirectory based on media type
	var subDir string
	switch mediaType {
	case "movie":
		subDir = "SlipStream/Movies"
	case "series", "episode":
		subDir = "SlipStream/Series"
	default:
		subDir = "SlipStream"
	}

	switch cfg.Type {
	case "transmission":
		client := transmission.New(transmission.Config{
			Host:     cfg.Host,
			Port:     cfg.Port,
			Username: cfg.Username,
			Password: cfg.Password,
			UseSSL:   cfg.UseSSL,
		})

		torrentID, err := client.AddURL(url, subDir)
		if err != nil {
			return "", fmt.Errorf("failed to add torrent: %w", err)
		}

		// Start the torrent
		if err := client.Start(torrentID); err != nil {
			s.logger.Warn().Err(err).Str("id", torrentID).Msg("Failed to start torrent")
		}

		s.logger.Info().Str("url", url).Str("torrentId", torrentID).Str("mediaType", mediaType).Str("subDir", subDir).Msg("Added torrent")
		return torrentID, nil

	default:
		return "", ErrUnsupportedClient
	}
}

// rowToClient converts a database row to a DownloadClient.
func (s *Service) rowToClient(row *sqlc.DownloadClient) *DownloadClient {
	client := &DownloadClient{
		ID:       row.ID,
		Name:     row.Name,
		Type:     row.Type,
		Host:     row.Host,
		Port:     int(row.Port),
		UseSSL:   row.UseSsl == 1,
		Priority: int(row.Priority),
		Enabled:  row.Enabled == 1,
	}

	if row.Username.Valid {
		client.Username = row.Username.String
	}
	if row.Password.Valid {
		client.Password = row.Password.String
	}
	if row.Category.Valid {
		client.Category = row.Category.String
	}
	if row.CreatedAt.Valid {
		client.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		client.UpdatedAt = row.UpdatedAt.Time
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
