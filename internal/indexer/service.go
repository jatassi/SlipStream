package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/indexer/cardigann"
)

var (
	ErrIndexerNotFound    = errors.New("indexer not found")
	ErrDefinitionNotFound = errors.New("definition not found")
	ErrInvalidIndexer     = errors.New("invalid indexer configuration")
)

// Service provides indexer operations using Cardigann definitions.
type Service struct {
	queries *sqlc.Queries
	manager *cardigann.Manager
	logger  zerolog.Logger
}

// NewService creates a new indexer service.
func NewService(db *sql.DB, manager *cardigann.Manager, logger zerolog.Logger) *Service {
	return &Service{
		queries: sqlc.New(db),
		manager: manager,
		logger:  logger.With().Str("component", "indexer").Logger(),
	}
}

// Get retrieves an indexer by ID.
func (s *Service) Get(ctx context.Context, id int64) (*IndexerDefinition, error) {
	row, err := s.queries.GetIndexer(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIndexerNotFound
		}
		return nil, fmt.Errorf("failed to get indexer: %w", err)
	}
	return s.rowToDefinition(row), nil
}

// List returns all indexers.
func (s *Service) List(ctx context.Context) ([]*IndexerDefinition, error) {
	rows, err := s.queries.ListIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexers: %w", err)
	}

	indexers := make([]*IndexerDefinition, 0, len(rows))
	for _, row := range rows {
		indexers = append(indexers, s.rowToDefinition(row))
	}
	return indexers, nil
}

// ListEnabled returns all enabled indexers.
func (s *Service) ListEnabled(ctx context.Context) ([]*IndexerDefinition, error) {
	rows, err := s.queries.ListEnabledIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled indexers: %w", err)
	}

	indexers := make([]*IndexerDefinition, 0, len(rows))
	for _, row := range rows {
		indexers = append(indexers, s.rowToDefinition(row))
	}
	return indexers, nil
}

// ListEnabledForMovies returns all enabled indexers that support movies.
func (s *Service) ListEnabledForMovies(ctx context.Context) ([]*IndexerDefinition, error) {
	rows, err := s.queries.ListEnabledMovieIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list movie indexers: %w", err)
	}

	indexers := make([]*IndexerDefinition, 0, len(rows))
	for _, row := range rows {
		indexers = append(indexers, s.rowToDefinition(row))
	}
	return indexers, nil
}

// ListEnabledForTV returns all enabled indexers that support TV shows.
func (s *Service) ListEnabledForTV(ctx context.Context) ([]*IndexerDefinition, error) {
	rows, err := s.queries.ListEnabledTVIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list TV indexers: %w", err)
	}

	indexers := make([]*IndexerDefinition, 0, len(rows))
	for _, row := range rows {
		indexers = append(indexers, s.rowToDefinition(row))
	}
	return indexers, nil
}

// ListEnabledByProtocol returns all enabled indexers with the specified protocol.
func (s *Service) ListEnabledByProtocol(ctx context.Context, protocol Protocol) ([]*IndexerDefinition, error) {
	// Get all enabled indexers and filter by protocol
	all, err := s.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*IndexerDefinition, 0)
	for _, idx := range all {
		if idx.Protocol == protocol {
			filtered = append(filtered, idx)
		}
	}
	return filtered, nil
}

// CreateIndexerInput is the input for creating a new indexer.
type CreateIndexerInput struct {
	Name           string          `json:"name"`
	DefinitionID   string          `json:"definitionId"`
	Settings       json.RawMessage `json:"settings,omitempty"`
	Categories     []int           `json:"categories"`
	SupportsMovies bool            `json:"supportsMovies"`
	SupportsTV     bool            `json:"supportsTV"`
	Priority       int             `json:"priority"`
	Enabled        bool            `json:"enabled"`
}

