package secretcinema

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/ports/internal/gazelle"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

const music = `{"status":"success","response":{"results":[{"groupId":3121,"groupName":"Abbey &amp; Road","artist":"The Band","groupYear":1969,"torrents":[
	{"torrentId":51010,"media":"CD","format":"FLAC","encoding":"Lossless","hasCue":true,"time":"2024-11-04 12:00:00","size":335544320,"snatches":5,"seeders":6,"leechers":0,"category":"Music"}]}]}}`

func TestAMusicTorrentIsTitledByArtistGroupAndFormat(t *testing.T) {
	base, _ := url.Parse("https://secretcinema.test/")
	ix := &ix{
		Settings:   gazelle.Settings{UseFreeleechToken: gazelle.Preferred},
		BaseURL:    base,
		Categories: setCapabilities().Categories,
		Clock:      datetimeutil.Clock{Now: time.Now(), Location: time.UTC},
	}
	res := &port.IndexerResponse{StatusCode: http.StatusOK, Headers: http.Header{"Content-Type": {"application/json"}}, Body: []byte(music)}

	infos, err := parseResponse(ix, res)
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 {
		t.Fatalf("got %d releases, want 1", len(infos))
	}
	r := infos[0]
	if want := "The Band - Abbey & Road (1969) [FLAC Lossless] [CD] [Cue]"; r.Title != want {
		t.Errorf("title = %q, want %q", r.Title, want)
	}
	if want := time.Date(2024, 11, 4, 10, 0, 0, 0, time.UTC); !r.PublishDate.Equal(want) {
		t.Errorf("published = %v, want %v", r.PublishDate, want)
	}
	if want := "https://secretcinema.test/torrents.php?action=download&id=51010&useToken=1"; r.DownloadURL != want {
		t.Errorf("link = %q, want %q", r.DownloadURL, want)
	}
}
