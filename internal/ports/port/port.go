// Package port is the kit every ported site is written against, and the
// framework that runs it: SlipStream's HttpIndexerBase. A port is Go
// shaped like its Prowlarr C# file, so an upstream diff ports line by
// line. A port declares what its site is (Entry) and supplies Prowlarr's
// strategy pieces (request generator, parser, login, ModifyRequest,
// Download). The framework owns everything else: the request loop,
// sign-in serialization, the one sign-in and one retry after a logged-out
// response, spacing, encodings, credential scoping, the undeclared-id
// skip, tiers and paging, release cleanup, the outcome each Prowlarr
// exception means for the port's auth family, Test and Recent.
//
// A site's package lives at internal/ports/<id>, a family's at
// internal/ports/internal/<family>. Either imports this package,
// internal/newznab, internal/indexerfields and internal/prowlarr/...
// only, never net/http's client, net's dialers, internal/searching,
// internal/cardigann or another site (lint rule ports-confined).
//
// Vocabulary map, C# to Go, for whoever ports an upstream diff:
//
//	TorrentIndexerBase<TSettings>          port.Define(Entry, Impl[S])
//	Name, IndexerUrls, LegacyUrls, ...     Entry fields of the same name, Go-cased
//	RateLimit                              Entry.RateLimit (a floor, never below 2 s)
//	SetCapabilities()                      Entry.Capabilities
//	caps.Categories.AddCategoryMapping     port.Map(tracker, newznab.X, desc)
//	NewznabStandardCategory.MoviesHD       newznab.MoviesHD
//	GetRequestGenerator().GetSearchRequests(Movie|Tv|Basic)   Impl.Movie, Impl.TV, Impl.Basic
//	IndexerPageableRequestChain            port.Chain (Add, AddTier)
//	HttpRequestBuilder / IndexerRequest    port.Get(u), port.Post(u), then AddQueryParam, AddFormParameter, SetHeader, ...
//	AllowAutoRedirect = false              r.NoRedirect()
//	GetParser().ParseResponse              Impl.Parse
//	IndexerResponse / HttpResponse         port.IndexerResponse
//	TorrentInfo                            port.TorrentInfo (same field names, Go initialisms:
//	                                       DownloadUrl is DownloadURL, ImdbId is ImdbID, Guid is GUID)
//	HttpUri                                port.NewURI
//	DoLogin, CheckIfLoginNeeded            the Login family's DoLogin and CheckIfLoginNeeded
//	ModifyRequest, GetDownloadRequest      Impl.ModifyRequest
//	Download(Uri)                          Impl.Download
//	Settings.BaseUrl                       ix.BaseURL
//	Capabilities.Categories.MapTorznabCapsToTrackers(cats)   ix.Categories.TrackerIDs(q.Categories)
//	_categories.MapTrackerCatToNewznab(id) ix.Categories.FromTracker(id)
//	MapTrackerCatDescToNewznab(desc)       ix.Categories.FromDesc(desc)
//	searchCriteria.SanitizedSearchTerm     q.SanitizedSearchTerm()
//	searchCriteria.SanitizedTvSearchString q.SanitizedTvSearchString()
//	searchCriteria.FullImdbId              q.FullImdbID()
//	searchCriteria.IsRssSearch             q.IsRssSearch()
//	searchCriteria.IsIdSearch              q.IsIDSearch()
//	ParseUtil.X, DateTimeUtil.X            parseutil.X, datetimeutil.X
//	DateTime.Now, the host's zone         ix.Clock, as the clock argument of datetimeutil.X
//	AngleSharp QuerySelector(All)          dom.Parse(...).QuerySelector(All)
//	IndexerException                       port.IndexerError
//	IndexerAuthException                   port.IndexerAuthError
//	RequestLimitReachedException           port.RequestLimitReached
//	TooManyRequestsException               port.TooManyRequests
//	ReleaseUnavailableException            port.ErrReleaseUnavailable
//	CookiesUpdater(null, null)             nothing: a logged-out response already brings a sign-in
//
// A line SlipStream changes on purpose carries a "// SlipStream:" comment
// that says why, so a sync keeps it.
package port

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
)

// ID is a port's id: the Prowlarr class name lowercased, or the catalog
// id Prowlarr's hidden list already names ("hdbits", "beyond-hd").
type ID string

// Port is one site, ready for the registry. Its only constructor is
// Define, so every Port passed Define's checks.
type Port struct {
	entry Entry
	run   runner
}

func (p Port) Entry() Entry { return p.entry }

