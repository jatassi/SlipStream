package importer

import (
	"context"
	"errors"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/module"
)

// ErrNotNamingProvider is returned when a module does not implement NamingProvider.
var ErrNotNamingProvider = errors.New("module does not implement NamingProvider")

// GetNamingSettings returns the current module naming settings as a key-value map.
func (s *Service) GetNamingSettings(ctx context.Context, moduleType string) (map[string]string, error) {
	rows, err := s.queries.ListModuleNamingSettings(ctx, moduleType)
	if err != nil {
		return nil, err
	}
	settings := make(map[string]string, len(rows))
	for _, row := range rows {
		settings[row.SettingKey] = row.SettingValue
	}
	return settings, nil
}

// UpsertNamingSettings writes module naming settings as key-value pairs.
func (s *Service) UpsertNamingSettings(ctx context.Context, moduleType string, settings map[string]string) error {
	for key, val := range settings {
		if err := s.queries.UpsertModuleNamingSetting(ctx, sqlc.UpsertModuleNamingSettingParams{
			ModuleType:   moduleType,
			SettingKey:   key,
			SettingValue: val,
		}); err != nil {
			return err
		}
	}
	return nil
}

// LoadModuleRenamer builds a Resolver for the given module by reading its
// naming settings from the database and falling back to module defaults.
func LoadModuleRenamer(ctx context.Context, queries *sqlc.Queries, mod module.Module) (*renamer.Resolver, error) {
	rows, err := queries.ListModuleNamingSettings(ctx, string(mod.ID()))
	if err != nil {
		return nil, err
	}

	dbSettings := make(map[string]string)
	for _, row := range rows {
		dbSettings[row.SettingKey] = row.SettingValue
	}

	namingProvider, ok := mod.(module.NamingProvider)
	if !ok {
		return nil, ErrNotNamingProvider
	}
	defaults := namingProvider.DefaultFileTemplates()

	settings := &renamer.Settings{
		RenameEpisodes:           true,
		RenameMovies:             true,
		ReplaceIllegalCharacters: true,
		Patterns:                 make(map[string]string),
	}

	for key, defaultVal := range defaults {
		if dbVal, ok := dbSettings[key]; ok {
			settings.Patterns[key] = dbVal
		} else {
			settings.Patterns[key] = defaultVal
		}
	}

	if v, ok := dbSettings["rename_enabled"]; ok {
		enabled := v == boolTrue
		settings.RenameEpisodes = enabled
		settings.RenameMovies = enabled
	}
	if v, ok := dbSettings["colon_replacement"]; ok {
		settings.ColonReplacement = renamer.ColonReplacement(v)
	}
	if v, ok := dbSettings["custom_colon_replacement"]; ok {
		settings.CustomColonReplacement = v
	}
	if v, ok := dbSettings["multi_episode_style"]; ok {
		settings.MultiEpisodeStyle = renamer.MultiEpisodeStyle(v)
	}

	return renamer.NewResolver(settings), nil
}
