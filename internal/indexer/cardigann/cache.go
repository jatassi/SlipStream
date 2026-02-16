package cardigann

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const (
	fileExtYML  = ".yml"
	fileExtYAML = ".yaml"
)

// Cache manages local caching of Cardigann definitions.
type Cache struct {
	definitionsDir string
	customDir      string
	memoryCache    map[string]*cachedDefinition
	metadataCache  map[string]*DefinitionMetadata
	mu             sync.RWMutex
	logger         *zerolog.Logger
}

// cachedDefinition holds a parsed definition with metadata.
type cachedDefinition struct {
	Definition *Definition
	LoadedAt   time.Time
	FilePath   string
	IsCustom   bool
}

// CacheConfig contains configuration for the definition cache.
type CacheConfig struct {
	DefinitionsDir string // Default: "./data/definitions"
	CustomDir      string // Default: "./data/definitions/custom"
}

// DefaultCacheConfig returns the default cache configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		DefinitionsDir: "./data/definitions",
		CustomDir:      "./data/definitions/custom",
	}
}

// NewCache creates a new definition cache.
func NewCache(cfg *CacheConfig, logger *zerolog.Logger) (*Cache, error) {
	if cfg.DefinitionsDir == "" {
		cfg.DefinitionsDir = DefaultCacheConfig().DefinitionsDir
	}
	if cfg.CustomDir == "" {
		cfg.CustomDir = DefaultCacheConfig().CustomDir
	}

	// Ensure directories exist
	if err := os.MkdirAll(cfg.DefinitionsDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create definitions directory: %w", err)
	}
	if err := os.MkdirAll(cfg.CustomDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create custom directory: %w", err)
	}

	subLogger := logger.With().Str("component", "cache").Logger()
	return &Cache{
		definitionsDir: cfg.DefinitionsDir,
		customDir:      cfg.CustomDir,
		memoryCache:    make(map[string]*cachedDefinition),
		metadataCache:  make(map[string]*DefinitionMetadata),
		logger:         &subLogger,
	}, nil
}

// Get retrieves a definition by ID, first checking memory cache, then disk.
func (c *Cache) Get(id string) (*Definition, error) {
	c.mu.RLock()
	if cached, ok := c.memoryCache[id]; ok {
		c.mu.RUnlock()
		return cached.Definition, nil
	}
	c.mu.RUnlock()

	// Try to load from disk
	def, isCustom, err := c.loadFromDisk(id)
	if err != nil {
		return nil, err
	}

	// Cache in memory
	c.mu.Lock()
	c.memoryCache[id] = &cachedDefinition{
		Definition: def,
		LoadedAt:   time.Now(),
		FilePath:   c.getFilePath(id, isCustom),
		IsCustom:   isCustom,
	}
	c.mu.Unlock()

	return def, nil
}

// GetMetadata retrieves metadata for a definition.
func (c *Cache) GetMetadata(id string) (*DefinitionMetadata, error) {
	c.mu.RLock()
	if meta, ok := c.metadataCache[id]; ok {
		c.mu.RUnlock()
		return meta, nil
	}
	c.mu.RUnlock()

	// Load the full definition to get metadata
	def, err := c.Get(id)
	if err != nil {
		return nil, err
	}

	meta := &DefinitionMetadata{
		ID:          def.ID,
		Name:        def.Name,
		Description: def.Description,
		Type:        def.Type,
		Language:    def.Language,
		Protocol:    def.GetProtocol(),
	}

	c.mu.Lock()
	c.metadataCache[id] = meta
	c.mu.Unlock()

	return meta, nil
}

// List returns metadata for all cached definitions.
func (c *Cache) List() ([]*DefinitionMetadata, error) {
	result := make([]*DefinitionMetadata, 0)
	seen := make(map[string]bool)

	// Load custom definitions first (they take priority)
	customDefs, err := c.listDirectory(c.customDir, true)
	if err != nil {
		c.logger.Warn().Err(err).Msg("Failed to list custom definitions")
	} else {
		for _, def := range customDefs {
			result = append(result, def)
			seen[def.ID] = true
		}
	}

	// Load standard definitions
	standardDefs, err := c.listDirectory(c.definitionsDir, false)
	if err != nil {
		c.logger.Warn().Err(err).Msg("Failed to list standard definitions")
	} else {
		for _, def := range standardDefs {
			if !seen[def.ID] {
				result = append(result, def)
				seen[def.ID] = true
			}
		}
	}

	return result, nil
}