// Create creates a new indexer.
func (s *Service) Create(ctx context.Context, input CreateIndexerInput) (*IndexerDefinition, error) {
	if err := s.validateInput(input); err != nil {
		return nil, err
	}

	// Verify definition exists
	if _, err := s.manager.GetDefinition(input.DefinitionID); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDefinitionNotFound, input.DefinitionID)
	}

	// Default priority
	if input.Priority == 0 {
		input.Priority = 50
	}

	// Serialize categories to JSON
	categoriesJSON, err := json.Marshal(input.Categories)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize categories: %w", err)
	}

	// Serialize settings
	settingsJSON := "{}"
	if input.Settings != nil {
		settingsJSON = string(input.Settings)
	}

	row, err := s.queries.CreateIndexer(ctx, sqlc.CreateIndexerParams{
		Name:           input.Name,
		DefinitionID:   input.DefinitionID,
		Settings:       toNullString(settingsJSON),
		Categories:     toNullString(string(categoriesJSON)),
		SupportsMovies: boolToInt64(input.SupportsMovies),
		SupportsTv:     boolToInt64(input.SupportsTV),
		Priority:       int64(input.Priority),
		Enabled:        boolToInt64(input.Enabled),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create indexer: %w", err)
	}

	s.logger.Info().
		Int64("id", row.ID).
		Str("name", input.Name).
		Str("definition", input.DefinitionID).
		Msg("Created indexer")

	return s.rowToDefinition(row), nil
}

// Update updates an existing indexer.
func (s *Service) Update(ctx context.Context, id int64, input CreateIndexerInput) (*IndexerDefinition, error) {
	if err := s.validateInput(input); err != nil {
		return nil, err
	}

	// Check if indexer exists
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}

	// Verify definition exists
	if _, err := s.manager.GetDefinition(input.DefinitionID); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDefinitionNotFound, input.DefinitionID)
	}

	// Serialize categories to JSON
	categoriesJSON, err := json.Marshal(input.Categories)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize categories: %w", err)
	}

	// Serialize settings
	settingsJSON := "{}"
	if input.Settings != nil {
		settingsJSON = string(input.Settings)
	}

	row, err := s.queries.UpdateIndexer(ctx, sqlc.UpdateIndexerParams{
		ID:             id,
		Name:           input.Name,
		DefinitionID:   input.DefinitionID,
		Settings:       toNullString(settingsJSON),
		Categories:     toNullString(string(categoriesJSON)),
		SupportsMovies: boolToInt64(input.SupportsMovies),
		SupportsTv:     boolToInt64(input.SupportsTV),
		Priority:       int64(input.Priority),
		Enabled:        boolToInt64(input.Enabled),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIndexerNotFound
		}
		return nil, fmt.Errorf("failed to update indexer: %w", err)
	}

	// Remove cached client to force recreation with new settings
	s.manager.RemoveClient(id)

	s.logger.Info().Int64("id", id).Str("name", input.Name).Msg("Updated indexer")
	return s.rowToDefinition(row), nil
}

// Delete deletes an indexer.
func (s *Service) Delete(ctx context.Context, id int64) error {
	// Check if indexer exists
	indexer, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := s.queries.DeleteIndexer(ctx, id); err != nil {
		return fmt.Errorf("failed to delete indexer: %w", err)
	}

	// Remove cached client
	s.manager.RemoveClient(id)

	s.logger.Info().Int64("id", id).Str("name", indexer.Name).Msg("Deleted indexer")
	return nil
}

// Count returns the total number of indexers.
func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.queries.CountIndexers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count indexers: %w", err)
	}
	return count, nil
}

// TestResult represents the result of testing an indexer connection.
type TestResult struct {
	Success      bool          `json:"success"`
	Message      string        `json:"message"`
	Capabilities *Capabilities `json:"capabilities,omitempty"`
}

// Test tests an indexer connection by ID.
func (s *Service) Test(ctx context.Context, id int64) (*TestResult, error) {
	indexer, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.TestConfig(ctx, TestConfigInput{
		DefinitionID: indexer.DefinitionID,
		Settings:     indexer.Settings,
	})
}

// TestConfigInput is the input for testing an indexer configuration.
type TestConfigInput struct {
	DefinitionID string          `json:"definitionId"`
	Settings     json.RawMessage `json:"settings"`
}

// TestConfig tests an indexer configuration without saving.
func (s *Service) TestConfig(ctx context.Context, input TestConfigInput) (*TestResult, error) {
	// Parse settings
	settings := make(map[string]string)
	if input.Settings != nil {
		if err := json.Unmarshal(input.Settings, &settings); err != nil {
			return &TestResult{
				Success: false,
				Message: fmt.Sprintf("Invalid settings format: %s", err.Error()),
			}, nil
		}
	}

	// Test the definition
	if err := s.manager.TestDefinition(ctx, input.DefinitionID, settings); err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Connection test failed: %s", err.Error()),
		}, nil
	}

	// Get capabilities
	caps, err := s.manager.GetCapabilities(input.DefinitionID)
	if err != nil {
		return &TestResult{
			Success: true,
			Message: "Successfully connected to indexer",
		}, nil
	}

	return &TestResult{
		Success:      true,
		Message:      "Successfully connected to indexer",
		Capabilities: caps,
	}, nil
}

