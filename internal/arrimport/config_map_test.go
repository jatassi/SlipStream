package arrimport

import (
	"encoding/json"
	"testing"

	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/notification"
)

func TestDownloadClientTypeMap(t *testing.T) {
	tests := []struct{ impl, expected string }{
		{"Transmission", "transmission"},
		{"QBittorrent", "qbittorrent"},
		{"Deluge", "deluge"},
		{"RTorrent", "rtorrent"},
		{"Vuze", "vuze"},
		{"Aria2", "aria2"},
		{"Flood", "flood"},
		{"UTorrent", "utorrent"},
		{"Hadouken", "hadouken"},
		{"DownloadStation", "downloadstation"}, // G14: no underscore
		{"FreeboxDownload", "freeboxdownload"}, // G14: no underscore
		{"Sabnzbd", ""},                        // G13: unsupported
		{"NzbGet", ""},                         // G13: unsupported
		{"TorrentBlackhole", ""},
		{"UsenetBlackhole", ""},
	}
	for _, tt := range tests {
		if got := downloadClientTypeMap[tt.impl]; got != tt.expected {
			t.Errorf("downloadClientTypeMap[%q] = %q, want %q", tt.impl, got, tt.expected)
		}
	}
}

func TestNotificationTypeMap(t *testing.T) {
	tests := []struct {
		impl     string
		expected notification.NotifierType
		exists   bool
	}{
		{"Discord", notification.NotifierDiscord, true},
		{"Telegram", notification.NotifierTelegram, true},
		{"Webhook", notification.NotifierWebhook, true},
		{"Email", notification.NotifierEmail, true},
		{"Slack", notification.NotifierSlack, true},
		{"Pushover", notification.NotifierPushover, true},
		{"Gotify", notification.NotifierGotify, true},
		{"Ntfy", notification.NotifierNtfy, true},
		{"Apprise", notification.NotifierApprise, true},
		{"Pushbullet", notification.NotifierPushbullet, true},
		{"Join", notification.NotifierJoin, true},
		{"Prowl", notification.NotifierProwl, true},
		{"Simplepush", notification.NotifierSimplepush, true},
		{"Signal", notification.NotifierSignal, true},
		{"CustomScript", notification.NotifierCustomScript, true},
		{"PlexServer", notification.NotifierPlex, true},
		{"Emby", "", false}, // unsupported
	}
	for _, tt := range tests {
		got, ok := notificationTypeMap[tt.impl]
		if ok != tt.exists {
			t.Errorf("notificationTypeMap[%q] exists=%v, want %v", tt.impl, ok, tt.exists)
		}
		if ok && got != tt.expected {
			t.Errorf("notificationTypeMap[%q] = %q, want %q", tt.impl, got, tt.expected)
		}
	}
}

