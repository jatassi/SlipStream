package cardigann

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/charmap"

	"github.com/jatassi/slipstream/internal/newznab"
)

// site answers an exchange's requests with recorded responses, keyed by
// method and address.
type site map[string]Response

func page(body string) Response { return Response{Status: http.StatusOK, Body: []byte(body)} }

// drive runs x to its end against s and returns the requests it sent.
func (s site) drive(t *testing.T, x Exchange) []Request {
	t.Helper()
	var sent []Request
	for {
		req, ok := x.Next()
		if !ok {
			return sent
		}
		sent = append(sent, req)
		key := req.Method + " " + req.URL.String()
		r, ok := s[key]
		if !ok {
			t.Fatalf("no recorded response for %s", key)
		}
		if err := x.Feed(r); err != nil {
			if _, more := x.Next(); more {
				t.Fatalf("the exchange goes on after Feed failed: %v", err)
			}
			return sent
		}
	}
}

func search(t *testing.T, d *Definition, q Query, s site) ([]Release, error) {
	t.Helper()
	x := d.Search(Settings{}, q, testClock)
	s.drive(t, x)
	rs, err := x.Result()
	for i := range rs {
		rs[i].Published = rs[i].Published.UTC()
	}
	return rs, err
}

func mustSearch(t *testing.T, d *Definition, q Query, s site) []Release {
	t.Helper()
	rs, err := search(t, d, q, s)
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	return rs
}

func titles(rs []Release) string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Title)
	}
	return strings.Join(out, ", ")
}

const requestsDefinition = `id: requests
links: [https://req.example/]
legacylinks: [https://old.example/]
settings:
  - {name: sort, type: select, default: added, options: {added: Added, seeders: Seeders}}
  - {name: freeleech, type: checkbox, default: false}
  - {name: apikey, type: text}
caps:
  categorymappings:
    - {id: 1, cat: Movies, desc: Movies}
    - {id: 2, cat: TV, desc: TV}
    - {id: 3, cat: TV/HD, desc: TV HD}
  modes:
    search: [q]
    tv-search: [q, season, ep]
search:
  paths:
    - path: movies
      categories: [1]
    - path: browse.php
      categories: [2, 3]
    - path: browse.php
      categories: ["!", 1]
    - path: "api/{{ .Keywords }}/x+y"
      method: post
  headers:
    Authorization: ["Bearer {{ .Config.apikey }}"]
  keywordsfilters:
    - name: re_replace
      args: ["[()]", ""]
  inputs:
    $raw: "{{ range .Categories }}c[]={{.}}&{{end}}kw={{ .Keywords }}"
    q: "{{ .Keywords }}"
    sort: "{{ .Config.sort }}"
    free: "{{ if .Config.freeleech }}1{{ else }}{{ end }}"
    t: "a!*()~b"
  rows:
    selector: tr
  fields:
    title:
      selector: a
`

func TestSearchRequestsEveryMatchingPath(t *testing.T) {
	d := mustLoad(t, requestsDefinition)
	const (
		browse = "https://req.example/browse.php?c[]=2&c[]=3&kw=Show%2B2020%2BS01E02&q=Show+2020+S01E02&sort=seeders&t=a!*()%7Eb"
		negate = "https://req.example/browse.php?kw=Show%2B2020%2BS01E02&q=Show+2020+S01E02&sort=seeders&t=a!*()%7Eb"
		api    = "https://req.example/api/Show%202020%20S01E02/x%20y"
	)
	s := site{"GET " + browse: page("<table></table>"), "GET " + negate: page(""), "POST " + api: page("")}
	settings := Settings{BaseURL: "https://old.example/", Values: map[string]string{"sort": "seeders", "freeleech": "false", "apikey": "KEY"}}
	x := d.Search(settings, Query{Mode: "tv-search", Text: "Show (2020)", Season: 1, Episode: "2", Categories: []newznab.ID{5000}}, testClock)
	sent := s.drive(t, x)
	if rs, err := x.Result(); err != nil || len(rs) != 0 {
		t.Fatalf("Result = %v, %v", rs, err)
	}
	var got []string
	for _, r := range sent {
		got = append(got, r.Method+" "+r.URL.String())
		if r.Header.Get("Authorization") != "Bearer KEY" || r.Purpose != ForSearch {
			t.Errorf("%s: Authorization %q, purpose %d", r.URL, r.Header.Get("Authorization"), r.Purpose)
		}
	}
	want := []string{"GET " + browse, "GET " + negate, "POST " + api}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requests =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	post := sent[2]
	if string(post.Body) != "c[]=2&c[]=3&kw=Show%2B2020%2BS01E02&q=Show+2020+S01E02&sort=seeders&t=a!*()%7Eb" ||
		post.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Errorf("POST body %q, content type %q", post.Body, post.Header.Get("Content-Type"))
	}
}