// GetClient creates or retrieves an indexer client for the given indexer ID.
func (s *Service) GetClient(ctx context.Context, id int64) (Indexer, error) {
	// Check if we already have a cached client
	if client, ok := s.manager.GetClient(id); ok {
		return client, nil
	}

	// Get the indexer definition
	indexer, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Parse settings
	settings := make(map[string]string)
	if indexer.Settings != nil {
		if err := json.Unmarshal(indexer.Settings, &settings); err != nil {
			return nil, fmt.Errorf("failed to parse settings: %w", err)
		}
	}

	// Create the client
	client, err := s.manager.CreateClientFromDefinition(
		indexer.DefinitionID,
		indexer.ID,
		indexer.Name,
		settings,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Cache the client
	s.manager.RegisterClient(id, client)

	return client, nil
}

// ListDefinitions returns all available Cardigann definitions.
func (s *Service) ListDefinitions() ([]*cardigann.DefinitionMetadata, error) {
	return s.manager.ListDefinitions()
}

// SearchDefinitions searches for definitions matching the query.
func (s *Service) SearchDefinitions(query string, filters cardigann.DefinitionFilters) ([]*cardigann.DefinitionMetadata, error) {
	return s.manager.SearchDefinitions(query, filters)
}

// GetDefinition retrieves a Cardigann definition by ID.
func (s *Service) GetDefinition(id string) (*cardigann.Definition, error) {
	return s.manager.GetDefinition(id)
}

// GetDefinitionSchema returns the settings schema for a definition.
func (s *Service) GetDefinitionSchema(id string) ([]cardigann.Setting, error) {
	return s.manager.GetSettingsSchema(id)
}

// UpdateDefinitions updates the definition cache from the remote repository.
func (s *Service) UpdateDefinitions(ctx context.Context) error {
	return s.manager.UpdateDefinitions(ctx)
}

// GetManager returns the Cardigann manager.
func (s *Service) GetManager() *cardigann.Manager {
	return s.manager
}

// validateInput validates the indexer input.
func (s *Service) validateInput(input CreateIndexerInput) error {
	if input.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidIndexer)
	}
	if input.DefinitionID == "" {
		return fmt.Errorf("%w: definition ID is required", ErrInvalidIndexer)
	}
	return nil
}

// rowToDefinition converts a database row to an IndexerDefinition.
func (s *Service) rowToDefinition(row *sqlc.Indexer) *IndexerDefinition {
	def := &IndexerDefinition{
		ID:             row.ID,
		Name:           row.Name,
		DefinitionID:   row.DefinitionID,
		SupportsMovies: row.SupportsMovies == 1,
		SupportsTV:     row.SupportsTv == 1,
		Priority:       int(row.Priority),
		Enabled:        row.Enabled == 1,
		Categories:     []int{},
	}

	// Parse settings
	if row.Settings.Valid && row.Settings.String != "" {
		def.Settings = json.RawMessage(row.Settings.String)
	}

	// Parse categories
	if row.Categories.Valid && row.Categories.String != "" {
		var categories []int
		if err := json.Unmarshal([]byte(row.Categories.String), &categories); err == nil {
			def.Categories = categories
		}
	}

	// Get protocol and privacy from the Cardigann definition
	if cardDef, err := s.manager.GetDefinition(row.DefinitionID); err == nil {
		def.Protocol = Protocol(cardDef.GetProtocol())
		def.Privacy = Privacy(cardDef.GetPrivacy())
		def.SupportsSearch = cardDef.SupportsSearch("search")
		def.SupportsRSS = true
	} else {
		// Defaults if definition not found
		def.Protocol = ProtocolTorrent
		def.Privacy = PrivacyPublic
		def.SupportsSearch = true
		def.SupportsRSS = true
	}

	if row.CreatedAt.Valid {
		def.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		def.UpdatedAt = row.UpdatedAt.Time
	}

	return def
}

// Helper functions

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
