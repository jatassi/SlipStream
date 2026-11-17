package entry

import (
	"time"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/works"
)

// Message is the provider-neutral message every provider maps onto its
// format. Lines are whole sentences, at most four.
type Message struct {
	Title string
	Lines []string
	// Link is zero for no link.
	Link Link
	// Poster is zero for no poster.
	Poster artwork.Ref
	Tone   Tone
	// Body is the Webhook's and Custom Script's objects.
	Body Body
	// Detail is History's opened entry beyond Lines. Providers never read
	// it.
	Detail Detail
}

// Detail is each file an entry placed and each stage a Grab reached. The
// web words the instants in the browser's zone.
type Detail struct {
	Files  []DetailFile
	Stages []StageAt
}

type DetailFile struct {
	// Items are in words: "Episode 1".
	Items   []string
	Quality string
	// PlacedBy is the files placed-by wire value.
	PlacedBy string
	// Replaced is the replaced qualities in words, "" for none.
	Replaced string
	// ReplacedLabel is the replaced file's label, "" for none.
	ReplacedLabel string
}

type StageAt struct {
	// Stage is the grabs Stage wire value.
	Stage string    `json:"stage"`
	At    time.Time `json:"at"`
}

// Link is a path in the web app. The dispatcher prefixes the Address and
// drops the link when no Address is set.
type Link struct{ Path string }

// WorkLink is w's Work page.
func WorkLink(w works.WorkID) Link { return Link{Path: "/works/" + w.String()} }

// HealthLink is one Health item on the Health page.
func HealthLink(category, key string) Link {
	return Link{Path: "/health#health-" + category + "-" + key}
}

type Tone uint8

const (
	Info Tone = iota
	// Good: imported, upgraded, cleared.
	Good
	// Warn: a Warning item.
	Warn
	// Fail: Grab failed, an Error item.
	Fail
)

// Body is the Webhook body's per-event objects, version 1. Fields are only
// ever added.
type Body struct {
	Work   *BodyWork  `json:"work,omitempty"`
	Files  []BodyFile `json:"files,omitempty"`
	Health []BodyItem `json:"health,omitempty"`
}

type BodyWork struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ContentType string `json:"contentType"`
}

type BodyFile struct {
	Items    []string `json:"items"`
	Quality  string   `json:"quality"`
	Replaced string   `json:"replaced,omitempty"`
	PlacedBy string   `json:"placedBy"`
}

type BodyItem struct {
	Category string `json:"category"`
	Key      string `json:"key"`
	Level    string `json:"level"`
	Subject  string `json:"subject"`
	Text     string `json:"text"`
}