func TestSearchSkipsARepeatedGet(t *testing.T) {
	d := mustLoad(t, `id: twice
links: [https://twice.example/]
caps: {modes: {search: [q]}}
search:
  paths:
    - path: "browse?q={{ .Keywords }}"
    - path: "browse?q={{ .Keywords }}"
      followredirect: true
  rows: {selector: tr}
  fields: {title: {selector: a}}
`)
	sent := site{"GET https://twice.example/browse?q=a%20b": page("")}.drive(t, d.Search(Settings{}, Query{Mode: "search", Text: "a b"}, testClock))
	if len(sent) != 1 || sent[0].FollowRedirects {
		t.Errorf("sent %d requests, the first following redirects: %v", len(sent), sent[0].FollowRedirects)
	}
}

const htmlDefinition = `id: htmlsite
type: public
links: [https://html.example/]
caps:
  categorymappings:
    - {id: 10, cat: Movies/HD, desc: "HD Movies"}
    - {id: 20, cat: TV, desc: "TV"}
  modes: {search: [q]}
search:
  paths: [{path: browse}]
  rows:
    selector: table.list tr.row
  fields:
    category:
      selector: a.cat
      attribute: href
      filters: [{name: querystring, args: cat}]
    title:
      selector: a.title
    details:
      selector: a.title
      attribute: href
    download:
      selector: a.dl
      attribute: href
    infohash:
      selector: td.hash
      optional: true
    size:
      selector: td.size
    seeders:
      selector: td.seed
    leechers:
      selector: td.leech
    grabs:
      selector: td.snatched
    date:
      selector: td.added
      filters: [{name: dateparse, args: "yyyy-MM-dd HH:mm"}]
    downloadvolumefactor:
      case:
        img.free: 0
        "*": 1
    uploadvolumefactor:
      text: 1
    imdbid:
      selector: a.imdb
      attribute: href
    minimumratio:
      text: 0.8
    minimumseedtime:
      text: 172800
`

const htmlPage = `<html><body><table class="list">
<tr class="row"><td><a class="cat" href="/browse?cat=10">HD</a></td><td><a class="title" href="/details.php?id=1">Movie One 2020 1080p</a></td>
<td><a class="dl" href="dl.php?id=1">get</a></td><td class="hash">0123456789abcdef0123456789abcdef01234567</td><td class="size">1.5 GB</td>
<td class="seed">1,234</td><td class="leech">5</td><td class="snatched">7</td><td class="added">2026-10-20 13:45</td><td><img class="free"><a class="imdb" href="https://imdb.example/title/tt0133093/">i</a></td></tr>
<tr class="row"><td><a class="cat" href="/browse?cat=20">TV</a></td><td><a class="title" href="/details.php?id=2">Show S01E01</a></td>
<td><a class="dl" href="magnet:?xt=urn:btih:89abcdef0123456789abcdef0123456789abcdef&amp;dn=Show">get</a></td><td class="size">700 MB</td>
<td class="seed">3</td><td class="leech">9000000</td><td class="snatched">0</td><td class="added">2026-10-21 08:00</td><td></td></tr>
<tr class="row"><td><a class="cat" href="/browse?cat=20">TV</a></td><td>no title</td><td><a class="dl" href="dl.php?id=3">get</a></td><td class="size">1 KB</td>
<td class="seed">1</td><td class="leech">1</td><td class="snatched">1</td><td class="added">2026-10-21 08:00</td><td></td></tr>
<tr class="row"><td><a class="cat" href="/browse?cat=10">HD</a></td><td><a class="title" href="/details.php?id=4">Movie One again</a></td>
<td><a class="dl" href="dl.php?id=1">get</a></td><td class="size">1 GB</td>
<td class="seed">1</td><td class="leech">1</td><td class="snatched">1</td><td class="added">2026-10-21 08:00</td><td></td></tr>
</table></body></html>`

