package defaults

import (
	"context"
	"fmt"
	"strconv"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// MediaType represents the media type for defaults
type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
)

// EntityType represents the type of entity for defaults
type EntityType string

const (
	EntityTypeRootFolder     EntityType = "root_folder"
	EntityTypeQualityProfile EntityType = "quality_profile"
	EntityTypeDownloadClient EntityType = "download_client"
	EntityTypeIndexer        EntityType = "indexer"
)

// DefaultEntry represents a default setting
type DefaultEntry struct {
	Key        string `json:"key"`
	EntityType string `json:"entityType"`
	MediaType  string `json:"mediaType"`
	EntityID   int64  `json:"entityId"`
}

// Service provides default management operations
type Service struct {
	queries *sqlc.Queries
}

// NewService creates a new defaults service
func NewService(queries *sqlc.Queries) *Service {
	return &Service{queries: queries}
}

// GetDefault returns the default entity for the given entity type and media type
func (s *Service) GetDefault(ctx context.Context, entityType EntityType, mediaType MediaType) (*DefaultEntry, error) {
	key := s.buildKey(entityType, mediaType)
	setting, err := s.queries.GetSetting(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get default: %w", err)
	}

	// Parse the value (should be entity ID as string)
	entityID, err := strconv.ParseInt(setting.Value, 10, 64)
	if err != nil {
		// If value is empty or invalid, return nil indicating no default set
		if setting.Value == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("invalid entity ID in default setting: %w", err)
	}

	return &DefaultEntry{
		Key:        setting.Key,
		EntityType: string(entityType),
		MediaType:  string(mediaType),
		EntityID:   entityID,
	}, nil
}

// SetDefault sets the default entity for the given entity type and media type
func (s *Service) SetDefault(ctx context.Context, entityType EntityType, mediaType MediaType, entityID int64) error {
	key := s.buildKey(entityType, mediaType)
	value := strconv.FormatInt(entityID, 10)

	_, err := s.queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   key,
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("failed to set default: %w", err)
	}

	return nil
}

// ClearDefault clears the default for the given entity type and media type
func (s *Service) ClearDefault(ctx context.Context, entityType EntityType, mediaType MediaType) error {
	key := s.buildKey(entityType, mediaType)
	err := s.queries.DeleteSetting(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to clear default: %w", err)
	}

	return nil
}

// GetAllDefaults returns all default settings
func (s *Service) GetAllDefaults(ctx context.Context) ([]*DefaultEntry, error) {
	settings, err := s.queries.ListSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all defaults: %w", err)
	}

	var defaults []*DefaultEntry
	for _, setting := range settings {
		if len(setting.Key) < 9 || setting.Key[:8] != "default_" {
			continue // Skip non-default settings
		}

		entityType, mediaType := s.parseKey(setting.Key)

		// Skip if key doesn't parse correctly
		if entityType == "" || mediaType == "" {
			continue
		}

		// Parse the value
		entityID, err := strconv.ParseInt(setting.Value, 10, 64)
		if err != nil {
			// Skip invalid values
			continue
		}

		defaults = append(defaults, &DefaultEntry{
			Key:        setting.Key,
			EntityType: entityType,
			MediaType:  mediaType,
			EntityID:   entityID,
		})
	}

	return defaults, nil
}

// GetDefaultsForEntityType returns all defaults for a specific entity type
func (s *Service) GetDefaultsForEntityType(ctx context.Context, entityType EntityType) ([]*DefaultEntry, error) {
	settings, err := s.queries.ListSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get defaults for entity type: %w", err)
	}

	prefix := fmt.Sprintf("default_%s_", entityType)
	var defaults []*DefaultEntry

	for _, setting := range settings {
		if len(setting.Key) < len(prefix) || setting.Key[:len(prefix)] != prefix {
			continue // Skip non-matching settings
		}

		_, mediaType := s.parseKey(setting.Key)

		// Skip if key doesn't parse correctly
		if mediaType == "" {
			continue
		}

		// Parse the value
		entityID, err := strconv.ParseInt(setting.Value, 10, 64)
		if err != nil {
			// Skip invalid values
			continue
		}

		defaults = append(defaults, &DefaultEntry{
			Key:        setting.Key,
			EntityType: string(entityType),
			MediaType:  mediaType,
			EntityID:   entityID,
		})
	}

	return defaults, nil
}

// buildKey creates the settings key for the given entity type and media type
func (s *Service) buildKey(entityType EntityType, mediaType MediaType) string {
	return fmt.Sprintf("default_%s_%s", entityType, mediaType)
}

// parseKey extracts the entity type and media type from a settings key
func (s *Service) parseKey(key string) (string, string) {
	// Expected format: default_{entity_type}_{media_type}
	if len(key) < 9 || key[:8] != "default_" {
		return "", ""
	}

	remaining := key[8:] // Remove "default_"

	// Find the last underscore to split entity type and media type
	lastUnderscore := -1
	for i := 0; i < len(remaining); i++ {
		if remaining[i] == '_' {
			lastUnderscore = i
		}
	}

	if lastUnderscore <= 0 || lastUnderscore == len(remaining)-1 {
		return "", ""
	}

	entityType := remaining[:lastUnderscore]
	mediaType := remaining[lastUnderscore+1:]

	return entityType, mediaType
}
