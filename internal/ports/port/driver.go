// Ported from Prowlarr's src/NzbDrone.Core/Indexers/HttpIndexerBase.cs, src/NzbDrone.Core/Indexers/IndexerBase.cs, src/NzbDrone.Core/Indexers/TorrentIndexerBase.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package port

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
	"github.com/jatassi/slipstream/internal/searching"
)

// driver is searching.Driver for kind native. It dispatches by site.Port.
type driver struct{ r Registry }

// runner is a Port's Impl with S erased.
type runner interface {
	search(ctx context.Context, s *searching.Site, e Entry, q searching.Query) ([]searching.Release, error)
	recent(ctx context.Context, s *searching.Site, e Entry) ([]searching.Release, error)
	fetch(ctx context.Context, s *searching.Site, e Entry, link searching.Link) (searching.Got, error)
	test(ctx context.Context, s *searching.Site, e Entry) (searching.Report, error)
}

// typed is Impl[S] behind runner, with S's settings and its Login's part
// in the request loop.
type typed[S any] struct {
	impl     Impl[S]
	settings []setting
	login    family[S]
	charset  encoding.Charset
}

// family is what the request loop does for one Login family.
type family[S any] struct {
	auth Auth
	// signIn is DoLogin, with the token a TokenLogin earns. It is nil for
	// a family that never signs in.
	signIn func(in *SignIn[S]) (token string, err error)
	// loggedOut is CheckIfLoginNeeded, nil for NoLogin.
	loggedOut func(r *IndexerResponse) bool
	rotates   bool
}

func familyOf[S any](l Login) family[S] {
	switch l := l.(type) {
	case FormLogin[S]:
		return family[S]{auth: FormCookie, signIn: noToken(l.DoLogin), loggedOut: orUnauthorized(l.CheckIfLoginNeeded)}
	case TokenLogin[S]:
		return family[S]{auth: BearerToken, signIn: l.DoLogin, loggedOut: orUnauthorized(l.CheckIfLoginNeeded)}
	case CookieLogin:
		return family[S]{auth: PastedCookie, loggedOut: orUnauthorized(l.CheckIfLoginNeeded), rotates: l.Rotates}
	case CodeLogin[S]:
		return family[S]{auth: FormWithCode, signIn: noToken(l.DoLogin), loggedOut: orUnauthorized(l.CheckIfLoginNeeded)}
	}
	return family[S]{auth: StatelessKey}
}

func noToken[S any](doLogin func(*SignIn[S]) error) func(*SignIn[S]) (string, error) {
	return func(in *SignIn[S]) (string, error) { return "", doLogin(in) }
}

// orUnauthorized is check, or Prowlarr's default: a 401 is logged out.
func orUnauthorized(check func(*IndexerResponse) bool) func(*IndexerResponse) bool {
	if check != nil {
		return check
	}
	return func(r *IndexerResponse) bool { return r.StatusCode == http.StatusUnauthorized }
}

func (f family[S]) needsSignIn(held *searching.Session) bool {
	return f.signIn != nil && (held == nil || f.auth == BearerToken && held.Token == "")
}

// port is the job's port, or the outcome a retired or unknown id
// answers. A retired id answers port_retired with the record's reason,
// so the Controller writes the stop as for any cause.
func (d driver) port(s *searching.Site) (Port, error) {
	if p, ok := d.r.byID[ID(s.Port)]; ok {
		s.Space(p.entry.RateLimit)
		return p, nil
	}
	if x, ok := d.r.retired[ID(s.Port)]; ok {
		return Port{}, &searching.Stopped{Cause: searching.PortRetired, Text: x.Reason}
	}
	return Port{}, &searching.Failing{Err: fmt.Errorf("this Node carries no port %q", s.Port)}
}

func (d driver) Search(ctx context.Context, s *searching.Site, q searching.Query) ([]searching.Release, error) {
	p, err := d.port(s)
	if err != nil {
		return nil, err
	}
	return contained(func() ([]searching.Release, error) { return p.run.search(ctx, s, p.entry, q) })
}