// Entry is what the port's site is. Define derives Fields, Auth, Key and
// SignInWords, so a port never lists them by hand.
type Entry struct {
	ID   ID
	Name string
	// Description is SlipStream's own sentence, per WRITING.md.
	Description string
	// Language is a BCP 47 tag, "en-US" when empty.
	Language string
	Privacy  Privacy
	// IndexerURLs are the known addresses, the first the default. An
	// Admin may type another.
	IndexerURLs []string
	// LegacyURLs are addresses a job runs at IndexerURLs[0] instead, as
	// Prowlarr migrates them.
	LegacyURLs []string
	// Hosts are the hosts besides the address's that the Indexer owns,
	// derived from the address: MTeamTp's "api." + domain. Credentials
	// and port headers reach only the address's host and these.
	Hosts func(base *url.URL) []string
	// Encoding is a name Encoding.GetEncoding takes, such as
	// "windows-1251", or "" for UTF-8. Responses are decoded and form
	// bodies encoded with it.
	Encoding string
	// UserAgent is a fixed User-Agent the site requires, sent unless a
	// pasted session carries its own.
	UserAgent string
	// RateLimit is the floor between requests to the site's hosts. Zero
	// is the Node's 2 s floor. Define refuses one below 2 s.
	RateLimit time.Duration
	// PageSize stops a request sequence at a short page. Zero is one page.
	PageSize int
	// QueryLimit is a new Indexer's default search limit, zero for none.
	QueryLimit Limit
	GrabLimit  Limit
	// Capabilities are static: the Controller has them before any Test.
	Capabilities Capabilities
	// Replaces are catalog ids the picker hides. Aliases are other
	// catalog ids for the site. Both key the port's Tracker.
	Replaces []string
	Aliases  []string
	// Upstream is read from the port's upstream.json, never written here.
	Upstream Upstream

	// Fields are S's, then the family's own: Define sets them.
	Fields []Field
	// Auth is Impl.Login's family: Define sets it.
	Auth Auth
	// Key names the credentials account_refused names ("passkey"), from
	// the fields tagged key. SignInWords names the ones
	// credentials_rejected names ("email or password"), from those
	// tagged signin. Define sets both.
	Key, SignInWords string
}

// Upstream is the port's provenance, from its upstream.json.
type Upstream struct {
	Commit string    `json:"commit"`
	Date   time.Time `json:"-"`
	// Files maps each Go file of the port to the Prowlarr files it is
	// ported from.
	Files map[string][]string `json:"files"`
	// Family is "gazelle", "avistaz", "speedapp" or "".
	Family string `json:"family"`
}

// ReadUpstream parses a port's embedded upstream.json. A malformed one
// panics at init, so every test that loads the registry fails.
func ReadUpstream(raw []byte) Upstream {
	var u struct {
		Upstream
		Date string `json:"date"`
	}
	if err := json.Unmarshal(raw, &u); err != nil {
		panic(fmt.Sprintf("port: upstream.json: %v", err))
	}
	date, err := time.Parse(time.DateOnly, u.Date)
	if err != nil || u.Commit == "" || len(u.Files) == 0 {
		panic(fmt.Sprintf("port: upstream.json needs a commit, a date as 2006-01-02 and its files, got %s", raw))
	}
	u.Upstream.Date = date
	return u.Upstream
}

type Privacy string

const (
	Private     Privacy = "private"
	SemiPrivate Privacy = "semi-private"
	Public      Privacy = "public"
)

// Limit is Prowlarr's QueryLimit and LimitsUnit.
type Limit struct {
	Count int
	// Daily is false for per hour.
	Daily bool
}

func PerHour(n int) Limit { return Limit{Count: n} }

func PerDay(n int) Limit { return Limit{Count: n, Daily: true} }

// Auth is the auth family. It decides Session(): StatelessKey fails over,
// the rest wait for home unless the Indexer allows failover.
type Auth int

const (
	StatelessKey Auth = iota + 1
	FormCookie
	BearerToken
	PastedCookie
	FormWithCode
)

// HoldsSession is Session() of the indexer contract.
func (a Auth) HoldsSession() bool { return a != StatelessKey }

// Capabilities are IndexerCapabilities: the params each mode takes and
// the category mapping.
type Capabilities struct {
	// SearchParams are {Q} when nil and Impl.Basic is set.
	SearchParams      []Param
	TvSearchParams    []Param
	MovieSearchParams []Param
	Categories        *newznab.Categories
	// NoRss is SupportsRss => false: the site lists no new uploads.
	NoRss bool
}

// Param is a Torznab parameter name.
type Param string

const (
	Q        Param = "q"
	Season   Param = "season"
	Ep       Param = "ep"
	ImdbID   Param = "imdbid"
	TmdbID   Param = "tmdbid"
	TvdbID   Param = "tvdbid"
	TvMazeID Param = "tvmazeid"
	RID      Param = "rid"
	Genre    Param = "genre"
	Year     Param = "year"
)

// Mapping is one AddCategoryMapping.
type Mapping struct {
	tracker string
	cat     newznab.Category
	desc    string
}

