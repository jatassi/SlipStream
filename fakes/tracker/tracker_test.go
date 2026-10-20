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
	ghost   = bencode.InfoHash{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	decoy   = bencode.InfoHash{}
)

const (
	seeder  = "-LT2000-seederseeder"
	leecher = "-LT2000-leecherleech"
)

// catalog holds orchard and ghost, a release the tracker deleted.
func catalog() *releases.Catalog {
	return &releases.Catalog{Passkey: passkey, Releases: []releases.Release{
		{Release: scenario.Release{Name: "orchard"}, InfoHash: orchard},
		{Release: scenario.Release{Name: "ghost", Unregistered: true}, InfoHash: ghost},
	}}
}

// fronts are the seeder fronts of orchard for two Nodes, and its bad peer
// front for one.
var fronts = fake.Fronts{
	{Target: tracker.SeedersTarget("orchard"), Node: "home", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:6001")}},
	{Target: tracker.SeedersTarget("orchard"), Node: "seedbox", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:6002")}},
	{Target: tracker.BadPeerTarget("orchard"), Node: "seedbox", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:6003")}},
}

type announce struct {
	// after is how long the announce waits after the one before, in the
	// in-memory tier.
	after   time.Duration
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
		time.Sleep(a.after)
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

var opts = []faketest.Option{faketest.WithReleases(catalog()), faketest.WithFronts(fronts), faketest.As("seedbox")}

func inBothTiers(t *testing.T, s scenario.Scenario, announces []announce, records []string) {
	t.Run("loopback", func(t *testing.T) {
		check(t, faketest.Loopback(t, tracker.Fake, s, opts...), announces, records)
	})
	t.Run("in memory", func(t *testing.T) { inMemory(t, s, announces, records) })
}

func inMemory(t *testing.T, s scenario.Scenario, announces []announce, records []string) {
	synctest.Test(t, func(t *testing.T) {
		check(t, faketest.InMemory(t, tracker.Fake, s, opts...), announces, records)
	})
}

func accepted(a announce) string { return record(a, "", "") }

func record(a announce, rule, failure string) string {
	return fmt.Sprintf(`tracker.announce seedbox {"passkey":%q,"infoHash":%q,"peerId":%q,"port":%d,"event":%q,"uploaded":0,"downloaded":0,"left":%d,"accepted":%t,"rule":%q,"failure":%q}`,
		a.passkey, a.hash, hex.EncodeToString([]byte(a.peer)), a.port, a.event, a.left, rule == "", rule, failure)
}

// failed is the reply to a failed announce, with interval and min interval
// in seconds unless they are empty.
func failed(reason, interval, minInterval string) string {
	timing := ""
	if interval != "" {
		timing = "8:intervali" + interval + "e12:min intervali" + minInterval + "e"
	}
	return fmt.Sprintf("200 OK d14:failure reason%d:%s%se", len(reason), reason, timing)
}

func TestALeecherGetsOnlyItsOwnSeederAndBadPeerFronts(t *testing.T) {
	s := scenario.Scenario{Seed: "tracker", Tracker: scenario.Tracker{Interval: 5 * time.Second}}
	none := "200 OK d8:intervali5e12:min intervali5e5:peers0:e"
	announces := []announce{
		{0, passkey, orchard, seeder, 6881, "started", 0, none},
		{0, passkey, orchard, leecher, 6882, "started", 4096, "200 OK d8:intervali5e12:min intervali5e5:peers12:\x7f\x00\x00\x01\x17\x72\x7f\x00\x00\x01\x17\x73e"},
		{0, passkey, orchard, leecher, 6882, "completed", 0, none},
		{0, passkey, orchard, leecher, 6882, "stopped", 0, none},
		{0, "wrongpasskey", orchard, leecher, 6882, "started", 4096, failed(tracker.FailurePasskey, "5", "5")},
		{0, passkey, decoy, leecher, 6882, "started", 4096, failed(tracker.FailureTorrent, "5", "5")},
	}
	records := []string{
		accepted(announces[0]),
		accepted(announces[1]),
		accepted(announces[2]),
		accepted(announces[3]),
		record(announces[4], tracker.RulePasskey, tracker.FailurePasskey),
		record(announces[5], tracker.RuleTorrent, tracker.FailureTorrent),
	}
	inBothTiers(t, s, announces, records)
}

func TestAScenarioWithoutAnIntervalAnswersThirtySeconds(t *testing.T) {
	a := announce{0, passkey, orchard, seeder, 6881, "started", 0, "200 OK d8:intervali30e12:min intervali30e5:peers0:e"}
	inBothTiers(t, scenario.Scenario{Seed: "tracker"}, []announce{a}, []string{accepted(a)})
}

var unit3dTracker = scenario.Tracker{Interval: 40 * time.Minute, MinInterval: 30 * time.Minute, Mode: scenario.UNIT3D}

const unit3dNone = "200 OK d8:intervali2400e12:min intervali1800e5:peers0:e"

func TestUNIT3DRejectsAStartedInsideTheWindow(t *testing.T) {
	s := scenario.Scenario{Seed: "tracker", Tracker: unit3dTracker}
	announces := []announce{
		{0, passkey, orchard, seeder, 6881, "started", 0, unit3dNone},
		{10 * time.Second, passkey, orchard, seeder, 6881, "started", 0, failed("You last announced 10 s ago. Please respect the min interval.", "2400", "1800")},
		{50 * time.Second, passkey, orchard, seeder, 6881, "stopped", 0, unit3dNone},
		{time.Minute, passkey, orchard, seeder, 6881, "started", 0, failed("You last announced 120 s ago. Please respect the min interval.", "2400", "1800")},
		{0, passkey, orchard, leecher, 6882, "started", 0, unit3dNone},
		{29 * time.Minute, passkey, orchard, seeder, 6881, "started", 0, unit3dNone},
	}
	records := []string{
		accepted(announces[0]),
		record(announces[1], "unit3d-duplicate", "You last announced 10 s ago. Please respect the min interval."),
		accepted(announces[2]),
		record(announces[3], "unit3d-window", "You last announced 120 s ago. Please respect the min interval."),
		accepted(announces[4]),
		accepted(announces[5]),
	}
	inMemory(t, s, announces, records)
}

func TestUNIT3DAcceptsCompletedAndStopped(t *testing.T) {
	s := scenario.Scenario{Seed: "tracker", Tracker: unit3dTracker}
	announces := []announce{
		{0, passkey, orchard, leecher, 6882, "started", 4096, "200 OK d8:intervali2400e12:min intervali1800e5:peers12:\x7f\x00\x00\x01\x17\x72\x7f\x00\x00\x01\x17\x73e"},
		{0, passkey, orchard, leecher, 6882, "completed", 0, unit3dNone},
		{0, passkey, orchard, leecher, 6882, "stopped", 0, unit3dNone},
	}
	inBothTiers(t, s, announces, []string{accepted(announces[0]), accepted(announces[1]), accepted(announces[2])})
}

func TestGazelleRejectsAnUnlistedPeerID(t *testing.T) {
	for name, c := range map[string]struct {
		allow    []string
		accepted string
		rejected string
	}{
		"libtorrent's by default": {nil, "-LT2000-seederseeder", "-SS1000-seederseeder"},
		"the listed prefixes":     {[]string{"-qB", "-TR"}, "-TR4000-seederseeder", "-LT2000-seederseeder"},
	} {
		t.Run(name, func(t *testing.T) {
			s := scenario.Scenario{Seed: "tracker", Tracker: scenario.Tracker{Interval: 5 * time.Second, Mode: scenario.Gazelle, AllowPeerIDs: c.allow}}
			announces := []announce{
				{0, passkey, orchard, c.accepted, 6881, "started", 0, "200 OK d8:intervali5e12:min intervali5e5:peers0:e"},
				{0, passkey, orchard, c.rejected, 6881, "started", 0, failed(tracker.FailurePeerID, "5", "5")},
			}
			inBothTiers(t, s, announces, []string{accepted(announces[0]), record(announces[1], tracker.RulePeerID, tracker.FailurePeerID)})
		})
	}
}

func TestAnUnregisteredTorrentFailsWithTheInterval(t *testing.T) {
	s := scenario.Scenario{Seed: "tracker", Tracker: scenario.Tracker{Interval: 5 * time.Second}}
	a := announce{0, passkey, ghost, seeder, 6881, "started", 0, failed(tracker.FailureUnregistered, "5", "5")}
	inBothTiers(t, s, []announce{a}, []string{record(a, tracker.RuleUnregistered, tracker.FailureUnregistered)})
}

func TestFailWithoutIntervalOmitsInterval(t *testing.T) {
	s := scenario.Scenario{Seed: "tracker", Tracker: scenario.Tracker{Interval: 5 * time.Second, FailWithoutInterval: true}}
	announces := []announce{
		{0, passkey, ghost, seeder, 6881, "started", 0, failed(tracker.FailureUnregistered, "", "")},
		{0, "wrongpasskey", orchard, seeder, 6881, "started", 0, failed(tracker.FailurePasskey, "", "")},
		{0, passkey, orchard, seeder, 6881, "started", 0, "200 OK d8:intervali5e12:min intervali5e5:peers0:e"},
	}
	records := []string{
		record(announces[0], tracker.RuleUnregistered, tracker.FailureUnregistered),
		record(announces[1], tracker.RulePasskey, tracker.FailurePasskey),
		accepted(announces[2]),
	}
	inBothTiers(t, s, announces, records)
}
