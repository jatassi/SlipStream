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
		enabled := v == "true"
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

	PopulateLegacyFields(settings, mod.ID())

	return renamer.NewResolver(settings), nil
}

// PopulateLegacyFields copies Patterns entries into the legacy typed fields
// so that existing resolver methods (ResolveMovieFilename, etc.) continue to work.
func PopulateLegacyFields(s *renamer.Settings, moduleType module.Type) {
	switch moduleType {
	case module.TypeMovie:
		s.MovieFolderFormat = s.Patterns["movie-folder"]
		s.MovieFileFormat = s.Patterns["movie-file"]
	case module.TypeTV:
		s.SeriesFolderFormat = s.Patterns["series-folder"]
		s.SeasonFolderFormat = s.Patterns["season-folder"]
		s.SpecialsFolderFormat = s.Patterns["specials-folder"]
		s.StandardEpisodeFormat = s.Patterns["episode-file.standard"]
		s.DailyEpisodeFormat = s.Patterns["episode-file.daily"]
		s.AnimeEpisodeFormat = s.Patterns["episode-file.anime"]
	}
}