// listDirectory lists definitions in a directory.
func (c *Cache) listDirectory(dir string, isCustom bool) ([]*DefinitionMetadata, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return make([]*DefinitionMetadata, 0), nil
		}
		return nil, err
	}

	result := make([]*DefinitionMetadata, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != fileExtYML && ext != fileExtYAML {
			continue
		}

		id := strings.TrimSuffix(name, ext)

		// Try to get from memory cache first
		c.mu.RLock()
		cached, ok := c.metadataCache[id]
		c.mu.RUnlock()

		if ok {
			result = append(result, cached)
			continue
		}

		// Load and parse the definition
		filePath := filepath.Join(dir, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			c.logger.Warn().Str("file", filePath).Err(err).Msg("Failed to read definition file")
			continue
		}

		def, err := ParseDefinition(data)
		if err != nil {
			c.logger.Warn().Str("file", filePath).Err(err).Msg("Failed to parse definition")
			continue
		}

		meta := &DefinitionMetadata{
			ID:          def.ID,
			Name:        def.Name,
			Description: def.Description,
			Type:        def.Type,
			Language:    def.Language,
			Protocol:    def.GetProtocol(),
		}

		// Cache metadata
		c.mu.Lock()
		c.metadataCache[id] = meta
		c.memoryCache[id] = &cachedDefinition{
			Definition: def,
			LoadedAt:   time.Now(),
			FilePath:   filePath,
			IsCustom:   isCustom,
		}
		c.mu.Unlock()

		result = append(result, meta)
	}

	return result, nil
}

// Store saves a definition to the cache.
func (c *Cache) Store(id string, data []byte) error {
	// Parse to validate
	def, err := ParseDefinition(data)
	if err != nil {
		return fmt.Errorf("invalid definition: %w", err)
	}

	// Write to disk
	filePath := filepath.Join(c.definitionsDir, id+".yml")
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write definition: %w", err)
	}

	// Update memory cache
	c.mu.Lock()
	c.memoryCache[id] = &cachedDefinition{
		Definition: def,
		LoadedAt:   time.Now(),
		FilePath:   filePath,
		IsCustom:   false,
	}
	c.metadataCache[id] = &DefinitionMetadata{
		ID:          def.ID,
		Name:        def.Name,
		Description: def.Description,
		Type:        def.Type,
		Language:    def.Language,
		Protocol:    def.GetProtocol(),
	}
	c.mu.Unlock()

	c.logger.Debug().Str("id", id).Str("path", filePath).Msg("Stored definition")
	return nil
}

// StoreCustom saves a custom definition.
func (c *Cache) StoreCustom(id string, data []byte) error {
	// Parse to validate
	def, err := ParseDefinition(data)
	if err != nil {
		return fmt.Errorf("invalid definition: %w", err)
	}

	// Write to custom directory
	filePath := filepath.Join(c.customDir, id+".yml")
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write custom definition: %w", err)
	}

	// Update memory cache
	c.mu.Lock()
	c.memoryCache[id] = &cachedDefinition{
		Definition: def,
		LoadedAt:   time.Now(),
		FilePath:   filePath,
		IsCustom:   true,
	}
	c.metadataCache[id] = &DefinitionMetadata{
		ID:          def.ID,
		Name:        def.Name,
		Description: def.Description,
		Type:        def.Type,
		Language:    def.Language,
		Protocol:    def.GetProtocol(),
	}
	c.mu.Unlock()

	c.logger.Debug().Str("id", id).Str("path", filePath).Msg("Stored custom definition")
	return nil
}

// StoreAll stores multiple definitions from a package.
func (c *Cache) StoreAll(definitions map[string][]byte) error {
	stored := 0
	failed := 0

	for id, data := range definitions {
		if err := c.Store(id, data); err != nil {
			c.logger.Warn().Str("id", id).Err(err).Msg("Failed to store definition")
			failed++
			continue
		}
		stored++
	}

	c.logger.Info().Int("stored", stored).Int("failed", failed).Msg("Stored definitions from package")
	return nil
}

