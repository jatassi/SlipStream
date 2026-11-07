package searching

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/searching/internal/torznab"
)

// CatalogCapabilities are a definition's capabilities, its modes by their
// Torznab t= value.
func CatalogCapabilities(c cardigann.Caps) Capabilities {
	out := Capabilities{Modes: map[string][]string{}}
	for mode, params := range c.Modes {
		if t, ok := torznab.Mode(mode); ok {
			out.Modes[t] = params
		}
	}
	for _, id := range c.Categories {
		out.Categories = append(out.Categories, int(id))
	}
	return out
}

// catalogDriver runs a catalog Indexer's Cardigann definition, sending
// each request its exchanges ask for through the Site.
type catalogDriver struct {
	// relogins is how many times a job signs in again after a response
	// reads as signed out.
	relogins int
}

// definitions are the definitions this Node compiled, by sha.
type definitions struct {
	mu    sync.Mutex
	bySHA map[string]*cardigann.Definition
}

// keptDefinitions is how many compiled definitions a Node keeps.
const keptDefinitions = 64

// load is d compiled, from the cache when its sha was compiled before.
func (ds *definitions) load(d Definition) (*cardigann.Definition, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if def, ok := ds.bySHA[d.SHA]; ok {
		return def, nil
	}
	def, err := cardigann.Load(d.YAML)
	if err != nil {
		return nil, err
	}
	if len(ds.bySHA) >= keptDefinitions {
		clear(ds.bySHA)
	}
	ds.bySHA[d.SHA] = def
	return def, nil
}

func (catalogDriver) settings(s *Site) cardigann.Settings {
	return cardigann.Settings{Values: s.Settings, BaseURL: s.Address.String()}
}

func clock() cardigann.Clock { return cardigann.Clock{Now: time.Now(), Location: time.Local} }

func (d catalogDriver) Search(ctx context.Context, s *Site, q Query) ([]Release, error) {
	mode := torznab.Element(q.Mode)
	if _, ok := s.def.Summary().Caps.Modes[mode]; !ok {
		return nil, &Failing{Err: fmt.Errorf("the definition has no %q search", q.Mode)}
	}
	cq := cardigann.Query{Mode: mode, Text: q.Text, IDs: q.IDs, Season: q.Season, Episode: q.Episode}
	for _, c := range s.Categories {
		cq.Categories = append(cq.Categories, newznab.ID(c)) //nolint:gosec // G115: Newznab category ids have at most six digits
	}
	return d.releases(ctx, s, func() *cardigann.Search { return s.def.Search(d.settings(s), cq, clock()) })
}

func (d catalogDriver) Recent(ctx context.Context, s *Site) ([]Release, error) {
	return d.releases(ctx, s, func() *cardigann.Search { return s.def.Recent(d.settings(s), clock()) })
}

func (d catalogDriver) releases(ctx context.Context, s *Site, start func() *cardigann.Search) ([]Release, error) {
	var x *cardigann.Search
	if err := d.run(ctx, s, func() cardigann.Exchange { x = start(); return x }); err != nil {
		return nil, err
	}
	found, err := x.Result()
	if err != nil {
		return nil, &Failing{Err: err}
	}
	out := make([]Release, len(found))
	for i, r := range found {
		out[i] = releaseOf(r)
	}
	return out, nil
}

func (d catalogDriver) Fetch(ctx context.Context, s *Site, link Link) (Got, error) {
	u, err := url.Parse(link.raw)
	if err != nil {
		return nil, &Failing{Err: errors.New("the download link isn't a URL")}
	}
	var x *cardigann.Download
	if err := d.run(ctx, s, func() cardigann.Exchange { x = s.def.Download(d.settings(s), u, clock()); return x }); err != nil {
		return nil, err
	}
	got, err := x.Result()
	if err != nil {
		return nil, &Failing{Err: err}
	}
	switch g := got.(type) {
	case cardigann.Torrent:
		return Torrent{Raw: g.Raw}, nil
	case cardigann.Magnet:
		return Magnet{Link: LinkOf(g.Link.String())}, nil
	}
	return nil, &Failing{Err: errors.New("the download led to nothing SlipStream can use")}
}