// Map is AddCategoryMapping(tracker, cat, desc). tracker is an int or a
// string, as in C#.
func Map[T ~int | ~string](tracker T, cat newznab.Category, desc string) Mapping {
	return Mapping{tracker: fmt.Sprint(tracker), cat: cat, desc: desc}
}

// Categories builds the mapping in order, as SetCapabilities does.
func Categories(m ...Mapping) *newznab.Categories {
	c := &newznab.Categories{}
	for _, mapping := range m {
		c.Add(mapping.tracker, mapping.cat, mapping.desc)
	}
	return c
}

// Impl is the port's strategy, S its settings struct. A nil generator is
// a mode the site lacks; Define checks it agrees with Capabilities.
type Impl[S any] struct {
	// Login is the auth family: NoLogin{}, FormLogin[S], TokenLogin[S],
	// CookieLogin{} or CodeLogin[S].
	Login Login
	// Movie, TV and Basic are the request generator's overloads. Recent is
	// Basic with an empty term, so a port writes no Recent.
	Movie, TV, Basic func(ix *Indexer[S], q SearchCriteria) (Chain, error)
	// Parse is ParseResponse: Releases, or one of the errors below.
	Parse func(ix *Indexer[S], r *IndexerResponse) ([]TorrentInfo, error)
	// ModifyRequest adds the port's credentials to every request to the
	// Indexer's hosts: searches, page fetches and downloads.
	ModifyRequest func(ix *Indexer[S], r *Request)
	// Download replaces the default fetch of a link.
	Download func(ix *Indexer[S], link *url.URL) (Got, error)
	// FilterReleases is CleanupReleases' override hook, such as SpeedApp's
	// FilterReleasesByQuery.
	FilterReleases func(ix *Indexer[S], q SearchCriteria, rs []TorrentInfo) []TorrentInfo
	// Derive reads values the port keeps in Node memory, once per
	// Indexer revision per Node: GazelleGames' passkey from its API key,
	// MyAnonamouse's user class. They reach the port as ix.Derived.
	Derive func(ix *Indexer[S]) (map[string]string, error)
}

// Login is one of the five auth families. Its methods are unexported, so
// the five are the only ones.
type Login interface {
	auth() Auth
	// settings is the type the family's hooks read, nil for a family
	// with none.
	settings() reflect.Type
}

// NoLogin is the stateless family: an API key, a passkey or nothing,
// which ModifyRequest or the generator puts on each request.
type NoLogin struct{}

// FormLogin is a sign-in form whose cookies are the session.
type FormLogin[S any] struct {
	DoLogin func(in *SignIn[S]) error
	// CheckIfLoginNeeded reads a response as logged out. Nil is
	// Prowlarr's default, a 401.
	CheckIfLoginNeeded func(r *IndexerResponse) bool
}

// TokenLogin signs in to a bearer token, which the session keeps.
// ModifyRequest nil sends "Authorization: Bearer <token>".
type TokenLogin[S any] struct {
	DoLogin            func(in *SignIn[S]) (token string, err error)
	CheckIfLoginNeeded func(r *IndexerResponse) bool
}

// CookieLogin is a browser cookie the Admin pasted, with its browser's
// User-Agent when UserAgent. A logged-out response is cookie_expired.
// Rotates reports every cookie a response sets as the changed session.
type CookieLogin struct {
	UserAgent          bool
	Rotates            bool
	CheckIfLoginNeeded func(r *IndexerResponse) bool
}

// CodeLogin is a sign-in form with a two-factor code. Define adds the
// sealed authenticator key field, and in.TwoFactorCode gives the code.
type CodeLogin[S any] struct {
	DoLogin            func(in *SignIn[S]) error
	CheckIfLoginNeeded func(r *IndexerResponse) bool
}

func (NoLogin) auth() Auth       { return StatelessKey }
func (FormLogin[S]) auth() Auth  { return FormCookie }
func (TokenLogin[S]) auth() Auth { return BearerToken }
func (CookieLogin) auth() Auth   { return PastedCookie }
func (CodeLogin[S]) auth() Auth  { return FormWithCode }

func (NoLogin) settings() reflect.Type       { return nil }
func (FormLogin[S]) settings() reflect.Type  { return reflect.TypeFor[S]() }
func (TokenLogin[S]) settings() reflect.Type { return reflect.TypeFor[S]() }
func (CookieLogin) settings() reflect.Type   { return nil }
func (CodeLogin[S]) settings() reflect.Type  { return reflect.TypeFor[S]() }

// Retired is a port SlipStream deleted because its site died: the Node
// answers its jobs with port_retired and Reason.
type Retired struct {
	ID   ID
	Name string
	// Reason ends "SlipStream no longer supports <Name>, because ...":
	// "the site closed".
	Reason string
}

// Moved is a port whose site moved to the catalog: the release that
// deletes it migrates its Indexers to Definition, mapping each setting by
// Fields, port field to definition field.
type Moved struct {
	ID         ID
	Definition string
	Fields     map[string]string
}
