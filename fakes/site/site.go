// Package site is fixture-site: the fictional sites Indexers reach, each
// at its own host under Domain, which each Node dials at its own front.
// A router picks the site by the request's Host. It serves a Torznab feed,
// Brook's RSS feed and a site for each Cardigann definition in
// fakes/site/definitions, all listing the world's releases. Every key,
// password and cookie derives from the Scenario's seed.
package site

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site/ports"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/ports/all"
	"github.com/jatassi/slipstream/internal/searching"
)

// List lists an unlisted release from now on: {"release": "<name>"}.
var List = fake.CueSpec{Name: "fixture-site-list"}

// Refuse has the Torznab API answer every request with Newznab error
// code, one of newznabErrors, from now on: {"code": 101}. Code 0 serves
// requests again.
var Refuse = fake.CueSpec{Name: "fixture-site-refuse"}

// Fake sends a Node's every host under Domain to that Node's front. An
// Admin enters an Indexer's URL with its site's host.
var Fake = fake.Spec{
	Name: "fixture-site",
	TLS:  fake.FrontTLS,
	New:  newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.Searching.Sites = append(e.Searching.Sites, searching.SiteAddress{Domain: Domain, At: at.AddrPort})
	},
	Online:   fake.NoOnlineRun("private trackers get fixtures only, #26 Q12"),
	Cues:     []fake.CueSpec{List, RSSItem, Expire, Maintenance, Refuse, ports.PortCase},
	Captures: []fake.CaptureSpec{CaptchaShown, SignedIn, Searched},
	Entered:  indexers,
}