// Test signs in when the definition signs in, with answer when the Admin
// answered a captcha, then reads the newest items. A sign-in that shows a
// captcha runs only with an answer, or when this Node holds no session,
// so a Test without one reads with the session held.
func (d catalogDriver) Test(ctx context.Context, s *Site, answer *CaptchaAnswer) (Report, error) {
	if sum := s.def.Summary(); sum.Login != cardigann.NoLogin {
		if err := d.signIn(ctx, s, nil, answer != nil || !sum.Captcha, answer); err != nil {
			return Report{}, err
		}
	}
	items, err := d.Recent(ctx, s)
	if err != nil {
		return Report{}, err
	}
	return Report{Capabilities: CatalogCapabilities(s.def.Summary().Caps), Items: len(items)}, nil
}

func (d catalogDriver) Captcha(ctx context.Context, s *Site) (Captcha, error) {
	x := s.def.CaptchaFetch(d.settings(s), clock())
	if _, _, err := d.drive(ctx, s, x, d.signInJar(s), false); err != nil {
		return Captcha{}, err
	}
	c, err := x.Result()
	if err != nil {
		return Captcha{}, &Failing{Err: err}
	}
	return Captcha{Image: c.Image, ContentType: c.ContentType, PreLogin: c.PreLogin}, nil
}

// run runs the exchange start makes with the session's cookies. It signs
// in first when the Indexer holds a session and this Node has none, and
// again, up to relogins times, when a response reads as signed out, each
// time starting the exchange over.
func (d catalogDriver) run(ctx context.Context, s *Site, start func() cardigann.Exchange) error {
	session := s.def.Summary().Login == cardigann.SessionLogin
	if session && s.slot.held() == nil {
		if err := d.signIn(ctx, s, nil, false, nil); err != nil {
			return err
		}
	}
	for tries := 0; ; tries++ {
		seen := s.slot.held()
		j := newJar(nil)
		if seen != nil {
			j = newJar(seen.Cookies)
		}
		out, _, err := d.drive(ctx, s, start(), j, true)
		if cookies, changed := j.held(); changed && seen != nil {
			s.slot.keep(seen, cookies)
		}
		switch {
		case err != nil:
			return err
		case !out:
			return nil
		case tries == d.relogins:
			return &Failing{Err: errors.New("the site signed SlipStream out again right after it signed in")}
		}
		if err := d.signIn(ctx, s, seen, false, nil); err != nil {
			return err
		}
	}
}

// signIn signs in, resuming from answer's pre-login state when the Admin
// answered a captcha. A definition whose sign-in holds a session keeps
// the cookies it ends with as the Indexer's session on this Node. A
// sign-in the site refuses stops the Indexer with its cause.
func (d catalogDriver) signIn(ctx context.Context, s *Site, seen *Session, must bool, answer *CaptchaAnswer) error {
	return s.signIn(ctx, seen, must, func(ctx context.Context) (*Session, error) {
		var c *cardigann.Captcha
		if answer != nil {
			c = &cardigann.Captcha{PreLogin: answer.PreLogin, Answer: answer.Text}
		}
		x := s.def.Login(d.settings(s), c, clock())
		_, j, err := d.drive(ctx, s, x, d.signInJar(s), false)
		if err != nil {
			return nil, err
		}
		if err := x.Result(); err != nil {
			return nil, refused(s.def, err)
		}
		if s.def.Summary().Login != cardigann.SessionLogin {
			return nil, nil
		}
		cookies, _ := j.held()
		out := &Session{Cookies: cookies, Earned: time.Now()}
		if held := s.slot.held(); held != nil {
			out.UserAgent, out.Pasted = held.UserAgent, held.Pasted
		}
		return out, nil
	})
}

// signInJar is a sign-in's jar: the pasted cookies, which every sign-in
// request sends beside its own.
func (catalogDriver) signInJar(s *Site) *jar {
	if held := s.slot.held(); held != nil && held.Pasted {
		return newJar(held.Cookies)
	}
	return newJar(nil)
}