func TestSearchParsesHTMLRows(t *testing.T) {
	d := mustLoad(t, htmlDefinition)
	rs := mustSearch(t, d, Query{Mode: "search", Text: "one"}, site{"GET https://html.example/browse": page(htmlPage)})
	hash1 := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67}
	hash2 := []byte{0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	want := []Release{
		{
			Title: "Movie One 2020 1080p", Details: "https://html.example/details.php?id=1", Link: "https://html.example/dl.php?id=1",
			Magnet:   "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=Movie+One+2020+1080p",
			InfoHash: hash1, Size: 1610612736, Published: time.Date(2026, 10, 20, 11, 45, 0, 0, time.UTC),
			Seeders: 1234, Leechers: 5, Grabs: 7, Categories: []newznab.ID{2040, 100010},
			IDs: map[string]string{"imdbid": "tt0133093"}, DownloadFactor: 0, UploadFactor: 1, MinRatio: 0.8, MinSeed: 48 * time.Hour,
		},
		{
			Title: "Show S01E01", Details: "https://html.example/details.php?id=2",
			Magnet:   "magnet:?xt=urn:btih:89abcdef0123456789abcdef0123456789abcdef&dn=Show",
			InfoHash: hash2, Size: 734003200, Published: time.Date(2026, 10, 21, 6, 0, 0, 0, time.UTC),
			Seeders: 3, Leechers: 0, Categories: []newznab.ID{5000, 100020}, DownloadFactor: 1, UploadFactor: 1, MinRatio: 0.8, MinSeed: 48 * time.Hour,
		},
	}
	if !reflect.DeepEqual(rs, want) {
		t.Errorf("Releases =\n%+v\nwant\n%+v", rs, want)
	}
}

func TestSearchRowsAfterRemoveAndResultLeak(t *testing.T) {
	d := mustLoad(t, `id: merged
links: [https://merged.example/]
caps: {modes: {search: [q]}}
search:
  paths: [{path: list}]
  rows:
    selector: tr
    after: 1
  fields:
    title: {selector: a}
    download: {selector: a, attribute: href}
    day: {selector: span.day}
    date:
      text: "{{ .Result.day }} 10:00"
      filters: [{name: dateparse, args: "yyyy-MM-dd HH:mm"}]
    size:
      selector: td.size
      remove: span
`)
	body := `<table>
<tr><td><a href="1">First</a><span class="day">2026-10-01</span></td></tr><tr><td class="size"><span>9 GB</span>1 MB</td></tr>
<tr><td><a href="2">Second</a></td></tr><tr><td class="size">2 MB</td></tr>
</table>`
	rs := mustSearch(t, d, Query{Mode: "search"}, site{"GET https://merged.example/list": page(body)})
	if len(rs) != 2 {
		t.Fatalf("Releases = %+v", rs)
	}
	oct1 := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	if rs[0].Size != 1<<20 || rs[1].Size != 2<<20 || !rs[0].Published.Equal(oct1) || !rs[1].Published.Equal(oct1) {
		t.Errorf("sizes %d, %d, dates %v, %v; want the second row to keep the first row's day", rs[0].Size, rs[1].Size, rs[0].Published, rs[1].Published)
	}
	_, err := search(t, d, Query{Mode: "search"}, site{"GET https://merged.example/list": page(`<table><tr><td><a href="1">Alone</a></td></tr></table>`)})
	if err == nil || !strings.Contains(err.Error(), "runs past the last row") {
		t.Errorf("a row short of its after rows: %v", err)
	}
}

func TestSearchDateHeaders(t *testing.T) {
	d := mustLoad(t, `id: headers
links: [https://headers.example/]
caps: {modes: {search: [q]}}
search:
  paths: [{path: list}]
  rows:
    selector: tr.row
    dateheaders:
      selector: td.day
      filters: [{name: dateparse, args: "yyyy-MM-dd"}]
  fields:
    title: {selector: a}
    download: {selector: a, attribute: href}
`)
	body := `<table>
<tr><td class="day">2026-10-02</td></tr>
<tr class="row"><td><a href="A">A</a></td></tr>
<tr class="row"><td><a href="B">B</a></td></tr>
<tr><td class="day">2026-10-01</td></tr>
<tr class="row"><td><a href="C">C</a></td></tr>
</table><table><tr class="row"><td><a href="D">D</a></td></tr></table>`
	rs := mustSearch(t, d, Query{Mode: "search"}, site{"GET https://headers.example/list": page(body)})
	got := map[string]string{}
	for _, r := range rs {
		got[r.Title] = r.Published.Format(time.DateOnly)
	}
	want := map[string]string{"A": "2026-10-01", "B": "2026-10-01", "C": "2026-09-30"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dates = %v, want %v (D has no header row and is dropped)", got, want)
	}
}

