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
	"github.com/slipstream/slipstream/internal/indexer/genericrss"
	indexermock "github.com/slipstream/slipstream/internal/indexer/mock"
)

// MockDefinitionID is the special definition ID for mock indexers.
const MockDefinitionID = "mock"

var (
	ErrIndexerNotFound    = errors.New("indexer not found")
	ErrDefinitionNotFound = errors.New("definition not found")
	ErrInvalidIndexer     = errors.New("invalid indexer configuration")
)

// HealthService is the interface for central health tracking.
type HealthService interface {
	RegisterItemStr(category, id, name string)
	UnregisterItemStr(category, id string)
	SetErrorStr(category, id, message string)
	ClearStatusStr(category, id string)
}

// Service provides indexer operations using Cardigann definitions.
type Service struct {
	queries       *sqlc.Queries
	manager       *cardigann.Manager
	logger        *zerolog.Logger
	healthService HealthService
}

// NewService creates a new indexer service.
func NewService(db *sql.DB, manager *cardigann.Manager, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "indexer").Logger()
	return &Service{
		queries: sqlc.New(db),
		manager: manager,
		logger:  &subLogger,
	}
}

// SetHealthService sets the central health service for registration tracking.
func (s *Service) SetHealthService(hs HealthService) {
	s.healthService = hs
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)
}

// InitializeDefinitions fetches the latest indexer definitions from the remote repository.
// This is called during startup with network retry logic.
func (s *Service) InitializeDefinitions(ctx context.Context) error {
	if s.manager == nil {
		return nil
	}
	return s.manager.Initialize(ctx)
}

// RegisterExistingIndexers registers all existing enabled indexers with the health service.
func (s *Service) RegisterExistingIndexers(ctx context.Context) error {
	if s.healthService == nil {
		return nil
	}

	indexers, err := s.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to list indexers for health registration: %w", err)
	}

	for _, indexer := range indexers {
		s.healthService.RegisterItemStr("indexers", fmt.Sprintf("%d", indexer.ID), indexer.Name)
	}

	s.logger.Info().Int("count", len(indexers)).Msg("Registered existing indexers with health service")
	return nil
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
	Name              string          `json:"name"`
	DefinitionID      string          `json:"definitionId"`
	Settings          json.RawMessage `json:"settings,omitempty"`
	Categories        []int           `json:"categories"`
	SupportsMovies    bool            `json:"supportsMovies"`
	SupportsTV        bool            `json:"supportsTv"`
	Priority          int             `json:"priority"`
	Enabled           bool            `json:"enabled"`
	AutoSearchEnabled *bool           `json:"autoSearchEnabled,omitempty"`
	RssEnabled        *bool           `json:"rssEnabled,omitempty"`
}

// UpdateIndexerInput is the input for updating an indexer (all fields optional for partial updates).
type UpdateIndexerInput struct {
	Name              *string         `json:"name,omitempty"`
	DefinitionID      *string         `json:"definitionId,omitempty"`
	Settings          json.RawMessage `json:"settings,omitempty"`
	Categories        []int           `json:"categories,omitempty"`
	SupportsMovies    *bool           `json:"supportsMovies,omitempty"`
	SupportsTV        *bool           `json:"supportsTv,omitempty"`
	Priority          *int            `json:"priority,omitempty"`
	Enabled           *bool           `json:"enabled,omitempty"`
	AutoSearchEnabled *bool           `json:"autoSearchEnabled,omitempty"`
	RssEnabled        *bool           `json:"rssEnabled,omitempty"`
}

// Create creates a new indexer.
func (s *Service) Create(ctx context.Context, input *CreateIndexerInput) (*IndexerDefinition, error) {
	if err := s.validateInput(input); err != nil {
		return nil, err
	}

	if err := s.validateDefinition(input.DefinitionID); err != nil {
		return nil, err
	}

	if input.Priority == 0 {
		input.Priority = 50
	}

	params, err := s.buildCreateParams(input)
	if err != nil {
		return nil, err
	}

	row, err := s.queries.CreateIndexer(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexer: %w", err)
	}

	s.logger.Info().Int64("id", row.ID).Str("name", input.Name).
		Str("definition", input.DefinitionID).Msg("Created indexer")

	if input.Enabled && s.healthService != nil {
		s.healthService.RegisterItemStr("indexers", fmt.Sprintf("%d", row.ID), input.Name)
	}

	return s.rowToDefinition(row), nil
}

