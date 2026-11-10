package grabs

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

// Actor is who did something: an Admin's account and name, or the API
// key, Account nil and Name empty. The name is kept beside the account,
// since an account can be deleted.
type Actor struct {
	Account *identity.AccountID
	Name    string
}

// actorOf is by as an Actor: its account and display name, or the API
// key.
func actorOf(by identity.Admin) Actor {
	a, ok := by.Account()
	if !ok {
		return Actor{}
	}
	p := a.Profile()
	return Actor{Account: &p.ID, Name: p.DisplayName}
}

// Words is the Admin's name, or "the API key".
func (a Actor) Words() string {
	if a.Name == "" {
		return "the API key"
	}
	return a.Name
}

// Overrides is what an Admin chose past the decision when grabbing from a
// Manual search, as grabs_grab.overrides keeps it. Names are kept as they
// were at the grab.
type Overrides struct {
	Lifted []Lifted `json:"lifted,omitempty"`
	// Edition is the name of the cut the Grab was grabbed as, "" unless
	// that is another Edition than its title's cut. Its file is stored
	// under that cut.
	Edition   string          `json:"edition,omitempty"`
	Client    *ClientOverride `json:"client,omitempty"`
	Unblocked *Unblocked      `json:"unblocked,omitempty"`
}

// Lifted is a soft refusal grabbed past, with its words then: "Score 5 is
// below the minimum of 10 at 1080p."
type Lifted struct {
	Code  targets.ReasonCode `json:"code"`
	Words string             `json:"words"`
}

// ClientOverride is the client chosen and the one Tracker routing would
// have sent the Grab to, "" when no Node took downloads then, both in
// words.
type ClientOverride struct {
	Sent   string `json:"sent"`
	Routed string `json:"routed,omitempty"`
}

// Unblocked is the Blocklist entry removed to grab the Release.
type Unblocked struct {
	Cause BlockCause `json:"cause"`
	Since time.Time  `json:"since"`
}

// overrideWords is each of g's overrides in a sentence: "Jackson grabbed
// it although score 5 is below the minimum of 10 at 1080p.", "Jackson
// grabbed it as the Director's Cut Edition.", "Jackson sent it to
// qBittorrent on Den. Tracker routing would have sent it to the built-in
// client on Den.", "Jackson removed it from the Blocklist to grab it."
func (g Grab) overrideWords() []string {
	if g.By == nil {
		return nil
	}
	name, o := g.By.Name, g.Overrides
	said := func(named, keyed string) string {
		if name == "" {
			return keyed
		}
		return named
	}
	var out []string
	for _, l := range o.Lifted {
		why := uncapitalized(strings.TrimSuffix(l.Words, "."))
		out = append(out, said(name+" grabbed it although "+why+".", "It was grabbed through the API key although "+why+"."))
	}
	if e := o.Edition; e != "" {
		e = editionWords(e)
		out = append(out, said(name+" grabbed it as the "+e+".", "It was grabbed through the API key as the "+e+"."))
	}
	if c := o.Client; c != nil {
		s := said(name+" sent it to "+c.Sent+".", "It was sent to "+c.Sent+" through the API key.")
		if c.Routed != "" {
			s += " Tracker routing would have sent it to " + c.Routed + "."
		}
		out = append(out, s)
	}
	if o.Unblocked != nil {
		out = append(out, said(name+" removed it from the Blocklist to grab it.", "It was removed from the Blocklist through the API key to grab it."))
	}
	return out
}

func uncapitalized(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[n:]
}

// clientWords is the client t names, read in r: "qBittorrent on Den", or
// "the built-in client on Den".
func clientWords(ctx context.Context, r storage.Reader[file], t downloadclients.Target) (string, error) {
	named, err := nodes.Names(ctx, r)
	if err != nil {
		return "", err
	}
	node := ""
	for _, n := range named {
		if n.ID == t.Node {
			node = n.Name
		}
	}
	if t.Client == nil {
		return "the built-in client on " + node, nil
	}
	e, err := downloadclients.ExternalOf(ctx, r, *t.Client)
	if err != nil {
		return "", err
	}
	return e.Name + " on " + node, nil
}

// editionWords names a cut as an Edition, once: "Extended" reads "Extended
// Edition" and "Special Edition" stays as it is.
func editionWords(cut string) string {
	if strings.HasSuffix(cut, "Edition") {
		return cut
	}
	return cut + " Edition"
}