func (d driver) Recent(ctx context.Context, s *searching.Site) ([]searching.Release, error) {
	p, err := d.port(s)
	if err != nil {
		return nil, err
	}
	return contained(func() ([]searching.Release, error) { return p.run.recent(ctx, s, p.entry) })
}

func (d driver) Fetch(ctx context.Context, s *searching.Site, link searching.Link) (searching.Got, error) {
	p, err := d.port(s)
	if err != nil {
		return nil, err
	}
	return contained(func() (searching.Got, error) { return p.run.fetch(ctx, s, p.entry, link) })
}

// Test signs in when the port holds a session, then runs Recent, or a
// Basic search for "test" when the site lists no new uploads. Any
// well-formed answer passes, zero items included. A CodeLogin reads the
// code the Admin typed from s.Code.
func (d driver) Test(ctx context.Context, s *searching.Site, _ *searching.CaptchaAnswer) (searching.Report, error) {
	p, err := d.port(s)
	if err != nil {
		return searching.Report{}, err
	}
	return contained(func() (searching.Report, error) { return p.run.test(ctx, s, p.entry) })
}

// contained is run's result, or failing when the port's code panics,
// such as on a selector that doesn't compile, so one port's bug never
// stops the Node.
func contained[T any](run func() (T, error)) (out T, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = &searching.Failing{Err: fmt.Errorf("the port's code failed: %v", p)}
		}
	}()
	return run()
}

// Captcha refuses: no port has a captcha step.
func (driver) Captcha(context.Context, *searching.Site) (searching.Captcha, error) {
	return searching.Captcha{}, &searching.Failing{Err: errors.New("a ported site has no captcha step")}
}

type generator[S any] = func(ix *Indexer[S], q SearchCriteria) (Chain, error)

// search is HttpIndexerBase.Fetch for one Query.
func (t *typed[S]) search(ctx context.Context, s *searching.Site, e Entry, q searching.Query) ([]searching.Release, error) {
	gen, crit, ok := t.criteria(e, q, s.Categories)
	if !ok {
		return nil, nil
	}
	return t.releases(ctx, s, e, gen, crit)
}

func (t *typed[S]) recent(ctx context.Context, s *searching.Site, e Entry) ([]searching.Release, error) {
	if e.Capabilities.NoRss {
		return nil, nil
	}
	return t.releases(ctx, s, e, t.impl.Basic, SearchCriteria{Categories: categories(s.Categories), Limit: e.PageSize})
}

func (t *typed[S]) releases(ctx context.Context, s *searching.Site, e Entry, gen generator[S], crit SearchCriteria) ([]searching.Release, error) {
	j, err := t.start(ctx, s, e)
	if err != nil {
		return nil, outcome(e.Auth, err)
	}
	infos, err := j.find(gen, crit)
	if err != nil {
		return nil, outcome(e.Auth, err)
	}
	return releasesOf(infos), nil
}

func (t *typed[S]) fetch(ctx context.Context, s *searching.Site, e Entry, link searching.Link) (searching.Got, error) {
	u, err := url.Parse(link.Reveal())
	if err != nil || u.Host == "" {
		return nil, &searching.Failing{Err: errors.New("the download link isn't a URL")}
	}
	j, err := t.start(ctx, s, e)
	if err != nil {
		return nil, outcome(e.Auth, err)
	}
	j.purpose = searching.Download
	got, err := j.download(u)
	if err != nil {
		return nil, outcome(e.Auth, err)
	}
	switch g := got.(type) {
	case Torrent:
		return searching.Torrent{Raw: g.Raw}, nil
	case Magnet:
		return searching.Magnet{Link: searching.LinkOf(g.Link)}, nil
	}
	return nil, &searching.Failing{Err: errors.New("the download led to nothing SlipStream can use")}
}

