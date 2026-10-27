package cardigann

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
)

// TestCatalog loads every definition in the directory that
// SLIPSTREAM_CARDIGANN_CATALOG names, such as an unzipped package.zip. Each
// must load, or be refused as using something SlipStream doesn't run.
func TestCatalog(t *testing.T) {
	files := catalogFiles(t)
	loaded, unsupported, broken := 0, 0, 0
	uses := map[Feature]int{}
	for _, f := range files {
		name, src := catalogFile(t, f)
		d, branches, err := load(src)
		for _, b := range branches {
			t.Logf("%s: a branch doesn't compile: %s", name, b)
		}
		broken += min(len(branches), 1)
		switch {
		case err == nil:
			loaded++
			for _, u := range d.Summary().Uses {
				uses[u]++
			}
		case errors.Is(err, errUnsupported):
			unsupported++
			t.Logf("%s: unsupported: %v", name, err)
		default:
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, c := range featureChecks {
		t.Logf("%4d use %s", uses[c.feature], c.feature)
	}
	t.Logf("%d definitions: %d load, %d unsupported, %d with a branch that doesn't compile", len(files), loaded, unsupported, broken)
	if loaded+unsupported != len(files) {
		t.Errorf("%d of %d definitions failed to read or compile", len(files)-loaded-unsupported, len(files))
	}
}

func catalogFiles(t *testing.T) []string {
	t.Helper()
	dir := os.Getenv("SLIPSTREAM_CARDIGANN_CATALOG")
	if dir == "" {
		t.Skip("SLIPSTREAM_CARDIGANN_CATALOG names no catalog directory")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no definitions in %s: %v", dir, err)
	}
	slices.Sort(files)
	return files
}

func catalogFile(t *testing.T, f string) (name string, src []byte) {
	t.Helper()
	src, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSuffix(filepath.Base(f), ".yml"), src
}

// TestCatalogSearches runs every definition's search exchange for each
// mode it offers, with its settings at their defaults, feeding each request an empty response. Every request must build, and an empty
// HTML or XML response must parse to no Releases. A JSON rows selector
// that finds no array in the empty response is counted, since Prowlarr
// fails there too.
func TestCatalogSearches(t *testing.T) {
	queries := map[string]Query{
		"search":       {Mode: "search", Text: "The Matrix"},
		"movie-search": {Mode: "movie-search", Text: "The Matrix", Year: 1999, IDs: map[string]string{"imdbid": "tt0133093", "tmdbid": "603"}, Categories: []newznab.ID{2000}},
		"tv-search":    {Mode: "tv-search", Text: "Severance", Season: 2, Episode: "3", IDs: map[string]string{"imdbid": "tt11280740", "tvdbid": "371980"}, Categories: []newznab.ID{5000}},
	}
	definitions, searches, requests, noArray := 0, 0, 0, 0
	for _, f := range catalogFiles(t) {
		name, src := catalogFile(t, f)
		d, err := Load(src)
		if err != nil {
			continue
		}
		definitions++
		runs := []*Search{d.Recent(Settings{}, testClock)}
		for mode, q := range queries {
			if _, ok := d.modes[mode]; ok {
				runs = append(runs, d.Search(Settings{}, q, testClock))
			}
		}
		for _, x := range runs {
			searches++
			sent, err := emptySearch(x)
			requests += sent
			switch {
			case err == nil:
			case strings.Contains(err.Error(), "finds no array of objects"):
				noArray++
			default:
				t.Errorf("%s: %v", name, err)
			}
		}
	}
	t.Logf("%d definitions: %d searches sent %d requests; %d JSON searches found no rows array in an empty response", definitions, searches, requests, noArray)
}

// emptySearch runs x, answering each request with an empty response, and
// reports how many requests it sent.
func emptySearch(x *Search) (int, error) {
	sent := 0
	for {
		req, ok := x.Next()
		if !ok {
			break
		}
		sent++
		if req.URL.Host == "" {
			return sent, fmt.Errorf("a request has no host: %s", req.URL)
		}
		if x.Feed(page("[]")) != nil {
			break
		}
	}
	_, err := x.Result()
	return sent, err
}

// TestCatalogLogins signs in with every definition that has a login
// block, its settings at their defaults, answering each request with an
// empty page. Every request must build. The sign-in then ends refused by
// the login test, or, for a form, finding no form on the empty page.
func TestCatalogLogins(t *testing.T) {
	ends := map[string]int{}
	for _, f := range catalogFiles(t) {
		name, src := catalogFile(t, f)
		d, err := Load(src)
		if err != nil || d.def.Login == nil {
			continue
		}
		x := d.Login(Settings{}, nil, testClock)
		for {
			req, ok := x.Next()
			if !ok {
				break
			}
			if req.URL.Host == "" {
				t.Errorf("%s: a request has no host: %s", name, req.URL)
			}
			if x.Feed(page("")) != nil {
				break
			}
		}
		err = x.Result()
		end := d.def.Login.Method + ": "
		switch {
		case err == nil, errors.Is(err, ErrSignInRefused):
			end += fmt.Sprint(err)
		case strings.Contains(err.Error(), "no form matching"):
			end += "no form"
		default:
			t.Errorf("%s: %v", name, err)
		}
		ends[end]++
	}
	for end, n := range ends {
		t.Logf("%4d %s", n, end)
	}
}

// TestCatalogYTS runs the catalog's YTS definition, a JSON API with
// multiple torrents per movie and fields read from the parent row, against
// a response written in the shape of YTS's.
func TestCatalogYTS(t *testing.T) {
	dir := os.Getenv("SLIPSTREAM_CARDIGANN_CATALOG")
	if dir == "" {
		t.Skip("SLIPSTREAM_CARDIGANN_CATALOG names no catalog directory")
	}
	_, src := catalogFile(t, filepath.Join(dir, "yts.yml"))
	d, err := Load(src)
	if err != nil {
		t.Fatal(err)
	}
	const body = `{"status": "ok", "data": {"movie_count": 1, "movies": [{"url": "https://yts.mx/movies/the-matrix-1999",
"imdb_code": "tt0133093", "title": "The Matrix", "title_long": "The Matrix (1999)", "year": 1999, "torrents": [
{"url": "https://yts.mx/torrent/download/ABC", "hash": "0123456789ABCDEF0123456789ABCDEF01234567", "quality": "1080p", "type": "bluray",
 "video_codec": "x264", "bit_depth": "8", "audio_channels": "5.1", "seeds": 100, "peers": 20, "size_bytes": 2147483648, "date_uploaded_unix": 1700000000},
{"url": "https://yts.mx/torrent/download/DEF", "hash": "89ABCDEF0123456789ABCDEF0123456789ABCDEF", "quality": "2160p", "type": "web",
 "video_codec": "x265", "bit_depth": "10", "audio_channels": "2.0", "seeds": 5, "peers": 1, "size_bytes": 5368709120, "date_uploaded_unix": 1700000100}]}]}}`
	const api = "GET https://movies-api.accel.li/api/v2/list_movies.json?query_term=tt0133093&limit=50&sort_by=date_added&order_by=desc"
	q := Query{Mode: "movie-search", Text: "The Matrix", IDs: map[string]string{"imdbid": "tt0133093"}, Categories: []newznab.ID{2000}}
	rs := mustSearch(t, d, q, site{api: page(body)})
	var got []string
	for _, r := range rs {
		got = append(got, fmt.Sprintf("%s | %s | %s | %s | %x | %d | %s | %d/%d | %v | %v | %v/%v",
			r.Title, r.Details, r.Link, r.Magnet, r.InfoHash, r.Size, r.Published.Format(time.RFC3339), r.Seeders, r.Leechers, r.Categories, r.IDs, r.DownloadFactor, r.UploadFactor))
	}
	want := []string{
		"The Matrix (1999) 1080p BRRip 5.1 x264 -YTS | https://yts.gg/movies/the-matrix-1999 | https://yts.gg/torrent/download/ABC | " +
			"magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=The+Matrix+%281999%29+1080p+BRRip+5.1+x264+-YTS | " +
			"0123456789abcdef0123456789abcdef01234567 | 2147483648 | 2023-11-14T22:13:20Z | 100/20 | [2040 100044] | map[imdbid:tt0133093] | 0/1",
		"The Matrix (1999) 2160p WEBRip 10Bit x265 -YTS | https://yts.gg/movies/the-matrix-1999 | https://yts.gg/torrent/download/DEF | " +
			"magnet:?xt=urn:btih:89abcdef0123456789abcdef0123456789abcdef&dn=The+Matrix+%281999%29+2160p+WEBRip+10Bit+x265+-YTS | " +
			"89abcdef0123456789abcdef0123456789abcdef | 5368709120 | 2023-11-14T22:15:00Z | 5/1 | [2045 100046] | map[imdbid:tt0133093] | 0/1",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Releases =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestCatalogDownloads runs every definition's download exchange, with its
// settings at their defaults, answering
// each request with a .torrent file. Every request must build. A
// definition whose selectors find no link, or whose infohash block reads
// an empty hash, in that response is counted, and any other failure is an
// error.
func TestCatalogDownloads(t *testing.T) {
	file := torrentResponse(t)
	definitions, blocks, requests := 0, 0, 0
	outcomes := map[string]int{}
	for _, f := range catalogFiles(t) {
		name, src := catalogFile(t, f)
		d, err := Load(src)
		if err != nil {
			continue
		}
		definitions++
		if d.def.Download != nil {
			blocks++
		}
		x := d.Download(Settings{}, mustURL(t, d.siteLink(Settings{})+"download-torrent-1/info/1?id=1"), testClock)
		for req, ok := x.Next(); ok; req, ok = x.Next() {
			requests++
			if req.URL.Host == "" {
				t.Errorf("%s: a request has no host: %s", name, req.URL)
			}
			if x.Feed(file) != nil {
				break
			}
		}
		outcome, err := downloadOutcome(x.Result())
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
		outcomes[outcome]++
	}
	t.Logf("%d definitions, %d with a download block: %d requests, %d torrents, %d magnets, %d whose selectors found no link, %d with an empty info hash",
		definitions, blocks, requests, outcomes["torrent"], outcomes["magnet"], outcomes["no link"], outcomes["no hash"])
}

func downloadOutcome(got Got, err error) (string, error) {
	switch {
	case err == nil:
		if _, ok := got.(Magnet); ok {
			return "magnet", nil
		}
		return "torrent", nil
	case err.Error() == "no download selector matched the download page":
		return "no link", nil
	case strings.HasPrefix(err.Error(), "read the download page's info hash"):
		return "no hash", nil
	}
	return "failed", err
}