// drive sends each request x asks for with j's cookies and feeds it the
// response. A sign-in request starts from a jar of j's cookies and its
// own, and the jar it ends with carries on to the next request. drive
// returns the last jar. With signedOut it stops at a response that reads
// as signed out, before x reads it, and reports it. An error Feed returns
// ends x, so Next stops, and x's Result returns it.
func (catalogDriver) drive(ctx context.Context, s *Site, x cardigann.Exchange, j *jar, signedOut bool) (bool, *jar, error) {
	base, cur := j, j
	for {
		req, ok := x.Next()
		if !ok {
			return false, cur, nil
		}
		if req.ReplaceCookies {
			cookies, _ := base.held()
			cur = newJar(cookies)
			if err := cur.add(req.Header.Get("Cookie"), req.URL.Hostname()); err != nil {
				return false, cur, &Failing{Err: err}
			}
		}
		res, err := s.exchange(ctx, req, cur)
		if err != nil {
			return false, cur, err
		}
		if signedOut && readsSignedOut(s.def, req, res) {
			return true, cur, nil
		}
		_ = x.Feed(res)
	}
}

// readsSignedOut reports whether res reads as the site having signed
// SlipStream out. A redirect to a magnet link never does. Nor does a page
// a download reads, since its selectors try links that may answer any
// page in place of a .torrent.
func readsSignedOut(def *cardigann.Definition, req cardigann.Request, res cardigann.Response) bool {
	switch {
	case strings.HasPrefix(strings.ToLower(res.Header.Get("Location")), "magnet:"):
		return false
	case req.Purpose == cardigann.ForDownload && res.Status == http.StatusOK:
		return false
	}
	return def.LoggedOut(res)
}

// exchange sends a Cardigann request with j's cookies and hands back its
// response with every Set-Cookie of the redirects it followed.
func (s *Site) exchange(ctx context.Context, r cardigann.Request, j *jar) (cardigann.Response, error) {
	req, err := http.NewRequestWithContext(ctx, r.Method, r.URL.String(), bytes.NewReader(r.Body))
	if err != nil {
		return cardigann.Response{}, &Failing{Err: fmt.Errorf("build the request: %w", withoutURL(err))}
	}
	req.Header = r.Header.Clone()
	if req.Header == nil {
		req.Header = http.Header{}
	}
	req.Header.Del("Cookie")
	p := Read
	if r.Purpose == cardigann.ForDownload {
		p = Download
	}
	before := len(j.taken())
	res, err := s.send(ctx, req, p, j, r.FollowRedirects)
	if err != nil {
		return cardigann.Response{}, err
	}
	header := res.Header.Clone()
	header.Del("Set-Cookie")
	for _, c := range j.taken()[before:] {
		header.Add("Set-Cookie", c)
	}
	return cardigann.Response{Status: res.Status, Header: header, Body: res.Body, URL: res.URL}, nil
}

// refused is the outcome of a sign-in that ended with err: a site that
// refused the credentials, the pasted cookie or an automatic sign-in at a
// captcha stops the Indexer.
func refused(def *cardigann.Definition, err error) error {
	if r, ok := errors.AsType[*cardigann.Rejected](err); ok {
		return &Stopped{Cause: CredentialsRejected, Text: r.Message}
	}
	switch {
	case errors.Is(err, cardigann.ErrCaptchaNeeded):
		return &Stopped{Cause: CaptchaNeeded}
	case errors.Is(err, cardigann.ErrSignInRefused) && def.Summary().Uses.Has(cardigann.LoginCookie):
		return &Stopped{Cause: CookieExpired}
	case errors.Is(err, cardigann.ErrSignInRefused):
		return &Stopped{Cause: CredentialsRejected}
	}
	return &Failing{Err: err}
}

func releaseOf(r cardigann.Release) Release {
	out := Release{
		Title: r.Title, Size: r.Size, Published: r.Published, Seeders: r.Seeders, Leechers: r.Leechers, Grabs: r.Grabs,
		Link: LinkOf(r.Link), Magnet: LinkOf(r.Magnet), IDs: r.IDs, Details: r.Details,
		Factors: Factors{Download: r.DownloadFactor, Upload: r.UploadFactor}, Floor: Floor{MinRatio: r.MinRatio, MinSeed: r.MinSeed},
	}
	if len(r.InfoHash) == len(bencode.InfoHash{}) {
		h := bencode.InfoHash(r.InfoHash)
		out.InfoHash = &h
	}
	for _, c := range r.Categories {
		out.Categories = append(out.Categories, int(c))
	}
	if len(out.IDs) == 0 {
		out.IDs = nil
	}
	return out
}