// test signs in, always for a sign-in family so a new password or a
// typed code is used. A CodeLogin with no authenticator key and no typed
// code reads with the session it holds, as a captcha site does.
func (t *typed[S]) test(ctx context.Context, s *searching.Site, e Entry) (searching.Report, error) {
	j, err := t.start(ctx, s, e)
	if err == nil && t.login.signIn != nil {
		must := t.login.auth != FormWithCode || s.Settings[authenticatorKey.Name] != "" || s.Code != ""
		err = j.signIn(nil, must)
	}
	if err != nil {
		return searching.Report{}, outcome(e.Auth, err)
	}
	crit := SearchCriteria{Categories: categories(s.Categories), Limit: e.PageSize}
	if e.Capabilities.NoRss {
		crit.SearchTerm = "test"
	}
	infos, err := j.find(t.impl.Basic, crit)
	if err != nil {
		return searching.Report{}, outcome(e.Auth, err)
	}
	return searching.Report{Capabilities: capabilitiesOf(e.Capabilities), Items: len(releasesOf(infos))}, nil
}

// criteria is the query as the mode's generator takes it. Each id the
// mode doesn't declare is dropped, and the text too when the mode takes
// none. ok is false for a mode the port lacks, or a query that has
// nothing left: an empty ok, never a failure.
func (t *typed[S]) criteria(e Entry, q searching.Query, cats []int) (generator[S], SearchCriteria, bool) {
	c := e.Capabilities
	var params []Param
	var gen generator[S]
	switch q.Mode {
	case "search":
		params, gen = c.SearchParams, t.impl.Basic
		if params == nil {
			params = []Param{Q}
		}
	case "tvsearch":
		params, gen = c.TvSearchParams, t.impl.TV
	case "movie":
		params, gen = c.MovieSearchParams, t.impl.Movie
	}
	crit := SearchCriteria{Season: seasonOf(q.Season), Episode: q.Episode, Categories: categories(cats), Limit: e.PageSize}
	if slices.Contains(params, Q) {
		crit.SearchTerm = q.Text
	}
	asked := q.Text != ""
	for key, v := range q.IDs {
		asked = true
		if !slices.Contains(params, Param(key)) {
			continue
		}
		n, _ := strconv.Atoi(v)
		switch Param(key) {
		case ImdbID:
			crit.ImdbID = v
		case TmdbID:
			crit.TmdbID = n
		case TvdbID:
			crit.TvdbID = n
		case TvMazeID:
			crit.TvMazeID = n
		case Q, Season, Ep, RID, Genre, Year:
		}
	}
	if gen == nil || asked && crit.IsRssSearch() {
		return nil, SearchCriteria{}, false
	}
	return gen, crit, true
}

func categories(ids []int) []newznab.ID {
	out := make([]newznab.ID, len(ids))
	for i, id := range ids {
		out[i] = newznab.ID(id) //nolint:gosec // G115: Newznab category ids have at most six digits
	}
	return out
}

// job is one job's run of a port: its Site, its Indexer and the hosts
// the Indexer's credentials may reach.
type job[S any] struct {
	ctx   context.Context
	site  *searching.Site
	e     Entry
	t     *typed[S]
	ix    *Indexer[S]
	hosts []string
	// purpose is Download for a fetch, so its body cap is a .torrent's.
	purpose searching.Purpose
}

// start is the job's Indexer, with what Impl.Derive read once per
// Indexer revision on this Node.
func (t *typed[S]) start(ctx context.Context, s *searching.Site, e Entry) (*job[S], error) {
	settings, err := decode[S](t.settings, s.Settings)
	if err != nil {
		return nil, &searching.Failing{Err: err}
	}
	base := baseURL(e, s.Address)
	j := &job[S]{ctx: ctx, site: s, e: e, t: t, hosts: []string{strings.ToLower(base.Host)}, purpose: searching.Read}
	if e.Hosts != nil {
		for _, h := range e.Hosts(base) {
			j.hosts = append(j.hosts, strings.ToLower(h))
		}
	}
	j.ix = &Indexer[S]{
		Settings: settings, BaseURL: base, Categories: e.Capabilities.Categories,
		Clock: datetimeutil.Clock{Now: time.Now(), Location: time.UTC}, j: j,
	}
	if t.impl.Derive != nil {
		if j.ix.Derived, err = s.Derived(ctx, func() (map[string]string, error) { return t.impl.Derive(j.ix) }); err != nil {
			return nil, err
		}
	}
	return j, nil
}

