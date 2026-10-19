// Package metadata is how SlipStream reads metadata: every call leaves
// through a Node holding the Metadata Role. The Controller fills an Ask for
// each call and folds the outcomes holders return into provider status; a
// holder runs a Content Type's provider half over its Node's call layer and
// keeps nothing durable.
package metadata

import (
	"maps"
	"sync"
	"time"
)

// ProviderID names one metadata provider, such as TMDB or OMDb. The core
// never names one; each provider package declares its own.
type ProviderID string

// Ask is what one metadata call needs that a holder doesn't keep. Types and
// provider clients read Language, Region and Ratings and pass the Ask on;
// only the call layer reads the keys and the cache policy, so no type can
// read or log a key.
type Ask struct {
	// Language is the metadata language, such as "en-US".
	Language string
	// Region is the release region, such as "US".
	Region string
	// Ratings asks the ratings providers too.
	Ratings bool

	keys  map[ProviderID]string
	fresh bool
	// cacheFor is how old a cached answer may be; zero serves none.
	cacheFor time.Duration
	// generation is the cache generation in effect; a holder whose cache
	// holds another drops it.
	generation uint64
	// tally collects the outcome of every call made for this Ask; nil
	// records nothing.
	tally *tally
}

type tally struct {
	mu       sync.Mutex
	outcomes []Outcome
}

func (t *tally) record(o Outcome) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.outcomes = append(t.outcomes, o)
}

type AskOptions struct {
	Language string
	Region   string
	Ratings  bool
	// Fresh bypasses the holder's cache and writes the answer back.
	Fresh bool
	// Keys are the keys in effect; a provider with none is called keyless.
	Keys map[ProviderID]string
}

// NewAsk builds an Ask outside the Controller, for tests and conformance
// suites that send a real key.
func NewAsk(o AskOptions) Ask {
	return Ask{Language: o.Language, Region: o.Region, Ratings: o.Ratings, keys: maps.Clone(o.Keys), fresh: o.Fresh}
}
