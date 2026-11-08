package indexers

import (
	"cmp"
	"errors"
	"net/url"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/indexerfields"
	"github.com/jatassi/slipstream/internal/ports/port"
)

var (
	// ErrNoPort is a native Indexer of a port this build doesn't carry.
	ErrNoPort = errors.New("SlipStream carries no ported site with that id")
	// ErrNotAnAddress is a native Indexer's address that isn't an http or
	// https URL.
	ErrNotAnAddress = errors.New("the address isn't an http or https URL")
	// ErrNoCookieSignIn is a browser cookie for a port that doesn't sign in
	// with one.
	ErrNoCookieSignIn = errors.New("the site doesn't sign in with a browser cookie")
	// ErrUserAgentNeeded is a cookie pasted without the User-Agent its port
	// sends with it.
	ErrUserAgentNeeded = errors.New("the site accepts the cookie only with its browser's User-Agent")
)

// portDraft is the port d names with its address and plain settings, and
// its sealed settings. The address is its first known one when d sets
// none.
func (x *Indexers) portDraft(d Draft) (Port, map[string]string, error) {
	e, ok := x.cfg.Ports.Lookup(d.Port)
	if !ok {
		return Port{}, nil, ErrNoPort
	}
	if e.Auth == port.PastedCookie && d.Cookie == nil {
		return Port{}, nil, ErrCookieNeeded
	}
	return resettledPort(Port{ID: e.ID, Address: cmp.Or(d.BaseAddress, e.IndexerURLs[0]), Entry: e}, d.Values, nil, d.Cookie)
}

// resettledPort is p with values on top of its plain settings and sealed,
// the sealed ones saved before, checking its address and a cookie pasted
// for it.
func resettledPort(p Port, values, sealed map[string]string, c *PastedCookie) (Port, map[string]string, error) {
	if u, err := url.Parse(p.Address); err != nil || u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
		return Port{}, nil, ErrNotAnAddress
	}
	if c != nil && p.Entry.Auth != port.PastedCookie {
		return Port{}, nil, ErrNoCookieSignIn
	}
	if c != nil && p.Entry.PastesUserAgent && c.UserAgent == "" {
		return Port{}, nil, ErrUserAgentNeeded
	}
	var err error
	p.Plain, sealed, err = indexerfields.Split(p.Entry.Fields, values, p.Plain, sealed)
	return p, sealed, err
}

// orDefault is l, or a port's default d when l is no limit.
func orDefault(l Limit, d port.Limit) Limit {
	if l.Count != 0 || d.Count == 0 {
		return l
	}
	if d.Daily {
		return Limit{Count: d.Count, Per: Day}
	}
	return Limit{Count: d.Count, Per: Hour}
}

// Ported is what a port's add form says beyond a definition's.
type Ported struct {
	// UserAgent is a pasted cookie the site accepts only with its
	// browser's User-Agent.
	UserAgent bool
	// Searches and Grabs are a new Indexer's default limits.
	Searches, Grabs Limit
	// Recent is a site that lists its new uploads, so RSS sync can use it.
	Recent bool
	// Spacing is the least time between two requests to the site.
	Spacing time.Duration
	// Synced is when the Prowlarr commit the port was synced to was made.
	Synced time.Time
}

func portForm(e port.Entry) Form {
	f := Form{
		Entry: portEntry(e), Links: slices.Clone(e.IndexerURLs), Session: e.Auth.HoldsSession(), Cookie: e.Auth == port.PastedCookie,
		Code: e.Auth == port.FormWithCode,
		Port: &Ported{
			UserAgent: e.PastesUserAgent, Searches: orDefault(Limit{}, e.QueryLimit), Grabs: orDefault(Limit{}, e.GrabLimit),
			Recent: !e.Capabilities.NoRss, Spacing: cmp.Or(e.RateLimit, port.Floor), Synced: e.Upstream.Date,
		},
	}
	for _, field := range e.Fields {
		f.Fields = append(f.Fields, FormField{Field: field, Sealed: indexerfields.Sealed(field)})
	}
	return f
}

// portEntry is e as the Add Indexer list shows it.
func portEntry(e port.Entry) Entry {
	out := Entry{
		ID: string(e.ID), Name: e.Name, Description: e.Description, Language: cmp.Or(e.Language, "en-US"),
		Privacy: cardigann.Privacy(e.Privacy), Origin: FromPort, Version: e.Upstream.Date,
	}
	for _, m := range e.Capabilities.Categories.Mappings() {
		if !slices.Contains(out.Categories, m.Newznab) {
			out.Categories = append(out.Categories, m.Newznab)
		}
	}
	return out
}

// hidden reports whether the catalog's definition id is left out of the
// list: a port replaces it or has it as its own id.
func (x *Indexers) hidden(id string) bool {
	_, replaced := x.cfg.Ports.Hiding(id)
	_, own := x.cfg.Ports.Lookup(port.ID(id))
	return replaced || own
}
