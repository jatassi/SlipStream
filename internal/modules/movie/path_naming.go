package movie

import (
	"context"
	"fmt"
	"strings"

	"github.com/slipstream/slipstream/internal/module"
)

var _ module.PathGenerator = (*pathGenerator)(nil)
var _ module.NamingProvider = (*namingProvider)(nil)

// --- PathGenerator ---

type pathGenerator struct{}

func (p *pathGenerator) DefaultTemplates() map[string]string {
	return map[string]string{
		"movie-folder": "{Movie Title} ({Year})",
		"movie-file":   "{Movie Title} ({Year}) - {Quality Title}",
	}
}

func (p *pathGenerator) AvailableVariables(contextName string) []module.TemplateVariable {
	common := []module.TemplateVariable{
		{Name: "Movie Title", Description: "Movie title", Example: "Inception", DataKey: "MovieTitle"},
		{Name: "Movie TitleYear", Description: "Title with year", Example: "Inception (2010)", DataKey: "MovieTitleYear"},
		{Name: "Movie CleanTitle", Description: "Title without special characters", Example: "Inception", DataKey: "MovieCleanTitle"},
		{Name: "Year", Description: "Release year", Example: "2010", DataKey: "MovieYear"},
		{Name: "IMDb", Description: "IMDb ID", Example: "tt1375666", DataKey: "ImdbID"},
		{Name: "TMDB", Description: "TMDB ID", Example: "27205", DataKey: "TmdbID"},
	}

	if contextName == "movie-file" {
		common = append(common, module.QualityVariables()...)
		common = append(common, module.MediaInfoVariables()...)
		common = append(common, module.MetadataVariables()...)
	}

	return common
}

func (p *pathGenerator) ConditionalSegments() []module.ConditionalSegment {
	return nil
}

func (p *pathGenerator) IsSpecialNode(_ context.Context, _ module.EntityType, _ int64) (bool, error) {
	return false, nil
}

func (p *pathGenerator) ResolveTemplate(template string, data map[string]any) (string, error) {
	return resolveTemplateGeneric(template, data)
}

// --- NamingProvider ---

type namingProvider struct{}

func (n *namingProvider) TokenContexts() []module.TokenContext {
	pg := &pathGenerator{}
	return []module.TokenContext{
		{
			Name:      "movie-folder",
			Label:     "Movie Folder Format",
			Variables: pg.AvailableVariables("movie-folder"),
			IsFolder:  true,
		},
		{
			Name:      "movie-file",
			Label:     "Movie File Format",
			Variables: pg.AvailableVariables("movie-file"),
			IsFolder:  false,
		},
	}
}

func (n *namingProvider) DefaultFileTemplates() map[string]string {
	return (&pathGenerator{}).DefaultTemplates()
}

func (n *namingProvider) FormatOptions() []module.FormatOption {
	return []module.FormatOption{
		{
			Key:          "colon_replacement",
			Label:        "Colon Replacement",
			Description:  "How to replace colons in filenames",
			Type:         "enum",
			EnumValues:   []string{"delete", "dash", "space_dash", "space_dash_space", "smart", "custom"},
			DefaultValue: "smart",
		},
	}
}

// resolveTemplateGeneric is a lightweight template resolver for validation/preview.
// It replaces {Key} tokens in a template with values from the data map.
//
//nolint:unparam // error return is intentional for interface conformance (PathGenerator.ResolveTemplate)
func resolveTemplateGeneric(template string, data map[string]any) (string, error) {
	result := template
	for key, val := range data {
		token := "{" + key + "}"
		result = strings.ReplaceAll(result, token, fmt.Sprintf("%v", val))
	}
	return result, nil
}
