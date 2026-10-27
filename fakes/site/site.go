// Package site is fixture-site: a private tracker's site, of which the
// skeleton serves the Torznab API and the .torrent downloads. Its API key
// is derived from the Scenario's seed.
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
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
)

// List lists an unlisted release from now on: {"release": "<name>"}.
var List = fake.CueSpec{Name: "fixture-site-list"}

// Fake reaches no Endpoints entry: an Admin enters an Indexer's URL.
var Fake = fake.Spec{
	Name:      "fixture-site",
	TLS:       fake.FrontTLS,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("private trackers get fixtures only, #26 Q12"),
	Cues:      []fake.CueSpec{List},
	Entered:   indexers,
}

// indexers are the Indexers an Admin added, each dialed at its feed's
// host from its home Node.
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
		} `json:"indexers"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("listIndexers: %w", err)
	}
	out := make([]fake.Entered, len(list.Indexers))
	for i, ix := range list.Indexers {
		out[i] = fake.Entered{What: "Indexer " + ix.Name, URL: "https://" + ix.FeedHost, NodeID: ix.HomeNodeID}
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
				TMDBID: r.TMDBID, IMDbID: imdb[r.TMDBID], Seeders: int64(r.Seeders), Torrent: r.Metainfo, Unlisted: unlisted[r.Name],
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
	site := newSite(APIKey(s), listed, announce)
	if err := fake.OnCue(env, List.Name, site.list); err != nil {
		return nil, err
	}
	return fake.HTTP(env, site.routes()), nil
}

// Release is one item the site lists, and its .torrent.
type Release struct {
	Name string
	// Size is what the listing advertises, not the payload's.
	Size int64
	// InfoHash is the torrent's v1 info hash, lowercase hex.
	InfoHash string
	TMDBID   int64
	IMDbID   string
	Seeders  int64
	Torrent  []byte
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

// TorznabPath is the Torznab feed's base, the Indexer URL an Admin enters
// less the site's origin.
const TorznabPath = "/torznab"

// URL is the Indexer URL of the site at at.
func URL(at fake.Address) string { return at.URL(TorznabPath) }

type site struct {
	apiKey   string
	releases []Release
	announce Announce

	mu     sync.Mutex
	listed map[string]bool
}

// Announce is the tracker URL a .torrent announces to when origin fetches
// it, and the tr of a magnet link origin reads.
type Announce func(origin string) (string, error)

// Routes serve releases behind apiKey. Each .torrent announces to the
// URL announce gives its downloader, and each magnet link names the URL
// it gives the reader of the feed or the redirect.
func Routes(apiKey string, releases []Release, announce Announce) fake.Routes {
	return newSite(apiKey, releases, announce).routes()
}

func newSite(apiKey string, releases []Release, announce Announce) *site {
	return &site{apiKey: apiKey, releases: releases, announce: announce, listed: map[string]bool{}}
}

func (s *site) routes() fake.Routes {
	return fake.Routes{
		"torznab":  {Pattern: "GET " + TorznabPath + "/api", Handler: s.api},
		"download": {Pattern: "GET /download/{file}", Handler: s.download},
	}
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
	switch q.Get("t") {
	case "caps":
		reply(w, caps)
	case "search":
		s.feed(w, r, func(rel Release) bool { return matches(rel.Name, q.Get("q")) })
	case "movie":
		tmdb, imdb := q.Get("tmdbid"), q.Get("imdbid")
		s.feed(w, r, func(rel Release) bool {
			return (tmdb == "" || tmdb == strconv.FormatInt(rel.TMDBID, 10)) && (imdb == "" || imdb == rel.IMDbID) && matches(rel.Name, q.Get("q"))
		})
	default:
		torznabError(w, 202, "No such function")
	}
}

// movies is the Newznab category every release is listed under.
const movies = 2000

// window is the offset and limit a feed request asks for, and whether it
// asks for movies when it names categories.
type window struct {
	offset, limit int
	movies        bool
}

func windowOf(q url.Values) window {
	w := window{limit: pageLimit, movies: q.Get("cat") == ""}
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n > 0 {
		w.offset = n
	}
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n < pageLimit {
		w.limit = n
	}
	for c := range strings.SplitSeq(q.Get("cat"), ",") {
		if n, err := strconv.Atoi(c); err == nil && n/1000 == movies/1000 {
			w.movies = true
		}
	}
	return w
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

const caps = `<?xml version="1.0" encoding="UTF-8"?>
<caps>
  <server title="fixture-site" />
  <limits default="100" max="100" />
  <searching>
    <search available="yes" supportedParams="q" />
    <tv-search available="no" supportedParams="q" />
    <movie-search available="yes" supportedParams="q,tmdbid,imdbid" />
  </searching>
  <categories>
    <category id="2000" name="Movies" />
  </categories>
</caps>
`

// feed lists the releases keep keeps in the window the request asks for.
// An extended request gets every attribute.
func (s *site) feed(w http.ResponseWriter, r *http.Request, keep func(Release) bool) {
	origin := "https://" + r.Host
	if r.TLS == nil {
		origin = "http://" + r.Host
	}
	win := windowOf(r.URL.Query())
	extended := r.URL.Query().Get("extended") == "1"
	var listed []Release
	for _, rel := range s.releases {
		if win.movies && keep(rel) && s.shown(rel) {
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
      <torznab:attr name="tmdbid" value="%d" />
      <torznab:attr name="imdbid" value="%s" />
%s%s    </item>
`, esc(rel.Name), esc(link), rel.Size, esc(link), esc(link), rel.Size, rel.Seeders, rel.InfoHash, rel.TMDBID, esc(rel.IMDbID), magnetAttr, extendedAttrs(rel, origin, extended))
	}
	b.WriteString("  </channel>\n</rss>\n")
	reply(w, b.String())
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
`, published, esc(origin+"/details/"+rel.InfoHash), movies, movies, rel.Seeders+rel.Leechers, rel.Grabs, download)
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
	meta, err := bencode.DecodeMetainfo(s.releases[i].Torrent)
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
