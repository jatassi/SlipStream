// Package cardigann implements a Cardigann-compatible indexer definition system.
// It parses YAML definition files and executes searches against arbitrary indexer sites.
package cardigann

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// StringOrArray is a type that can unmarshal from either a string or an array of strings.
// When unmarshaled, it always stores as a single string (joining array elements if needed).
type StringOrArray string

// UnmarshalYAML implements custom YAML unmarshaling for StringOrArray.
func (s *StringOrArray) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*s = StringOrArray(value.Value)
		return nil
	case yaml.SequenceNode:
		var arr []string
		if err := value.Decode(&arr); err != nil {
			return err
		}
		// Join array elements - for headers, typically there's only one value
		if len(arr) > 0 {
			*s = StringOrArray(strings.Join(arr, ", "))
		}
		return nil
	default:
		return fmt.Errorf("cannot unmarshal %v into StringOrArray", value.Kind)
	}
}

// Definition represents a parsed Cardigann YAML definition file.
// These definitions describe how to interact with a torrent/usenet indexer site.
type Definition struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Language     string   `yaml:"language"`
	Type         string   `yaml:"type"`     // public, private, semi-private
	Encoding     string   `yaml:"encoding"` // UTF-8, etc.
	RequestDelay float64  `yaml:"requestDelay"`
	Links        []string `yaml:"links"`
	LegacyLinks  []string `yaml:"legacylinks"`

	Caps     Capabilities   `yaml:"caps"`
	Settings []Setting      `yaml:"settings"`
	Login    *LoginBlock    `yaml:"login"`
	Search   SearchBlock    `yaml:"search"`
	Download *DownloadBlock `yaml:"download"`
}

// Capabilities describes what search modes and categories the indexer supports.
type Capabilities struct {
	CategoryMappings []CategoryMapping   `yaml:"categorymappings"`
	Modes            map[string][]string `yaml:"modes"` // search, tv-search, movie-search -> supported params
	AllowRawSearch   bool                `yaml:"allowrawsearch"`
}

// CategoryMapping maps indexer-specific category IDs to standard Newznab categories.
type CategoryMapping struct {
	ID      string `yaml:"id"`
	Cat     string `yaml:"cat"`  // Newznab category name (e.g., "Movies/HD")
	Desc    string `yaml:"desc"` // Human-readable description
	Default bool   `yaml:"default"`
}

// Setting defines a user-configurable option for the indexer.
type Setting struct {
	Name    string            `yaml:"name" json:"name"`
	Type    string            `yaml:"type" json:"type"` // text, password, checkbox, select, info, info_cookie, info_flaresolverr
	Label   string            `yaml:"label" json:"label"`
	Default string            `yaml:"default" json:"default,omitempty"`
	Options map[string]string `yaml:"options" json:"options,omitempty"` // For select type
}

// LoginBlock defines how to authenticate with the indexer.
type LoginBlock struct {
	Path           string                       `yaml:"path"`
	Method         string                       `yaml:"method"` // post, form, cookie, oneurl
	Form           string                       `yaml:"form"`   // CSS selector for form element
	Selectors      bool                         `yaml:"selectors"`
	Inputs         map[string]string            `yaml:"inputs"`
	SelectorInputs map[string]SelectorDef       `yaml:"selectorinputs"`
	Error          []ErrorSelector              `yaml:"error"`
	Test           TestBlock                    `yaml:"test"`
	Captcha        *CaptchaBlock                `yaml:"captcha"`
	Cookies        []string                     `yaml:"cookies"` // Required cookie names
	Headers        map[string]StringOrArray     `yaml:"headers"`
}

// SelectorDef defines how to extract a value using a CSS selector.
type SelectorDef struct {
	Selector  string   `yaml:"selector"`
	Attribute string   `yaml:"attribute"`
	Filters   []Filter `yaml:"filters"`
}

// ErrorSelector defines how to detect and extract error messages.
type ErrorSelector struct {
	Selector string          `yaml:"selector"`
	Message  *TextOrSelector `yaml:"message"`
}

// TextOrSelector can be either static text or a selector definition.
type TextOrSelector struct {
	Text     string `yaml:"text"`
	Selector string `yaml:"selector"`
}

// TestBlock defines how to verify successful authentication.
type TestBlock struct {
	Path     string `yaml:"path"`
	Selector string `yaml:"selector"`
}

// CaptchaBlock defines CAPTCHA handling (not fully implemented).
type CaptchaBlock struct {
	Type     string `yaml:"type"` // image, recaptcha, etc.
	Selector string `yaml:"selector"`
	Input    string `yaml:"input"`
	SiteKey  string `yaml:"sitekey"`
}

// SearchBlock defines how to execute searches and parse results.
type SearchBlock struct {
	Paths                []SearchPath             `yaml:"paths"`
	Inputs               map[string]string        `yaml:"inputs"`
	KeywordsFilters      []Filter                 `yaml:"keywordsfilters"`
	PreprocessingFilters []Filter                 `yaml:"preprocessingfilters"`
	Headers              map[string]StringOrArray `yaml:"headers"`
	Rows                 RowSelector              `yaml:"rows"`
	Fields               map[string]Field         `yaml:"fields"`
	Error                []ErrorSelector          `yaml:"error"`
}