func TestColonReplacementMap(t *testing.T) {
	tests := []struct {
		input    int
		expected renamer.ColonReplacement
	}{
		{0, renamer.ColonDelete}, {1, renamer.ColonDash}, {2, renamer.ColonSpaceDash},
		{3, renamer.ColonSpaceDashSpace}, {4, renamer.ColonSmart},
	}
	for _, tt := range tests {
		if got := colonReplacementMap[tt.input]; got != tt.expected {
			t.Errorf("colonReplacementMap[%d] = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMultiEpisodeStyleMap(t *testing.T) {
	tests := []struct {
		input    int
		expected renamer.MultiEpisodeStyle
	}{
		{0, renamer.StyleExtend}, {1, renamer.StyleDuplicate}, {2, renamer.StyleRepeat},
		{3, renamer.StyleScene}, {4, renamer.StyleRange}, {5, renamer.StylePrefixedRange},
	}
	for _, tt := range tests {
		if got := multiEpisodeStyleMap[tt.input]; got != tt.expected {
			t.Errorf("multiEpisodeStyleMap[%d] = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFlattenQualityProfileItems(t *testing.T) {
	// Real Sonarr "HD-1080p" profile items
	itemsJSON := json.RawMessage(`[
		{"quality":{"id":0,"name":"Unknown"},"items":[],"allowed":false},
		{"quality":{"id":1,"name":"SDTV"},"items":[],"allowed":false},
		{"id":1000,"name":"WEB 480p","items":[
			{"quality":{"id":12,"name":"WEBRip-480p"},"items":[],"allowed":false},
			{"quality":{"id":8,"name":"WEBDL-480p"},"items":[],"allowed":false}
		],"allowed":false},
		{"quality":{"id":2,"name":"DVD"},"items":[],"allowed":false},
		{"quality":{"id":4,"name":"HDTV-720p"},"items":[],"allowed":false},
		{"quality":{"id":9,"name":"HDTV-1080p"},"items":[],"allowed":true},
		{"id":1001,"name":"WEB 720p","items":[
			{"quality":{"id":14,"name":"WEBRip-720p"},"items":[],"allowed":false},
			{"quality":{"id":5,"name":"WEBDL-720p"},"items":[],"allowed":false}
		],"allowed":false},
		{"id":1002,"name":"WEB 1080p","items":[
			{"quality":{"id":15,"name":"WEBRip-1080p"},"items":[],"allowed":true},
			{"quality":{"id":3,"name":"WEBDL-1080p"},"items":[],"allowed":true}
		],"allowed":true},
		{"quality":{"id":7,"name":"Bluray-1080p"},"items":[],"allowed":true},
		{"quality":{"id":6,"name":"Bluray-720p"},"items":[],"allowed":false},
		{"quality":{"id":20,"name":"Bluray-1080p Remux"},"items":[],"allowed":false},
		{"quality":{"id":16,"name":"HDTV-2160p"},"items":[],"allowed":false},
		{"id":1003,"name":"WEB 2160p","items":[
			{"quality":{"id":17,"name":"WEBRip-2160p"},"items":[],"allowed":false},
			{"quality":{"id":18,"name":"WEBDL-2160p"},"items":[],"allowed":false}
		],"allowed":false},
		{"quality":{"id":19,"name":"Bluray-2160p"},"items":[],"allowed":false},
		{"quality":{"id":21,"name":"Bluray-2160p Remux"},"items":[],"allowed":false}
	]`)

	items, cutoff, warnings := flattenQualityProfileItems(SourceTypeSonarr, itemsJSON, 9)

	if len(items) != 17 {
		t.Fatalf("expected 17 items, got %d", len(items))
	}

	allowedIDs := map[int]bool{}
	for _, item := range items {
		if item.Allowed {
			allowedIDs[item.Quality.ID] = true
		}
	}

	// HDTV-1080p (source 9 → SS 8) should be allowed
	if !allowedIDs[8] {
		t.Error("HDTV-1080p (SS ID 8) should be allowed")
	}
	// WEBRip-1080p (source 15 → SS 9) should be allowed (in allowed group)
	if !allowedIDs[9] {
		t.Error("WEBRip-1080p (SS ID 9) should be allowed")
	}
	// WEBDL-1080p (source 3 → SS 10) should be allowed
	if !allowedIDs[10] {
		t.Error("WEBDL-1080p (SS ID 10) should be allowed")
	}
	// Bluray-1080p (source 7 → SS 11) should be allowed
	if !allowedIDs[11] {
		t.Error("Bluray-1080p (SS ID 11) should be allowed")
	}
	// SDTV (source 1 → SS 1) should NOT be allowed
	if allowedIDs[1] {
		t.Error("SDTV (SS ID 1) should NOT be allowed")
	}

	// Cutoff: source 9 (HDTV-1080p) → SS 8
	if cutoff != 8 {
		t.Errorf("expected cutoff 8 (HDTV-1080p), got %d", cutoff)
	}

	_ = warnings
}

func TestFlattenQualityProfileItems_GroupDisabled(t *testing.T) {
	// Group with allowed=false should override member allowed=true (G3)
	itemsJSON := json.RawMessage(`[
		{"id":1002,"name":"WEB 1080p","items":[
			{"quality":{"id":15,"name":"WEBRip-1080p"},"items":[],"allowed":true},
			{"quality":{"id":3,"name":"WEBDL-1080p"},"items":[],"allowed":true}
		],"allowed":false}
	]`)

	items, _, _ := flattenQualityProfileItems(SourceTypeSonarr, itemsJSON, 0)

	for _, item := range items {
		if item.Quality.ID == 9 && item.Allowed { // WEBRip-1080p → SS 9
			t.Error("WEBRip-1080p should be disallowed (parent group disabled)")
		}
		if item.Quality.ID == 10 && item.Allowed { // WEBDL-1080p → SS 10
			t.Error("WEBDL-1080p should be disallowed (parent group disabled)")
		}
	}
}

func TestTranslateDownloadClientSettings(t *testing.T) {
	settings := json.RawMessage(`{
		"host": "111.nl116.seedit4.me",
		"port": 8101,
		"useSsl": false,
		"urlBase": "/transmission/",
		"username": "seedit4me",
		"password": "eWdiNrG*Rmww",
		"tvCategory": "sonarr",
		"recentTvPriority": 0,
		"olderTvPriority": 0,
		"addPaused": false
	}`)

	result := translateDownloadClientSettings(settings, SourceTypeSonarr)

	if result.Host != "111.nl116.seedit4.me" {
		t.Errorf("host = %q", result.Host)
	}
	if result.Port != 8101 {
		t.Errorf("port = %d", result.Port)
	}
	if result.Username != "seedit4me" {
		t.Errorf("username = %q", result.Username)
	}
	if result.Password != "eWdiNrG*Rmww" {
		t.Errorf("password = %q", result.Password)
	}
	if result.UseSSL {
		t.Error("useSsl should be false")
	}
	if result.Category != "sonarr" {
		t.Errorf("category = %q", result.Category)
	}
	if result.URLBase != "/transmission/" {
		t.Errorf("urlBase = %q", result.URLBase)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

func TestTranslateDownloadClientSettings_Radarr(t *testing.T) {
	settings := json.RawMessage(`{
		"host": "localhost",
		"port": 9091,
		"movieCategory": "radarr"
	}`)

	result := translateDownloadClientSettings(settings, SourceTypeRadarr)

	if result.Category != "radarr" {
		t.Errorf("category = %q, want %q", result.Category, "radarr")
	}
}

func TestTranslateIndexerSettings(t *testing.T) {
	settings := json.RawMessage(`{
		"baseUrl": "https://torznab.example.com",
		"apiKey": "abc123",
		"apiPath": "/api",
		"categories": [2000, 5000, 5040],
		"minimumSeeders": 1
	}`)

	result, categories, warnings := translateIndexerSettings(settings)

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatal(err)
	}

	if _, ok := parsed["baseUrl"]; ok {
		t.Error("baseUrl should be removed")
	}
	if parsed["url"] != "https://torznab.example.com" {
		t.Error("url should be set from baseUrl")
	}
	if parsed["apiKey"] != "abc123" {
		t.Error("apiKey should be preserved")
	}
	if _, ok := parsed["apiPath"]; ok {
		t.Error("apiPath should be removed")
	}
	if _, ok := parsed["minimumSeeders"]; ok {
		t.Error("minimumSeeders should be removed")
	}
	if _, ok := parsed["categories"]; ok {
		t.Error("categories should be extracted from settings")
	}
	if len(categories) != 3 || categories[0] != 2000 || categories[1] != 5000 || categories[2] != 5040 {
		t.Errorf("categories = %v", categories)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestTranslateNotificationSettings_Discord(t *testing.T) {
	settings := json.RawMessage(`{"webHookUrl":"https://discord.com/hook","username":"bot"}`)

	result, warnings := translateNotificationSettings("Discord", settings)

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatal(err)
	}

	if _, ok := parsed["webHookUrl"]; ok {
		t.Error("webHookUrl should be renamed")
	}
	if parsed["webhookUrl"] != "https://discord.com/hook" {
		t.Error("webhookUrl should be set")
	}
	if parsed["username"] != "bot" {
		t.Error("username should be preserved")
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestHasRedactedCredentials(t *testing.T) {
	redacted := json.RawMessage(`{"host":"localhost","password":"********"}`)
	if !hasRedactedCredentials(redacted) {
		t.Error("should detect redacted credentials")
	}
	clean := json.RawMessage(`{"host":"localhost","password":"real"}`)
	if hasRedactedCredentials(clean) {
		t.Error("should not flag clean credentials")
	}
}

func TestIndexerImplementationToDefinitionID(t *testing.T) {
	tests := []struct{ impl, expected string }{
		{"Torznab", "torznab"},
		{"Newznab", "newznab"},
		{"IPTorrents", "iptorrents"},
		{"Nyaa", "nyaa"},
		{"Torrentleech", "torrentleech"}, // fallback lowercase
		{"SomeCustom", "somecustom"},     // fallback lowercase
	}
	for _, tt := range tests {
		if got := indexerImplementationToDefinitionID(tt.impl); got != tt.expected {
			t.Errorf("indexerImplementationToDefinitionID(%q) = %q, want %q", tt.impl, got, tt.expected)
		}
	}
}
