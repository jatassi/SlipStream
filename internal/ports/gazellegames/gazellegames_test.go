package gazellegames

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

func newIndexer(s gazelleGamesSettings) *indexer {
	base, _ := url.Parse("https://gazellegames.fixture-site.fake.localhost/")
	return &indexer{
		Settings:   s,
		BaseURL:    base,
		Categories: Port.Entry().Capabilities.Categories,
		Clock:      datetimeutil.Clock{Now: time.Now(), Location: time.UTC},
		Derived:    map[string]string{"passkey": "pk"},
	}
}

func TestPlatformsGoAsRepeatedArtistChecks(t *testing.T) {
	ix := newIndexer(gazelleGamesSettings{SearchGroupNames: true, FreeleechOnly: true})
	q := getRequest(ix, getBasicSearchParameters(ix, "The.Quiet.Orchard", port.SearchCriteria{Categories: []newznab.ID{newznab.PCGames.ID}})).URL.Query()
	if got, want := q["artistcheck[]"], []string{"DOS", "Windows"}; !slices.Equal(got, want) {
		t.Errorf("artistcheck[] = %q, want %q", got, want)
	}
	if q.Get("groupname") != "The Quiet Orchard" || q.Has("searchstr") || q.Get("freetorrent") != "1" {
		t.Errorf("query = %v, want groupname=The Quiet Orchard and freetorrent=1", q)
	}
}

func TestFreeTorrentReadsAsEnumTryParse(t *testing.T) {
	for value, want := range map[nsString]gazelleGamesFreeTorrent{
		"0": normal, "1": freeLeech, "2": neutral, "FREELEECH": freeLeech, "neutral": neutral,
		"FreeLeech, Neutral": either, "": normal, "gold": normal,
	} {
		if got := tryParseFreeTorrent(value); got != want {
			t.Errorf("tryParseFreeTorrent(%q) = %d, want %d", value, got, want)
		}
	}
}

func TestATitleGetsTheYearOnlyWhenItHasNone(t *testing.T) {
	year := nsInt(2021)
	for title, want := range map[nsString]string{
		"The Quiet Orchard":      "The Quiet Orchard (2021)",
		"The Quiet Orchard 2021": "The Quiet Orchard 2021",
		"The_Quiet_Orchard_2021": "The_Quiet_Orchard_2021 (2021)",
		"The Quiet Orchard-2101": "The Quiet Orchard-2101",
	} {
		if got := getTitle(gazelleGamesGroup{Year: &year}, gazelleGamesTorrent{ReleaseTitle: title}); got != want {
			t.Errorf("getTitle(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestFreeleechOnlyKeepsOnlyFreeleechTorrents(t *testing.T) {
	body := `{"status":"success","response":{"1":{"Artists":[{"name":"Windows"}],"Year":"2021","Torrents":{
		"10":{"ReleaseTitle":"The Quiet Orchard","FreeTorrent":"0","Size":"1","Time":"2026-06-01 12:00:00","TorrentType":"Torrent"},
		"11":{"ReleaseTitle":"The Quiet Orchard v2","FreeTorrent":"0","LowSeedFL":true,"Size":"1","Time":"2026-06-01 12:00:00","TorrentType":"Torrent"}}}}}`
	res := &port.IndexerResponse{StatusCode: http.StatusOK, Headers: http.Header{"Content-Type": {"application/json"}}, Body: []byte(body)}
	infos, err := parseResponse(newIndexer(gazelleGamesSettings{FreeleechOnly: true}), res)
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 || infos[0].Title != "The Quiet Orchard v2 (2021) [Windows]" {
		t.Errorf("releases = %+v, want only the low-seed freeleech one", infos)
	}
}
