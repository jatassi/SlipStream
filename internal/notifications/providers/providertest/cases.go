package providertest

import "github.com/jatassi/slipstream/internal/notifications/providers"

// Cases are each provider's fixture connection, which its goldens are
// built with and notify-sink's tests send to the sink.
var Cases = map[providers.ID]Case{
	"apprise": {
		Settings: Raw(map[string]any{"serverUrl": "http://apprise.lan:8000", "tags": []string{"family", "phones"}, "username": "slipstream"}),
		Secrets:  Raw(map[string]any{"key": "apprise-key", "password": "apprise-password"}),
	},
	"discord": {
		Settings: Raw(map[string]any{"username": "SlipStream"}),
		Secrets:  Raw(map[string]any{"webhookUrl": "https://discord.com/api/webhooks/1180000000000000000/discord-token"}),
	},
	"email": {
		Settings: Raw(map[string]any{"addresses": []string{"sam@example.com", "alex@example.com"}}),
	},
	"gotify": {
		Settings: Raw(map[string]any{"serverUrl": "https://gotify.example.com", "priority": 8}),
		Secrets:  Raw(map[string]any{"appToken": "gotify-app-token"}), //nolint:gosec // a fixture token
	},
	"ntfy": {
		Settings: Raw(map[string]any{"priority": "4"}),
		Secrets:  Raw(map[string]any{"topic": "slipstream-ntfy-topic", "accessToken": "tk_ntfy"}),
	},
	"pushbullet": {
		Settings: Raw(map[string]any{"device": "ujpah72o0sjAoRtnM0jc"}),
		Secrets:  Raw(map[string]any{"accessToken": "o.pushbullet-token"}),
	},
	"pushover": {
		Settings: Raw(map[string]any{"devices": []string{"phone", "tablet"}, "priority": "1", "sound": "pushover"}),
		Secrets:  Raw(map[string]any{"appToken": "pushover-app-token", "userKey": "pushover-user-key"}),
	},
	"script": {
		Settings: Raw(map[string]any{"node": "n1", "path": "/opt/notify.sh", "args": []string{"--quiet"}, "stop": 45}),
	},
	"slack": {
		Secrets: Raw(map[string]any{"webhookUrl": "https://hooks.slack.com/services/T0000/B0000/slack-token"}),
	},
	"telegram": {
		Settings: Raw(map[string]any{"chatId": "-1001234567890", "topicId": 42, "silent": true}),
		Secrets:  Raw(map[string]any{"botToken": "123456:telegram-token"}),
	},
	"webhook": {
		Settings: Raw(map[string]any{"method": "PUT", "username": "slipstream"}),
		Secrets: Raw(map[string]any{
			"url":      "https://hooks.example.com/slipstream/webhook-token",
			"password": "webhook-password",
			"headers":  []providers.Header{{Name: "X-Hook-Secret", Value: "webhook-header-secret"}},
		}),
	},
}