func TestSearchAndMatch(t *testing.T) {
	d := mustLoad(t, `id: andmatch
links: [https://and.example/]
caps: {modes: {search: [q], movie-search: [q, imdbid]}}
search:
  paths: [{path: list}]
  rows:
    selector: tr
    filters: [{name: andmatch}]
  fields:
    title: {selector: a}
    download: {selector: a, attribute: href}
`)
	body := `<table><tr><td><a href="1">The Big Lebowski 1998</a></td></tr><tr><td><a href="2">Big Fish</a></td></tr>
<tr><td><a href="3">lebowski fest</a></td></tr><tr><td><a href="4">Something else</a></td></tr></table>`
	s := site{"GET https://and.example/list": page(body)}
	tests := []struct {
		q    Query
		want string
	}{
		{Query{Mode: "search", Text: "The Big Lebowski"}, "The Big Lebowski 1998"},
		{Query{Mode: "search", Text: "Lebowski"}, "The Big Lebowski 1998, lebowski fest"},
		{Query{Mode: "movie-search", Text: "Lebowski", IDs: map[string]string{"imdbid": "tt0118715"}}, "The Big Lebowski 1998, Big Fish, lebowski fest, Something else"},
		{Query{Mode: "search"}, "The Big Lebowski 1998, Big Fish, lebowski fest, Something else"},
	}
	for _, tt := range tests {
		if got := titles(mustSearch(t, d, tt.q, s)); got != tt.want {
			t.Errorf("%+v: %s, want %s", tt.q, got, tt.want)
		}
	}
}

const jsonDefinition = `id: jsonsite
type: private
links: [https://json.example/]
caps:
  categorymappings:
    - {id: 1, cat: Movies}
  modes: {search: [q]}
search:
  paths:
    - path: api
      response: {type: json}
  rows:
    selector: data.movies
    attribute: torrents
    multiple: true
    missingAttributeEqualsNoResults: true
    count: {selector: data.movie_count}
  fields:
    movie: {selector: ..title}
    quality: {selector: quality}
    title: {text: "{{ .Result.movie }} {{ .Result.quality }}"}
    category: {text: 1}
    download: {selector: url}
    infohash: {selector: hash}
    size: {selector: size_bytes}
    seeders: {selector: seeds}
    leechers: {selector: peers}
    date:
      selector: date_uploaded
      filters: [{name: dateparse, args: "MM/dd/yyyy HH:mm:ss"}]
    downloadvolumefactor:
      selector: freeleech
      case: {True: 0, False: 1}
    tmdbid:
      selector: tmdb
`

func TestSearchParsesJSONRows(t *testing.T) {
	d := mustLoad(t, jsonDefinition)
	body := `{"data": {"movie_count": 2, "movies": [
  {"title": "Alpha", "torrents": [
    {"quality": "720p", "url": "/t/1", "hash": "0123456789ABCDEF0123456789ABCDEF01234567", "size_bytes": 1000, "seeds": 4, "peers": 6, "date_uploaded": "2026-10-20T13:45:00Z", "freeleech": true, "tmdb": null},
    {"quality": "1080p", "url": "https://cdn.example/t/2", "hash": "", "size_bytes": 2000, "seeds": 1, "peers": 0, "date_uploaded": "2026-10-21T00:00:00Z", "freeleech": false, "tmdb": 603}]},
  {"title": "No torrents"}]}}`
	rs := mustSearch(t, d, Query{Mode: "search"}, site{"GET https://json.example/api": page(body)})
	want := []Release{
		{
			Title: "Alpha 720p", Link: "https://json.example/t/1",
			InfoHash: []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67},
			Size:     1000, Published: time.Date(2026, 10, 20, 13, 45, 0, 0, time.UTC), Seeders: 4, Leechers: 6,
			Categories: []newznab.ID{2000}, DownloadFactor: 0,
		},
		{
			Title: "Alpha 1080p", Link: "https://cdn.example/t/2", Size: 2000, Published: time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
			Seeders: 1, Categories: []newznab.ID{2000}, DownloadFactor: 1, IDs: map[string]string{"tmdbid": "603"},
		},
	}
	if !reflect.DeepEqual(rs, want) {
		t.Errorf("Releases =\n%+v\nwant\n%+v", rs, want)
	}
	if rs := mustSearch(t, d, Query{Mode: "search"}, site{"GET https://json.example/api": page(`{"data": {"movie_count": 0}}`)}); len(rs) != 0 {
		t.Errorf("a count of 0: %+v", rs)
	}
	_, err := search(t, d, Query{Mode: "search"}, site{"GET https://json.example/api": page(`{"data": {"movies": [{"title": "A", "torrents": [{"quality": "x"}]}]}}`)})
	if err == nil || !strings.Contains(err.Error(), "field download") {
		t.Errorf("a JSON row missing a required field: %v, want the response to fail", err)
	}
}

