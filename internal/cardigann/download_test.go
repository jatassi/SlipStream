package cardigann

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/magnet"
)

const hash40 = "0123456789abcdef0123456789abcdef01234567"

func torrentFile(t *testing.T) []byte {
	t.Helper()
	b, err := bencode.Encode(map[string]any{
		"announce": "https://tracker.example/announce",
		"info":     map[string]any{"name": "Film.mkv", "length": 100, "piece length": 16384, "pieces": strings.Repeat("x", 20)},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func torrentResponse(t *testing.T) Response {
	return Response{Status: http.StatusOK, Header: http.Header{"Content-Type": {"application/x-bittorrent"}}, Body: torrentFile(t)}
}

func redirect(to string) Response {
	return Response{Status: http.StatusFound, Header: http.Header{"Location": {to}}}
}

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func download(t *testing.T, d *Definition, s site) (Got, []Request, error) {
	t.Helper()
	x := d.Download(Settings{}, mustURL(t, link), testClock)
	sent := s.drive(t, x)
	got, err := x.Result()
	return got, sent, err
}

func requestLines(sent []Request) []string {
	var out []string
	for _, r := range sent {
		out = append(out, r.Method+" "+r.URL.String())
	}
	return out
}

func wantRequests(t *testing.T, sent []Request, want ...string) {
	t.Helper()
	if got := requestLines(sent); !reflect.DeepEqual(got, want) {
		t.Errorf("requests =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func wantTorrent(t *testing.T, got Got, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	tor, ok := got.(Torrent)
	if !ok {
		t.Fatalf("Result = %#v, want a Torrent", got)
	}
	if _, err := bencode.DecodeMetainfo(tor.Raw); err != nil {
		t.Errorf("the Torrent doesn't decode: %v", err)
	}
}

func wantMagnet(t *testing.T, got Got, err error, want magnet.Link) {
	t.Helper()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	m, ok := got.(Magnet)
	if !ok {
		t.Fatalf("Result = %#v, want a Magnet", got)
	}
	if !reflect.DeepEqual(m.Link, want) {
		t.Errorf("magnet = %+v, want %+v", m.Link, want)
	}
}

func hash40Info(t *testing.T) bencode.InfoHash {
	t.Helper()
	h, err := bencode.ParseInfoHash(hash40)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

const downloadBase = `id: dl
links: [https://dl.example/]
caps: {modes: {search: [q]}}
search:
  path: browse
  headers:
    X-Search: ["{{ .Config.sitelink }}"]
  rows: {selector: tr}
  fields: {title: {selector: a}}
`

const link = "https://dl.example/details.php?id=42&id=43&name=a%20b"

func TestDownloadWithNoBlockFetchesTheLink(t *testing.T) {
	d := mustLoad(t, downloadBase)
	got, sent, err := download(t, d, site{"GET " + link: torrentResponse(t)})
	wantTorrent(t, got, err)
	wantRequests(t, sent, "GET "+link)
	r := sent[0]
	if !r.FollowRedirects || r.Purpose != ForDownload || r.Header.Get("X-Search") != "https://dl.example/" {
		t.Errorf("follow redirects %v, purpose %d, X-Search %q", r.FollowRedirects, r.Purpose, r.Header.Get("X-Search"))
	}
}

func TestDownloadReadsTheResponse(t *testing.T) {
	d := mustLoad(t, downloadBase)
	webPage := Response{Status: http.StatusOK, Header: http.Header{"Content-Type": {"text/html; charset=utf-8"}}, Body: []byte("d<html>")}
	for name, tc := range map[string]struct {
		r    Response
		want string
	}{
		"not found":      {Response{Status: http.StatusNotFound}, "the download link responded with HTTP 404"},
		"a web page":     {webPage, "the download link responded with a web page"},
		"not a torrent":  {page("dx"), "read the .torrent file: "},
		"empty":          {page(""), "read the .torrent file: "},
		"an odd magnet":  {redirect("magnet:?dn=x"), "read the magnet link the download redirected to: "},
		"left redirects": {redirect("https://elsewhere.example/f.torrent"), "the download link responded with HTTP 302"},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := download(t, d, site{"GET " + link: tc.r})
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Errorf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestDownloadRedirectedToAMagnetIsAMagnet(t *testing.T) {
	d := mustLoad(t, downloadBase)
	to := "magnet:?xt=urn:btih:" + hash40 + "&dn=Film&tr=udp%3A%2F%2Ftracker.example%3A80"
	got, _, err := download(t, d, site{"GET " + link: redirect(to)})
	wantMagnet(t, got, err, magnet.Link{InfoHash: hash40Info(t), Name: "Film", Trackers: []string{"udp://tracker.example:80"}})
}

func TestDownloadVariablesFromTheLink(t *testing.T) {
	d := mustLoad(t, downloadBase+`download:
  headers:
    X-Vars: ["{{ .DownloadUri.AbsoluteUri }}|{{ .DownloadUri.AbsolutePath }}|{{ .DownloadUri.Scheme }}|{{ .DownloadUri.Host }}|{{ .DownloadUri.Port }}|{{ .DownloadUri.PathAndQuery }}|{{ .DownloadUri.Query }}|{{ .DownloadUri.Query.id }}|{{ .DownloadUri.Query.name }}"]
`)
	_, sent, _ := download(t, d, site{"GET " + link: torrentResponse(t)})
	want := link + "|/details.php|https|dl.example|443|/details.php?id=42&id=43&name=a%20b|?id=42&id=43&name=a%20b|42|a b"
	if got := sent[0].Header.Get("X-Vars"); got != want {
		t.Errorf("X-Vars =\n%s\nwant\n%s", got, want)
	}
	if sent[0].Header.Get("X-Search") != "" {
		t.Error("download headers didn't replace the search headers")
	}
}

func TestDownloadMethodPost(t *testing.T) {
	d := mustLoad(t, downloadBase+"download:\n  method: post\n")
	got, sent, err := download(t, d, site{"POST " + link: torrentResponse(t)})
	wantTorrent(t, got, err)
	wantRequests(t, sent, "POST "+link)
}

func TestDownloadBeforePostsThenFetches(t *testing.T) {
	d := mustLoad(t, downloadBase+`download:
  before:
    path: "takethanks.php"
    method: post
    inputs:
      torrentid: "{{ .DownloadUri.Query.id }}"
      thanks: 1
`)
	const thanks = "https://dl.example/takethanks.php"
	got, sent, err := download(t, d, site{"POST " + thanks: redirect("/details.php"), "GET " + link: torrentResponse(t)})
	wantTorrent(t, got, err)
	wantRequests(t, sent, "POST "+thanks, "GET "+link)
	before := sent[0]
	if string(before.Body) != "torrentid=42&thanks=1" || before.Header.Get("Referer") != link ||
		before.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || before.FollowRedirects {
		t.Errorf("before body %q, header %v, follow redirects %v", before.Body, before.Header, before.FollowRedirects)
	}
}

func TestDownloadBeforeGetWithQuerySeparator(t *testing.T) {
	d := mustLoad(t, downloadBase+`download:
  before:
    path: ajax.php
    queryseparator: ";"
    inputs:
      action: say_thanks
      id: "{{ .DownloadUri.Query.id }}"
`)
	const ajax = "https://dl.example/ajax.php?action=say_thanks;id=42"
	_, sent, err := download(t, d, site{"GET " + ajax: page("ok"), "GET " + link: torrentResponse(t)})
	if err != nil {
		t.Fatal(err)
	}
	wantRequests(t, sent, "GET "+ajax, "GET "+link)
}

func TestDownloadBeforeFailureEndsTheDownload(t *testing.T) {
	d := mustLoad(t, downloadBase+"download:\n  before:\n    path: thanks.php\n")
	_, _, err := download(t, d, site{"GET https://dl.example/thanks.php": {Status: http.StatusForbidden}})
	if err == nil || err.Error() != "the download before request responded with HTTP 403" {
		t.Errorf("err = %v", err)
	}
}

func TestDownloadBeforePathSelector(t *testing.T) {
	d := mustLoad(t, downloadBase+`download:
  before:
    pathselector:
      selector: a.thanks
      attribute: href
`)
	const thanks = "https://dl.example/thanks.php?id=42"
	s := site{"GET " + link: page(`<a class="thanks" href="thanks.php?id=42">Thanks</a>`), "GET " + thanks: page("ok")}
	_, sent, err := download(t, d, s)
	if err == nil || !strings.HasPrefix(err.Error(), "read the .torrent file") {
		t.Errorf("err = %v", err)
	}
	wantRequests(t, sent, "GET "+link, "GET "+thanks, "GET "+link)
}

const selectorsDownload = `download:
  selectors:
    - selector: a.missing
      attribute: href
    - selector: a.first
      attribute: href
    - selector: a.second
      attribute: href
      filters:
        - {name: replace, args: ["/get/", "/dl/"]}
`

const selectorsPage = `<a class="first" href="/bad.torrent">1</a><a class="second" href="/get/42.torrent">2</a>`

func TestDownloadSelectorsTestEachLink(t *testing.T) {
	d := mustLoad(t, downloadBase+selectorsDownload)
	s := site{
		"GET " + link:                          page(selectorsPage),
		"GET https://dl.example/bad.torrent":   {Status: http.StatusOK, Header: http.Header{"Content-Type": {"text/html"}}, Body: []byte("<html>")},
		"GET https://dl.example/dl/42.torrent": torrentResponse(t),
	}
	got, sent, err := download(t, d, s)
	wantTorrent(t, got, err)
	wantRequests(t, sent, "GET "+link, "GET "+link, "GET https://dl.example/bad.torrent", "GET "+link, "GET https://dl.example/dl/42.torrent")
}

func TestDownloadSelectorsWithPostFetchAgain(t *testing.T) {
	d := mustLoad(t, downloadBase+selectorsDownload+"  method: post\n")
	s := site{
		"GET " + link:                           page(selectorsPage),
		"GET https://dl.example/bad.torrent":    page("<html>"),
		"GET https://dl.example/dl/42.torrent":  torrentResponse(t),
		"POST https://dl.example/dl/42.torrent": torrentResponse(t),
	}
	got, sent, err := download(t, d, s)
	wantTorrent(t, got, err)
	wantRequests(t, sent, "GET "+link, "GET "+link, "GET https://dl.example/bad.torrent", "GET "+link,
		"GET https://dl.example/dl/42.torrent", "POST https://dl.example/dl/42.torrent")
}

func TestDownloadWithoutTestLinkTorrent(t *testing.T) {
	d := mustLoad(t, "testlinktorrent: false\n"+downloadBase+selectorsDownload)
	s := site{"GET " + link: page(selectorsPage), "GET https://dl.example/bad.torrent": page("not a torrent")}
	_, sent, err := download(t, d, s)
	if err == nil || !strings.HasPrefix(err.Error(), "read the .torrent file") {
		t.Errorf("err = %v", err)
	}
	wantRequests(t, sent, "GET "+link, "GET "+link, "GET https://dl.example/bad.torrent")
}

func TestDownloadSelectorTestRedirectedToAMagnet(t *testing.T) {
	d := mustLoad(t, downloadBase+"download:\n  selectors:\n    - {selector: a, attribute: href}\n")
	to := "magnet:?xt=urn:btih:" + hash40
	s := site{"GET " + link: page(`<a href="/go/42">get</a>`), "GET https://dl.example/go/42": redirect(to)}
	got, sent, err := download(t, d, s)
	wantMagnet(t, got, err, magnet.Link{InfoHash: hash40Info(t)})
	wantRequests(t, sent, "GET "+link, "GET https://dl.example/go/42")
}

func TestDownloadSelectorMagnetLink(t *testing.T) {
	d := mustLoad(t, downloadBase+"download:\n  selectors:\n    - {selector: \"a[href^=\\\"magnet:\\\"]\", attribute: href}\n")
	s := site{"GET " + link: page(`<a href="magnet:?xt=urn:btih:` + hash40 + `&amp;dn=Film">m</a>`)}
	got, sent, err := download(t, d, s)
	wantMagnet(t, got, err, magnet.Link{InfoHash: hash40Info(t), Name: "Film"})
	wantRequests(t, sent, "GET "+link)
}

func TestDownloadSelectorsNoneMatch(t *testing.T) {
	d := mustLoad(t, downloadBase+selectorsDownload)
	_, _, err := download(t, d, site{"GET " + link: page("<p>gone</p>")})
	if err == nil || err.Error() != "no download selector matched the download page" {
		t.Errorf("err = %v", err)
	}
}

const infohashDownload = `download:
  infohash:
    hash:
      selector: li.hash
      filters: [{name: regexp, args: "([A-Fa-f0-9]{40})"}]
    title:
      selector: h1
`

func TestDownloadInfohashBuildsATrackerlessMagnet(t *testing.T) {
	d := mustLoad(t, downloadBase+infohashDownload)
	s := site{"GET " + link: page(`<h1>Film 2020</h1><li class="hash">Hash: ` + strings.ToUpper(hash40) + `</li>`)}
	got, sent, err := download(t, d, s)
	wantMagnet(t, got, err, magnet.Link{InfoHash: hash40Info(t), Name: "Film 2020"})
	wantRequests(t, sent, "GET "+link)
}

func TestDownloadInfohashFallsThroughToTheLink(t *testing.T) {
	d := mustLoad(t, downloadBase+infohashDownload)
	s := site{"GET " + link: page(`<li class="hash">` + hash40 + `</li>`)}
	_, sent, err := download(t, d, s)
	if err == nil || !strings.HasPrefix(err.Error(), "read the .torrent file") {
		t.Errorf("err = %v", err)
	}
	wantRequests(t, sent, "GET "+link, "GET "+link)
}

func TestDownloadInfohashUsesTheBeforeResponse(t *testing.T) {
	d := mustLoad(t, downloadBase+`download:
  before:
    path: api/info
    inputs: {id: "{{ .DownloadUri.Query.id }}"}
  infohash:
    usebeforeresponse: true
    hash: {selector: ":root", filters: [{name: regexp, args: "([a-f0-9]{40})"}]}
    title: {selector: ":root", filters: [{name: regexp, args: "\"name\": \"(.+?)\""}]}
`)
	const api = "https://dl.example/api/info?id=42"
	s := site{"GET " + api: page(`{"hash": "` + hash40 + `", "name": "Film"}`)}
	got, sent, err := download(t, d, s)
	wantMagnet(t, got, err, magnet.Link{InfoHash: hash40Info(t), Name: "Film"})
	wantRequests(t, sent, "GET "+api)
}

func TestLoadRefusesAnInfohashWithoutTitle(t *testing.T) {
	_, err := Load([]byte(downloadBase + "download:\n  infohash:\n    hash: {selector: li}\n"))
	if err == nil || err.Error() != "its download infohash block needs both a hash and a title selector" {
		t.Errorf("err = %v", err)
	}
}
