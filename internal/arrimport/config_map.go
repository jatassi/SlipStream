package arrimport

import (
	"encoding/json"
	"strings"

	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/notification"
)

// downloadClientTypeMap maps Sonarr/Radarr Implementation names to SlipStream client types.
// Empty string = unsupported (skip during import).
var downloadClientTypeMap = map[string]string{
	"Transmission":          "transmission",
	"QBittorrent":           "qbittorrent",
	"Deluge":                "deluge",
	"RTorrent":              "rtorrent",
	"Vuze":                  "vuze",
	"Aria2":                 "aria2",
	"Flood":                 "flood",
	"UTorrent":              "utorrent",
	"Hadouken":              "hadouken",
	"DownloadStation":       "downloadstation", // G14: no underscore
	"FreeboxDownload":       "freeboxdownload", // G14: no underscore
	"Sabnzbd":               "",                // G13: not in validClientTypes
	"NzbGet":                "",                // G13: not in validClientTypes, factory unimplemented
	"TorrentBlackhole":      "",
	"UsenetBlackhole":       "",
	"NzbVortex":             "",
	"UsenetDownloadStation": "",
	"PneumaticClient":       "",
}

// notificationTypeMap maps Sonarr/Radarr Implementation names to SlipStream NotifierType. (G16)
var notificationTypeMap = map[string]notification.NotifierType{
	"Discord":      notification.NotifierDiscord,
	"Telegram":     notification.NotifierTelegram,
	"Webhook":      notification.NotifierWebhook,
	"Email":        notification.NotifierEmail,
	"Slack":        notification.NotifierSlack,
	"Pushover":     notification.NotifierPushover,
	"Gotify":       notification.NotifierGotify,
	"Ntfy":         notification.NotifierNtfy,
	"Apprise":      notification.NotifierApprise,
	"Pushbullet":   notification.NotifierPushbullet,
	"Join":         notification.NotifierJoin,
	"Prowl":        notification.NotifierProwl,
	"Simplepush":   notification.NotifierSimplepush,
	"Signal":       notification.NotifierSignal,
	"CustomScript": notification.NotifierCustomScript,
	"PlexServer":   notification.NotifierPlex,
}

// colonReplacementMap maps Sonarr/Radarr ColonReplacementFormat int to SlipStream type. (G15)
var colonReplacementMap = map[int]renamer.ColonReplacement{
	0: renamer.ColonDelete,
	1: renamer.ColonDash,
	2: renamer.ColonSpaceDash,
	3: renamer.ColonSpaceDashSpace,
	4: renamer.ColonSmart,
}

// multiEpisodeStyleMap maps Sonarr MultiEpisodeStyle int to SlipStream type. (G15)
var multiEpisodeStyleMap = map[int]renamer.MultiEpisodeStyle{
	0: renamer.StyleExtend,
	1: renamer.StyleDuplicate,
	2: renamer.StyleRepeat,
	3: renamer.StyleScene,
	4: renamer.StyleRange,
	5: renamer.StylePrefixedRange,
}

// notificationSettingsKeyMap renames source settings keys to SlipStream settings keys, per notification type.
var notificationSettingsKeyMap = map[string]map[string]string{
	"discord":  {"webHookUrl": "webhookUrl"},
	"slack":    {"webHookUrl": "webhookUrl"},
	"plex":     {"authToken": "token"},
	"email":    {"server": "host", "requireEncryption": "useSsl"},
	"gotify":   {"server": "url", "appToken": "token"},
	"ntfy":     {"server": "url"},
	"pushover": {"appToken": "token"},
}

func indexerImplementationToDefinitionID(impl string) string {
	known := map[string]string{
		"Torznab":    "torznab",
		"Newznab":    "newznab",
		"IPTorrents": "iptorrents",
		"Nyaa":       "nyaa",
	}
	if id, ok := known[impl]; ok {
		return id
	}
	return strings.ToLower(impl)
}

type dlClientSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	UseSSL   bool
	APIKey   string
	Category string
	URLBase  string
	Warnings []string
}