// Delete removes a definition from the cache.
func (c *Cache) Delete(id string) error {
	c.mu.Lock()
	cached, ok := c.memoryCache[id]
	if ok {
		delete(c.memoryCache, id)
		delete(c.metadataCache, id)
	}
	c.mu.Unlock()

	// Delete from disk
	if ok && cached.FilePath != "" {
		if err := os.Remove(cached.FilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete definition file: %w", err)
		}
	}

	return nil
}

// Clear removes all cached definitions from memory (keeps disk cache).
func (c *Cache) Clear() {
	c.mu.Lock()
	c.memoryCache = make(map[string]*cachedDefinition)
	c.metadataCache = make(map[string]*DefinitionMetadata)
	c.mu.Unlock()

	c.logger.Debug().Msg("Cleared memory cache")
}

// ClearDisk removes all cached definitions from disk.
func (c *Cache) ClearDisk() error {
	// Clear standard definitions (not custom)
	entries, err := os.ReadDir(c.definitionsDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read definitions directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != fileExtYML && ext != fileExtYAML {
			continue
		}

		filePath := filepath.Join(c.definitionsDir, name)
		if err := os.Remove(filePath); err != nil {
			c.logger.Warn().Str("file", filePath).Err(err).Msg("Failed to delete definition file")
		}
	}

	// Clear memory cache
	c.Clear()

	c.logger.Info().Msg("Cleared disk cache")
	return nil
}

// Exists checks if a definition exists in the cache.
func (c *Cache) Exists(id string) bool {
	c.mu.RLock()
	if _, ok := c.memoryCache[id]; ok {
		c.mu.RUnlock()
		return true
	}
	c.mu.RUnlock()

	// Check disk
	customPath := filepath.Join(c.customDir, id+".yml")
	if _, err := os.Stat(customPath); err == nil {
		return true
	}

	standardPath := filepath.Join(c.definitionsDir, id+".yml")
	if _, err := os.Stat(standardPath); err == nil {
		return true
	}

	return false
}

// Count returns the number of cached definitions.
func (c *Cache) Count() (int, error) {
	defs, err := c.List()
	if err != nil {
		return 0, err
	}
	return len(defs), nil
}

// loadFromDisk loads a definition from disk.
func (c *Cache) loadFromDisk(id string) (*Definition, bool, error) {
	// Try custom directory first
	customPath := filepath.Join(c.customDir, id+".yml")
	if data, err := os.ReadFile(customPath); err == nil {
		def, err := ParseDefinition(data)
		if err != nil {
			return nil, false, fmt.Errorf("failed to parse custom definition: %w", err)
		}
		return def, true, nil
	}

	// Try standard directory
	standardPath := filepath.Join(c.definitionsDir, id+".yml")
	if data, err := os.ReadFile(standardPath); err == nil {
		def, err := ParseDefinition(data)
		if err != nil {
			return nil, false, fmt.Errorf("failed to parse definition: %w", err)
		}
		return def, false, nil
	}

	return nil, false, fmt.Errorf("definition not found: %s", id)
}

// getFilePath returns the file path for a definition.
func (c *Cache) getFilePath(id string, isCustom bool) string {
	if isCustom {
		return filepath.Join(c.customDir, id+".yml")
	}
	return filepath.Join(c.definitionsDir, id+".yml")
}

// GetDefinitionsDir returns the definitions directory path.
func (c *Cache) GetDefinitionsDir() string {
	return c.definitionsDir
}

// GetCustomDir returns the custom definitions directory path.
func (c *Cache) GetCustomDir() string {
	return c.customDir
}

// IsCustom checks if a definition is from the custom directory.
func (c *Cache) IsCustom(id string) bool {
	c.mu.RLock()
	if cached, ok := c.memoryCache[id]; ok {
		c.mu.RUnlock()
		return cached.IsCustom
	}
	c.mu.RUnlock()

	// Check disk
	customPath := filepath.Join(c.customDir, id+".yml")
	if _, err := os.Stat(customPath); err == nil {
		return true
	}

	return false
}
