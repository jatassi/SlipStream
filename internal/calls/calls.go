// Package calls is the shared way out of a Node to a key-holding Role
// provider (ADR 0021): a token bucket per provider per holder, per-key
// gates, tries with jittered backoff, Retry-After waits, classification,
// and keys on the query, in a header or through a login. The core never
// names a provider; each provider's client package declares its own.
package calls

import (
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// MaxBody bounds every answer a Line reads. A metadata image travels whole
// in one unary call, so this is also the largest image.
const MaxBody = 16 << 20

// ID names one provider, such as TMDB or OMDb.
type ID string

// Provider is one source's protocol facts, declared by its client package
// from its own timings.
type Provider struct {
	ID    ID
	Label string
	// Rate and Burst are the token bucket per provider per holder.
	Rate  rate.Limit
	Burst int
	// Key puts the key on the query and Header puts it in a header, such as
	// X-API-Key. Auth logs in with it for a bearer token. A provider sets at
	// most one; with none it is called keyless.
	Key    func(q url.Values, key string)
	Header func(h http.Header, key string)
	Auth   *Auth
	// Classify reads an answer the shared rules don't settle, such as a 200
	// whose body says the daily allowance is spent. Nil leaves every answer
	// to the shared rules.
	Classify func(status int, h http.Header, body []byte, now time.Time) (Outcome, bool)
	// KeyPage is where an Admin gets a key, opened in the browser only.
	KeyPage string
	// Attribution is what the provider's terms make SlipStream show beside
	// it, nil when they ask for none.
	Attribution *Attribution
}

// Auth is how a provider trades its key, and a PIN when one is set, for a
// bearer token.
type Auth struct {
	// Login is the JSON POST that logs in; pin is empty when none is set.
	Login func(key, pin string) (u *url.URL, body []byte)
	// Token reads a 200 login's token and when it expires.
	Token func(body []byte) (token string, expires time.Time, err error)
	// Refused is which credential a refused login names: CredentialKey
	// when the body can't tell.
	Refused func(status int, body []byte) Credential
	// Pin is whether the provider takes an optional PIN beside its key, so
	// Set key offers one.
	Pin bool
}

// Credential is the part of a login a provider refused.
type Credential uint8

const (
	CredentialKey Credential = iota + 1
	CredentialPin
)

// Attribution is a provider's required credit: its words, and the key of
// the logo the web app bundles for it, empty when it bundles none.
type Attribution struct {
	Text, Logo string
}

type Timings struct {
	// Attempts bounds the tries of one call, the first included.
	Attempts int `yaml:"attempts"`
	// Try bounds one request, its body included.
	Try time.Duration `yaml:"try"`
	// RetryMin and RetryMax bound the jittered backoff between tries.
	RetryMin time.Duration `yaml:"retry_min"`
	RetryMax time.Duration `yaml:"retry_max"`
	// Wait is how long one call may wait on a 429's Retry-After in all;
	// a longer one marks the provider limited until it ends.
	Wait time.Duration `yaml:"wait"`
}

// Credentials is what one call may send: each provider's key and the PIN
// beside it, and the tally its outcome goes to.
type Credentials struct {
	// Keys are the keys in effect; a provider with none is called keyless.
	Keys map[ID]string
	// Pins are the PINs beside Keys, for a provider that takes one.
	Pins map[ID]string
	// Tally collects the outcome of every call; nil records none.
	Tally *Tally
}

// Tally is the outcomes of the calls one holder request made, which the
// holder answers to the Controller.
type Tally struct {
	mu       sync.Mutex
	outcomes []Outcome
}

func (t *Tally) record(o Outcome) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.outcomes = append(t.outcomes, o)
}

// Outcomes is every outcome recorded so far, in order.
func (t *Tally) Outcomes() []Outcome {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.outcomes)
}
