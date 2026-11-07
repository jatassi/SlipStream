package port

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/searching"
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
	j       *job[S]
}

// URL is BaseURL joined with path, as Settings.BaseUrl + "path". A query
// in path stays the query.
func (ix *Indexer[S]) URL(path string) *url.URL {
	p, query, _ := strings.Cut(strings.TrimLeft(path, "/"), "?")
	return ix.BaseURL.ResolveReference(&url.URL{Path: p, RawQuery: query})
}

// Fetch sends r through the same loop as a search request: the session,
// ModifyRequest, spacing, and one sign-in and retry when the response
// reads as logged out. Parsers and Download use it for extra pages.
func (ix *Indexer[S]) Fetch(r *Request) (*IndexerResponse, error) { return ix.j.send(r) }

// Token is the bearer token a TokenLogin holds, "" for none.
func (ix *Indexer[S]) Token() string {
	if held := ix.j.site.Held(); held != nil {
		return held.Token
	}
	return ""
}

// SignIn is a DoLogin's handle: its requests carry only the cookies the
// sign-in itself earns, and the pasted ones.
type SignIn[S any] struct {
	Settings S
	BaseURL  *url.URL
	j        *job[S]
	in       *searching.SigningIn
}

// Do sends one sign-in request. It never signs in again and reads no
// logged-out page.
func (in *SignIn[S]) Do(r *Request) (*IndexerResponse, error) {
	req, o, err := in.j.request(r, false)
	if err != nil {
		return nil, err
	}
	res, err := in.in.Do(in.j.ctx, req, o)
	if err != nil {
		return nil, err
	}
	return in.j.response(r, res), nil
}

// UseCookie makes the cookie header the session in place of a form
// sign-in: GreatPosterWall's optional cookie for a two-factor account.
func (in *SignIn[S]) UseCookie(header string) error {
	if err := in.in.UseCookie(header, in.BaseURL.Host); err != nil {
		return &searching.Failing{Err: err}
	}
	return nil
}

// TwoFactorCode is the code for this sign-in: the RFC 6238 code of the
// authenticator key, else the code the Admin typed for this Test. ok is
// false with neither, and a DoLogin that needs one then returns
// ErrTwoFactorCodeNeeded.
func (in *SignIn[S]) TwoFactorCode() (code string, ok bool) {
	if key := in.j.site.Settings[authenticatorKey.Name]; key != "" {
		if code, err := totp(key, time.Now()); err == nil {
			return code, true
		}
	}
	return in.j.site.Code, in.j.site.Code != ""
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

var (
	dashes       = regexp.MustCompile(`\p{Pd}+`)
	singleQuotes = regexp.MustCompile("[`´‘’]")
	imdbID       = regexp.MustCompile(`^(?:tt)?(\d{1,8})$`)
)

func (q SearchCriteria) SanitizedSearchTerm() string {
	term := singleQuotes.ReplaceAllString(dashes.ReplaceAllString(q.SearchTerm, "-"), "'")
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) || strings.ContainsRune("-._()@/'[]+%", r) {
			return r
		}
		return -1
	}, term)
}

func (q SearchCriteria) SanitizedTvSearchString() string {
	return strings.TrimSpace(q.SanitizedSearchTerm() + " " + q.EpisodeSearchString())
}

// EpisodeSearchString is "S01E02", "S01" for a whole season, or
// "2026.06.01" for a daily episode, whose Season is the year and Episode
// "06/01".
func (q SearchCriteria) EpisodeSearchString() string {
	if q.Season == 0 {
		return ""
	}
	if day, err := time.Parse("2006 01/02", fmt.Sprintf("%d %s", q.Season, q.Episode)); err == nil {
		return day.Format("2006.01.02")
	}
	if strings.TrimSpace(q.Episode) == "" {
		return fmt.Sprintf("S%02d", q.Season)
	}
	if ep, ok := coerceInt(q.Episode); ok {
		return fmt.Sprintf("S%02dE%02d", q.Season, ep)
	}
	return fmt.Sprintf("S%02dE%s", q.Season, q.Episode)
}

// coerceInt is ParseUtil.CoerceInt: the digits of s, read with "," and
// "." as thousands separators, 0 for none. ok is false where C# throws.
func coerceInt(s string) (int, bool) {
	var digits strings.Builder
	commas, dots := false, false
	for _, r := range s {
		switch {
		case unicode.IsDigit(r):
			digits.WriteRune(r)
		case r == ',':
			commas = true
		case r == '.':
			dots = true
		}
	}
	if commas && dots {
		return 0, false
	}
	n, err := strconv.Atoi(cmp.Or(digits.String(), "0"))
	return n, err == nil
}

// FullImdbID is the IMDb id as "tt" and at least seven digits, "" for
// none.
func (q SearchCriteria) FullImdbID() string {
	if n := q.ImdbNumber(); n != 0 {
		return fmt.Sprintf("tt%07d", n)
	}
	return ""
}

