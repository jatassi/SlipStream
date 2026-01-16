package notification

// FieldType represents the type of a settings field
type FieldType string

const (
	FieldTypeText     FieldType = "text"
	FieldTypePassword FieldType = "password"
	FieldTypeNumber   FieldType = "number"
	FieldTypeBool     FieldType = "bool"
	FieldTypeSelect   FieldType = "select"
	FieldTypeURL      FieldType = "url"
)

// SelectOption represents an option for select fields
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// SettingsField describes a single configuration field
type SettingsField struct {
	Name        string         `json:"name"`
	Label       string         `json:"label"`
	Type        FieldType      `json:"type"`
	Required    bool           `json:"required,omitempty"`
	Placeholder string         `json:"placeholder,omitempty"`
	HelpText    string         `json:"helpText,omitempty"`
	Default     any            `json:"default,omitempty"`
	Options     []SelectOption `json:"options,omitempty"`
	Advanced    bool           `json:"advanced,omitempty"`
}

// NotifierSchema describes a notification provider's capabilities
type NotifierSchema struct {
	Type        NotifierType    `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InfoURL     string          `json:"infoUrl,omitempty"`
	Fields      []SettingsField `json:"fields"`
}

// SchemaRegistry contains schemas for all supported notifier types
var SchemaRegistry = map[NotifierType]NotifierSchema{
	NotifierDiscord: {
		Type:        NotifierDiscord,
		Name:        "Discord",
		Description: "Send notifications via Discord webhook",
		InfoURL:     "https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks",
		Fields: []SettingsField{
			{Name: "webhookUrl", Label: "Webhook URL", Type: FieldTypeURL, Required: true, HelpText: "Discord channel webhook URL"},
			{Name: "username", Label: "Username", Type: FieldTypeText, Placeholder: "SlipStream", HelpText: "Override the webhook's default username"},
			{Name: "avatarUrl", Label: "Avatar URL", Type: FieldTypeURL, HelpText: "Override the webhook's default avatar", Advanced: true},
			{Name: "includePosters", Label: "Include Posters", Type: FieldTypeBool, Default: true, HelpText: "Include movie/series posters in embeds"},
		},
	},
	NotifierTelegram: {
		Type:        NotifierTelegram,
		Name:        "Telegram",
		Description: "Send notifications via Telegram bot",
		InfoURL:     "https://core.telegram.org/bots#how-do-i-create-a-bot",
		Fields: []SettingsField{
			{Name: "botToken", Label: "Bot Token", Type: FieldTypePassword, Required: true, HelpText: "Token from @BotFather"},
			{Name: "chatId", Label: "Chat ID", Type: FieldTypeText, Required: true, HelpText: "Chat, group, or channel ID"},
			{Name: "topicId", Label: "Topic ID", Type: FieldTypeNumber, HelpText: "Forum topic ID (optional)", Advanced: true},
			{Name: "silent", Label: "Send Silently", Type: FieldTypeBool, HelpText: "Send without notification sound"},
			{Name: "includeLinks", Label: "Include Links", Type: FieldTypeBool, Default: true, HelpText: "Include TMDb/IMDb links"},
		},
	},
	NotifierWebhook: {
		Type:        NotifierWebhook,
		Name:        "Webhook",
		Description: "Send notifications to a custom HTTP endpoint",
		Fields: []SettingsField{
			{Name: "url", Label: "Webhook URL", Type: FieldTypeURL, Required: true, HelpText: "URL to POST notifications to"},
			{Name: "method", Label: "HTTP Method", Type: FieldTypeSelect, Default: "POST", Options: []SelectOption{
				{Value: "POST", Label: "POST"},
				{Value: "PUT", Label: "PUT"},
			}},
			{Name: "username", Label: "Username", Type: FieldTypeText, HelpText: "Basic auth username", Advanced: true},
			{Name: "password", Label: "Password", Type: FieldTypePassword, HelpText: "Basic auth password", Advanced: true},
			{Name: "headers", Label: "Custom Headers", Type: FieldTypeText, HelpText: "JSON object of headers", Advanced: true},
		},
	},
	NotifierEmail: {
		Type:        NotifierEmail,
		Name:        "Email",
		Description: "Send notifications via SMTP email",
		Fields: []SettingsField{
			{Name: "server", Label: "SMTP Server", Type: FieldTypeText, Required: true, HelpText: "SMTP server hostname"},
			{Name: "port", Label: "Port", Type: FieldTypeNumber, Required: true, Default: 587, HelpText: "SMTP port (usually 587 or 465)"},
			{Name: "useTLS", Label: "Use TLS", Type: FieldTypeBool, Default: true, HelpText: "Enable TLS encryption"},
			{Name: "username", Label: "Username", Type: FieldTypeText, HelpText: "SMTP authentication username"},
			{Name: "password", Label: "Password", Type: FieldTypePassword, HelpText: "SMTP authentication password"},
			{Name: "from", Label: "From Address", Type: FieldTypeText, Required: true, HelpText: "Sender email address"},
			{Name: "to", Label: "To Address", Type: FieldTypeText, Required: true, HelpText: "Recipient email address(es), comma-separated"},
			{Name: "cc", Label: "CC", Type: FieldTypeText, HelpText: "CC recipients, comma-separated", Advanced: true},
			{Name: "bcc", Label: "BCC", Type: FieldTypeText, HelpText: "BCC recipients, comma-separated", Advanced: true},
		},
	},
	NotifierSlack: {
		Type:        NotifierSlack,
		Name:        "Slack",
		Description: "Send notifications via Slack webhook",
		InfoURL:     "https://api.slack.com/messaging/webhooks",
		Fields: []SettingsField{
			{Name: "webhookUrl", Label: "Webhook URL", Type: FieldTypeURL, Required: true, HelpText: "Slack incoming webhook URL"},
			{Name: "username", Label: "Username", Type: FieldTypeText, Placeholder: "SlipStream", HelpText: "Override bot username"},
			{Name: "iconEmoji", Label: "Icon Emoji", Type: FieldTypeText, Placeholder: ":movie_camera:", HelpText: "Emoji for bot icon (e.g., :movie_camera:)", Advanced: true},
			{Name: "channel", Label: "Channel", Type: FieldTypeText, HelpText: "Override default channel (e.g., #movies)", Advanced: true},
		},
	},
	NotifierPushover: {
		Type:        NotifierPushover,
		Name:        "Pushover",
		Description: "Send push notifications via Pushover",
		InfoURL:     "https://pushover.net/api",
		Fields: []SettingsField{
			{Name: "userKey", Label: "User Key", Type: FieldTypePassword, Required: true, HelpText: "Your Pushover user key"},
			{Name: "apiToken", Label: "API Token", Type: FieldTypePassword, Required: true, HelpText: "Your Pushover application API token"},
			{Name: "devices", Label: "Devices", Type: FieldTypeText, HelpText: "Specific device names, comma-separated (leave empty for all)", Advanced: true},
			{Name: "priority", Label: "Priority", Type: FieldTypeSelect, Default: "0", Options: []SelectOption{
				{Value: "-2", Label: "Lowest"},
				{Value: "-1", Label: "Low"},
				{Value: "0", Label: "Normal"},
				{Value: "1", Label: "High"},
				{Value: "2", Label: "Emergency"},
			}},
			{Name: "sound", Label: "Sound", Type: FieldTypeText, Placeholder: "pushover", HelpText: "Notification sound name", Advanced: true},
		},
	},
	NotifierGotify: {
		Type:        NotifierGotify,
		Name:        "Gotify",
		Description: "Send notifications to a Gotify server",
		InfoURL:     "https://gotify.net/docs/",
		Fields: []SettingsField{
			{Name: "serverUrl", Label: "Server URL", Type: FieldTypeURL, Required: true, HelpText: "Gotify server URL"},
			{Name: "appToken", Label: "App Token", Type: FieldTypePassword, Required: true, HelpText: "Application token from Gotify"},
			{Name: "priority", Label: "Priority", Type: FieldTypeNumber, Default: 5, HelpText: "Message priority (0-10)"},
		},
	},
	NotifierNtfy: {
		Type:        NotifierNtfy,
		Name:        "ntfy",
		Description: "Send notifications via ntfy.sh or self-hosted ntfy",
		InfoURL:     "https://ntfy.sh/docs/",
		Fields: []SettingsField{
			{Name: "serverUrl", Label: "Server URL", Type: FieldTypeURL, Default: "https://ntfy.sh", HelpText: "ntfy server URL"},
			{Name: "topic", Label: "Topic", Type: FieldTypeText, Required: true, HelpText: "Topic name to publish to"},
			{Name: "username", Label: "Username", Type: FieldTypeText, HelpText: "Authentication username", Advanced: true},
			{Name: "password", Label: "Password", Type: FieldTypePassword, HelpText: "Authentication password", Advanced: true},
			{Name: "priority", Label: "Priority", Type: FieldTypeSelect, Default: "3", Options: []SelectOption{
				{Value: "1", Label: "Min"},
				{Value: "2", Label: "Low"},
				{Value: "3", Label: "Default"},
				{Value: "4", Label: "High"},
				{Value: "5", Label: "Max"},
			}},
		},
	},
	NotifierPushbullet: {
		Type:        NotifierPushbullet,
		Name:        "Pushbullet",
		Description: "Send notifications via Pushbullet",
		InfoURL:     "https://docs.pushbullet.com/",
		Fields: []SettingsField{
			{Name: "accessToken", Label: "Access Token", Type: FieldTypePassword, Required: true, HelpText: "Pushbullet access token"},
			{Name: "deviceId", Label: "Device ID", Type: FieldTypeText, HelpText: "Specific device ID (leave empty for all)", Advanced: true},
			{Name: "channelTag", Label: "Channel Tag", Type: FieldTypeText, HelpText: "Channel tag for broadcast", Advanced: true},
		},
	},
	NotifierJoin: {
		Type:        NotifierJoin,
		Name:        "Join",
		Description: "Send notifications via Join by joaoapps",
		InfoURL:     "https://joaoapps.com/join/api/",
		Fields: []SettingsField{
			{Name: "apiKey", Label: "API Key", Type: FieldTypePassword, Required: true, HelpText: "Join API key"},
			{Name: "deviceId", Label: "Device ID", Type: FieldTypeText, HelpText: "Target device ID (leave empty for all)"},
			{Name: "priority", Label: "Priority", Type: FieldTypeSelect, Default: "2", Options: []SelectOption{
				{Value: "-2", Label: "Lowest"},
				{Value: "-1", Label: "Low"},
				{Value: "0", Label: "Normal"},
				{Value: "1", Label: "High"},
				{Value: "2", Label: "Highest"},
			}},
		},
	},
	NotifierApprise: {
		Type:        NotifierApprise,
		Name:        "Apprise",
		Description: "Send notifications via Apprise API",
		InfoURL:     "https://github.com/caronc/apprise-api",
		Fields: []SettingsField{
			{Name: "serverUrl", Label: "Server URL", Type: FieldTypeURL, Required: true, HelpText: "Apprise API server URL"},
			{Name: "configKey", Label: "Config Key", Type: FieldTypeText, HelpText: "Named configuration key (optional)"},
			{Name: "urls", Label: "Notification URLs", Type: FieldTypeText, HelpText: "Apprise notification URLs, one per line"},
			{Name: "tags", Label: "Tags", Type: FieldTypeText, HelpText: "Filter by tags", Advanced: true},
		},
	},
	NotifierCustomScript: {
		Type:        NotifierCustomScript,
		Name:        "Custom Script",
		Description: "Execute a custom script for notifications",
		Fields: []SettingsField{
			{Name: "path", Label: "Script Path", Type: FieldTypeText, Required: true, HelpText: "Full path to the script to execute"},
			{Name: "arguments", Label: "Arguments", Type: FieldTypeText, HelpText: "Additional arguments to pass to the script", Advanced: true},
		},
	},
}

// GetSchema returns the schema for a notifier type
func GetSchema(t NotifierType) (NotifierSchema, bool) {
	schema, ok := SchemaRegistry[t]
	return schema, ok
}

// GetAllSchemas returns all available notifier schemas
func GetAllSchemas() []NotifierSchema {
	schemas := make([]NotifierSchema, 0, len(SchemaRegistry))
	for _, schema := range SchemaRegistry {
		schemas = append(schemas, schema)
	}
	return schemas
}