func translateDownloadClientSettings(settings json.RawMessage, sourceType SourceType) dlClientSettings {
	var result dlClientSettings
	var parsed map[string]any
	if err := json.Unmarshal(settings, &parsed); err != nil {
		result.Warnings = append(result.Warnings, "failed to parse download client settings: "+err.Error())
		return result
	}

	result.Host, _ = parsed["host"].(string)
	if p, ok := parsed["port"].(float64); ok {
		result.Port = int(p)
	}
	result.Username, _ = parsed["username"].(string)
	result.Password, _ = parsed["password"].(string)
	if ssl, ok := parsed["useSsl"].(bool); ok {
		result.UseSSL = ssl
	}
	result.APIKey, _ = parsed["apiKey"].(string)
	result.URLBase, _ = parsed["urlBase"].(string)

	switch sourceType {
	case SourceTypeSonarr:
		result.Category, _ = parsed["tvCategory"].(string)
	case SourceTypeRadarr:
		result.Category, _ = parsed["movieCategory"].(string)
	}

	return result
}

func translateIndexerSettings(settings json.RawMessage) (translatedSettings json.RawMessage, categories []int, warnings []string) {
	var parsed map[string]any
	if err := json.Unmarshal(settings, &parsed); err != nil {
		return settings, nil, []string{"failed to parse indexer settings: " + err.Error()}
	}

	if cats, ok := parsed["categories"].([]any); ok {
		for _, c := range cats {
			if n, ok := c.(float64); ok {
				categories = append(categories, int(n))
			}
		}
	}
	delete(parsed, "categories")
	delete(parsed, "animeCategories")

	if base, ok := parsed["baseUrl"]; ok {
		parsed["url"] = base
		delete(parsed, "baseUrl")
	}

	for _, key := range []string{
		"apiPath", "multiLanguages", "failDownloads", "animeStandardFormatSearch",
		"removeYear", "requiredFlags", "minimumSeeders", "seedCriteria",
		"rejectBlocklistedTorrentHashesWhileGrabbing",
	} {
		delete(parsed, key)
	}

	translatedSettings, _ = json.Marshal(parsed)
	return translatedSettings, categories, nil
}

func translateNotificationSettings(impl string, settings json.RawMessage) (translatedSettings json.RawMessage, warnings []string) {
	var parsed map[string]any
	if err := json.Unmarshal(settings, &parsed); err != nil {
		return settings, []string{"failed to parse notification settings: " + err.Error()}
	}

	typeLower := strings.ToLower(impl)
	if keyMap, ok := notificationSettingsKeyMap[typeLower]; ok {
		for oldKey, newKey := range keyMap {
			if val, exists := parsed[oldKey]; exists {
				parsed[newKey] = val
				delete(parsed, oldKey)
			}
		}
	}

	translatedSettings, _ = json.Marshal(parsed)
	return translatedSettings, nil
}

