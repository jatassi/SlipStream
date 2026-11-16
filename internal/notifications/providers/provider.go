// Package providers is the registry of notification providers: each one's
// fields, how it builds a request from a Message and how it reads the
// response. It is pure. The dispatcher hands what it builds to a Node and
// converts the reply back into these types.
package providers

import (
	"net/url"
	"time"
)

// ID names a provider. It is stored on each connection, so it never changes
// once shipped.
type ID string

type Provider struct {
	ID     ID
	Label  string
	Fields []Field
	Image  ImageWay
	// Host is the host a Node must reach to send to a connection with
	// settings, for the reach test. ok is false for a provider that sends
	// to no host (Email, Custom Script).
	Host     func(settings Values, e Endpoints) (host *url.URL, ok bool)
	Build    func(Payload) (Outbound, error)
	Classify func(r Reply, now time.Time) Verdict
	// Words overrides the generic sentence for a code this provider words
	// its own way.
	Words map[ReasonCode]string
}

// Table is every provider, in the order the sheet lists them.
var Table = []Provider{}

// ByID is the provider in Table with id.
func ByID(id ID) (Provider, bool) {
	for _, p := range Table {
		if p.ID == id {
			return p, true
		}
	}
	return Provider{}, false
}
