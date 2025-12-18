package quality

import (
	"encoding/json"
	"time"
)

// Quality represents a quality tier.
type Quality struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`     // "bluray", "webdl", "hdtv", etc.
	Resolution int    `json:"resolution"` // 480, 720, 1080, 2160
	Weight     int    `json:"weight"`     // Higher = better quality
}

// QualityItem represents a quality in a profile with its allowed status.
type QualityItem struct {
	Quality Quality `json:"quality"`
	Allowed bool    `json:"allowed"`
}

// Profile represents a quality profile.
type Profile struct {
	ID        int64         `json:"id"`
	Name      string        `json:"name"`
	Cutoff    int           `json:"cutoff"` // Quality ID at which upgrades stop
	Items     []QualityItem `json:"items"`  // Ordered list of qualities
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// CreateProfileInput is used when creating a new profile.
type CreateProfileInput struct {
	Name   string        `json:"name"`
	Cutoff int           `json:"cutoff"`
	Items  []QualityItem `json:"items"`
}

// UpdateProfileInput is used when updating a profile.
type UpdateProfileInput struct {
	Name   string        `json:"name"`
	Cutoff int           `json:"cutoff"`
	Items  []QualityItem `json:"items"`
}

// PredefinedQualities are the standard quality definitions.
var PredefinedQualities = []Quality{
	{ID: 1, Name: "SDTV", Source: "tv", Resolution: 480, Weight: 1},
	{ID: 2, Name: "DVD", Source: "dvd", Resolution: 480, Weight: 2},
	{ID: 3, Name: "WEBRip-480p", Source: "webrip", Resolution: 480, Weight: 3},
	{ID: 4, Name: "HDTV-720p", Source: "tv", Resolution: 720, Weight: 4},
	{ID: 5, Name: "WEBRip-720p", Source: "webrip", Resolution: 720, Weight: 5},
	{ID: 6, Name: "WEBDL-720p", Source: "webdl", Resolution: 720, Weight: 6},
	{ID: 7, Name: "Bluray-720p", Source: "bluray", Resolution: 720, Weight: 7},
	{ID: 8, Name: "HDTV-1080p", Source: "tv", Resolution: 1080, Weight: 8},
	{ID: 9, Name: "WEBRip-1080p", Source: "webrip", Resolution: 1080, Weight: 9},
	{ID: 10, Name: "WEBDL-1080p", Source: "webdl", Resolution: 1080, Weight: 10},
	{ID: 11, Name: "Bluray-1080p", Source: "bluray", Resolution: 1080, Weight: 11},
	{ID: 12, Name: "Remux-1080p", Source: "remux", Resolution: 1080, Weight: 12},
	{ID: 13, Name: "HDTV-2160p", Source: "tv", Resolution: 2160, Weight: 13},
	{ID: 14, Name: "WEBRip-2160p", Source: "webrip", Resolution: 2160, Weight: 14},
	{ID: 15, Name: "WEBDL-2160p", Source: "webdl", Resolution: 2160, Weight: 15},
	{ID: 16, Name: "Bluray-2160p", Source: "bluray", Resolution: 2160, Weight: 16},
	{ID: 17, Name: "Remux-2160p", Source: "remux", Resolution: 2160, Weight: 17},
}

// qualityByID is a lookup map for qualities by ID.
var qualityByID map[int]Quality

func init() {
	qualityByID = make(map[int]Quality)
	for _, q := range PredefinedQualities {
		qualityByID[q.ID] = q
	}
}

// GetQualityByID returns a quality by its ID.
func GetQualityByID(id int) (Quality, bool) {
	q, ok := qualityByID[id]
	return q, ok
}

// GetQualityByName finds a quality by name.
func GetQualityByName(name string) (Quality, bool) {
	for _, q := range PredefinedQualities {
		if q.Name == name {
			return q, true
		}
	}
	return Quality{}, false
}

// DefaultProfile returns a default "Any" profile that accepts all qualities.
func DefaultProfile() Profile {
	items := make([]QualityItem, len(PredefinedQualities))
	for i, q := range PredefinedQualities {
		items[i] = QualityItem{
			Quality: q,
			Allowed: true,
		}
	}
	return Profile{
		Name:   "Any",
		Cutoff: 11, // Bluray-1080p
		Items:  items,
	}
}

// HD1080pProfile returns a profile targeting 1080p content.
func HD1080pProfile() Profile {
	items := make([]QualityItem, len(PredefinedQualities))
	for i, q := range PredefinedQualities {
		items[i] = QualityItem{
			Quality: q,
			Allowed: q.Resolution >= 720 && q.Resolution <= 1080,
		}
	}
	return Profile{
		Name:   "HD-1080p",
		Cutoff: 11, // Bluray-1080p
		Items:  items,
	}
}

// Ultra4KProfile returns a profile targeting 4K content.
func Ultra4KProfile() Profile {
	items := make([]QualityItem, len(PredefinedQualities))
	for i, q := range PredefinedQualities {
		items[i] = QualityItem{
			Quality: q,
			Allowed: q.Resolution >= 1080,
		}
	}
	return Profile{
		Name:   "Ultra-HD",
		Cutoff: 16, // Bluray-2160p
		Items:  items,
	}
}

// SerializeItems converts quality items to JSON for database storage.
func SerializeItems(items []QualityItem) (string, error) {
	data, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeserializeItems parses JSON quality items from database.
func DeserializeItems(data string) ([]QualityItem, error) {
	var items []QualityItem
	if err := json.Unmarshal([]byte(data), &items); err != nil {
		return nil, err
	}
	return items, nil
}

// IsAcceptable checks if a quality is acceptable for this profile.
func (p *Profile) IsAcceptable(qualityID int) bool {
	for _, item := range p.Items {
		if item.Quality.ID == qualityID && item.Allowed {
			return true
		}
	}
	return false
}

// IsUpgrade checks if candidate quality is an upgrade over current quality.
func (p *Profile) IsUpgrade(currentQualityID, candidateQualityID int) bool {
	// If we're at or above cutoff, no upgrades
	currentQuality, ok := GetQualityByID(currentQualityID)
	if ok && currentQuality.Weight >= p.getCutoffWeight() {
		return false
	}

	// Check if candidate is better
	candidateQuality, ok := GetQualityByID(candidateQualityID)
	if !ok {
		return false
	}

	// Must be allowed
	if !p.IsAcceptable(candidateQualityID) {
		return false
	}

	// Must be higher weight
	return candidateQuality.Weight > currentQuality.Weight
}

// getCutoffWeight returns the weight of the cutoff quality.
func (p *Profile) getCutoffWeight() int {
	if q, ok := GetQualityByID(p.Cutoff); ok {
		return q.Weight
	}
	return 0
}
