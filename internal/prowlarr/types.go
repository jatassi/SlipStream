// Package prowlarr provides integration with Prowlarr for indexer management.
package prowlarr

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/indexer/types"
)

// protocolFromString converts a string protocol name to the Protocol type.
func protocolFromString(s string) types.Protocol {
	switch strings.ToLower(s) {
	case "torrent":
		return types.ProtocolTorrent
	case "usenet":
		return types.ProtocolUsenet
	default:
		return types.ProtocolTorrent
	}
}

// IndexerMode represents the active indexer management mode.
type IndexerMode string

const (
	ModeSlipStream IndexerMode = "slipstream"
	ModeProwlarr   IndexerMode = "prowlarr"
)

// Config holds Prowlarr connection and behavior configuration.
type Config struct {
	ID                    int64           `json:"id"`
	Enabled               bool            `json:"enabled"`
	URL                   string          `json:"url"`
	APIKey                string          `json:"apiKey"`
	MovieCategories       []int           `json:"movieCategories"`
	TVCategories          []int           `json:"tvCategories"`
	Timeout               int             `json:"timeout"`
	SkipSSLVerify         bool            `json:"skipSslVerify"`
	Capabilities          *Capabilities   `json:"capabilities,omitempty"`
	CapabilitiesUpdatedAt *time.Time      `json:"capabilitiesUpdatedAt,omitempty"`
	CreatedAt             time.Time       `json:"createdAt"`
	UpdatedAt             time.Time       `json:"updatedAt"`
}

// DefaultMovieCategories returns the default Newznab movie category IDs.
func DefaultMovieCategories() []int {
	return []int{2000, 2010, 2020, 2030, 2040, 2045, 2050, 2060}
}

// DefaultTVCategories returns the default Newznab TV category IDs.
func DefaultTVCategories() []int {
	return []int{5000, 5010, 5020, 5030, 5040, 5045, 5050, 5060, 5070, 5080}
}

// Indexer represents an indexer configured in Prowlarr.
type Indexer struct {
	ID           int              `json:"id"`
	Name         string           `json:"name"`
	Protocol     types.Protocol   `json:"protocol"`
	Privacy      types.Privacy    `json:"privacy,omitempty"`
	Priority     int              `json:"priority"`
	Enable       bool             `json:"enable"`
	Status       IndexerStatus    `json:"status"`
	Capabilities IndexerCaps      `json:"capabilities,omitempty"`
	Fields       []IndexerField   `json:"fields,omitempty"`
}

// IndexerStatus represents the health status of a Prowlarr indexer.
type IndexerStatus int

const (
	IndexerStatusHealthy  IndexerStatus = 0
	IndexerStatusWarning  IndexerStatus = 1
	IndexerStatusDisabled IndexerStatus = 2
	IndexerStatusFailed   IndexerStatus = 3
)