// SearchPath defines a search endpoint, optionally restricted to certain categories.
type SearchPath struct {
	Path       string          `yaml:"path"`
	Categories []string        `yaml:"categories"`
	Inputs     map[string]string `yaml:"inputs"`    // Path-specific inputs
	Method     string          `yaml:"method"`     // GET or POST
	Response   *ResponseConfig `yaml:"response"`
	Followredirect bool        `yaml:"followredirect"`
}

// ResponseConfig specifies the response format.
type ResponseConfig struct {
	Type      string `yaml:"type"`      // json, xml, html (default)
	NoResultsMessage string `yaml:"noresultsmessage"`
}

// RowSelector defines how to find result rows in the response.
type RowSelector struct {
	Selector    string       `yaml:"selector"`
	Attribute   string       `yaml:"attribute"` // For JSON: extract this nested object from each row
	After       int          `yaml:"after"`     // Skip N rows (e.g., header row)
	Remove      string       `yaml:"remove"`    // Remove elements matching this selector
	Multiple    bool         `yaml:"multiple"`
	DateHeaders *DateHeaders `yaml:"dateheaders"`
	Count       *CountBlock  `yaml:"count"`
	MissingAttributeEquals string `yaml:"missingAttributeEqualsNoResults"`
}

// DateHeaders handles sites that group results by date with header rows.
type DateHeaders struct {
	Selector string   `yaml:"selector"`
	Filters  []Filter `yaml:"filters"`
}

// CountBlock validates result count.
type CountBlock struct {
	Selector string   `yaml:"selector"`
	Filters  []Filter `yaml:"filters"`
}

// Field defines how to extract a single piece of data from a result row.
type Field struct {
	Selector  string            `yaml:"selector"`
	Attribute string            `yaml:"attribute"` // href, src, value, etc.
	Text      string            `yaml:"text"`      // Static value
	Remove    string            `yaml:"remove"`    // Remove elements before extracting
	Optional  bool              `yaml:"optional"`
	Default   string            `yaml:"default"`
	Filters   []Filter          `yaml:"filters"`
	Case      map[string]string `yaml:"case"` // Value mapping
}

// Filter transforms extracted values.
type Filter struct {
	Name string      `yaml:"name"`
	Args interface{} `yaml:"args"` // string, []string, or nil
}

// DownloadBlock defines how to construct download URLs.
type DownloadBlock struct {
	Selectors []DownloadSelector `yaml:"selectors"`
	Before    *BeforeRequest     `yaml:"before"`
	InfoHash  *InfoHashBlock     `yaml:"infohash"`
	Method    string             `yaml:"method"`
}

// DownloadSelector defines a selector for finding download links.
type DownloadSelector struct {
	Selector  string `yaml:"selector"`
	Attribute string `yaml:"attribute"`
	Filters   []Filter `yaml:"filters"`
}

// BeforeRequest defines a request to make before downloading.
type BeforeRequest struct {
	Path    string                   `yaml:"path"`
	Method  string                   `yaml:"method"`
	Inputs  map[string]string        `yaml:"inputs"`
	Headers map[string]StringOrArray `yaml:"headers"`
}

// InfoHashBlock defines how to extract magnet link info.
type InfoHashBlock struct {
	Hash  Field `yaml:"hash"`
	Title Field `yaml:"title"`
}

// ParseDefinition parses a Cardigann YAML definition from bytes.
func ParseDefinition(data []byte) (*Definition, error) {
	var def Definition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse definition YAML: %w", err)
	}
	return &def, nil
}

// ParseDefinitionFile parses a Cardigann YAML definition from a file.
func ParseDefinitionFile(path string) (*Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read definition file: %w", err)
	}
	return ParseDefinition(data)
}

// GetBaseURL returns the primary URL for this indexer.
func (d *Definition) GetBaseURL() string {
	if len(d.Links) > 0 {
		return d.Links[0]
	}
	return ""
}

// GetProtocol determines if this is a torrent or usenet indexer.
// This is inferred from the category mappings.
func (d *Definition) GetProtocol() string {
	// Check category mappings for usenet-specific categories
	for _, cat := range d.Caps.CategoryMappings {
		// Usenet categories typically include things like "TV/SD", "Movies/HD" without torrent-specific markers
		// For now, default to torrent - most definitions are torrent indexers
		_ = cat
	}
	return "torrent"
}

// GetPrivacy returns the privacy level (public, private, semi-private).
func (d *Definition) GetPrivacy() string {
	if d.Type == "" {
		return "public"
	}
	return d.Type
}

// HasLogin returns true if this indexer requires authentication.
func (d *Definition) HasLogin() bool {
	return d.Login != nil && d.Login.Method != ""
}

// SupportsSearch returns true if the indexer supports the given search mode.
func (d *Definition) SupportsSearch(mode string) bool {
	if d.Caps.Modes == nil {
		return false
	}
	_, ok := d.Caps.Modes[mode]
	return ok
}