type sourceProfileItem struct {
	Quality *struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"quality"`
	Items   []sourceProfileItem `json:"items"`
	Allowed bool                `json:"allowed"`
}

// extractAllowedSourceIDs walks the quality profile tree and returns which source quality IDs are allowed.
// Group allowed=false overrides child items (G3).
func extractAllowedSourceIDs(items []sourceProfileItem) map[int]bool {
	allowed := map[int]bool{}
	walkProfileItems(items, true, allowed)
	return allowed
}

func walkProfileItems(items []sourceProfileItem, parentAllowed bool, result map[int]bool) {
	for _, item := range items {
		if item.Quality != nil {
			result[item.Quality.ID] = parentAllowed && item.Allowed
		}
		if len(item.Items) > 0 {
			walkProfileItems(item.Items, parentAllowed && item.Allowed, result)
		}
	}
}

func buildFullQualityItems(allowedSSIDs map[int]bool) []quality.QualityItem {
	items := make([]quality.QualityItem, 0, 17)
	for id := 1; id <= 17; id++ {
		q, ok := quality.GetQualityByID(id)
		if !ok {
			continue
		}
		items = append(items, quality.QualityItem{
			Quality: q,
			Allowed: allowedSSIDs[id],
		})
	}
	return items
}

func mapCutoff(sourceType SourceType, cutoffID int, items []quality.QualityItem) (mappedID int, warnings []string) {
	if ssID := MapQualityID(sourceType, cutoffID, ""); ssID != nil {
		return int(*ssID), nil
	}
	// Fallback: highest-weight allowed quality (G12)
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Allowed {
			var warnings []string
			if cutoffID != 0 {
				warnings = append(warnings, "unmapped cutoff quality ID, using fallback")
			}
			return items[i].Quality.ID, warnings
		}
	}
	return 0, nil
}

func flattenQualityProfileItems(sourceType SourceType, itemsJSON json.RawMessage, cutoffID int) (items []quality.QualityItem, cutoff int, warnings []string) {
	var srcItems []sourceProfileItem
	if err := json.Unmarshal(itemsJSON, &srcItems); err != nil {
		warnings = append(warnings, "failed to parse quality profile items: "+err.Error())
	}

	allowedSourceIDs := extractAllowedSourceIDs(srcItems)

	allowedSSIDs := map[int]bool{}
	for srcID, allowed := range allowedSourceIDs {
		if ssID := MapQualityID(sourceType, srcID, ""); ssID != nil {
			allowedSSIDs[int(*ssID)] = allowed
		}
	}

	items = buildFullQualityItems(allowedSSIDs)
	cutoff, cutoffWarnings := mapCutoff(sourceType, cutoffID, items)
	warnings = append(warnings, cutoffWarnings...)

	return items, cutoff, warnings
}

func translateNamingConfig(src *SourceNamingConfig, sourceType SourceType, current *importer.ImportSettings) (settings *importer.ImportSettings, warnings []string) {
	result := *current // shallow copy

	result.ReplaceIllegalCharacters = src.ReplaceIllegalCharacters
	if cr, ok := colonReplacementMap[src.ColonReplacementFormat]; ok {
		result.ColonReplacement = cr
	} else {
		warnings = append(warnings, "unknown colon replacement format, keeping current")
	}

	switch sourceType {
	case SourceTypeSonarr:
		translateSonarrNaming(src, &result)
	case SourceTypeRadarr:
		translateRadarrNaming(src, &result)
	}

	return &result, warnings
}

func translateSonarrNaming(src *SourceNamingConfig, result *importer.ImportSettings) {
	result.RenameEpisodes = src.RenameEpisodes
	if mes, ok := multiEpisodeStyleMap[src.MultiEpisodeStyle]; ok {
		result.MultiEpisodeStyle = mes
	}
	setIfNonEmpty(&result.StandardEpisodeFormat, src.StandardEpisodeFormat)
	setIfNonEmpty(&result.DailyEpisodeFormat, src.DailyEpisodeFormat)
	setIfNonEmpty(&result.AnimeEpisodeFormat, src.AnimeEpisodeFormat)
	setIfNonEmpty(&result.SeriesFolderFormat, src.SeriesFolderFormat)
	setIfNonEmpty(&result.SeasonFolderFormat, src.SeasonFolderFormat)
	setIfNonEmpty(&result.SpecialsFolderFormat, src.SpecialsFolderFormat)
}

func translateRadarrNaming(src *SourceNamingConfig, result *importer.ImportSettings) {
	result.RenameMovies = src.RenameMovies
	setIfNonEmpty(&result.MovieFileFormat, src.StandardMovieFormat)
	setIfNonEmpty(&result.MovieFolderFormat, src.MovieFolderFormat)
}

func setIfNonEmpty(target *string, value string) {
	if value != "" {
		*target = value
	}
}

func hasRedactedCredentials(settings json.RawMessage) bool {
	var parsed map[string]any
	if err := json.Unmarshal(settings, &parsed); err != nil {
		return false
	}
	for _, v := range parsed {
		if s, ok := v.(string); ok && s == "********" {
			return true
		}
	}
	return false
}