// baseURL is address, or IndexerURLs[0] for a legacy address, with a
// trailing slash.
func baseURL(e Entry, address *url.URL) *url.URL {
	u := *address
	for _, legacy := range e.LegacyURLs {
		if strings.TrimSuffix(legacy, "/") == strings.TrimSuffix(address.String(), "/") {
			home, _ := url.Parse(e.IndexerURLs[0])
			u = *home
		}
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
		u.RawPath = ""
	}
	return &u
}

// find runs the chain gen builds: each tier only when the tiers before it
// found nothing, each sequence until a page shorter than PageSize.
func (j *job[S]) find(gen generator[S], crit SearchCriteria) ([]TorrentInfo, error) {
	chain, err := gen(j.ix, crit)
	if err != nil {
		return nil, err
	}
	var infos []TorrentInfo
	for _, tier := range chain.tiers {
		for _, seq := range tier {
			for _, r := range seq {
				page, err := j.page(r)
				if err != nil {
					return nil, err
				}
				infos = append(infos, slices.DeleteFunc(slices.Clone(page), untitled)...)
				if j.e.PageSize == 0 || len(page) < j.e.PageSize {
					break
				}
			}
		}
		if len(infos) > 0 {
			break
		}
	}
	if f := j.t.impl.FilterReleases; f != nil {
		infos = f(j.ix, crit, infos)
	}
	return infos, nil
}

func untitled(i TorrentInfo) bool { return strings.TrimSpace(i.Title) == "" }

// page is one request of a search, parsed. A search the site redirects
// off the Indexer's hosts fails, since its response is another site's.
func (j *job[S]) page(r *Request) ([]TorrentInfo, error) {
	res, err := j.send(r)
	if err != nil {
		return nil, err
	}
	if res.HasHTTPRedirect && !r.noFollow {
		return nil, &searching.Failing{Err: errors.New("the site redirected the search off the Indexer's hosts")}
	}
	return j.t.impl.Parse(j.ix, res)
}

// download is Impl.Download, or by default a GET with the Indexer's
// credentials on its hosts. A redirect off them is fetched again from
// where it points, with no credentials.
func (j *job[S]) download(u *url.URL) (Got, error) {
	if j.t.impl.Download != nil {
		return j.t.impl.Download(j.ix, u)
	}
	res, err := j.send(Get(u))
	if err == nil && res.HasHTTPRedirect && !magnet.Is(res.RedirectURL) {
		var away *url.URL
		if away, err = url.Parse(res.RedirectURL); err == nil {
			res, err = j.send(Get(away))
		}
	}
	switch {
	case err != nil:
		return nil, err
	case res.HasHTTPRedirect && magnet.Is(res.RedirectURL):
		return Magnet{Link: res.RedirectURL}, nil
	case res.StatusCode == http.StatusNotFound:
		return nil, ErrReleaseUnavailable
	case res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices:
		return nil, &searching.Failing{Err: fmt.Errorf("the Indexer responded with status %d", res.StatusCode)}
	}
	return Torrent{Raw: res.Body}, nil
}

// send is FetchIndexerResponse for one request. A session family with no
// session on this Node signs in first. A response the family reads as
// logged out signs in once, one sign-in at a time per Indexer on this
// Node and none when another job signed in meanwhile, and goes again
// once. A pasted cookie never signs in: logged out is cookie_expired. A
// request to another host goes bare, so it reads no logged-out page.
func (j *job[S]) send(r *Request) (*IndexerResponse, error) {
	if !j.on(r.URL) {
		return j.do(r)
	}
	f := j.t.login
	if f.needsSignIn(j.site.Held()) {
		if err := j.signIn(nil, false); err != nil {
			return nil, err
		}
	}
	seen := j.site.Held()
	res, err := j.do(r)
	if err != nil || f.loggedOut == nil || !f.loggedOut(res) {
		return res, err
	}
	if f.signIn == nil {
		return nil, &searching.Stopped{Cause: searching.CookieExpired}
	}
	if err := j.signIn(seen, false); err != nil {
		return nil, err
	}
	if res, err = j.do(r); err != nil || !f.loggedOut(res) {
		return res, err
	}
	return nil, &searching.Failing{Err: errors.New("the site signed SlipStream out again right after it signed in")}
}

