// Package site is fixture-site: a private tracker's site, of which the
// skeleton serves the Torznab API and the .torrent downloads. Its API key
// is derived from the Scenario's seed.
package site

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

// Fake reaches no Endpoints entry: an Admin enters an Indexer's URL.
var Fake = fake.Spec{
	Name:      "fixture-site",
	TLS:       true,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	imdb := map[int64]string{}
	if s.TMDB != nil {
		for _, m := range s.TMDB.Movies {
			imdb[m.ID] = m.IMDbID
		}
	}
	var listed []Release
	if env.Releases == nil {
		return fake.HTTP(env, Routes(APIKey(s), nil)), nil
	}
	for _, r := range env.Releases.Releases {
		listed = append(listed, Release{
			Name: r.Name, Size: r.Size, InfoHash: r.InfoHash.String(),
			TMDBID: r.TMDBID, IMDbID: imdb[r.TMDBID], Seeders: int64(r.Seeders), Torrent: r.Metainfo,
		})
	}
	return fake.HTTP(env, Routes(APIKey(s), listed)), nil
}

// Release is one item the site lists, and its .torrent.
type Release struct {
	Name string
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
}

// Routes serve releases behind apiKey.
func Routes(apiKey string, releases []Release) fake.Routes {
	s := &site{apiKey: apiKey, releases: releases}
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

func (s *site) download(w http.ResponseWriter, r *http.Request) {
	hash, ok := strings.CutSuffix(r.PathValue("file"), ".torrent")
	for _, rel := range s.releases {
		if ok && rel.InfoHash == hash {
			w.Header().Set("Content-Type", "application/x-bittorrent")
			_, _ = w.Write(rel.Torrent)
			return
		}
	}
	http.NotFound(w, r)
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
