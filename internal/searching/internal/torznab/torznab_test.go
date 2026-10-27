package torznab_test

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/searching/internal/torznab"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCapsKeepEachAvailableModeItsParametersAndTheCategories(t *testing.T) {
	for _, tc := range []struct{ file, want string }{
		{"prowlarr-caps.xml", `map[movie:[q imdbid tmdbid] search:[q] tvsearch:[q season ep imdbid tvdbid tmdbid]] [2000 2040]`},
		{"jackett-caps.xml", `map[book:[q] movie:[q imdbid] search:[q] tvsearch:[q season ep]] [2000]`},
	} {
		t.Run(tc.file, func(t *testing.T) {
			c, err := torznab.ParseCaps(fixture(t, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprint(c.Modes, " ", c.Categories); got != tc.want {
				t.Fatalf("caps = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestAnErrorDocumentIsNoCapsAndReadsAsAnError(t *testing.T) {
	_, err := torznab.ParseCaps(fixture(t, "error.xml"))
	if want := "read the caps: the document is <error>, not <caps>"; err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %s", err, want)
	}
	if e, ok := torznab.ErrorOf(fixture(t, "error.xml")); !ok || e != (torznab.Error{Code: 100, Description: "Invalid API Key"}) {
		t.Fatalf("ErrorOf = %+v, %v; want code 100", e, ok)
	}
	if _, ok := torznab.ErrorOf(fixture(t, "prowlarr-results.xml")); ok {
		t.Fatal("a results feed read as an error")
	}
}

func show(it torznab.Item) string {
	hash := ""
	if it.InfoHash != nil {
		hash = it.InfoHash.String()
	}
	source := ""
	if s := it.Source; s != nil {
		source = fmt.Sprintf(" source=%s/%s/%s/%s", s.Origin, s.ID, s.Name, s.Privacy)
	}
	return fmt.Sprintf("%q %d %s link=%s magnet=%s seeders=%d leechers=%d grabs=%d hash=%s cats=%v ids=%v factors=%v/%v floor=%v/%v details=%s%s",
		it.Title, it.Size, it.Published.Format("2006-01-02T15:04Z"), it.Link, it.Magnet, it.Seeders, it.Leechers, it.Grabs, hash, it.Categories, it.IDs,
		it.DownloadFactor, it.UploadFactor, it.MinRatio, it.MinSeed, it.Details, source)
}

func TestItemsKeepEveryAttributeWithProwlarrsDefaults(t *testing.T) {
	for _, tc := range []struct {
		file string
		want []string
	}{
		{"prowlarr-results.xml", []string{
			`"Night.Harbor.2019.1080p.BluRay.x264-GRP" 8589934592 2020-02-03T10:00Z link=https://prowlarr.example.test/1/download?apikey=k&id=41 magnet= seeders=57 leechers=3 grabs=120 hash=0123456789abcdef0123456789abcdef01234567 cats=[2040] ids=map[imdbid:tt0000501 tmdbid:501] factors=0.5/2 floor=1.5/48h0m0s details=https://tracker.example.test/details/41 source=prowlarr/1/Example Tracker/private`,
			`"Night.Harbor.2019.720p.WEB.h264-OTHER" 2147483648 2020-02-04T10:00Z link=https://prowlarr.example.test/1/download?apikey=k&id=42 magnet= seeders=3 leechers=0 grabs=0 hash= cats=[] ids=map[] factors=1/1 floor=0/0s details=`,
		}},
		{"jackett-results.xml", []string{
			`"Night Harbor 2019 2160p UHD BluRay" 53687091200 2020-02-05T10:00Z link=http://jackett.example.test:9117/dl/example/?jackett_apikey=k&path=77&file=Night+Harbor magnet= seeders=9 leechers=1 grabs=12 hash=fedcba9876543210fedcba9876543210fedcba98 cats=[2000] ids=map[imdbid:tt0000501] factors=0/1 floor=0/0s details=https://tracker.example.test/details/77 source=jackett/example/Example Tracker/private`,
			`"Night Harbor 2019 Sample" 1048576 2020-02-05T11:00Z link=http://jackett.example.test:9117/dl/example/?jackett_apikey=k&path=78 magnet= seeders=0 leechers=0 grabs=0 hash= cats=[2000] ids=map[] factors=1/1 floor=0/0s details=`,
		}},
		{"magnet-results.xml", []string{
			`"Night Harbor 2019 1080p Magnet Enclosure" 1000 0001-01-01T00:00Z link= magnet=magnet:?xt=urn:btih:0123456789ABCDEF0123456789ABCDEF01234567&dn=Night+Harbor seeders=4 leechers=0 grabs=0 hash=0123456789abcdef0123456789abcdef01234567 cats=[] ids=map[] factors=1/1 floor=0/0s details=`,
			`"Night Harbor 2019 720p Magnet Link" 2000 0001-01-01T00:00Z link= magnet=magnet:?xt=urn:btih:AERUKZ4JVPG66AJDIVSYTK6N54ASGRLH seeders=0 leechers=0 grabs=0 hash=0123456789abcdef0123456589abcdef01234567 cats=[] ids=map[] factors=1/1 floor=0/0s details=`,
			`"Night Harbor 2019 480p Magnet Attr" 3000 0001-01-01T00:00Z link= magnet=magnet:?xt=urn:btih:fedcba9876543210fedcba9876543210fedcba98&tr=udp%3A%2F%2Fopen.example.test%3A6969 seeders=0 leechers=0 grabs=0 hash=fedcba9876543210fedcba9876543210fedcba98 cats=[] ids=map[] factors=1/1 floor=0/0s details=`,
			`"Night Harbor 2019 2160p Torrent And Magnet" 4000 0001-01-01T00:00Z link=https://prowlarr.example.test/1/download?apikey=k&id=43 magnet=magnet:?xt=urn:btih:00112233445566778899aabbccddeeff00112233 seeders=0 leechers=0 grabs=0 hash=00112233445566778899aabbccddeeff00112233 cats=[] ids=map[] factors=1/1 floor=0/0s details=`,
			`"Night Harbor 2019 Remux Hash Attr" 5000 0001-01-01T00:00Z link= magnet=magnet:?xt=urn:btih:00112233445566778899aabbccddeeff00112233 seeders=0 leechers=0 grabs=0 hash=aaaabbbbccccddddeeeeffff0000111122223333 cats=[] ids=map[] factors=1/1 floor=0/0s details=`,
		}},
	} {
		t.Run(tc.file, func(t *testing.T) {
			items, err := torznab.ParseFeed(fixture(t, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			got := make([]string, len(items))
			for i, it := range items {
				got[i] = show(it)
			}
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Fatalf("items =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tc.want, "\n"))
			}
		})
	}
}

func TestAPageCarriesTheQueryExtendedAndItsWindow(t *testing.T) {
	base, err := url.Parse("https://feed.example.test/1")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		q    torznab.Query
		want string
	}{
		{"ids", torznab.Query{Mode: "tvsearch", IDs: map[string]string{"tvdbid": "501", "imdbid": "tt0000501"}}, "apikey=k&extended=1&imdbid=tt0000501&limit=100&offset=200&t=tvsearch&tvdbid=501"},
		{"text, an episode and categories", torznab.Query{Mode: "tvsearch", Text: "Lantern Coast", Season: 2, Episode: "5", Categories: []int{5000, 5040}}, "apikey=k&cat=5000%2C5040&ep=5&extended=1&limit=100&offset=200&q=Lantern+Coast&season=2&t=tvsearch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := torznab.API(base, "k", tc.q.Page(200, 100))
			if u.Path != "/1/api" || u.RawQuery != tc.want {
				t.Fatalf("api = %s ? %s, want /1/api ? %s", u.Path, u.RawQuery, tc.want)
			}
		})
	}
}
