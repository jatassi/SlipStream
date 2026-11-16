package providers

import (
	"encoding/json"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/notice"
)

// Values are a connection's settings or its opened secrets, by field key.
type Values map[string]json.RawMessage

// Poster is the image a message carries, in the provider's ImageWay.
type Poster struct {
	// Bytes, Type and Name are set for Bytes.
	Bytes []byte
	Type  string
	Name  string
	// URL is set for PublicURL and BodyURL.
	URL string
}

// Payload is everything a provider builds one message from.
type Payload struct {
	Kind notice.Kind
	At   time.Time
	// Message's Link is already absolute, or zero.
	Message   notice.Message
	Settings  Values
	Secrets   Values
	Poster    *Poster
	Endpoints Endpoints
	// Accepted are the Email addresses that already took this message.
	Accepted []string
}
