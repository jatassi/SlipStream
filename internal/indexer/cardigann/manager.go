package cardigann

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/indexer/types"
)

// Manager manages Cardigann definitions and indexer clients.
type Manager struct {
	repo           *Repository
	cache          *Cache
	clients        map[int64]*Client
	clientsMu      sync.RWMutex
	logger         *zerolog.Logger
	lastUpdate     time.Time
	lastAttempt    time.Time // Tracks last update attempt (even if failed)
	updateMu       sync.Mutex
	autoUpdate     bool
	updateInterval time.Duration
	cookieStore    CookieStore
	modeCheckFunc  func() bool // Returns true if SlipStream mode (definitions should be used)
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
func NewManager(cfg *ManagerConfig, logger *zerolog.Logger) (*Manager, error) {
	repo := NewRepository(&cfg.Repository, logger)
	cache, err := NewCache(&cfg.Cache, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	subLogger := logger.With().Str("component", "manager").Logger()
	return &Manager{
		repo:           repo,
		cache:          cache,
		clients:        make(map[int64]*Client),
		logger:         &subLogger,
		autoUpdate:     cfg.AutoUpdate,
		updateInterval: cfg.UpdateInterval,
	}, nil
}

// SetCookieStore sets the cookie store for persistent session cookies.
func (m *Manager) SetCookieStore(store CookieStore) {
	m.cookieStore = store
}

// SetModeCheckFunc sets a function that returns true when SlipStream mode is active.
// When in Prowlarr mode, definition updates are skipped.
func (m *Manager) SetModeCheckFunc(fn func() bool) {
	m.modeCheckFunc = fn
}

// isSlipStreamMode returns true if we should use SlipStream definitions.
// Defaults to true if no mode check function is set.
func (m *Manager) isSlipStreamMode() bool {
	if m.modeCheckFunc == nil {
		return true
	}
	return m.modeCheckFunc()
}

// Initialize loads definitions and optionally updates from remote.
func (m *Manager) Initialize(ctx context.Context) error {
	// Load last update time from disk
	m.loadLastUpdateTime()

	// Try to update from remote if in SlipStream mode
	if m.autoUpdate && m.isSlipStreamMode() {
		if err := m.UpdateDefinitions(ctx); err != nil {
			m.logger.Warn().Err(err).Msg("Failed to update definitions from remote, using cached versions")
		}
	} else if !m.isSlipStreamMode() {
		m.logger.Debug().Msg("Skipping definition update: Prowlarr mode is active")
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

	// Skip if not in SlipStream mode
	if !m.isSlipStreamMode() {
		m.logger.Debug().Msg("Skipping definition update: Prowlarr mode is active")
		return nil
	}

	// Check if we've updated successfully within the configured interval (default 24h)
	if time.Since(m.lastUpdate) < m.updateInterval {
		m.logger.Debug().
			Time("lastUpdate", m.lastUpdate).
			Dur("interval", m.updateInterval).
			Msg("Skipping update, updated within interval")
		return nil
	}

	// Also throttle attempts (even failed ones) to once per day
	if time.Since(m.lastAttempt) < m.updateInterval {
		m.logger.Debug().
			Time("lastAttempt", m.lastAttempt).
			Msg("Skipping update, attempted recently")
		return nil
	}

	m.lastAttempt = time.Now()
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
	m.saveLastUpdateTime()
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
		if !matchesTextQuery(meta, query) || !matchesFilters(meta, &filters) {
			continue
		}
		results = append(results, meta)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results, nil
}

func matchesTextQuery(meta *DefinitionMetadata, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(meta.Name), query) ||
		strings.Contains(strings.ToLower(meta.Description), query) ||
		strings.Contains(strings.ToLower(meta.ID), query)
}

func matchesFilters(meta *DefinitionMetadata, filters *DefinitionFilters) bool {
	if filters.Protocol != "" && meta.Protocol != filters.Protocol {
		return false
	}
	if filters.Privacy != "" && meta.Type != filters.Privacy {
		return false
	}
	if filters.Language != "" && meta.Language != filters.Language {
		return false
	}
	return true
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
	var defID = strings.ToLower(strings.ReplaceAll(indexerDef.Name, " ", ""))

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
		Definition:  def,
		IndexerDef:  indexerDef,
		Settings:    settings,
		Logger:      m.logger,
		CookieStore: m.cookieStore,
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

// lastUpdateFileName is the name of the file storing the last update timestamp.
const lastUpdateFileName = ".last_update"

// loadLastUpdateTime loads the last update time from disk.
func (m *Manager) loadLastUpdateTime() {
	filePath := filepath.Join(m.cache.GetDefinitionsDir(), lastUpdateFileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		// File doesn't exist or can't be read, use zero time
		return
	}

	t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		m.logger.Warn().Err(err).Msg("Failed to parse last update time")
		return
	}

	m.lastUpdate = t
	m.logger.Debug().Time("lastUpdate", t).Msg("Loaded last update time from disk")
}

// saveLastUpdateTime saves the last update time to disk.
func (m *Manager) saveLastUpdateTime() {
	filePath := filepath.Join(m.cache.GetDefinitionsDir(), lastUpdateFileName)
	data := m.lastUpdate.Format(time.RFC3339)
	if err := os.WriteFile(filePath, []byte(data), 0o600); err != nil {
		m.logger.Warn().Err(err).Msg("Failed to save last update time")
	}
}
