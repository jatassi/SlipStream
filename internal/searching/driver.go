package searching

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
)

// Kind is how an Indexer is reached: the protocol its driver speaks.
type Kind uint8

const (
	Catalog Kind = iota + 1
	Torznab
	RSS
)

// Driver is one kind's protocol on the Node. Every method runs inside one
// job. It reaches the network only through Site.Do and returns domain
// values, or a *Limited, *Failing or *Stopped outcome.
type Driver interface {
	Search(ctx context.Context, s *Site, q Query) ([]Release, error)
	Recent(ctx context.Context, s *Site) ([]Release, error)
	Fetch(ctx context.Context, s *Site, link Link) (Got, error)
	// Test signs in when the Indexer signs in, with answer when the
	// Admin answered a captcha, then reads its capabilities and its
	// newest items. Zero items pass.
	Test(ctx context.Context, s *Site, answer *CaptchaAnswer) (Report, error)
	// Captcha fetches the sign-in page's captcha.
	Captcha(ctx context.Context, s *Site) (Captcha, error)
}

// Job is one call for one Indexer, everything a Node needs for it, since a
// Worker keeps nothing durable (ADR 0003). Settings, APIKey and Session
// hold opened secrets for this job only.
type Job struct {
	// ID joins a repeated call to the run of the first.
	ID       string
	Indexer  string
	Revision int64
	Kind     Kind
	// Address is the Indexer's base address.
	Address string
	// Definition is a catalog Indexer's, nil for every other kind.
	Definition *Definition
	Settings   map[string]string
	APIKey     string
	Session    *Session
	Categories []int
	// RSS is how an RSS feed's items are read.
	RSS  RSSReading
	Call Call
}

// Definition is the Cardigann definition version a catalog Indexer runs.
type Definition struct {
	YAML []byte
	SHA  string
}

// Call is what a Job asks: a SearchCall, RecentCall, FetchCall, TestCall
// or CaptchaCall.
type Call interface{ call() }

type (
	SearchCall struct{ Query Query }
	RecentCall struct{}
	FetchCall  struct{ Link Link }
	// TestCall signs in with Captcha when the Admin answered one.
	TestCall    struct{ Captcha *CaptchaAnswer }
	CaptchaCall struct{}
)

func (SearchCall) call()  {}
func (RecentCall) call()  {}
func (FetchCall) call()   {}
func (TestCall) call()    {}
func (CaptchaCall) call() {}

// Captcha is a sign-in page's captcha image and the state a sign-in with
// its answer resumes from.
type Captcha struct {
	// Image is empty when the sign-in page shows no captcha this time.
	Image       []byte
	ContentType string
	// PreLogin is opaque: the sign-in page's cookies, its form's action and
	// its inputs.
	PreLogin []byte
}

// CaptchaAnswer is an Admin's answer to a Captcha.
type CaptchaAnswer struct {
	PreLogin []byte
	Text     string
}

// Query is one search as the Controller built it from a Work's terms and
// the Indexer's capabilities.
type Query struct {
	// Mode is "search", "tvsearch" or "movie".
	Mode string
	// IDs are the mode's id parameters the Work has.
	IDs map[string]string
	// Text is empty when an id is sent.
	Text    string
	Season  int
	Episode string
}

// Answer is a job's one outcome and what it produced.
type Answer struct {
	// Started is when the job's first request left the Node's spacer, zero
	// when it sent none.
	Started time.Time
	// Outcome is nil when the site responded, else a *Limited, *Failing or
	// *Stopped.
	Outcome error
	// Session is set when the job changed the Indexer's session on the
	// Node.
	Session  *Session
	Releases []Release
	Got      Got
	Report   *Report
	Captcha  *Captcha
}

// Release is one listing, normalized. The Node filters nothing beyond the
// categories it asked for.
type Release struct {
	Title     string
	Size      int64
	Published time.Time
	Seeders   int
	Leechers  int
	Grabs     int
	// Link is the .torrent link, zero for a magnet-only Release.
	Link     Link
	Magnet   Link
	InfoHash *bencode.InfoHash
	// Categories are Newznab category ids.
	Categories []int
	// IDs are imdbid, tmdbid and tvdbid as the site sent them.
	IDs     map[string]string
	Factors Factors
	Floor   Floor
	// Details is the Release's page on the site.
	Details string
	// Source is the site a Torznab feed says the Release came from.
	Source *Source
}

// Factors are the share of a download and an upload the site counts:
// freeleech is Download 0.
type Factors struct{ Download, Upload float64 }

// Floor is the seeding the site asks of the Release.
type Floor struct {
	MinRatio float64
	MinSeed  time.Duration
}

type Source struct {
	ID      string
	Name    string
	Privacy string
	// Origin is "prowlarr" or "jackett".
	Origin string
}

// Got is what a fetched link leads to: a Torrent or a Magnet.
type Got interface{ got() }

type (
	// Torrent is a .torrent's bytes as the site sent them.
	Torrent struct{ Raw []byte }
	// Magnet is the magnet link a download link redirected to.
	Magnet struct{ Link Link }
)

func (Torrent) got() {}
func (Magnet) got()  {}

// Report is what a Test learned beside its outcome.
type Report struct {
	Capabilities Capabilities
	Items        int
	// Sources are the different source names the items named.
	Sources []string
	// RSS is how an RSS feed's items are read, nil for every other kind
	// and for a feed SlipStream can't use, whose Refusal says why.
	RSS     *RSSReading
	Refusal RSSRefusal
}

// Capabilities are an Indexer's search modes, by their t= value, each with
// the parameters it takes, and its categories.
type Capabilities struct {
	Modes      map[string][]string
	Categories []int
}

// Session is one Indexer's cookies and User-Agent on one Node.
type Session struct {
	Cookies   []Cookie
	UserAgent string
	// Pasted is a session seeded from a browser cookie the Admin pasted.
	Pasted bool
	Earned time.Time
}

type Cookie struct {
	Name, Value, Domain, Path string
	Expires                   time.Time
	Secure, HostOnly          bool
}

func (k Kind) String() string {
	switch k {
	case Catalog:
		return "catalog"
	case Torznab:
		return "Torznab"
	case RSS:
		return "RSS"
	}
	return "unknown"
}
