// Package site is fixture-site: a private tracker's site, of which the
// skeleton serves the Torznab API and the .torrent downloads. Its API key
// is derived from the Scenario's seed.
package site

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
)

// Fake reaches no Endpoints entry: an Admin enters an Indexer's URL.
var Fake = fake.Spec{
	Name:      "fixture-site",
	TLS:       fake.FrontTLS,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("private trackers get fixtures only, #26 Q12"),
	Entered:   indexers,
}

// indexers are the Indexers an Admin added, each dialed from its home
// Node.
func indexers(get func(path string) ([]byte, error)) ([]fake.Entered, error) {
	body, err := get("/api/v1/indexers")
	if err != nil {
		return nil, err
	}
	var list struct {
		Indexers []struct {
			Name       string `json:"name"`
			URL        string `json:"url"`
			HomeNodeID string `json:"homeNodeId"`
		} `json:"indexers"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("listIndexers: %w", err)
	}
	out := make([]fake.Entered, len(list.Indexers))
	for i, ix := range list.Indexers {
		out[i] = fake.Entered{What: "Indexer " + ix.Name, URL: ix.URL, NodeID: ix.HomeNodeID}
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
				TMDBID: r.TMDBID, IMDbID: imdb[r.TMDBID], Seeders: int64(r.Seeders), Torrent: r.Metainfo,
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
	return fake.HTTP(env, Routes(APIKey(s), listed, announce)), nil
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
}

// Announce is the tracker URL a .torrent announces to when origin fetches
// it.
type Announce func(origin string) (string, error)

// Routes serve releases behind apiKey. Each .torrent announces to the
// URL announce gives its downloader.
func Routes(apiKey string, releases []Release, announce Announce) fake.Routes {
	s := &site{apiKey: apiKey, releases: releases, announce: announce}
	return fake.Routes{
		"torznab":  {Pattern: "GET " + TorznabPath + "/api", Handler: s.api},
		"download": {Pattern: "GET /download/{file}", Handler: s.download},
	}
}

func (s *site) api(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("apikey") != s.apiKey {
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

func (s *site) feed(w http.ResponseWriter, r *http.Request, keep func(Release) bool) {
	origin := "https://" + r.Host
	if r.TLS == nil {
		origin = "http://" + r.Host
	}
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:torznab="http://torznab.com/schemas/2015/feed">
  <channel>
    <title>fixture-site</title>
`)
	for _, rel := range s.releases {
		if !keep(rel) {
			continue
		}
		link := origin + "/download/" + rel.InfoHash + ".torrent"
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
    </item>
`, esc(rel.Name), esc(link), rel.Size, esc(link), esc(link), rel.Size, rel.Seeders, rel.InfoHash, rel.TMDBID, esc(rel.IMDbID))
	}
	b.WriteString("  </channel>\n</rss>\n")
	reply(w, b.String())
}

func esc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// download serves a release's .torrent announcing to the downloader's
// own tracker URL. It copies the info dictionary's bytes verbatim, so the
// info hash is the catalog's.
func (s *site) download(w http.ResponseWriter, r *http.Request) {
	hash, ok := strings.CutSuffix(r.PathValue("file"), ".torrent")
	i := slices.IndexFunc(s.releases, func(rel Release) bool { return ok && rel.InfoHash == hash })
	if i < 0 {
		http.NotFound(w, r)
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
