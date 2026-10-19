package storing

import (
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
)

// identities is a fake file identity: paths naming one folder share a
// key, and a path not listed can't be read.
type identities map[string]string

func (ids identities) same(a, b string) bool {
	ka, oka := ids[a]
	kb, okb := ids[b]
	return oka && okb && ka == kb
}

func TestJudgeOverlapByIdentity(t *testing.T) {
	movies, tv, gone := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	ids := identities{
		"/": "root", "/srv": "srv", "/srv/media": "media",
		"/srv/media/movies": "movies", "/srv/media/movies/new": "movies/new",
		"/srv/tv": "tv", "/mnt/tv-link": "tv", "/mnt": "mnt",
		"/var/lib/slipstream": "data", "/var/lib": "varlib", "/var": "var",
		"/var/lib/slipstream/x": "data/x",
		"/dl":                   "dl", "/dl/incomplete": "dl/incomplete",
		"/srv/other": "other",
	}
	others := []Other{{movies, "/srv/media/movies"}, {tv, "/srv/tv"}, {gone, "/srv/gone"}}
	for _, c := range []struct {
		name, folder string
		want         *Refusal
	}{
		{"inside a root", "/srv/media/movies/new", &Refusal{ReasonInsideRoot, movies}},
		{"equal to a root", "/srv/media/movies", &Refusal{ReasonInsideRoot, movies}},
		{"contains a root", "/srv/media", &Refusal{ReasonContainsRoot, movies}},
		{"a link to a root", "/mnt/tv-link", &Refusal{ReasonInsideRoot, tv}},
		{"inside data", "/var/lib/slipstream/x", &Refusal{ReasonInsideData, storage.UUID{}}},
		{"equal to data", "/var/lib/slipstream", &Refusal{ReasonInsideData, storage.UUID{}}},
		{"contains data", "/var/lib", &Refusal{ReasonContainsData, storage.UUID{}}},
		{"inside downloads", "/dl/incomplete", &Refusal{ReasonInsideDownloads, storage.UUID{}}},
		{"roots before reserved folders", "/", &Refusal{ReasonContainsRoot, movies}},
		{"an unreadable root holds nothing", "/srv/gone", nil},
		{"apart from all", "/srv/other", nil},
	} {
		got := judge(Unix, c.folder, append(apartFrom(others), forRoot("/var/lib/slipstream", "/dl")...), ids.same)
		if (got == nil) != (c.want == nil) || got != nil && *got != *c.want {
			t.Errorf("%s: judge(%q) = %+v, want %+v", c.name, c.folder, got, c.want)
		}
	}
}

func TestJudgeDownloadsWithoutRoots(t *testing.T) {
	ids := identities{`C:\`: "c", `C:\Downloads`: "dl", `c:\downloads`: "dl", `C:\Media`: "media"}
	if got := judge(Windows, `C:\`, forRoot("", `c:\downloads`), ids.same); got == nil || got.Reason != ReasonContainsDownloads {
		t.Errorf("judge(C:\\) = %+v, want contains_downloads", got)
	}
	if got := judge(Windows, `C:\Downloads`, forRoot("", `c:\downloads`), ids.same); got == nil || got.Reason != ReasonInsideDownloads {
		t.Errorf("judge(C:\\Downloads) = %+v, want inside_downloads", got)
	}
	if got := judge(Windows, `C:\Media`, forRoot("", `c:\downloads`), ids.same); got != nil {
		t.Errorf("judge(C:\\Media) = %+v, want usable", got)
	}
}