func (s IndexerStatus) String() string {
	switch s {
	case IndexerStatusHealthy:
		return "Healthy"
	case IndexerStatusWarning:
		return "Warning"
	case IndexerStatusDisabled:
		return "Disabled"
	case IndexerStatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// IndexerCaps represents the capabilities of a Prowlarr indexer.
type IndexerCaps struct {
	SupportsSearch bool     `json:"supportsSearch"`
	SupportsTV     bool     `json:"supportsTvSearch"`
	SupportsMovies bool     `json:"supportsMovieSearch"`
	Categories     []int    `json:"categories,omitempty"`
}

// IndexerField represents a configuration field for a Prowlarr indexer.
type IndexerField struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// Capabilities represents Prowlarr's aggregated capabilities from the caps endpoint.
type Capabilities struct {
	Server     ServerInfo    `json:"server,omitempty"`
	Limits     LimitsInfo    `json:"limits,omitempty"`
	Searching  SearchingInfo `json:"searching,omitempty"`
	Categories []Category    `json:"categories,omitempty"`
}

// ServerInfo contains Prowlarr server information.
type ServerInfo struct {
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

// LimitsInfo contains rate/result limits.
type LimitsInfo struct {
	Max     int `json:"max,omitempty"`
	Default int `json:"default,omitempty"`
}

// SearchingInfo describes supported search types.
type SearchingInfo struct {
	Search      SearchTypeInfo `json:"search,omitempty"`
	TVSearch    SearchTypeInfo `json:"tv-search,omitempty"`
	MovieSearch SearchTypeInfo `json:"movie-search,omitempty"`
}

// SearchTypeInfo describes a specific search type's capabilities.
type SearchTypeInfo struct {
	Available       bool     `json:"available"`
	SupportedParams []string `json:"supportedParams,omitempty"`
}

// Category represents a Newznab category.
type Category struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	Subcategories []Category `json:"subcategories,omitempty"`
}

// ConnectionStatus represents the current Prowlarr connection state.
type ConnectionStatus struct {
	Connected   bool       `json:"connected"`
	LastChecked *time.Time `json:"lastChecked,omitempty"`
	Error       string     `json:"error,omitempty"`
	Version     string     `json:"version,omitempty"`
}

// TorznabFeed represents the root RSS feed from a Torznab response.
type TorznabFeed struct {
	XMLName xml.Name       `xml:"rss"`
	Channel TorznabChannel `xml:"channel"`
}

// TorznabChannel represents the channel element in a Torznab RSS feed.
type TorznabChannel struct {
	Title       string        `xml:"title"`
	Description string        `xml:"description"`
	Items       []TorznabItem `xml:"item"`
}

// TorznabItem represents a single release in a Torznab response.
type TorznabItem struct {
	Title       string             `xml:"title"`
	GUID        string             `xml:"guid"`
	Link        string             `xml:"link"`
	Comments    string             `xml:"comments,omitempty"`
	PubDate     string             `xml:"pubDate"`
	Size        int64              `xml:"size"`
	Description string             `xml:"description,omitempty"`
	Enclosure   TorznabEnclosure   `xml:"enclosure"`
	Attributes  []TorznabAttribute `xml:"attr"`
}

// TorznabEnclosure represents the enclosure element with download info.
type TorznabEnclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// TorznabAttribute represents a torznab:attr element with extended info.
type TorznabAttribute struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// TorznabCaps represents the capabilities XML response.
type TorznabCaps struct {
	XMLName    xml.Name           `xml:"caps"`
	Server     TorznabServer      `xml:"server"`
	Limits     TorznabLimits      `xml:"limits"`
	Searching  TorznabSearching   `xml:"searching"`
	Categories TorznabCategories  `xml:"categories"`
}

// TorznabServer represents server info in capabilities.
type TorznabServer struct {
	Title   string `xml:"title,attr"`
	Version string `xml:"version,attr"`
}

// TorznabLimits represents limits in capabilities.
type TorznabLimits struct {
	Max     int `xml:"max,attr"`
	Default int `xml:"default,attr"`
}

// TorznabSearching represents searching capabilities.
type TorznabSearching struct {
	Search      TorznabSearchType `xml:"search"`
	TVSearch    TorznabSearchType `xml:"tv-search"`
	MovieSearch TorznabSearchType `xml:"movie-search"`
}

// TorznabSearchType represents a search type capability.
type TorznabSearchType struct {
	Available       string `xml:"available,attr"`
	SupportedParams string `xml:"supportedParams,attr"`
}

// TorznabCategories is a container for category elements.
type TorznabCategories struct {
	Categories []TorznabCategory `xml:"category"`
}

// TorznabCategory represents a category in capabilities.
type TorznabCategory struct {
	ID            int               `xml:"id,attr"`
	Name          string            `xml:"name,attr"`
	Subcategories []TorznabCategory `xml:"subcat"`
}

// GetAttribute returns the value of a torznab attribute by name.
func (item *TorznabItem) GetAttribute(name string) string {
	for _, attr := range item.Attributes {
		if attr.Name == name {
			return attr.Value
		}
	}
	return ""
}

// GetIntAttribute returns the integer value of a torznab attribute.
func (item *TorznabItem) GetIntAttribute(name string, defaultVal int) int {
	val := item.GetAttribute(name)
	if val == "" {
		return defaultVal
	}
	var result int
	if _, err := json.Marshal(val); err == nil {
		json.Unmarshal([]byte(val), &result)
	}
	return result
}

// GetFloatAttribute returns the float value of a torznab attribute.
func (item *TorznabItem) GetFloatAttribute(name string, defaultVal float64) float64 {
	val := item.GetAttribute(name)
	if val == "" {
		return defaultVal
	}
	var result float64
	if _, err := json.Marshal(val); err == nil {
		json.Unmarshal([]byte(val), &result)
	}
	return result
}

// ToReleaseInfo converts a TorznabItem to a standard ReleaseInfo.
func (item *TorznabItem) ToReleaseInfo(indexerName string) types.ReleaseInfo {
	var pubDate time.Time
	for _, layout := range []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
	} {
		if parsed, err := time.Parse(layout, item.PubDate); err == nil {
			pubDate = parsed
			break
		}
	}

	downloadURL := item.Link
	if downloadURL == "" && item.Enclosure.URL != "" {
		downloadURL = item.Enclosure.URL
	}

	size := item.Size
	if size == 0 && item.Enclosure.Length > 0 {
		size = item.Enclosure.Length
	}

	protocol := types.ProtocolTorrent
	if item.Enclosure.Type == "application/x-nzb" {
		protocol = types.ProtocolUsenet
	}

	return types.ReleaseInfo{
		GUID:        item.GUID,
		Title:       item.Title,
		Description: item.Description,
		DownloadURL: downloadURL,
		InfoURL:     item.Comments,
		Size:        size,
		PublishDate: pubDate,
		IndexerName: indexerName,
		Protocol:    protocol,
	}
}

// ToTorrentInfo converts a TorznabItem to a TorrentInfo with extended attributes.
func (item *TorznabItem) ToTorrentInfo(indexerName string) types.TorrentInfo {
	release := item.ToReleaseInfo(indexerName)

	return types.TorrentInfo{
		ReleaseInfo:          release,
		Seeders:              item.GetIntAttribute("seeders", 0),
		Leechers:             item.GetIntAttribute("leechers", 0),
		InfoHash:             item.GetAttribute("infohash"),
		MinimumRatio:         item.GetFloatAttribute("minimumratio", 0),
		MinimumSeedTime:      int64(item.GetIntAttribute("minimumseedtime", 0)),
		DownloadVolumeFactor: item.GetFloatAttribute("downloadvolumefactor", 1),
		UploadVolumeFactor:   item.GetFloatAttribute("uploadvolumefactor", 1),
	}
}

// SearchRequest represents a search request to Prowlarr.
type SearchRequest struct {
	Query      string
	Type       string // "search", "tvsearch", "movie"
	Categories []int
	ImdbID     string
	TmdbID     int
	TvdbID     int
	Season     int
	Episode    int
	Limit      int
	Offset     int
}

// SearchResponse represents aggregated search results from Prowlarr.
type SearchResponse struct {
	Results       []types.TorrentInfo `json:"results"`
	TotalResults  int                 `json:"totalResults"`
	IndexersUsed  int                 `json:"indexersUsed"`
	Errors        []SearchError       `json:"errors,omitempty"`
}

// SearchError represents an error from an indexer during search.
type SearchError struct {
	IndexerName string `json:"indexerName"`
	Message     string `json:"message"`
}

// ContentType represents what content types an indexer should be used for.
type ContentType string

const (
	ContentTypeMovies ContentType = "movies"
	ContentTypeSeries ContentType = "series"
	ContentTypeBoth   ContentType = "both"
)

// IndexerSettings holds per-indexer configuration stored in SlipStream.
type IndexerSettings struct {
	ProwlarrIndexerID int64       `json:"prowlarrIndexerId"`
	Priority          int         `json:"priority"`
	ContentType       ContentType `json:"contentType"`
	MovieCategories   []int       `json:"movieCategories,omitempty"`
	TVCategories      []int       `json:"tvCategories,omitempty"`
	SuccessCount      int64       `json:"successCount"`
	FailureCount      int64       `json:"failureCount"`
	LastFailureAt     *time.Time  `json:"lastFailureAt,omitempty"`
	LastFailureReason string      `json:"lastFailureReason,omitempty"`
	CreatedAt         time.Time   `json:"createdAt"`
	UpdatedAt         time.Time   `json:"updatedAt"`
}

// IndexerSettingsInput is used for creating/updating indexer settings.
type IndexerSettingsInput struct {
	Priority        int         `json:"priority"`
	ContentType     ContentType `json:"contentType"`
	MovieCategories []int       `json:"movieCategories,omitempty"`
	TVCategories    []int       `json:"tvCategories,omitempty"`
}

// IndexerWithSettings combines Prowlarr indexer data with SlipStream settings.
type IndexerWithSettings struct {
	Indexer
	Settings *IndexerSettings `json:"settings,omitempty"`
}
