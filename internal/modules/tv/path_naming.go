package tv

import (
	"context"
	"fmt"
	"strings"

	tvlib "github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/module"
)

var _ module.PathGenerator = (*pathGenerator)(nil)
var _ module.NamingProvider = (*namingProvider)(nil)

// --- PathGenerator ---

type pathGenerator struct {
	tvSvc *tvlib.Service
}

func (p *pathGenerator) DefaultTemplates() map[string]string {
	return map[string]string{
		"series-folder":         "{Series Title}",
		"season-folder":         "Season {season}",
		"specials-folder":       "Specials",
		"episode-file.standard": "{Series Title} - S{season:00}E{episode:00} - {Quality Title}",
		"episode-file.daily":    "{Series Title} - {Air-Date} - {Episode Title} {Quality Full}",
		"episode-file.anime":    "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}",
	}
}

func (p *pathGenerator) AvailableVariables(contextName string) []module.TemplateVariable {
	switch contextName {
	case "series-folder":
		return []module.TemplateVariable{
			{Name: "Series Title", Description: "Series title", Example: "Breaking Bad", DataKey: "SeriesTitle"},
			{Name: "Series TitleYear", Description: "Title with year", Example: "Breaking Bad (2008)", DataKey: "SeriesTitleYear"},
			{Name: "Series CleanTitle", Description: "Title without special characters", Example: "Breaking Bad", DataKey: "SeriesCleanTitle"},
		}
	case "season-folder", "specials-folder":
		return []module.TemplateVariable{
			{Name: "season", Description: "Season number", Example: "01", DataKey: "SeasonNumber"},
		}
	default: // episode-file and variants
		vars := []module.TemplateVariable{
			{Name: "Series Title", Description: "Series title", Example: "Breaking Bad", DataKey: "SeriesTitle"},
			{Name: "season", Description: "Season number", Example: "01", DataKey: "SeasonNumber"},
			{Name: "episode", Description: "Episode number", Example: "01", DataKey: "EpisodeNumber"},
			{Name: "Episode Title", Description: "Episode title", Example: "Pilot", DataKey: "EpisodeTitle"},
			{Name: "Air-Date", Description: "Air date (YYYY-MM-DD)", Example: "2008-01-20", DataKey: "AirDate"},
			{Name: "absolute", Description: "Absolute episode number (anime)", Example: "001", DataKey: "AbsoluteNumber"},
		}
		vars = append(vars, qualityVariables()...)
		vars = append(vars, mediaInfoVariables()...)
		vars = append(vars, metadataVariables()...)
		return vars
	}
}

func (p *pathGenerator) ConditionalSegments() []module.ConditionalSegment {
	return []module.ConditionalSegment{
		{
			Name:         "SeasonFolder",
			Label:        "Use Season Folders",
			Description:  "Organize episodes into season subfolders",
			DefaultValue: true,
			DataKey:      "SeasonFolder",
		},
	}
}

func (p *pathGenerator) IsSpecialNode(ctx context.Context, entityType module.EntityType, entityID int64) (bool, error) {
	if entityType != module.EntitySeason {
		return false, nil
	}
	season, err := p.tvSvc.GetSeasonByID(ctx, entityID)
	if err != nil {
		return false, err
	}
	return season.SeasonNumber == 0, nil
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
			Name:      "series-folder",
			Label:     "Series Folder Format",
			Variables: pg.AvailableVariables("series-folder"),
			IsFolder:  true,
		},
		{
			Name:      "season-folder",
			Label:     "Season Folder Format",
			Variables: pg.AvailableVariables("season-folder"),
			IsFolder:  true,
		},
		{
			Name:      "specials-folder",
			Label:     "Specials Folder Format",
			Variables: pg.AvailableVariables("specials-folder"),
			IsFolder:  true,
		},
		{
			Name:      "episode-file",
			Label:     "Episode File Format",
			Variables: pg.AvailableVariables("episode-file"),
			Variants:  []string{"standard", "daily", "anime"},
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
		{
			Key:          "multi_episode_style",
			Label:        "Multi-Episode Style",
			Description:  "How multi-episode filenames are formatted",
			Type:         "enum",
			EnumValues:   []string{"extend", "duplicate", "repeat", "scene", "range", "prefixed_range"},
			DefaultValue: "extend",
		},
	}
}

// --- Shared helpers ---

func qualityVariables() []module.TemplateVariable {
	return []module.TemplateVariable{
		{Name: "Quality Title", Description: "Quality with source (e.g., HDTV-720p)", Example: "Bluray-1080p", DataKey: "QualityTitle"},
		{Name: "Quality Full", Description: "Full quality string", Example: "HDTV-720p Proper", DataKey: "QualityFull"},
	}
}

func mediaInfoVariables() []module.TemplateVariable {
	return []module.TemplateVariable{
		{Name: "MediaInfo VideoCodec", Description: "Video codec", Example: "x264", DataKey: "VideoCodec"},
		{Name: "MediaInfo AudioCodec", Description: "Audio codec", Example: "DTS", DataKey: "AudioCodec"},
		{Name: "MediaInfo AudioChannels", Description: "Audio channels", Example: "5.1", DataKey: "AudioChannels"},
		{Name: "MediaInfo VideoDynamicRange", Description: "Dynamic range", Example: "HDR", DataKey: "VideoDynamicRange"},
	}
}

func metadataVariables() []module.TemplateVariable {
	return []module.TemplateVariable{
		{Name: "Release Group", Description: "Release group name", Example: "SPARKS", DataKey: "ReleaseGroup"},
		{Name: "Edition Tags", Description: "Edition info", Example: "Director's Cut", DataKey: "EditionTags"},
	}
}

func resolveTemplateGeneric(template string, data map[string]any) (string, error) {
	result := template
	for key, val := range data {
		token := "{" + key + "}"
		result = strings.ReplaceAll(result, token, fmt.Sprintf("%v", val))
	}
	return result, nil
}
