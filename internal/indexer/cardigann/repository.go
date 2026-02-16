package cardigann

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Repository handles fetching and updating Cardigann definitions from remote sources.
type Repository struct {
	httpClient *http.Client
	logger     *zerolog.Logger
	config     RepositoryConfig
}

// RepositoryConfig contains configuration for the definition repository.
type RepositoryConfig struct {
	BaseURL        string        // Default: "https://indexers.prowlarr.com"
	Branch         string        // Default: "master"
	Version        string        // Default: "v10"
	RequestTimeout time.Duration // Default: 60s
	UserAgent      string        // Default: "SlipStream/1.0"
}

// DefaultRepositoryConfig returns the default repository configuration.
func DefaultRepositoryConfig() RepositoryConfig {
	return RepositoryConfig{
		BaseURL:        "https://indexers.prowlarr.com",
		Branch:         "master",
		Version:        "11",
		RequestTimeout: 60 * time.Second,
		UserAgent:      "SlipStream/1.0",
	}
}

// DefinitionMetadata contains metadata about a definition without the full content.
type DefinitionMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // public, private, semi-private
	Language    string `json:"language"`
	Protocol    string `json:"protocol"` // torrent, usenet
}

// NewRepository creates a new definition repository.
func NewRepository(cfg *RepositoryConfig, logger *zerolog.Logger) *Repository {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultRepositoryConfig().BaseURL
	}
	if cfg.Branch == "" {
		cfg.Branch = DefaultRepositoryConfig().Branch
	}
	if cfg.Version == "" {
		cfg.Version = DefaultRepositoryConfig().Version
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = DefaultRepositoryConfig().RequestTimeout
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultRepositoryConfig().UserAgent
	}

	subLogger := logger.With().Str("component", "repository").Logger()
	return &Repository{
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
		logger: &subLogger,
		config: *cfg,
	}
}

// buildURL constructs a URL for the repository.
func (r *Repository) buildURL(path string) string {
	return fmt.Sprintf("%s/%s/%s/%s", r.config.BaseURL, r.config.Branch, r.config.Version, path)
}

// FetchDefinitionList retrieves the list of available definitions from the remote repository.
func (r *Repository) FetchDefinitionList(ctx context.Context) ([]DefinitionMetadata, error) {
	url := r.buildURL("")
	r.logger.Debug().Str("url", url).Msg("Fetching definition list")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", r.config.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch definition list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var metadata []DefinitionMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	r.logger.Info().Int("count", len(metadata)).Msg("Fetched definition list")
	return metadata, nil
}

// FetchDefinition retrieves a single definition by ID.
func (r *Repository) FetchDefinition(ctx context.Context, id string) (*Definition, error) {
	url := r.buildURL(id)
	r.logger.Debug().Str("url", url).Str("id", id).Msg("Fetching definition")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", r.config.UserAgent)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch definition: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("definition not found: %s", id)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	def, err := ParseDefinition(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse definition: %w", err)
	}

	r.logger.Debug().Str("id", id).Str("name", def.Name).Msg("Fetched definition")
	return def, nil
}

// FetchPackage downloads and extracts all definitions as a ZIP package.
// Returns a map of definition ID to raw YAML content.
func (r *Repository) FetchPackage(ctx context.Context) (map[string][]byte, error) {
	url := r.buildURL("package.zip")
	r.logger.Info().Str("url", url).Str("version", r.config.Version).Msg("Fetching definition package")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", r.config.UserAgent)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		r.logger.Error().Int("status", resp.StatusCode).Str("body", string(body)).Str("url", url).Msg("Failed to fetch package")
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the entire ZIP into memory
	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read package: %w", err)
	}

	r.logger.Debug().Int("size", len(zipData)).Msg("Downloaded package")

	// Extract definitions from ZIP
	definitions, err := r.extractPackage(zipData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract package: %w", err)
	}

	r.logger.Info().Int("count", len(definitions)).Msg("Extracted definitions from package")
	return definitions, nil
}

// extractPackage extracts YAML definitions from a ZIP archive.
func (r *Repository) extractPackage(zipData []byte) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP: %w", err)
	}

	definitions := make(map[string][]byte)

	for _, file := range reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Only process YAML files
		ext := strings.ToLower(filepath.Ext(file.Name))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		// Extract definition ID from filename
		baseName := filepath.Base(file.Name)
		id := strings.TrimSuffix(baseName, ext)

		// Read file contents
		rc, err := file.Open()
		if err != nil {
			r.logger.Warn().Str("file", file.Name).Err(err).Msg("Failed to open file in ZIP")
			continue
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			r.logger.Warn().Str("file", file.Name).Err(err).Msg("Failed to read file in ZIP")
			continue
		}

		definitions[id] = content
	}

	return definitions, nil
}

// FetchDefinitionRaw retrieves raw YAML content for a definition.
func (r *Repository) FetchDefinitionRaw(ctx context.Context, id string) ([]byte, error) {
	url := r.buildURL(id)
	r.logger.Debug().Str("url", url).Str("id", id).Msg("Fetching raw definition")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", r.config.UserAgent)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch definition: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("definition not found: %s", id)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// GetConfig returns the current repository configuration.
func (r *Repository) GetConfig() RepositoryConfig {
	return r.config
}
