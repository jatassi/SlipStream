package providers

import "encoding/json"

type FieldKind string

const (
	Text   FieldKind = "text"
	URL    FieldKind = "url"
	Number FieldKind = "number"
	Select FieldKind = "select"
	Switch FieldKind = "switch"
	// List is addresses, devices, tags or Apprise URLs.
	List FieldKind = "list"
	// Headers is name and value rows, each value secret.
	Headers FieldKind = "headers"
	// NodeField is Custom Script's Node.
	NodeField FieldKind = "node"
	// Duration is Custom Script's Stop the script after.
	Duration FieldKind = "duration"
)

// Field is one setting a connection of a provider takes, in the order the
// sheet shows it.
type Field struct {
	Key   string
	Label string
	Kind  FieldKind
	// Secret: sealed at rest and never sent to the browser.
	Secret bool
	// Optional: a secret the Admin may remove, or a setting they may leave
	// empty.
	Optional bool
	// Details: under Details in the sheet.
	Details bool
	Default json.RawMessage
	Help    string
	// Options are a Select's choices.
	Options []Option
	// Min and Max bound a Number.
	Min, Max *int
	// Shown is what of a secret value the API may show.
	Shown Shown
}

type Option struct {
	Value string
	Label string
}

// Shown is what of a secret value the API may show.
type Shown uint8

const (
	ShownNone Shown = iota
	// ShownHost: the host of a secret URL, such as a Webhook's.
	ShownHost
)

// ImageWay is how a provider takes a poster.
type ImageWay uint8

const (
	// NoImage: Pushbullet, Custom Script.
	NoImage ImageWay = iota
	// Bytes: an attached image (Discord, Telegram, Pushover, ntfy, Apprise,
	// Email).
	Bytes
	// PublicURL: the metadata source's own image URL (Slack, Gotify).
	PublicURL
	// BodyURL: the public URL inside the Webhook body.
	BodyURL
)