func (j *job[S]) do(r *Request) (*IndexerResponse, error) {
	req, o, err := j.request(r, true)
	if err != nil {
		return nil, err
	}
	res, err := j.site.Do(j.ctx, req, o)
	if err != nil {
		return nil, err
	}
	return j.response(r, res), nil
}

func (j *job[S]) on(u *url.URL) bool { return slices.Contains(j.hosts, strings.ToLower(u.Host)) }

// signIn runs the family's DoLogin under s.SignIn and holds the session
// it earned: the sign-in's cookies, and the token for a TokenLogin. must
// signs in even when this Node holds a session.
func (j *job[S]) signIn(seen *searching.Session, must bool) error {
	return j.site.SignIn(j.ctx, seen, must, func(_ context.Context, in *searching.SigningIn) (*searching.Session, error) {
		token, err := j.t.login.signIn(&SignIn[S]{Settings: j.ix.Settings, BaseURL: j.ix.BaseURL, j: j, in: in})
		if err != nil {
			return nil, err
		}
		if j.t.login.auth == BearerToken && token == "" {
			return nil, &searching.Failing{Err: errors.New("the site's sign-in response has no token")}
		}
		out := &searching.Session{Cookies: in.Cookies(), Token: token, Earned: time.Now()}
		if held := j.site.Held(); held != nil {
			out.UserAgent = held.UserAgent
		}
		return out, nil
	})
}

// request is r as the Site sends it. On the Indexer's hosts it carries
// every header the port set, and with credentials ModifyRequest's or the
// bearer token. Anywhere else it carries none of them, and its redirects
// go anywhere.
func (j *job[S]) request(r *Request, credentials bool) (*http.Request, searching.Send, error) {
	if r.err != nil {
		return nil, searching.Send{}, &searching.Failing{Err: fmt.Errorf("build the request's body: %w", r.err)}
	}
	on := j.on(r.URL)
	if on && credentials {
		r = r.clone()
		switch {
		case j.t.impl.ModifyRequest != nil:
			j.t.impl.ModifyRequest(j.ix, r)
		case j.t.login.auth == BearerToken:
			r.Header.Set("Authorization", "Bearer "+j.ix.Token())
		}
	}
	body, contentType := j.body(r)
	req, err := http.NewRequestWithContext(j.ctx, r.Method, r.URL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, searching.Send{}, &searching.Failing{Err: fmt.Errorf("build the request: %w", searching.WithoutURL(err))}
	}
	o := searching.Send{Purpose: j.purpose, NoFollow: r.noFollow, Rotates: j.t.login.rotates}
	if on {
		req.Header = r.Header.Clone()
		o.Hosts = j.hosts
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if j.e.UserAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", j.e.UserAgent)
	}
	return req, o, nil
}

func (j *job[S]) body(r *Request) (body []byte, contentType string) {
	switch {
	case r.json != nil:
		return r.json, "application/json"
	case r.form != nil:
		pairs := make([]string, len(r.form))
		for i, kv := range r.form {
			pairs[i] = url.QueryEscape(kv[0]) + "=" + url.QueryEscape(string(j.t.charset.GetBytes(kv[1])))
		}
		return []byte(strings.Join(pairs, "&")), "application/x-www-form-urlencoded"
	}
	return nil, ""
}

func (j *job[S]) response(r *Request, res *searching.Response) *IndexerResponse {
	out := &IndexerResponse{
		StatusCode: res.Status, Headers: res.Header, Body: res.Body, Content: j.t.charset.GetString(res.Body), URL: res.URL, Request: r,
	}
	if loc := res.Header.Get("Location"); loc != "" && res.Status >= http.StatusMultipleChoices && res.Status < http.StatusBadRequest {
		out.HasHTTPRedirect = true
		out.RedirectURL = loc
		if to, err := res.URL.Parse(loc); err == nil {
			out.RedirectURL = to.String()
		}
	}
	return out
}