func (s *Service) validateDefinition(definitionID string) error {
	if definitionID == MockDefinitionID || definitionID == genericrss.DefinitionID {
		return nil
	}
	if _, err := s.manager.GetDefinition(definitionID); err != nil {
		return fmt.Errorf("%w: %s", ErrDefinitionNotFound, definitionID)
	}
	return nil
}

func (s *Service) buildCreateParams(input *CreateIndexerInput) (sqlc.CreateIndexerParams, error) {
	categoriesJSON, err := json.Marshal(input.Categories)
	if err != nil {
		return sqlc.CreateIndexerParams{}, fmt.Errorf("failed to serialize categories: %w", err)
	}

	settingsJSON := s.serializeSettings(input.Settings)

	return sqlc.CreateIndexerParams{
		Name:              input.Name,
		DefinitionID:      input.DefinitionID,
		Settings:          toNullString(settingsJSON),
		Categories:        toNullString(string(categoriesJSON)),
		SupportsMovies:    boolToInt64(input.SupportsMovies),
		SupportsTv:        boolToInt64(input.SupportsTV),
		Priority:          int64(input.Priority),
		Enabled:           boolToInt64(input.Enabled),
		AutoSearchEnabled: boolToInt64(optBool(input.AutoSearchEnabled, true)),
		RssEnabled:        boolToInt64(optBool(input.RssEnabled, true)),
	}, nil
}

func (s *Service) serializeSettings(settings json.RawMessage) string {
	if settings != nil {
		s.logger.Debug().RawJSON("settings", settings).Msg("Creating indexer with settings")
		return string(settings)
	}
	s.logger.Warn().Msg("Creating indexer with nil settings")
	return "{}"
}

func optBool(ptr *bool, defaultVal bool) bool {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

// Update updates an existing indexer with partial update support.
func (s *Service) Update(ctx context.Context, id int64, input *UpdateIndexerInput) (*IndexerDefinition, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	params, err := s.buildUpdateParams(id, existing, input)
	if err != nil {
		return nil, err
	}

	row, err := s.queries.UpdateIndexer(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIndexerNotFound
		}
		return nil, fmt.Errorf("failed to update indexer: %w", err)
	}

	s.manager.RemoveClient(id)
	enabled := optBool(input.Enabled, existing.Enabled)
	name := optStr(input.Name, existing.Name)
	s.updateHealthRegistration(id, name, enabled, existing.Enabled)

	s.logger.Info().Int64("id", id).Str("name", name).Msg("Updated indexer")
	return s.rowToDefinition(row), nil
}

func (s *Service) buildUpdateParams(id int64, existing *IndexerDefinition, input *UpdateIndexerInput) (sqlc.UpdateIndexerParams, error) {
	name := optStr(input.Name, existing.Name)
	definitionID := optStr(input.DefinitionID, existing.DefinitionID)

	if name == "" {
		return sqlc.UpdateIndexerParams{}, fmt.Errorf("%w: name is required", ErrInvalidIndexer)
	}
	if definitionID == "" {
		return sqlc.UpdateIndexerParams{}, fmt.Errorf("%w: definition ID is required", ErrInvalidIndexer)
	}

	if err := s.validateDefinition(definitionID); err != nil {
		return sqlc.UpdateIndexerParams{}, err
	}

	categories := existing.Categories
	if input.Categories != nil {
		categories = input.Categories
	}
	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return sqlc.UpdateIndexerParams{}, fmt.Errorf("failed to serialize categories: %w", err)
	}

	settingsJSON := s.mergeSettings(input.Settings, existing.Settings)

	return sqlc.UpdateIndexerParams{
		ID:                id,
		Name:              name,
		DefinitionID:      definitionID,
		Settings:          toNullString(settingsJSON),
		Categories:        toNullString(string(categoriesJSON)),
		SupportsMovies:    boolToInt64(optBool(input.SupportsMovies, existing.SupportsMovies)),
		SupportsTv:        boolToInt64(optBool(input.SupportsTV, existing.SupportsTV)),
		Priority:          int64(optInt(input.Priority, existing.Priority)),
		Enabled:           boolToInt64(optBool(input.Enabled, existing.Enabled)),
		AutoSearchEnabled: boolToInt64(optBool(input.AutoSearchEnabled, existing.AutoSearchEnabled)),
		RssEnabled:        boolToInt64(optBool(input.RssEnabled, existing.RssEnabled)),
	}, nil
}