func TestSearchNoResultsMessage(t *testing.T) {
	src := `id: nomsg
links: [https://nomsg.example/]
caps: {modes: {search: [q]}}
search:
  paths:
    - path: api
      response: {type: json, noResultsMessage: "%s"}
  rows: {selector: data}
  fields: {title: {selector: name}}
`
	for msg, body := range map[string]string{"No Torrents Found": `{"error": "No Torrents Found"}`, "": "  "} {
		d := mustLoad(t, strings.Replace(src, "%s", msg, 1))
		if rs, err := search(t, d, Query{Mode: "search"}, site{"GET https://nomsg.example/api": page(body)}); err != nil || len(rs) != 0 {
			t.Errorf("noResultsMessage %q over %q: %v, %v", msg, body, rs, err)
		}
	}
	d := mustLoad(t, strings.Replace(src, "%s", "nothing", 1))
	if _, err := search(t, d, Query{Mode: "search"}, site{"GET https://nomsg.example/api": page(`{"error": "down"}`)}); err == nil {
		t.Error("a JSON response without rows or the message passed")
	}
}

func TestSearchError(t *testing.T) {
	d := mustLoad(t, `id: errs
links: [https://errs.example/]
caps: {modes: {search: [q]}}
search:
  paths: [{path: list}]
  error:
    - selector: 'div.error:not(:contains("There are no results found."))'
    - selector: ':root:contains("Database error.")'
      message: {text: "the database is down"}
  rows: {selector: tr}
  fields: {title: {selector: a}}
`)
	tests := []struct{ body, want string }{
		{`<div class="error">There are no results found.</div>`, ""},
		{`<div class="error"> Too many searches. </div>`, "Too many searches."},
		{`<p>Database error.</p>`, "the database is down"},
	}
	for _, tt := range tests {
		_, err := search(t, d, Query{Mode: "search"}, site{"GET https://errs.example/list": page(tt.body)})
		var rej *Rejected
		switch {
		case tt.want == "" && err != nil:
			t.Errorf("%s: %v", tt.body, err)
		case tt.want != "" && (!errors.As(err, &rej) || rej.Message != tt.want || !rej.Search):
			t.Errorf("%s: %v, want a search Rejected with %q", tt.body, err, tt.want)
		}
	}
}

func TestSearchParsesXMLRows(t *testing.T) {
	d := mustLoad(t, `id: rss
links: [https://rss.example/]
caps:
  categories: {1: Movies}
  modes: {search: [q]}
search:
  paths:
    - path: rss
      response: {type: xml}
  rows: {selector: item}
  fields:
    title: {selector: title}
    download: {selector: link}
    size: {selector: enclosure, attribute: length}
    category: {text: 1}
    date:
      selector: pubDate
      filters: [{name: dateparse, args: "ddd, dd MMM yyyy HH:mm:ss zzz"}]
`)
	body := `<?xml version="1.0" encoding="UTF-8"?><rss><channel><item><title>Feed Item</title><link>https://rss.example/get/1</link>
<pubDate>Tue, 20 Oct 2026 13:45:00 +00:00</pubDate><enclosure url="x" length="4096"/></item></channel></rss>`
	rs := mustSearch(t, d, Query{Mode: "search"}, site{"GET https://rss.example/rss": page(body)})
	want := []Release{{Title: "Feed Item", Link: "https://rss.example/get/1", Size: 4096, Published: time.Date(2026, 10, 20, 13, 45, 0, 0, time.UTC), Categories: []newznab.ID{2000}}}
	if !reflect.DeepEqual(rs, want) {
		t.Errorf("Releases =\n%+v\nwant\n%+v", rs, want)
	}
}

func TestSearchEncodesBothWays(t *testing.T) {
	d := mustLoad(t, `id: cyr
encoding: windows-1251
links: [https://cyr.example/]
caps: {modes: {search: [q]}}
search:
  paths: [{path: list}]
  inputs: {q: "{{ .Keywords }}"}
  rows: {selector: tr}
  fields: {title: {selector: a}}
`)
	body, err := charmap.Windows1251.NewEncoder().String(`<table><tr><td><a>Тест</a></td></tr></table>`)
	if err != nil {
		t.Fatal(err)
	}
	r := page(body)
	r.Header = http.Header{"Content-Type": {"text/html; charset=utf-8"}}
	rs := mustSearch(t, d, Query{Mode: "search", Text: "Привет мир"}, site{"GET https://cyr.example/list?q=%CF%F0%E8%E2%E5%F2+%EC%E8%F0": r})
	if titles(rs) != "Тест" {
		t.Errorf("titles = %q", titles(rs))
	}
}