// outcome is a port's error as the contract's outcome for its auth
// family. The Site's own outcomes pass unchanged, and anything else is
// failing.
func outcome(a Auth, err error) error {
	if err == nil {
		return nil
	}
	if ae, ok := errors.AsType[*AuthError](err); ok {
		switch a {
		case StatelessKey:
			return &searching.Stopped{Cause: searching.AccountRefused, Text: ae.Message}
		case PastedCookie:
			return &searching.Stopped{Cause: searching.CookieExpired}
		case FormCookie, BearerToken, FormWithCode:
		}
		return &searching.Stopped{Cause: searching.CredentialsRejected, Text: ae.Message}
	}
	if le, ok := errors.AsType[*LimitError](err); ok {
		return &searching.Limited{RetryAfter: le.After}
	}
	if errors.Is(err, ErrTwoFactorCodeNeeded) && a == FormWithCode {
		return &searching.Stopped{Cause: searching.CodeNeeded}
	}
	return err
}

// releasesOf is CleanupReleases and the conversion to the contract's
// Release: no release without a title, one per GUID.
func releasesOf(infos []TorrentInfo) []searching.Release {
	var out []searching.Release
	seen := map[string]bool{}
	for _, i := range infos {
		guid := i.GUID
		for _, alt := range []string{i.DownloadURL, i.MagnetURL, i.InfoURL} {
			if strings.TrimSpace(guid) == "" {
				guid = alt
			}
		}
		if untitled(i) || seen[guid] {
			continue
		}
		seen[guid] = true
		out = append(out, releaseOf(i))
	}
	return out
}

func releaseOf(i TorrentInfo) searching.Release {
	r := searching.Release{
		Title: i.Title, Size: i.Size, Published: i.PublishDate, Seeders: i.Seeders, Leechers: i.Peers - i.Seeders, Grabs: i.Grabs,
		Link: searching.LinkOf(i.DownloadURL), Magnet: searching.LinkOf(i.MagnetURL), Details: i.InfoURL,
		Factors: searching.Factors{Download: i.DownloadVolumeFactor, Upload: i.UploadVolumeFactor},
		Floor:   searching.Floor{MinRatio: i.MinimumRatio, MinSeed: time.Duration(i.MinimumSeedTime) * time.Second},
	}
	if h, err := hex.DecodeString(i.InfoHash); err == nil && len(h) == len(bencode.InfoHash{}) {
		r.InfoHash = (*bencode.InfoHash)(h)
	} else if m, err := magnet.Parse(i.MagnetURL); err == nil {
		r.InfoHash = &m.InfoHash
	}
	for _, c := range i.Categories {
		r.Categories = append(r.Categories, int(c.ID))
	}
	ids := map[string]string{}
	if i.ImdbID > 0 {
		ids["imdbid"] = fmt.Sprintf("tt%07d", i.ImdbID)
	}
	if i.TmdbID > 0 {
		ids["tmdbid"] = strconv.Itoa(i.TmdbID)
	}
	if i.TvdbID > 0 {
		ids["tvdbid"] = strconv.Itoa(i.TvdbID)
	}
	if len(ids) > 0 {
		r.IDs = ids
	}
	for _, f := range i.IndexerFlags {
		r.Flags = append(r.Flags, searching.Flag(f))
	}
	return r
}

// capabilitiesOf are the static capabilities as Test reports them: the
// modes by their t= value and the mapped categories.
func capabilitiesOf(c Capabilities) searching.Capabilities {
	out := searching.Capabilities{Modes: map[string][]string{}}
	for mode, params := range map[string][]Param{"search": c.SearchParams, "tvsearch": c.TvSearchParams, "movie": c.MovieSearchParams} {
		if mode == "search" && params == nil {
			params = []Param{Q}
		}
		if params == nil {
			continue
		}
		names := make([]string, len(params))
		for i, p := range params {
			names[i] = string(p)
		}
		out.Modes[mode] = names
	}
	for _, m := range c.Categories.Mappings() {
		if id := int(m.Newznab); !slices.Contains(out.Categories, id) {
			out.Categories = append(out.Categories, id)
		}
	}
	return out
}

// seasonOf reads no season as 0, as the C# reads a null Season wherever
// a port compares it.
func seasonOf(season *int) int {
	if season == nil {
		return 0
	}
	return *season
}
