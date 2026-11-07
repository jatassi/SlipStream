package port

import (
	"net/http"
	"net/url"

	"github.com/jatassi/slipstream/internal/newznab"
)

// Indexer is one job's view of one Indexer that runs the port: C#'s
// `this`, with the settings opened and decoded.
type Indexer[S any] struct {
	Settings S
	// BaseURL is the Indexer's address, IndexerURLs[0] for a legacy one,
	// with a trailing slash.
	BaseURL    *url.URL
	Categories *newznab.Categories
	// Derived is what Impl.Derive read for this revision on this Node.
	Derived map[string]string
}

// URL is BaseURL joined with path, as Settings.BaseUrl + "path".
func (ix *Indexer[S]) URL(_ string) *url.URL { panic("port: Indexer.URL is not implemented") }

// Fetch sends r through the same loop as a search request: the session,
// ModifyRequest, spacing, and one sign-in and retry when the response
// reads as logged out. Parsers and Download use it for extra pages.
func (ix *Indexer[S]) Fetch(_ *Request) (*IndexerResponse, error) {
	panic("port: Indexer.Fetch is not implemented")
}

// Token is the bearer token a TokenLogin holds, "" for none.
func (ix *Indexer[S]) Token() string { panic("port: Indexer.Token is not implemented") }

// SignIn is a DoLogin's handle: its requests carry only the cookies the
// sign-in itself earns, and the pasted ones.
type SignIn[S any] struct {
	Settings S
	BaseURL  *url.URL
}

// Do sends one sign-in request. It never signs in again and reads no
// logged-out page.
func (in *SignIn[S]) Do(_ *Request) (*IndexerResponse, error) {
	panic("port: SignIn.Do is not implemented")
}

// UseCookie makes the cookie header the session in place of a form
// sign-in: GreatPosterWall's optional cookie for a two-factor account.
func (in *SignIn[S]) UseCookie(_ string) error {
	panic("port: SignIn.UseCookie is not implemented")
}

// TwoFactorCode is the code for this sign-in: the RFC 6238 code of the
// authenticator key, else the code the Admin typed for this Test. ok is
// false with neither, and a DoLogin that needs one then returns
// ErrTwoFactorCodeNeeded.
func (in *SignIn[S]) TwoFactorCode() (code string, ok bool) {
	panic("port: SignIn.TwoFactorCode is not implemented")
}

// SearchCriteria is SearchCriteriaBase with the Movie and Tv criteria
// folded in. The framework already dropped every id the mode's params
// don't declare, and filled Limit and Offset from PageSize.
type SearchCriteria struct {
	SearchTerm string
	// ImdbID is "tt0076759" or "".
	ImdbID     string
	TmdbID     int
	TvdbID     int
	TvMazeID   int
	Season     int
	Episode    string
	Categories []newznab.ID
	Limit      int
	Offset     int
}

func (q SearchCriteria) SanitizedSearchTerm() string {
	panic("port: SanitizedSearchTerm is not implemented")
}

func (q SearchCriteria) SanitizedTvSearchString() string {
	panic("port: SanitizedTvSearchString is not implemented")
}

func (q SearchCriteria) EpisodeSearchString() string {
	panic("port: EpisodeSearchString is not implemented")
}

func (q SearchCriteria) FullImdbID() string { panic("port: FullImdbID is not implemented") }

func (q SearchCriteria) ImdbNumber() int { panic("port: ImdbNumber is not implemented") }

func (q SearchCriteria) IsRssSearch() bool { panic("port: IsRssSearch is not implemented") }

// Chain is IndexerPageableRequestChain: tiers of request sequences. A
// tier runs only when every tier before it found nothing, and a sequence
// stops at a page shorter than PageSize.
type Chain struct{}

func (c *Chain) Add(_ ...*Request) { panic("port: Chain.Add is not implemented") }

func (c *Chain) AddTier() { panic("port: Chain.AddTier is not implemented") }

// One is a chain of one request, the common case.
func One(_ *Request) (Chain, error) { panic("port: One is not implemented") }

// Request is HttpRequestBuilder's result. Every header a port sets is
// treated as a credential: it reaches only the Indexer's hosts and is
// dropped on a redirect anywhere else.
type Request struct {
	Method string
	URL    *url.URL
	Header http.Header
}

func Get(_ *url.URL) *Request { panic("port: Get is not implemented") }

func Post(_ *url.URL) *Request { panic("port: Post is not implemented") }

func (r *Request) AddQueryParam(_, _ string) *Request {
	panic("port: Request.AddQueryParam is not implemented")
}

func (r *Request) AddFormParameter(_, _ string) *Request {
	panic("port: Request.AddFormParameter is not implemented")
}

func (r *Request) SetHeader(_, _ string) *Request {
	panic("port: Request.SetHeader is not implemented")
}

func (r *Request) SetJSON(_ any) *Request { panic("port: Request.SetJSON is not implemented") }

// NoRedirect is AllowAutoRedirect = false: the response is the redirect.
func (r *Request) NoRedirect() *Request { panic("port: Request.NoRedirect is not implemented") }

func (r *Request) Accept(_ string) *Request { panic("port: Request.Accept is not implemented") }

// URI is NzbDrone.Common's HttpUri, for the links a parser builds.
type URI struct{}

func NewURI(_ *url.URL) URI { panic("port: NewURI is not implemented") }

func (h URI) CombinePath(_ string) URI { panic("port: URI.CombinePath is not implemented") }

func (h URI) AddQueryParam(_, _ string) URI { panic("port: URI.AddQueryParam is not implemented") }

func (h URI) RemoveQueryParam(_ string) URI {
	panic("port: URI.RemoveQueryParam is not implemented")
}

func (h URI) FullURI() string { panic("port: URI.FullURI is not implemented") }

// IndexerResponse is what a request got, its body decoded with
// Entry.Encoding.
type IndexerResponse struct {
	StatusCode      int
	Headers         http.Header
	Content         string
	Body            []byte
	URL             *url.URL
	Request         *Request
	HasHTTPRedirect bool
	RedirectURL     string
}

// Cookies is GetCookies: the cookies the response set, by name.
func (r *IndexerResponse) Cookies() map[string]string {
	panic("port: IndexerResponse.Cookies is not implemented")
}