func TestSearchFailures(t *testing.T) {
	d := mustLoad(t, `id: fails
links: [https://fails.example/]
caps: {modes: {search: [q]}}
search:
  paths: [{path: list}, {path: more}]
  rows: {selector: tr}
  fields: {title: {selector: a}}
`)
	tests := []struct {
		name string
		resp Response
		want string
	}{
		{"redirect", Response{Status: http.StatusFound, Header: http.Header{"Location": {"/elsewhere"}}}, "redirected the search to https://fails.example/elsewhere"},
		{"login redirect", Response{Status: http.StatusFound, Header: http.Header{"Location": {"/login.php?x=1"}}}, "redirected the search to its login page"},
		{"refresh", Response{Status: http.StatusOK, Header: http.Header{"Refresh": {"0; url=/wait"}}}, "redirected the search to https://fails.example/wait"},
		{"status", Response{Status: http.StatusNotFound}, "responded with HTTP 404"},
	}
	for _, tt := range tests {
		x := d.Search(Settings{}, Query{Mode: "search"}, testClock)
		sent := site{"GET https://fails.example/list": tt.resp}.drive(t, x)
		_, err := x.Result()
		if len(sent) != 1 || err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: %d requests, %v, want %q", tt.name, len(sent), err, tt.want)
		}
		if err := x.Feed(page("")); err == nil {
			t.Errorf("%s: Feed after the end passed", tt.name)
		}
	}
	x := d.Search(Settings{}, Query{Mode: "search"}, testClock)
	if _, err := x.Result(); err == nil {
		t.Error("Result before the exchange ended passed")
	}
	if _, err := d.Search(Settings{}, Query{Mode: "radio-search"}, testClock).Result(); err == nil {
		t.Error("an unknown mode passed")
	}
}

func TestEpisodeSearchString(t *testing.T) {
	tests := []struct {
		season  int
		episode string
		want    string
	}{
		{0, "3", ""},
		{1, "", "S01"},
		{1, "2", "S01E02"},
		{12, "103", "S12E103"},
		{2026, "10/27", "2026.10.27"},
	}
	for _, tt := range tests {
		if got := episodeSearchString(tt.season, tt.episode, testClock); got != tt.want {
			t.Errorf("episodeSearchString(%d, %q) = %q, want %q", tt.season, tt.episode, got, tt.want)
		}
	}
}

func TestBaseScope(t *testing.T) {
	d := mustLoad(t, `id: vars
links: [https://a.example/, https://b.example/]
legacylinks: [https://old.example/]
settings:
  - {name: user, type: text}
  - {name: sort, type: select, default: "2", options: {1: newest, 2: seeders}}
  - {name: free, type: checkbox, default: true}
  - {name: off, type: checkbox, default: true}
  - {name: about, type: info, default: Read this.}
caps: {modes: {search: [q]}}
search:
  paths: [{path: s}]
  rows: {selector: tr}
  fields: {title: {selector: a}}
`)
	tests := []struct {
		settings Settings
		want     scope
	}{
		{Settings{BaseURL: "https://old.example/", Values: map[string]string{"off": "False"}}, scope{
			".Config.sitelink": textValue("https://a.example/"), ".Config.user": nullValue, ".Config.sort": textValue("2"),
			".Config.free": textValue(".True"), ".Config.off": nullValue,
		}},
		{Settings{BaseURL: "https://b.example/", Values: map[string]string{"user": "", "sort": "1", "free": "false", "off": "true"}}, scope{
			".Config.sitelink": textValue("https://b.example/"), ".Config.user": textValue(""), ".Config.sort": textValue("1"),
			".Config.free": nullValue, ".Config.off": textValue(".True"),
		}},
	}
	for _, tt := range tests {
		got := d.baseScope(tt.settings, testClock)
		tt.want[".True"], tt.want[".False"], tt.want[".Today.Year"] = textValue("True"), nullValue, textValue("2026")
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("baseScope(%+v) =\n%v\nwant\n%v", tt.settings, got, tt.want)
		}
	}
}
