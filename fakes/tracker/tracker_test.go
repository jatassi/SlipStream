package tracker_test

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
)

const passkey = "0123456789abcdef0123456789abcdef"

var (
	orchard = bencode.InfoHash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	decoy   = bencode.InfoHash{}
)

func catalog() *releases.Catalog {
	return &releases.Catalog{Passkey: passkey, Releases: []releases.Release{{Release: scenario.Release{Name: "orchard"}, InfoHash: orchard}}}
}

// fronts are the seeder fronts of orchard for two Nodes.
var fronts = fake.Fronts{
	{Target: tracker.SeedersTarget("orchard"), Node: "home", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:6001")}},
	{Target: tracker.SeedersTarget("orchard"), Node: "seedbox", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:6002")}},
}

type announce struct {
	passkey string
	hash    bencode.InfoHash
	peer    string
	port    int
	event   string
	left    int64
	want    string
}

func get(t *testing.T, f faketest.Served, path string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.Address.URL(path), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.Status + " " + string(body)
}

func check(t *testing.T, f faketest.Served, announces []announce, records []string) {
	t.Helper()
	for _, a := range announces {
		q := url.Values{
			"info_hash":  {string(a.hash[:])},
			"peer_id":    {a.peer},
			"port":       {strconv.Itoa(a.port)},
			"uploaded":   {"0"},
			"downloaded": {"0"},
			"left":       {strconv.FormatInt(a.left, 10)},
			"compact":    {"1"},
		}
		if a.event != "" {
			q.Set("event", a.event)
		}
		if got := get(t, f, "/announce/"+a.passkey+"?"+q.Encode()); got != a.want {
			t.Errorf("announce %s %s = %q, want %q", a.peer, a.event, got, a.want)
		}
	}
	var got []string
	for _, r := range f.Records(t) {
		got = append(got, r.Kind+" "+r.Node+" "+string(r.Data))
	}
	if !slices.Equal(got, records) {
		t.Errorf("records =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(records, "\n"))
	}
}

func inBothTiers(t *testing.T, s scenario.Scenario, announces []announce, records []string) {
	opts := []faketest.Option{faketest.WithReleases(catalog()), faketest.WithFronts(fronts), faketest.As("seedbox")}
	t.Run("loopback", func(t *testing.T) {
		check(t, faketest.Loopback(t, tracker.Fake, s, opts...), announces, records)
	})
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			check(t, faketest.InMemory(t, tracker.Fake, s, opts...), announces, records)
		})
	})
}

func record(a announce, failure string) string {
	return fmt.Sprintf(`tracker.announce seedbox {"passkey":%q,"infoHash":%q,"peerId":%q,"port":%d,"event":%q,"uploaded":0,"downloaded":0,"left":%d,"failure":%q}`,
		a.passkey, a.hash, hex.EncodeToString([]byte(a.peer)), a.port, a.event, a.left, failure)
}

func TestALeecherGetsOnlyItsOwnSeederFront(t *testing.T) {
	seeder, leecher := "-LT2000-seederseeder", "-LT2000-leecherleech"
	s := scenario.Scenario{Seed: "tracker", Tracker: scenario.Tracker{Interval: 5 * time.Second}}
	none := "200 OK d8:intervali5e12:min intervali5e5:peers0:e"
	announces := []announce{
		{passkey, orchard, seeder, 6881, "started", 0, none},
		{passkey, orchard, leecher, 6882, "started", 4096, "200 OK d8:intervali5e12:min intervali5e5:peers6:\x7f\x00\x00\x01\x17\x72e"},
		{passkey, orchard, leecher, 6882, "completed", 0, none},
		{passkey, orchard, leecher, 6882, "stopped", 0, none},
		{"wrongpasskey", orchard, leecher, 6882, "started", 4096, "200 OK d14:failure reason22:passkey not recognizede"},
		{passkey, decoy, leecher, 6882, "started", 4096, "200 OK d14:failure reason40:torrent not registered with this trackere"},
	}
	var records []string
	for i, a := range announces {
		records = append(records, record(a, []string{4: tracker.FailurePasskey, 5: tracker.FailureTorrent}[i]))
	}
	inBothTiers(t, s, announces, records)
}

func TestAScenarioWithoutAnIntervalAnswersThirtySeconds(t *testing.T) {
	a := announce{passkey, orchard, "-LT2000-seederseeder", 6881, "started", 0, "200 OK d8:intervali30e12:min intervali30e5:peers0:e"}
	inBothTiers(t, scenario.Scenario{Seed: "tracker"}, []announce{a}, []string{record(a, "")})
}
