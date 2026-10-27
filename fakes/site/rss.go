package site

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

// RSSItem lists an unlisted release in Brook's RSS feed from now on, and
// in no other feed: {"release": "<name>"}.
var RSSItem = fake.CueSpec{Name: "fixture-site-rss-item"}

// Brook is the label of the site that has only an RSS feed.
const Brook = "brook"

// RSSPath is the path of Brook's feed.
const RSSPath = "/rss"

// The plays of Brook's feed, which RSSURL puts in the address: items it
// can't be read by, or none.
const (
	PlayNoItems      = "no-items"
	PlayNoLink       = "no-link"
	PlayRepeatedGUID = "repeated-guid"
	PlayNoSize       = "no-size"
)

// RSSURL is the address of Brook's feed, which an Admin enters, playing
// play unless it is empty, in the scheme the front at serves.
func RSSURL(at fake.Address, play string) string {
	scheme := "http"
	if at.TLS {
		scheme = "https"
	}
	u := scheme + "://" + Host(Brook) + RSSPath
	if play != "" {
		u += "?" + url.Values{"play": {play}}.Encode()
	}
	return u
}

// CookieName is the name of Brook's session cookie.
const CookieName = "brook_session"

// Cookie is Brook's session cookie in a world with Scenario s, as an Admin
// pastes it: name=value.
func Cookie(s scenario.Scenario) string {
	r := node.NewStreams(s.Seed + "/fixture-site").Stream("cookie")
	return fmt.Sprintf("%s=%016x", CookieName, r.Uint64())
}

func (s *site) rssItem(_ context.Context, a ListArgs) (any, error) {
	if !slices.ContainsFunc(s.releases, func(r Release) bool { return r.Name == a.Release && r.Unlisted }) {
		return nil, fmt.Errorf("the site has no unlisted release %q", a.Release)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	was := s.rssListed[a.Release]
	s.rssListed[a.Release] = true
	return wasListed{was}, nil
}

// rss serves Brook's feed. It takes no cookie or its own session cookie,
// and refuses any other with 403, as a site does once a session expires.
func (s *site) rss(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil && CookieName+"="+c.Value != s.cookie {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	origin := "https://" + r.Host
	if r.TLS == nil {
		origin = "http://" + r.Host
	}
	play := r.URL.Query().Get("play")
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:torrent="http://xmlns.ezrss.it/0.1/">
  <channel>
    <title>Brook</title>
`)
	for _, rel := range s.releases {
		if play == PlayNoItems || !s.inRSS(rel) {
			continue
		}
		link := origin + "/download/" + rel.InfoHash + ".torrent"
		if rel.Link == scenario.MagnetLink {
			magnet, err := s.magnet(rel, fake.OriginOf(r.Context()))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			link = magnet
		}
		guid := rel.InfoHash
		if play == PlayRepeatedGUID {
			guid = Brook
		}
		enclosure := fmt.Sprintf(`      <enclosure url="%s" length="%d" type="application/x-bittorrent" />`+"\n", esc(link), rel.Size)
		switch play {
		case PlayNoSize:
			enclosure = fmt.Sprintf(`      <enclosure url="%s" type="application/x-bittorrent" />`+"\n", esc(link))
		case PlayNoLink:
			enclosure = ""
		}
		fmt.Fprintf(&b, `    <item>
      <title>%s</title>
      <guid isPermaLink="false">%s</guid>
      <pubDate>%s</pubDate>
      <description>Seeders: %d</description>
%s      <torrent:infoHash>%s</torrent:infoHash>
    </item>
`, esc(rel.Name), guid, published, rel.Seeders, enclosure, rel.InfoHash)
	}
	b.WriteString("  </channel>\n</rss>\n")
	reply(w, b.String())
}

// inRSS reports whether Brook's feed lists rel: every feed does, or the
// RSSItem cue listed it.
func (s *site) inRSS(rel Release) bool {
	if s.shown(rel) {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rssListed[rel.Name]
}
