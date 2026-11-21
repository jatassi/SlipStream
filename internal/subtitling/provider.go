// Package subtitling is the Subtitles Role's Node half and its wire: what
// a Subtitle Provider is, SubtitlesService on a holder, and Remote, the
// Controller's typed client of it. A holder keeps no work between calls:
// keys, maps, gates and Timing sync's constants arrive inside each call,
// and the Controller checks every answer.
package subtitling

import (
	"context"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/subtitles/cues"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
)

type ProviderID = calls.ID

// Kind is what a provider searches: a film by its ids, or an episode by
// its series ids and Aired numbers. Content Types answer kinds; providers
// never name a Content Type.
type Kind uint8

const (
	Feature Kind = iota + 1
	Episode
)

// Spec is a provider's facts, the same on every Node.
type Spec struct {
	ID    ProviderID
	Label string
	Kinds []Kind
	// Keyed: the provider needs the Admin's key.
	Keyed   bool
	KeyPage string
	// Position is its default place in the search order.
	Position int
}

// Provider is one hand-written client.
type Provider interface {
	Spec() Spec
	// CheckKey answers the key's account and plan, with the allowance in
	// the call's outcome, or a *calls.Failed whose Outcome is Refused with
	// the provider's message.
	CheckKey(ctx context.Context, c calls.Credentials) (Account, error)
	Search(ctx context.Context, c calls.Credentials, q Query) ([]Offer, error)
	// Download answers one offer's bytes as the provider sends them: an
	// archive or a file.
	Download(ctx context.Context, c calls.Credentials, o Offer) (Download, error)
}

type Account struct{ Name, Plan string }

// Query is one want's search: the Subject and the wanted language and
// kind.
type Query struct {
	Subject  Subject
	Language language.Code
	Forced   bool
}

// Subject is what providers are searched for: ids, title and year, and for
// an episode the Aired season and numbers of every Item the file holds.
type Subject struct {
	Kind       Kind
	IMDb, TMDB string
	TVDB       string
	Title      string
	Year       int
	Season     int
	Episodes   []int
}

// Offer is one candidate as a provider describes it, parsed into domain
// facts at the client's boundary. Key is opaque and provider-scoped, never
// a URL; Key and Provider together name it everywhere.
type Offer struct {
	Provider ProviderID
	Key      string
	// Releases are the release names it was made for, the provider's
	// first.
	Releases []string
	Language language.Code
	Forced   bool
	HI       bool
	// AI is machine or AI translated.
	AI bool
	// Pack is several episodes in one download.
	Pack      bool
	Season    int
	Episodes  []int
	Downloads int
	Rating    float64
	// Image: the provider says its files are image subtitles.
	Image bool
	// Page is the provider's page for it, shown, never fetched.
	Page string
}

type Download struct {
	Name  string
	Bytes []byte
}

// Verdict is how a Fetch or Sync ended on the holder.
type Verdict uint8

const (
	Passed Verdict = iota + 1
	FailedTiming
	Unreadable
	Image
	WrongEpisode
	TooLarge
	// Unfetched is a candidate the provider didn't serve; the outcomes say
	// why.
	Unfetched
)

// FetchAsk is one Fetch: the candidate, its key, the want, the file's map
// and the Controller's gate and constants.
type FetchAsk struct {
	Offer     Offer
	Key       string
	Want      Query
	Map       []byte
	Gate      timing.Gate
	Constants timing.Constants
	FPS       float64
}

// SyncAsk is one Sync: bytes the Controller holds, timed against a map.
type SyncAsk struct {
	Name      string
	Bytes     []byte
	Language  language.Code
	Map       []byte
	Gate      timing.Gate
	Constants timing.Constants
	FPS       float64
}

// Fetched is a holder's answer as the Controller receives it, unchecked.
type Fetched struct {
	// Key is the candidate the holder says it fetched.
	Key     string
	Verdict Verdict
	// Words say why it is unreadable.
	Words    string
	Format   cues.Format
	Original []byte
	Retimed  []byte
	// Claimed is the holder's timing: its segments and numbers.
	Claimed     timing.Fit
	Members     []Member
	MembersLeft int
}

// Member is one more episode of a pack, as a converted original.
type Member struct {
	Name     string
	Season   int
	Episodes []int
	Format   cues.Format
	Bytes    []byte
}
