package tracker_test

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

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
	return &releases.Catalog{Passkey: passkey, Releases: []releases.Release{{InfoHash: orchard}}}
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

func check(t *testing.T, f faketest.Served, announces []announce, records string) {
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
	if got := get(t, f, "/records"); got != "200 OK "+records {
		t.Errorf("records =\n%s\nwant\n200 OK %s", got, records)
	}
}

func inBothTiers(t *testing.T, s scenario.Scenario, announces []announce, records string) {
	t.Run("loopback", func(t *testing.T) {
		check(t, faketest.Loopback(t, tracker.Fake, s, faketest.WithReleases(catalog())), announces, records)
	})
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			check(t, faketest.InMemory(t, tracker.Fake, s, faketest.WithReleases(catalog())), announces, records)
		})
	})
}

func TestTheTrackerAnswersEveryOtherPeer(t *testing.T) {
	seeder, leecher := "-LT2000-seederseeder", "-LT2000-leecherleech"
	s := scenario.Scenario{Seed: "tracker", Tracker: scenario.Tracker{Interval: 5 * time.Second}}
	inBothTiers(t, s, []announce{
		{passkey, orchard, seeder, 6881, "started", 0, "200 OK d8:intervali5e12:min intervali5e5:peers0:e"},
		{passkey, orchard, leecher, 6882, "started", 4096, "200 OK d8:intervali5e12:min intervali5e5:peers6:\x7f\x00\x00\x01\x1a\xe1e"},
		{passkey, orchard, seeder, 6881, "", 0, "200 OK d8:intervali5e12:min intervali5e5:peers6:\x7f\x00\x00\x01\x1a\xe2e"},
		{passkey, orchard, leecher, 6882, "stopped", 0, "200 OK d8:intervali5e12:min intervali5e5:peers6:\x7f\x00\x00\x01\x1a\xe1e"},
		{passkey, orchard, seeder, 6881, "", 0, "200 OK d8:intervali5e12:min intervali5e5:peers0:e"},
		{"wrongpasskey", orchard, leecher, 6882, "started", 4096, "200 OK d14:failure reason22:passkey not recognizede"},
		{passkey, decoy, leecher, 6882, "started", 4096, "200 OK d14:failure reason40:torrent not registered with this trackere"},
	}, `[`+
		`{"passkey":"0123456789abcdef0123456789abcdef","infoHash":"0102030405060708090a0b0c0d0e0f1011121314","peerId":"2d4c54323030302d736565646572736565646572","source":"127.0.0.1","port":6881,"event":"started","uploaded":0,"downloaded":0,"left":0,"failure":""},`+
		`{"passkey":"0123456789abcdef0123456789abcdef","infoHash":"0102030405060708090a0b0c0d0e0f1011121314","peerId":"2d4c54323030302d6c6565636865726c65656368","source":"127.0.0.1","port":6882,"event":"started","uploaded":0,"downloaded":0,"left":4096,"failure":""},`+
		`{"passkey":"0123456789abcdef0123456789abcdef","infoHash":"0102030405060708090a0b0c0d0e0f1011121314","peerId":"2d4c54323030302d736565646572736565646572","source":"127.0.0.1","port":6881,"event":"","uploaded":0,"downloaded":0,"left":0,"failure":""},`+
		`{"passkey":"0123456789abcdef0123456789abcdef","infoHash":"0102030405060708090a0b0c0d0e0f1011121314","peerId":"2d4c54323030302d6c6565636865726c65656368","source":"127.0.0.1","port":6882,"event":"stopped","uploaded":0,"downloaded":0,"left":0,"failure":""},`+
		`{"passkey":"0123456789abcdef0123456789abcdef","infoHash":"0102030405060708090a0b0c0d0e0f1011121314","peerId":"2d4c54323030302d736565646572736565646572","source":"127.0.0.1","port":6881,"event":"","uploaded":0,"downloaded":0,"left":0,"failure":""},`+
		`{"passkey":"wrongpasskey","infoHash":"0102030405060708090a0b0c0d0e0f1011121314","peerId":"2d4c54323030302d6c6565636865726c65656368","source":"127.0.0.1","port":6882,"event":"started","uploaded":0,"downloaded":0,"left":4096,"failure":"passkey not recognized"},`+
		`{"passkey":"0123456789abcdef0123456789abcdef","infoHash":"0000000000000000000000000000000000000000","peerId":"2d4c54323030302d6c6565636865726c65656368","source":"127.0.0.1","port":6882,"event":"started","uploaded":0,"downloaded":0,"left":4096,"failure":"torrent not registered with this tracker"}`+
		`]`)
}

func TestAScenarioWithoutAnIntervalAnswersThirtySeconds(t *testing.T) {
	inBothTiers(t, scenario.Scenario{Seed: "tracker"}, []announce{
		{passkey, orchard, "-LT2000-seederseeder", 6881, "started", 0, "200 OK d8:intervali30e12:min intervali30e5:peers0:e"},
	}, `[{"passkey":"0123456789abcdef0123456789abcdef","infoHash":"0102030405060708090a0b0c0d0e0f1011121314","peerId":"2d4c54323030302d736565646572736565646572","source":"127.0.0.1","port":6881,"event":"started","uploaded":0,"downloaded":0,"left":0,"failure":""}]`)
}
