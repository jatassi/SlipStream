package cardigann

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/indexer/types"
)

// Manager manages Cardigann definitions and indexer clients.
type Manager struct {
	repo          *Repository
	cache         *Cache
	clients       map[int64]*Client
	clientsMu     sync.RWMutex
	logger        zerolog.Logger
	lastUpdate    time.Time
	updateMu      sync.Mutex
	autoUpdate    bool
	updateInterval time.Duration
}

// ManagerConfig contains configuration for the definition manager.
type ManagerConfig struct {
	Repository     RepositoryConfig
	Cache          CacheConfig
	AutoUpdate     bool          // Default: true
	UpdateInterval time.Duration // Default: 24h
}

// DefaultManagerConfig returns the default manager configuration.
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		Repository:     DefaultRepositoryConfig(),
		Cache:          DefaultCacheConfig(),
		AutoUpdate:     true,
		UpdateInterval: 24 * time.Hour,
	}
}

// NewManager creates a new definition manager.
func NewManager(cfg ManagerConfig, logger zerolog.Logger) (*Manager, error) {
	repo := NewRepository(cfg.Repository, logger)
	cache, err := NewCache(cfg.Cache, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return &Manager{
		repo:           repo,
		cache:          cache,
		clients:        make(map[int64]*Client),
		logger:         logger.With().Str("component", "manager").Logger(),
		autoUpdate:     cfg.AutoUpdate,
		updateInterval: cfg.UpdateInterval,
	}, nil
}

// Initialize loads definitions and optionally updates from remote.
func (m *Manager) Initialize(ctx context.Context) error {
	// Try to update from remote
	if m.autoUpdate {
		if err := m.UpdateDefinitions(ctx); err != nil {
			m.logger.Warn().Err(err).Msg("Failed to update definitions from remote, using cached versions")
		}
	}

	// Load definitions from cache
	count, err := m.cache.Count()
	if err != nil {
		return fmt.Errorf("failed to count cached definitions: %w", err)
	}

	m.logger.Info().Int("count", count).Msg("Initialized definition manager")
	return nil
}

// UpdateDefinitions fetches the latest definitions from the remote repository.
func (m *Manager) UpdateDefinitions(ctx context.Context) error {
	m.updateMu.Lock()
	defer m.updateMu.Unlock()

	// Check if we've updated recently
	if time.Since(m.lastUpdate) < time.Minute {
		m.logger.Debug().Msg("Skipping update, updated recently")
		return nil
	}

	m.logger.Info().Msg("Updating definitions from remote repository")

	// Fetch the package
	definitions, err := m.repo.FetchPackage(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch definitions package: %w", err)
	}

	// Store all definitions
	if err := m.cache.StoreAll(definitions); err != nil {
		return fmt.Errorf("failed to store definitions: %w", err)
	}

	m.lastUpdate = time.Now()
	m.logger.Info().Int("count", len(definitions)).Msg("Updated definitions from remote")

	return nil
}

// GetDefinition retrieves a definition by ID.
func (m *Manager) GetDefinition(id string) (*Definition, error) {
	return m.cache.Get(id)
}

// GetDefinitionMetadata retrieves metadata for a definition.
func (m *Manager) GetDefinitionMetadata(id string) (*DefinitionMetadata, error) {
	return m.cache.GetMetadata(id)
}

// ListDefinitions returns metadata for all available definitions.
func (m *Manager) ListDefinitions() ([]*DefinitionMetadata, error) {
	return m.cache.List()
}

// SearchDefinitions searches for definitions matching the query.
func (m *Manager) SearchDefinitions(query string, filters DefinitionFilters) ([]*DefinitionMetadata, error) {
	all, err := m.cache.List()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []*DefinitionMetadata

	for _, meta := range all {
		// Apply text search
		if query != "" {
			nameMatch := strings.Contains(strings.ToLower(meta.Name), query)
			descMatch := strings.Contains(strings.ToLower(meta.Description), query)
			idMatch := strings.Contains(strings.ToLower(meta.ID), query)
			if !nameMatch && !descMatch && !idMatch {
				continue
			}
		}

		// Apply filters
		if filters.Protocol != "" && meta.Protocol != filters.Protocol {
			continue
		}
		if filters.Privacy != "" && meta.Type != filters.Privacy {
			continue
		}
		if filters.Language != "" && meta.Language != filters.Language {
			continue
		}

		results = append(results, meta)
	}

	// Sort by name
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results, nil
}

// DefinitionFilters contains filter options for searching definitions.
type DefinitionFilters struct {
	Protocol string // torrent, usenet
	Privacy  string // public, private, semi-private
	Language string // en-US, etc.
}

// GetSettingsSchema returns the settings schema for a definition.
func (m *Manager) GetSettingsSchema(id string) ([]Setting, error) {
	def, err := m.cache.Get(id)
	if err != nil {
		return nil, err
	}
	return def.Settings, nil
}

// CreateClient creates a new indexer client for a configured indexer.
func (m *Manager) CreateClient(indexerDef *types.IndexerDefinition, settings map[string]string) (*Client, error) {
	// Get the Cardigann definition
	// The definition_id is stored in the Settings JSON
	var defID string

	// For now, we'll use the indexer name to look up the definition
	// In the new schema, we'll have a dedicated definition_id field
	defID = strings.ToLower(strings.ReplaceAll(indexerDef.Name, " ", ""))

	def, err := m.cache.Get(defID)
	if err != nil {
		return nil, fmt.Errorf("definition not found: %w", err)
	}

	client, err := NewClient(ClientConfig{
		Definition: def,
		IndexerDef: indexerDef,
		Settings:   settings,
		Logger:     m.logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

// CreateClientFromDefinition creates a new indexer client from a definition ID.
func (m *Manager) CreateClientFromDefinition(defID string, indexerID int64, name string, settings map[string]string) (*Client, error) {
	def, err := m.cache.Get(defID)
	if err != nil {
		return nil, fmt.Errorf("definition not found: %w", err)
	}

	// Build a minimal IndexerDefinition
	indexerDef := &types.IndexerDefinition{
		ID:             indexerID,
		Name:           name,
		Protocol:       types.Protocol(def.GetProtocol()),
		Privacy:        types.Privacy(def.GetPrivacy()),
		SupportsSearch: def.SupportsSearch("search"),
		SupportsMovies: def.SupportsSearch("movie-search"),
		SupportsTV:     def.SupportsSearch("tv-search"),
		SupportsRSS:    true,
	}

	client, err := NewClient(ClientConfig{
		Definition: def,
		IndexerDef: indexerDef,
		Settings:   settings,
		Logger:     m.logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

// RegisterClient registers a client for an indexer ID.
func (m *Manager) RegisterClient(indexerID int64, client *Client) {
	m.clientsMu.Lock()
	m.clients[indexerID] = client
	m.clientsMu.Unlock()
}

// GetClient retrieves a registered client by indexer ID.
func (m *Manager) GetClient(indexerID int64) (*Client, bool) {
	m.clientsMu.RLock()
	client, ok := m.clients[indexerID]
	m.clientsMu.RUnlock()
	return client, ok
}

// RemoveClient removes a registered client.
func (m *Manager) RemoveClient(indexerID int64) {
	m.clientsMu.Lock()
	delete(m.clients, indexerID)
	m.clientsMu.Unlock()
}

// TestDefinition tests a definition with the given settings.
func (m *Manager) TestDefinition(ctx context.Context, defID string, settings map[string]string) error {
	client, err := m.CreateClientFromDefinition(defID, 0, "Test", settings)
	if err != nil {
		return fmt.Errorf("failed to create test client: %w", err)
	}

	return client.Test(ctx)
}

// GetCapabilities returns the capabilities for a definition.
func (m *Manager) GetCapabilities(defID string) (*types.Capabilities, error) {
	def, err := m.cache.Get(defID)
	if err != nil {
		return nil, err
	}

	// Create a temporary client to get capabilities
	client, err := NewClient(ClientConfig{
		Definition: def,
		Logger:     m.logger,
	})
	if err != nil {
		return nil, err
	}

	return client.Capabilities(), nil
}

// NeedsUpdate checks if definitions should be updated.
func (m *Manager) NeedsUpdate() bool {
	if !m.autoUpdate {
		return false
	}
	return time.Since(m.lastUpdate) > m.updateInterval
}

// GetLastUpdate returns the time of the last definition update.
func (m *Manager) GetLastUpdate() time.Time {
	return m.lastUpdate
}

// GetRepository returns the repository instance.
func (m *Manager) GetRepository() *Repository {
	return m.repo
}

// GetCache returns the cache instance.
func (m *Manager) GetCache() *Cache {
	return m.cache
}

// Close cleans up resources.
func (m *Manager) Close() error {
	m.clientsMu.Lock()
	m.clients = make(map[int64]*Client)
	m.clientsMu.Unlock()
	return nil
}