func (q SearchCriteria) ImdbNumber() int {
	m := imdbID.FindStringSubmatch(q.ImdbID)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func (q SearchCriteria) IsRssSearch() bool {
	return strings.TrimSpace(q.SearchTerm) == "" && !q.IsIDSearch()
}

func (q SearchCriteria) IsIDSearch() bool {
	return q.ImdbID != "" || q.TmdbID != 0 || q.TvdbID != 0 || q.TvMazeID != 0
}

// Chain is IndexerPageableRequestChain: tiers of request sequences. A
// tier runs only when every tier before it found nothing, and a sequence
// stops at a page shorter than PageSize.
type Chain struct{ tiers [][][]*Request }

func (c *Chain) Add(seq ...*Request) {
	if len(seq) == 0 {
		return
	}
	if len(c.tiers) == 0 {
		c.tiers = [][][]*Request{nil}
	}
	last := len(c.tiers) - 1
	c.tiers[last] = append(c.tiers[last], seq)
}

func (c *Chain) AddTier() {
	if len(c.tiers) == 0 || len(c.tiers[len(c.tiers)-1]) == 0 {
		return
	}
	c.tiers = append(c.tiers, nil)
}

// One is a chain of one request, the common case.
func One(r *Request) (Chain, error) {
	var c Chain
	c.Add(r)
	return c, nil
}

// Request is HttpRequestBuilder's result. Every header a port sets is
// treated as a credential: it reaches only the Indexer's hosts.
type Request struct {
	Method   string
	URL      *url.URL
	Header   http.Header
	form     [][2]string
	json     []byte
	err      error
	noFollow bool
}

func Get(u *url.URL) *Request { return newRequest(http.MethodGet, u) }

func Post(u *url.URL) *Request { return newRequest(http.MethodPost, u) }

func newRequest(method string, u *url.URL) *Request {
	c := *u
	return &Request{Method: method, URL: &c, Header: http.Header{}}
}

// AddQueryParam appends k=v to the query, escaped as Uri.EscapeDataString
// escapes.
func (r *Request) AddQueryParam(k, v string) *Request {
	pair := escapeData(k) + "=" + escapeData(v)
	if r.URL.RawQuery == "" {
		r.URL.RawQuery = pair
	} else {
		r.URL.RawQuery += "&" + pair
	}
	return r
}

// AddFormParameter adds k=v to the form body, encoded in Entry.Encoding.
func (r *Request) AddFormParameter(k, v string) *Request {
	r.form = append(r.form, [2]string{k, v})
	return r
}

func (r *Request) SetHeader(k, v string) *Request {
	r.Header.Set(k, v)
	return r
}

func (r *Request) SetJSON(v any) *Request {
	r.json, r.err = json.Marshal(v)
	return r
}

// NoRedirect is AllowAutoRedirect = false: the response is the redirect.
func (r *Request) NoRedirect() *Request {
	r.noFollow = true
	return r
}

func (r *Request) Accept(mime string) *Request { return r.SetHeader("Accept", mime) }

func (r *Request) clone() *Request {
	c := *r
	u := *r.URL
	c.URL, c.Header = &u, r.Header.Clone()
	return &c
}

// escapeData is Uri.EscapeDataString: every byte but the unreserved ones
// percent-encoded, a space as %20.
func escapeData(s string) string { return strings.ReplaceAll(url.QueryEscape(s), "+", "%20") }

// URI is NzbDrone.Common's HttpUri, for the links a parser builds.
type URI struct{ u url.URL }

func NewURI(base *url.URL) URI { return URI{u: *base} }

// CombinePath is the path with p after it, one slash between.
func (h URI) CombinePath(p string) URI {
	switch {
	case strings.TrimSpace(p) == "":
	case strings.TrimSpace(h.u.Path) == "":
		h.u.Path = p
	default:
		h.u.Path = strings.TrimRight(h.u.Path, "/") + "/" + strings.TrimLeft(p, "/")
	}
	h.u.RawPath = ""
	return h
}

func (h URI) AddQueryParam(k, v string) URI {
	pair := escapeData(k) + "=" + escapeData(v)
	if h.u.RawQuery == "" {
		h.u.RawQuery = pair
	} else {
		h.u.RawQuery += "&" + pair
	}
	return h
}

func (h URI) RemoveQueryParam(k string) URI {
	var kept []string
	for pair := range strings.SplitSeq(h.u.RawQuery, "&") {
		name, _, _ := strings.Cut(pair, "=")
		if n, err := url.QueryUnescape(name); pair != "" && (err != nil || n != k) {
			kept = append(kept, pair)
		}
	}
	h.u.RawQuery = strings.Join(kept, "&")
	return h
}

func (h URI) FullURI() string { return h.u.String() }

// IndexerResponse is what a request got, its body decoded with
// Entry.Encoding.
type IndexerResponse struct {
	StatusCode int
	Headers    http.Header
	Content    string
	Body       []byte
	// URL is where the response came from, after any redirect.
	URL     *url.URL
	Request *Request
	// HasHTTPRedirect is a redirect the request didn't follow, and
	// RedirectURL where it points.
	HasHTTPRedirect bool
	RedirectURL     string
}

// Cookies is GetCookies: the cookies the response set, by name.
func (r *IndexerResponse) Cookies() map[string]string {
	out := map[string]string{}
	for _, c := range (&http.Response{Header: r.Headers}).Cookies() {
		out[c.Name] = c.Value
	}
	return out
}
