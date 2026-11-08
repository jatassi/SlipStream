package gazelle

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

type testSettings struct{ Settings }

func indexer(token GazelleFreeleechTokenAction) *port.Indexer[testSettings] {
	base, _ := url.Parse("https://gazelle.test/")
	return &port.Indexer[testSettings]{
		Settings:   testSettings{Settings{UseFreeleechToken: token}},
		BaseURL:    base,
		Categories: port.Categories(port.Map(1, newznab.Audio, "Music"), port.Map(2, newznab.PC, "Applications")),
		Clock:      datetimeutil.Clock{Now: time.Now(), Location: time.UTC},
	}
}

func parse(t *testing.T, ix *port.Indexer[testSettings], body string) []port.TorrentInfo {
	t.Helper()
	p := parser[testSettings]{site: Site[testSettings]{GetDownloadURL: GetDownloadURL[testSettings]}}
	res := &port.IndexerResponse{StatusCode: http.StatusOK, Headers: http.Header{"Content-Type": {"application/json"}}, Body: []byte(body)}
	infos, err := p.parseResponse(ix, res)
	if err != nil {
		t.Fatal(err)
	}
	return infos
}

const grouped = `{"status":"success","response":{"results":[{"groupId":466,"groupName":"Abbey &amp; Road","artist":"The Band","groupYear":1969,"torrents":[
	{"torrentId":1,"media":"CD","format":"FLAC","encoding":"Lossless","hasCue":true,"scene":true,"time":"2016-11-30 22:05:03","size":979581415,"snatches":35,"seeders":13,"leechers":2,"category":"Music","canUseToken":true},
	{"torrentId":2,"media":"WEB","format":"MP3","encoding":"320","time":"2016-12-01 10:00:00","size":"1000","seeders":"1","leechers":"0","canUseToken":false},
	{"torrentId":3,"media":"WEB","format":"MP3","encoding":"V0","time":"2016-12-02 10:00:00","size":1000,"seeders":1,"leechers":0,"isNeutralLeech":true,"canUseToken":false}]}]}}`

func TestAGroupsTorrentsAreTitledByArtistGroupAndFormat(t *testing.T) {
	infos := parse(t, indexer(Never), grouped)
	if len(infos) != 3 {
		t.Fatalf("got %d releases, want 3", len(infos))
	}
	cd := infos[2]
	if want := "The Band - Abbey & Road (1969) [FLAC Lossless] [CD] [Cue]"; cd.Title != want {
		t.Errorf("title = %q, want %q", cd.Title, want)
	}
	if want := time.Date(2016, 11, 30, 22, 5, 3, 0, time.UTC); !cd.PublishDate.Equal(want) {
		t.Errorf("published = %v, want %v", cd.PublishDate, want)
	}
	if cd.Size != 979581415 || cd.Seeders != 13 || cd.Peers != 15 || cd.Grabs != 35 {
		t.Errorf("size, seeders, peers, grabs = %d, %d, %d, %d", cd.Size, cd.Seeders, cd.Peers, cd.Grabs)
	}
	if len(cd.IndexerFlags) != 1 || cd.IndexerFlags[0] != port.Scene {
		t.Errorf("flags = %v, want scene", cd.IndexerFlags)
	}
	if cd.InfoURL != "https://gazelle.test/torrents.php?id=466&torrentid=1" {
		t.Errorf("details = %q", cd.InfoURL)
	}
	if neutral := infos[0]; neutral.DownloadVolumeFactor != 0 || neutral.UploadVolumeFactor != 0 {
		t.Errorf("a neutral leech's factors = %v, %v, want 0, 0", neutral.DownloadVolumeFactor, neutral.UploadVolumeFactor)
	}
}

func TestRequiredTokensListOnlyTorrentsThatTakeATokenOrAreFree(t *testing.T) {
	infos := parse(t, indexer(Required), grouped)
	var ids []string
	for _, i := range infos {
		ids = append(ids, i.DownloadURL)
	}
	want := []string{
		"https://gazelle.test/torrents.php?action=download&id=3",
		"https://gazelle.test/torrents.php?action=download&id=1&usetoken=1",
	}
	if len(ids) != len(want) || ids[0] != want[0] || ids[1] != want[1] {
		t.Errorf("download links = %v, want %v", ids, want)
	}
}

func TestASearchFiltersByTrackerCategoryAndSpacesDots(t *testing.T) {
	parameters := GetBasicSearchParameters(indexer(Never), port.SearchCriteria{Categories: []newznab.ID{newznab.PC.ID}}, "Abbey.Road")
	if got, want := parameters.GetQueryString(), "action=browse&order_by=time&order_way=desc&searchstr=Abbey+Road&filter_cat[2]=1"; got != want {
		t.Errorf("query = %q, want %q", got, want)
	}
}

func TestAnErrorStatusQuotesTheSitesError(t *testing.T) {
	p := parser[testSettings]{site: Site[testSettings]{GetDownloadURL: GetDownloadURL[testSettings]}}
	res := &port.IndexerResponse{StatusCode: http.StatusBadRequest, Body: []byte(`{"status":"failure","error":"no such torrent"}`)}
	_, err := p.parseResponse(indexer(Never), res)
	if _, ok := errors.AsType[*port.SiteError](err); !ok || !strings.Contains(err.Error(), `"no such torrent"`) {
		t.Errorf("err = %v, want a site error that quotes the site's", err)
	}
}

func TestASignInRedirectIsAcceptedAndItsFormAgainIsRejected(t *testing.T) {
	for _, c := range []struct {
		name     string
		response port.IndexerResponse
		rejected bool
	}{
		{"redirect", port.IndexerResponse{StatusCode: http.StatusFound, HasHTTPRedirect: true}, false},
		{"form again", port.IndexerResponse{StatusCode: http.StatusOK}, true},
	} {
		err := checkForLoginError(&c.response)
		_, isAuth := errors.AsType[*port.AuthError](err)
		if isAuth != c.rejected || !c.rejected && err != nil {
			t.Errorf("%s: err = %v, want rejected %v", c.name, err, c.rejected)
		}
	}
}
