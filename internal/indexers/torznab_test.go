package indexers_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/indexers"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCapsKeepEachAvailableModeAndItsParameters(t *testing.T) {
	for _, tc := range []struct{ file, want string }{
		{"prowlarr-caps.xml", `{"modes":{"movie":["q","imdbid","tmdbid"],"search":["q"],"tvsearch":["q","season","ep","imdbid","tvdbid","tmdbid"]}}`},
		{"jackett-caps.xml", `{"modes":{"book":["q"],"movie":["q","imdbid"],"search":["q"],"tvsearch":["q","season","ep"]}}`},
	} {
		t.Run(tc.file, func(t *testing.T) {
			c, err := indexers.ParseCaps(fixture(t, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(c)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("caps = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestAnErrorDocumentIsNoCaps(t *testing.T) {
	_, err := indexers.ParseCaps(fixture(t, "error.xml"))
	if want := "read the caps: the document is <error>, not <caps>"; err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %s", err, want)
	}
}

func TestItemsKeepTheirTorrentSeedersHashAndIDs(t *testing.T) {
	for _, tc := range []struct {
		file string
		want []string
	}{
		{"prowlarr-results.xml", []string{
			`"Night.Harbor.2019.1080p.BluRay.x264-GRP" 8589934592 https://prowlarr.example.test/1/download?apikey=k&id=41 seeders=57 hash=0123456789abcdef0123456789abcdef01234567 ids=map[imdbid:tt0000501 tmdbid:501]`,
			`"Night.Harbor.2019.720p.WEB.h264-OTHER" 2147483648 https://prowlarr.example.test/1/download?apikey=k&id=42 seeders=3 hash= ids=map[]`,
		}},
		{"jackett-results.xml", []string{
			`"Night Harbor 2019 2160p UHD BluRay" 53687091200 http://jackett.example.test:9117/dl/example/?jackett_apikey=k&path=77&file=Night+Harbor seeders=9 hash=fedcba9876543210fedcba9876543210fedcba98 ids=map[imdbid:tt0000501]`,
			`"Night Harbor 2019 Sample" 1048576 http://jackett.example.test:9117/dl/example/?jackett_apikey=k&path=78 seeders=0 hash= ids=map[]`,
		}},
	} {
		t.Run(tc.file, func(t *testing.T) {
			items, err := indexers.ParseItems(fixture(t, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			got := make([]string, len(items))
			for i, it := range items {
				got[i] = fmt.Sprintf("%q %d %s seeders=%d hash=%s ids=%v", it.Title, it.Size, it.Link, it.Seeders, it.InfoHash, it.IDs)
			}
			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Fatalf("items =\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}

func TestQueryTakesTheFirstIDTheModeListsElseText(t *testing.T) {
	caps := indexers.Caps{Modes: map[string][]string{
		"search":   {"q"},
		"tvsearch": {"q", "season", "tvdbid", "imdbid"},
	}}
	ids := map[string]string{"imdbid": "tt0000901", "tvdbid": "901"}
	for _, tc := range []struct {
		name  string
		terms indexers.Terms
		want  string
	}{
		{"the first id the caps list", indexers.Terms{Mode: "tvsearch", IDs: ids, Query: "Lantern Coast"}, "map[t:tvsearch tvdbid:901]"},
		{"an id the type lacks is skipped", indexers.Terms{Mode: "tvsearch", IDs: map[string]string{"imdbid": "tt0000901"}, Query: "Lantern Coast"}, "map[imdbid:tt0000901 t:tvsearch]"},
		{"no id the caps list", indexers.Terms{Mode: "tvsearch", IDs: map[string]string{"tmdbid": "901"}, Query: "Lantern Coast"}, "map[q:Lantern Coast t:search]"},
		{"a mode the caps lack", indexers.Terms{Mode: "book", IDs: ids, Query: "Lantern Coast"}, "map[q:Lantern Coast t:search]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := fmt.Sprint(indexers.Query(caps, tc.terms)); got != tc.want {
				t.Fatalf("query = %s, want %s", got, tc.want)
			}
		})
	}
}
