package module

// QualityItem represents a quality tier defined by a module.
type QualityItem struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`     // "bluray", "webdl", "tv", etc.
	Resolution int    `json:"resolution"` // 480, 720, 1080, 2160
	Weight     int    `json:"weight"`     // Higher = better quality
}

// QualityResult is the result of parsing quality from a release title.
type QualityResult struct {
	Quality QualityItem
	Proper  bool
}
