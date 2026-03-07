package module

// TemplateVariable describes a variable available for path/naming templates.
type TemplateVariable struct {
	Name        string // e.g., "Movie Title", "Series Title", "season"
	Description string // Human-readable description
	Example     string // Example output value
	DataKey     string // Key in TokenData/PathData (e.g., "MovieTitle", "SeasonNumber")
}

// ConditionalSegment declares an optional path segment with a toggle.
type ConditionalSegment struct {
	Name         string // Condition name (e.g., "SeasonFolder")
	Label        string // UI label (e.g., "Use Season Folders")
	Description  string
	DefaultValue bool   // Default for new entities
	DataKey      string // Key in TokenData that carries the per-entity value
}

// TokenContext describes a named scope of template variables for the settings UI.
// Each context corresponds to a naming template (e.g., "movie-file", "episode-file").
type TokenContext struct {
	Name      string             // Context name (e.g., "movie-folder", "episode-file")
	Label     string             // UI label (e.g., "Movie Folder Format", "Episode File Format")
	Variables []TemplateVariable // Variables available in this context
	Variants  []string           // Format sub-variants (e.g., ["standard", "daily", "anime"])
	IsFolder  bool               // true for folder templates, false for file templates
}

// FormatOption declares a module-specific naming option.
type FormatOption struct {
	Key          string // Setting key (e.g., "multi_episode_style", "colon_replacement")
	Label        string // UI label
	Description  string
	Type         string   // "enum" or "bool"
	EnumValues   []string // For enum type: allowed values
	DefaultValue string   // Default value as string
}