// indexers are the Indexers an Admin added, each dialed at its feed's
// host, or its definition's base address, from its home Node.
func indexers(get func(path string) ([]byte, error)) ([]fake.Entered, error) {
	body, err := get("/api/v1/indexers")
	if err != nil {
		return nil, err
	}
	var list struct {
		Indexers []struct {
			Name       string `json:"name"`
			FeedHost   string `json:"feedHost"`
			HomeNodeID string `json:"homeNodeId"`
			Definition *struct {
				BaseAddress string `json:"baseAddress"`
			} `json:"definition"`
		} `json:"indexers"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("listIndexers: %w", err)
	}
	out := make([]fake.Entered, len(list.Indexers))
	for i, ix := range list.Indexers {
		at := "https://" + ix.FeedHost
		if ix.Definition != nil {
			at = ix.Definition.BaseAddress
		}
		out[i] = fake.Entered{What: "Indexer " + ix.Name, URL: at, NodeID: ix.HomeNodeID}
	}
	return out, nil
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	imdb := map[int64]string{}
	if s.TMDB != nil {
		for _, m := range s.TMDB.Movies {
			imdb[m.ID] = m.IMDbID
		}
	}
	unlisted := map[string]bool{}
	for _, r := range s.Releases {
		unlisted[r.Name] = r.Unlisted
	}
	var listed []Release
	passkey := ""
	if env.Releases != nil {
		passkey = env.Releases.Passkey
		for _, r := range env.Releases.Releases {
			size := r.Size
			if r.AdvertisedSize > 0 {
				size = int64(r.AdvertisedSize)
			}
			listed = append(listed, Release{
				Name: r.Name, Size: size, InfoHash: r.InfoHash.String(),
				TMDBID: r.TMDBID, IMDbID: imdb[r.TMDBID], TVDB: r.TVDB, Seeders: int64(r.Seeders), Torrent: r.Metainfo, Unlisted: unlisted[r.Name],
				Link: r.Link, Trackerless: r.Trackerless,
			})
		}
	}
	announce := func(origin string) (string, error) {
		at, ok := env.Fronts.Find(fault.Target(tracker.Fake.Name), origin)
		if !ok {
			return "", fmt.Errorf("no tracker front for %q", origin)
		}
		return at.URL("/announce/" + passkey), nil
	}
	site := newSite(s, listed, announce)
	var err error
	if site.captchas, err = env.Capture(CaptchaShown.Name); err != nil {
		return nil, err
	}
	if site.signIns, err = env.Capture(SignedIn.Name); err != nil {
		return nil, err
	}
	if site.searches, err = env.Capture(Searched.Name); err != nil {
		return nil, err
	}
	for _, cue := range []error{
		fake.OnCue(env, List.Name, site.list),
		fake.OnCue(env, RSSItem.Name, site.rssItem),
		fake.OnCue(env, Expire.Name, site.expire),
		fake.OnCue(env, Maintenance.Name, site.maintain),
		fake.OnCue(env, Refuse.Name, site.refuse),
	} {
		if cue != nil {
			return nil, cue
		}
	}
	routes := site.routes()
	if _, err := ports.Serve(env, all.Registry(), ports.Embedded, routes); err != nil {
		return nil, err
	}
	return fake.HTTP(env, routes), nil
}

// Release is one item the site lists, and its .torrent.
type Release struct {
	Name string
	// Size is what the listing advertises, not the payload's.
	Size int64
	// InfoHash is the torrent's v1 info hash, lowercase hex.
	InfoHash string
	// TMDBID and IMDbID name a movie, and TVDB the episode or season
	// pack, Episode 0, of a series.
	TMDBID  int64
	IMDbID  string
	TVDB    *scenario.ReleaseEpisode
	Seeders int64
	Torrent []byte
	// Unlisted leaves it out of every feed until the List cue lists it.
	Unlisted bool
	// Link is how the feeds link it: a magnet or redirect release's item
	// carries its magnet link as the magneturl attr too.
	Link scenario.Link
	// Trackerless leaves tr out of its magnet link.
	Trackerless bool
	// Leechers, Grabs and Freeleech are what an extended feed lists.
	Leechers  int64
	Grabs     int64
	Freeleech bool
}

// APIKey is the site's API key in a world with Scenario s.
func APIKey(s scenario.Scenario) string {
	r := node.NewStreams(s.Seed + "/fixture-site").Stream("apikey")
	return fmt.Sprintf("%016x%016x", r.Uint64(), r.Uint64())
}

// Domain is the domain every site of fixture-site has its host under.
const Domain = "fixture-site.fake.localhost"

// Host is the host of the site whose label is label.
func Host(label string) string { return label + "." + Domain }

// Torznab is the label of the site that serves a Torznab feed.
const Torznab = "torznab"

// Prowlarr is the label of the site whose feed names a source in each
// item as Prowlarr does: Source, private, with Prowlarr's id 1.
const Prowlarr = "prowlarr"

// Source is the source the Prowlarr site's items name.
const Source = "Fixture Source"

// TorznabPath is the Torznab feed's base, the Indexer URL an Admin enters
// less the site's origin.
const TorznabPath = "/torznab"

// Origin is the scheme and host of the site whose label is label, in the
// scheme the front at serves.
func Origin(at fake.Address, label string) string {
	scheme := "http"
	if at.TLS {
		scheme = "https"
	}
	return scheme + "://" + Host(label)
}

// URL is the Indexer URL of the Torznab site, in the scheme the front at
// serves.
func URL(at fake.Address) string { return Origin(at, Torznab) + TorznabPath }

type site struct {
	scenario scenario.Scenario
	apiKey   string
	releases []Release
	announce Announce

	// cookie is Brook's session cookie, name=value.
	cookie string

	// captchas and signIns record what the catalog sites showed and who
	// signed in, nil where no event log keeps them.
	captchas *fake.Capture
	signIns  *fake.Capture
	searches *fake.Capture

	mu        sync.Mutex
	listed    map[string]bool
	rssListed map[string]bool
	refusal   int
	// sessions are the catalog sites' sessions, by token.
	sessions map[string]session
	// minted counts the tokens and pre-login nonces handed out.
	minted int
	// down are the catalog sites in maintenance, by label.
	down map[string]bool
	// expired are the sites whose pasted session cookie no longer works.
	expired map[string]bool
	// thanked are the releases each Larkspur session thanked, by token
	// and info hash.
	thanked map[string]bool
}

// Announce is the tracker URL a .torrent announces to when origin fetches
// it, and the tr of a magnet link origin reads.
type Announce func(origin string) (string, error)

// Routes serve releases on every site of the world with Scenario s. Each
// .torrent announces to the URL announce gives its downloader, and each
// magnet link names the URL it gives the reader of the feed or the
// redirect. They keep no capture.
func Routes(s scenario.Scenario, releases []Release, announce Announce) fake.Routes {
	return newSite(s, releases, announce).routes()
}

func newSite(s scenario.Scenario, releases []Release, announce Announce) *site {
	return &site{
		scenario: s, apiKey: APIKey(s), releases: releases, announce: announce, cookie: Cookie(s),
		listed: map[string]bool{}, rssListed: map[string]bool{}, sessions: map[string]session{}, down: map[string]bool{},
		expired: map[string]bool{}, thanked: map[string]bool{},
	}
}

// routes are every site's routes, each named <site>.<route> and served
// only at its site's host.
func (s *site) routes() fake.Routes {
	out := fake.Routes{}
	add := func(label string, routes fake.Routes) {
		for name, rt := range routes {
			method, path, _ := strings.Cut(rt.Pattern, " ")
			h := rt.Handler
			if catalogSite(label) {
				h = s.serving(label, h)
			}
			out[label+"."+name] = fake.Route{Pattern: method + " " + Host(label) + path, Handler: h}
		}
	}
	for _, label := range []string{Torznab, Prowlarr} {
		add(label, fake.Routes{
			"api":      {Pattern: "GET " + TorznabPath + "/api", Handler: s.api},
			"download": {Pattern: "GET /download/{file}", Handler: s.download},
		})
	}
	add(Brook, fake.Routes{
		"rss":      {Pattern: "GET " + RSSPath, Handler: s.rss},
		"download": {Pattern: "GET /download/{file}", Handler: s.download},
	})
	add(Larkspur, s.larkspur())
	add(Quarry, s.quarry())
	add(Cinder, s.cinder())
	add(Marsh, s.marsh())
	add(Ferry, s.ferry())
	return out
}

// ListArgs names the release the List cue lists.
type ListArgs struct {
	Release string `json:"release"`
}

type wasListed struct {
	Was bool `json:"was"`
}

func (s *site) list(_ context.Context, a ListArgs) (any, error) {
	if !slices.ContainsFunc(s.releases, func(r Release) bool { return r.Name == a.Release && r.Unlisted }) {
		return nil, fmt.Errorf("the site has no unlisted release %q", a.Release)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	was := s.listed[a.Release]
	s.listed[a.Release] = true
	return wasListed{was}, nil
}

// RefuseArgs is the Newznab error code the Refuse cue answers with.
type RefuseArgs struct {
	Code int `json:"code"`
}

func (s *site) refuse(_ context.Context, a RefuseArgs) (any, error) {
	if _, ok := newznabErrors[a.Code]; !ok && a.Code != 0 {
		return nil, fmt.Errorf("the site gives no Newznab error %d", a.Code)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refusal = a.Code
	return struct{}{}, nil
}

// Search is a Searched record's data: the site and the parameters of one
// search it served, each empty when the search left it out.
type Search struct {
	Site   string `json:"site"`
	T      string `json:"t"`
	Q      string `json:"q,omitempty"`
	TVDBID string `json:"tvdbid,omitempty"`
	Season string `json:"season,omitempty"`
	Ep     string `json:"ep,omitempty"`
}

func (s *site) searched(ctx context.Context, rec Search) {
	if s.searches != nil {
		s.searches.Record(ctx, rec)
	}
}

func (s *site) refusing() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refusal
}

// shown reports whether the feeds list rel.
func (s *site) shown(rel Release) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !rel.Unlisted || s.listed[rel.Name]
}

// PlayKey is an API key the site refuses with Newznab error code: 100 to
// 107, 200 to 203, 300, 500 ("Request limit reached") or 910. PlayKey(429) gets
// HTTP 429 with a Retry-After of 2 seconds.
func PlayKey(code int) string { return fmt.Sprintf("play-%d", code) }

// newznabErrors are the descriptions Newznab gives its error codes.
var newznabErrors = map[int]string{
	100: "Incorrect user credentials", 101: "Account suspended", 102: "Insufficient privileges", 103: "Registration denied",
	104: "Registrations are closed", 105: "Invalid registration (Email Address Taken)", 106: "Invalid registration (Email Address Bad Format)",
	107: "Registration Failed (Data error)", 200: "Missing parameter", 201: "Incorrect parameter", 202: "No such function",
	203: "Function not available", 300: "No such item", 500: "Request limit reached", 910: "API Disabled",
}

func (s *site) api(w http.ResponseWriter, r *http.Request) {
	if code := s.refusing(); code != 0 {
		torznabError(w, code, newznabErrors[code])
		return
	}
	q := r.URL.Query()
	key := q.Get("apikey")
	if key == PlayKey(http.StatusTooManyRequests) {
		w.Header().Set("Retry-After", "2")
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}
	for code, description := range newznabErrors {
		if key == PlayKey(code) {
			torznabError(w, code, description)
			return
		}
	}
	if key != s.apiKey {
		torznabError(w, 100, "Invalid API Key")
		return
	}
	label, _, _ := strings.Cut(r.Host, ".")
	if t := q.Get("t"); t != "caps" {
		s.searched(r.Context(), Search{Site: label, T: t, Q: q.Get("q"), TVDBID: q.Get("tvdbid"), Season: q.Get("season"), Ep: q.Get("ep")})
	}
	switch q.Get("t") {
	case "caps":
		reply(w, capsOf(label))
	case "tvsearch":
		s.feed(w, r, func(rel Release) bool {
			return rel.TVDB != nil && episodeMatches(*rel.TVDB, q) && matches(rel.Name, q.Get("q"))
		})
	case "search":
		s.feed(w, r, func(rel Release) bool { return matches(rel.Name, q.Get("q")) })
	case "movie":
		tmdb, imdb := q.Get("tmdbid"), q.Get("imdbid")
		s.feed(w, r, func(rel Release) bool {
			return rel.TVDB == nil && (tmdb == "" || tmdb == strconv.FormatInt(rel.TMDBID, 10)) && (imdb == "" || imdb == rel.IMDbID) && matches(rel.Name, q.Get("q"))
		})
	default:
		torznabError(w, 202, "No such function")
	}
}

// family is the Newznab category a release is listed under: Movies for a
// movie and TV for an episode.
func (rel Release) family() int {
	if rel.TVDB != nil {
		return 5000
	}
	return 2000
}

// window is the offset and limit a feed request asks for, and the
// category families it asks for, every family when it names none.
type window struct {
	offset, limit int
	families      map[int]bool
}

func windowOf(q url.Values) window {
	w := window{limit: pageLimit}
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n > 0 {
		w.offset = n
	}
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n < pageLimit {
		w.limit = n
	}
	if q.Get("cat") == "" {
		return w
	}
	w.families = map[int]bool{}
	for c := range strings.SplitSeq(q.Get("cat"), ",") {
		if n, err := strconv.Atoi(c); err == nil {
			w.families[n/1000*1000] = true
		}
	}
	return w
}

func (w window) asks(rel Release) bool { return w.families == nil || w.families[rel.family()] }

// episodeMatches is whether a tvsearch for q lists the release of ep: of
// the series q's tvdbid names, in q's season, with q's episode or as a
// pack of that season. A parameter q leaves out matches any.
func episodeMatches(ep scenario.ReleaseEpisode, q url.Values) bool {
	is := func(param string, n int64) bool {
		v := q.Get(param)
		return v == "" || v == strconv.FormatInt(n, 10)
	}
	return is("tvdbid", ep.Series) && is("season", int64(ep.Season)) && (ep.Episode == 0 || is("ep", int64(ep.Episode)))
}

// pageLimit is the caps' default and largest page.
const pageLimit = 100

// matches is whether every word of query is a word of a release's name,
// which separates words with dots.
func matches(name, query string) bool {
	words := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool { return r == '.' || r == ' ' || r == '_' || r == '-' })
	for _, w := range strings.Fields(strings.ToLower(query)) {
		if !slices.Contains(words, w) {
			return false
		}
	}
	return true
}

// capsOf is the caps of the site whose label is label: the Torznab site
// searches TV by TVDB id, season and episode, and the Prowlarr site
// searches TV only as free text.
func capsOf(label string) string {
	tv := `    <tv-search available="yes" supportedParams="q,season,ep,tvdbid" />
`
	if label == Prowlarr {
		tv = `    <tv-search available="no" supportedParams="q" />
`
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<caps>
  <server title="fixture-site" />
  <limits default="100" max="100" />
  <searching>
    <search available="yes" supportedParams="q" />
` + tv + `    <movie-search available="yes" supportedParams="q,tmdbid,imdbid" />
  </searching>
  <categories>
    <category id="2000" name="Movies" />
    <category id="5000" name="TV" />
  </categories>
</caps>
`
}

// feed lists the releases keep keeps in the window the request asks for.
// An extended request gets every attribute.
func (s *site) feed(w http.ResponseWriter, r *http.Request, keep func(Release) bool) {
	origin := originOf(r)
	win := windowOf(r.URL.Query())
	extended := r.URL.Query().Get("extended") == "1"
	var source string
	if label, _, _ := strings.Cut(r.Host, "."); label == Prowlarr {
		source = `      <prowlarrindexer id="1" type="private">` + Source + "</prowlarrindexer>\n"
	}
	var listed []Release
	for _, rel := range s.releases {
		if win.asks(rel) && keep(rel) && s.shown(rel) {
			listed = append(listed, rel)
		}
	}
	listed = listed[min(win.offset, len(listed)):]
	listed = listed[:min(win.limit, len(listed))]
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:torznab="http://torznab.com/schemas/2015/feed">
  <channel>
    <title>fixture-site</title>
`)
	for _, rel := range listed {
		link := origin + "/download/" + rel.InfoHash + ".torrent"
		var magnetAttr string
		if rel.Link.Magnet() {
			magnet, err := s.magnet(rel, fake.OriginOf(r.Context()))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if rel.Link == scenario.MagnetLink {
				link = magnet
			}
			magnetAttr = fmt.Sprintf(`      <torznab:attr name="magneturl" value="%s" />`+"\n", esc(magnet))
		}
		fmt.Fprintf(&b, `    <item>
      <title>%s</title>
      <guid>%s</guid>
      <size>%d</size>
      <link>%s</link>
      <enclosure url="%s" length="%d" type="application/x-bittorrent" />
      <torznab:attr name="seeders" value="%d" />
      <torznab:attr name="infohash" value="%s" />
%s%s%s%s    </item>
`, esc(rel.Name), esc(link), rel.Size, esc(link), esc(link), rel.Size, rel.Seeders, rel.InfoHash, idAttrs(rel), magnetAttr, extendedAttrs(rel, origin, extended), source)
	}
	b.WriteString("  </channel>\n</rss>\n")
	reply(w, b.String())
}

// idAttrs are the id attributes of rel's item: an episode's tvdbid, else
// a movie's tmdbid and imdbid.
func idAttrs(rel Release) string {
	if rel.TVDB != nil {
		return fmt.Sprintf(`      <torznab:attr name="tvdbid" value="%d" />`+"\n", rel.TVDB.Series)
	}
	return fmt.Sprintf(`      <torznab:attr name="tmdbid" value="%d" />
      <torznab:attr name="imdbid" value="%s" />
`, rel.TMDBID, esc(rel.IMDbID))
}

// originOf is the scheme and host r was sent to.
func originOf(r *http.Request) string {
	if r.TLS == nil {
		return "http://" + r.Host
	}
	return "https://" + r.Host
}

// extendedAttrs are the attributes only an extended feed lists: every one
// a Release carries, as a private tracker sets them.
func extendedAttrs(rel Release, origin string, extended bool) string {
	if !extended {
		return ""
	}
	download := 1
	if rel.Freeleech {
		download = 0
	}
	return fmt.Sprintf(`      <pubDate>%s</pubDate>
      <comments>%s</comments>
      <category>%d</category>
      <torznab:attr name="category" value="%d" />
      <torznab:attr name="peers" value="%d" />
      <torznab:attr name="grabs" value="%d" />
      <torznab:attr name="downloadvolumefactor" value="%d" />
      <torznab:attr name="uploadvolumefactor" value="1" />
      <torznab:attr name="minimumratio" value="1" />
      <torznab:attr name="minimumseedtime" value="172800" />
`, published, esc(origin+"/details/"+rel.InfoHash), rel.family(), rel.family(), rel.Seeders+rel.Leechers, rel.Grabs, download)
}

// published is when the site says every release was posted.
const published = "Mon, 01 Jun 2026 12:00:00 +0000"

func esc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// magnet is rel's magnet link as origin reads it: its info hash, its
// name and, unless it's trackerless, origin's tracker URL.
func (s *site) magnet(rel Release, origin string) (string, error) {
	link := "magnet:?xt=urn:btih:" + rel.InfoHash + "&dn=" + url.QueryEscape(rel.Name)
	if rel.Trackerless {
		return link, nil
	}
	tr, err := s.announce(origin)
	if err != nil {
		return "", err
	}
	return link + "&tr=" + url.QueryEscape(tr), nil
}

// download serves a release's .torrent announcing to the downloader's
// own tracker URL. It copies the info dictionary's bytes verbatim, so the
// info hash is the catalog's. A redirect release's answers 302 to its
// magnet link instead.
func (s *site) download(w http.ResponseWriter, r *http.Request) {
	hash, ok := strings.CutSuffix(r.PathValue("file"), ".torrent")
	i := slices.IndexFunc(s.releases, func(rel Release) bool { return ok && rel.InfoHash == hash })
	if i < 0 {
		http.NotFound(w, r)
		return
	}
	if s.releases[i].Link == scenario.RedirectLink {
		magnet, err := s.magnet(s.releases[i], fake.OriginOf(r.Context()))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, magnet, http.StatusFound)
		return
	}
	s.torrent(w, r, s.releases[i])
}

// torrent serves rel's .torrent announcing to the downloader's own
// tracker URL, with the info dictionary's bytes kept verbatim.
func (s *site) torrent(w http.ResponseWriter, r *http.Request, rel Release) {
	meta, err := bencode.DecodeMetainfo(rel.Torrent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	announce, err := s.announce(fake.OriginOf(r.Context()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/x-bittorrent")
	_, _ = w.Write(fmt.Appendf(nil, "d8:announce%d:%s4:info%se", len(announce), announce, meta.RawInfo))
}

func torznabError(w http.ResponseWriter, code int, description string) {
	reply(w, fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<error code="%d" description="%s" />
`, code, description))
}

func reply(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(body))
}