func (s *Service) mergeSettings(input, existing json.RawMessage) string {
	if input != nil {
		return string(input)
	}
	if existing != nil {
		return string(existing)
	}
	return "{}"
}

func (s *Service) updateHealthRegistration(id int64, name string, enabled, wasEnabled bool) {
	if s.healthService == nil {
		return
	}
	idStr := fmt.Sprintf("%d", id)
	if enabled && !wasEnabled {
		s.healthService.RegisterItemStr("indexers", idStr, name)
	} else if !enabled && wasEnabled {
		s.healthService.UnregisterItemStr("indexers", idStr)
	}
}

func optStr(ptr *string, defaultVal string) string {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func optInt(ptr *int, defaultVal int) int {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
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

	// Unregister from health service (only if it was enabled and tracked)
	if indexer.Enabled && s.healthService != nil {
		s.healthService.UnregisterItemStr("indexers", fmt.Sprintf("%d", id))
	}

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
	// Mock indexers always pass the test
	if input.DefinitionID == MockDefinitionID {
		return &TestResult{
			Success:      true,
			Message:      "Mock indexer connection successful",
			Capabilities: indexermock.NewClient(nil).Capabilities(),
		}, nil
	}

	// Generic RSS indexers test by fetching the feed
	if input.DefinitionID == genericrss.DefinitionID {
		settings := make(map[string]string)
		if input.Settings != nil {
			if err := json.Unmarshal(input.Settings, &settings); err != nil {
				return &TestResult{
					Success: false,
					Message: fmt.Sprintf("Invalid settings format: %s", err.Error()),
				}, nil
			}
		}
		client := genericrss.NewClient(nil, settings)
		if err := client.Test(ctx); err != nil {
			return &TestResult{
				Success: false,
				Message: fmt.Sprintf("RSS feed test failed: %s", err.Error()),
			}, nil
		}
		return &TestResult{
			Success:      true,
			Message:      "Successfully connected to RSS feed",
			Capabilities: client.Capabilities(),
		}, nil
	}

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
	caps, capsErr := s.manager.GetCapabilities(input.DefinitionID)
	if capsErr != nil {
		//nolint:nilerr // Capabilities optional for test success
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
	indexer, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if indexer.DefinitionID == MockDefinitionID {
		return indexermock.NewClient(indexer), nil
	}

	if indexer.DefinitionID == genericrss.DefinitionID {
		return s.createGenericRSSClient(indexer)
	}

	return s.getOrCreateCardigannClient(id, indexer)
}

func (s *Service) createGenericRSSClient(indexer *IndexerDefinition) (Indexer, error) {
	settings := make(map[string]string)
	if indexer.Settings != nil {
		if err := json.Unmarshal(indexer.Settings, &settings); err != nil {
			return nil, fmt.Errorf("failed to parse generic-rss settings: %w", err)
		}
	}
	return genericrss.NewClient(indexer, settings), nil
}

func (s *Service) getOrCreateCardigannClient(id int64, indexer *IndexerDefinition) (Indexer, error) {
	if client, ok := s.manager.GetClient(id); ok {
		if len(client.GetSettings()) > 0 {
			s.logger.Debug().Int64("id", id).Int("settingsCount", len(client.GetSettings())).Msg("Returning cached client with settings")
			return client, nil
		}
		s.logger.Debug().Int64("id", id).Msg("Cached client has empty settings, recreating")
		s.manager.RemoveClient(id)
	}

	settings, err := s.parseIndexerSettings(id, indexer.Settings)
	if err != nil {
		return nil, err
	}

	client, err := s.manager.CreateClientFromDefinition(indexer.DefinitionID, indexer.ID, indexer.Name, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	s.manager.RegisterClient(id, client)
	return client, nil
}

func (s *Service) parseIndexerSettings(id int64, raw json.RawMessage) (map[string]string, error) {
	settings := make(map[string]string)
	if raw != nil {
		s.logger.Debug().Int64("id", id).RawJSON("rawSettings", raw).Msg("Parsing indexer settings")
		if err := json.Unmarshal(raw, &settings); err != nil {
			return nil, fmt.Errorf("failed to parse settings: %w", err)
		}
		s.logger.Debug().Int64("id", id).Int("settingsCount", len(settings)).Msg("Parsed indexer settings")
	} else {
		s.logger.Warn().Int64("id", id).Msg("Indexer has nil settings")
	}
	return settings, nil
}

// ListDefinitions returns all available definitions (Cardigann + built-in).
func (s *Service) ListDefinitions() ([]*cardigann.DefinitionMetadata, error) {
	defs, err := s.manager.ListDefinitions()
	if err != nil {
		return nil, err
	}

	defs = append(defs, &cardigann.DefinitionMetadata{
		ID:          genericrss.DefinitionID,
		Name:        "Generic RSS",
		Description: "Generic RSS/Atom/TorrentPotato feed",
		Type:        "public",
		Language:    "en-US",
		Protocol:    "torrent",
	})

	return defs, nil
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
	if id == genericrss.DefinitionID {
		return genericrss.DefinitionSchema(), nil
	}
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
func (s *Service) validateInput(input *CreateIndexerInput) error {
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
		ID:                row.ID,
		Name:              row.Name,
		DefinitionID:      row.DefinitionID,
		SupportsMovies:    row.SupportsMovies == 1,
		SupportsTV:        row.SupportsTv == 1,
		Priority:          int(row.Priority),
		Enabled:           row.Enabled == 1,
		AutoSearchEnabled: row.AutoSearchEnabled == 1,
		RssEnabled:        row.RssEnabled == 1,
		Categories:        []int{},
	}

	if row.Settings.Valid && row.Settings.String != "" {
		def.Settings = json.RawMessage(row.Settings.String)
	}

	if row.Categories.Valid && row.Categories.String != "" {
		var categories []int
		if err := json.Unmarshal([]byte(row.Categories.String), &categories); err == nil {
			def.Categories = categories
		}
	}

	s.setDefinitionProtocol(def, row.DefinitionID)

	if row.CreatedAt.Valid {
		def.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		def.UpdatedAt = row.UpdatedAt.Time
	}

	return def
}

func (s *Service) setDefinitionProtocol(def *IndexerDefinition, definitionID string) {
	switch definitionID {
	case MockDefinitionID:
		def.Protocol = ProtocolTorrent
		def.Privacy = PrivacyPrivate
		def.SupportsSearch = true
		def.SupportsRSS = true
	case genericrss.DefinitionID:
		def.Protocol = ProtocolTorrent
		def.Privacy = PrivacyPrivate
		def.SupportsSearch = false
		def.SupportsRSS = true
	default:
		if cardDef, err := s.manager.GetDefinition(definitionID); err == nil {
			def.Protocol = Protocol(cardDef.GetProtocol())
			def.Privacy = Privacy(cardDef.GetPrivacy())
			def.SupportsSearch = cardDef.SupportsSearch("search")
			def.SupportsRSS = true
		} else {
			def.Protocol = ProtocolTorrent
			def.Privacy = PrivacyPublic
			def.SupportsSearch = true
			def.SupportsRSS = true
		}
	}
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
