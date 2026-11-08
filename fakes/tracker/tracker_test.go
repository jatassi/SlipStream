package tracker_test

import (
	"encoding/hex"
	"encoding/json"
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

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
)

var passkey = tracker.Passkey(scenario.Scenario{Seed: "tracker"}, "torznab")

var (
	orchard = bencode.InfoHash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	ghost   = bencode.InfoHash{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	decoy   = bencode.InfoHash{}
)

// size is orchard's payload size.
const size = 1 << 20

const (
	seeder  = "-LT2000-seederseeder"
	leecher = "-LT2000-leecherleech"
)

// catalog holds orchard and ghost, a release the tracker deleted.
func catalog() *releases.Catalog {
	return &releases.Catalog{Releases: []releases.Release{
		{Release: scenario.Release{Name: "orchard"}, InfoHash: orchard, Size: size},
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
		if r.Kind == tracker.Announces.Kind {
			got = append(got, r.Kind+" "+r.Node+" "+string(r.Data))
		}
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

// replied is the reply to an accepted announce: the swarm's seeders,
// leechers and completed downloads, interval and min interval in seconds,
// and the compact peers.
func replied(complete, incomplete, downloaded int, interval, minInterval, peers string) string {
	return fmt.Sprintf("200 OK d8:completei%de10:downloadedi%de10:incompletei%de8:intervali%se12:min intervali%se5:peers%d:%se",
		complete, downloaded, incomplete, interval, minInterval, len(peers), peers)
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
	announces := []announce{
		{0, passkey, orchard, seeder, 6881, "started", 0, replied(1, 0, 0, "5", "5", "")},
		{0, passkey, orchard, leecher, 6882, "started", 4096, replied(1, 1, 0, "5", "5", "\x7f\x00\x00\x01\x17\x72\x7f\x00\x00\x01\x17\x73")},
		{0, passkey, orchard, leecher, 6882, "completed", 0, replied(2, 0, 1, "5", "5", "")},
		{0, passkey, orchard, leecher, 6882, "stopped", 0, replied(1, 0, 1, "5", "5", "")},
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
	a := announce{0, passkey, orchard, seeder, 6881, "started", 0, replied(1, 0, 0, "30", "30", "")}
	inBothTiers(t, scenario.Scenario{Seed: "tracker"}, []announce{a}, []string{accepted(a)})
}

var unit3dTracker = scenario.Tracker{Interval: 40 * time.Minute, MinInterval: 30 * time.Minute, Mode: scenario.UNIT3D}

func TestUNIT3DRejectsAStartedInsideTheWindow(t *testing.T) {
	s := scenario.Scenario{Seed: "tracker", Tracker: unit3dTracker}
	announces := []announce{
		{0, passkey, orchard, seeder, 6881, "started", 0, replied(1, 0, 0, "2400", "1800", "")},
		{10 * time.Second, passkey, orchard, seeder, 6881, "started", 0, failed("You last announced 10 s ago. Please respect the min interval.", "2400", "1800")},
		{50 * time.Second, passkey, orchard, seeder, 6881, "stopped", 0, replied(0, 0, 0, "2400", "1800", "")},
		{time.Minute, passkey, orchard, seeder, 6881, "started", 0, failed("You last announced 120 s ago. Please respect the min interval.", "2400", "1800")},
		{0, passkey, orchard, leecher, 6882, "started", 0, replied(1, 0, 0, "2400", "1800", "")},
		{29 * time.Minute, passkey, orchard, seeder, 6881, "started", 0, replied(2, 0, 0, "2400", "1800", "")},
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
		{0, passkey, orchard, leecher, 6882, "started", 4096, replied(0, 1, 0, "2400", "1800", "\x7f\x00\x00\x01\x17\x72\x7f\x00\x00\x01\x17\x73")},
		{0, passkey, orchard, leecher, 6882, "completed", 0, replied(1, 0, 1, "2400", "1800", "")},
		{0, passkey, orchard, leecher, 6882, "stopped", 0, replied(0, 0, 1, "2400", "1800", "")},
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
				{0, passkey, orchard, c.accepted, 6881, "started", 0, replied(1, 0, 0, "5", "5", "")},
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
		{0, passkey, orchard, seeder, 6881, "started", 0, replied(1, 0, 0, "5", "5", "")},
	}
	records := []string{
		record(announces[0], tracker.RuleUnregistered, tracker.FailureUnregistered),
		record(announces[1], tracker.RulePasskey, tracker.FailurePasskey),
		accepted(announces[2]),
	}
	inBothTiers(t, s, announces, records)
}

// send is one announce of orchard by account's passkey, after the one
// before in the in-memory tier.
type send struct {
	after      time.Duration
	account    string
	peer       string
	event      string
	left       int64
	downloaded int64
	uploaded   int64
}

func (a send) query() string {
	q := url.Values{
		"info_hash":  {string(orchard[:])},
		"peer_id":    {a.peer},
		"port":       {"6881"},
		"uploaded":   {strconv.FormatInt(a.uploaded, 10)},
		"downloaded": {strconv.FormatInt(a.downloaded, 10)},
		"left":       {strconv.FormatInt(a.left, 10)},
		"compact":    {"1"},
	}
	if a.event != "" {
		q.Set("event", a.event)
	}
	return "/announce/" + tracker.Passkey(scenario.Scenario{Seed: "tracker"}, a.account) + "?" + q.Encode()
}

// seedingRecords plays sends with tracker tr in memory and waits for
// then, and returns each seeding record as the time since the first send,
// the account, the seed time and the flags that are set, and each reply.
func seedingRecords(t *testing.T, cat *releases.Catalog, tr scenario.Tracker, sends []send, then time.Duration) (records, replies []string) {
	synctest.Test(t, func(t *testing.T) {
		f := faketest.InMemory(t, tracker.Fake, scenario.Scenario{Seed: "tracker", Tracker: tr}, faketest.WithReleases(cat), faketest.As("seedbox"))
		start := time.Now()
		for _, a := range sends {
			time.Sleep(a.after)
			replies = append(replies, get(t, f, a.query()))
		}
		time.Sleep(then)
		synctest.Wait()
		for _, r := range f.Records(t) {
			if r.Kind == tracker.Seeding.Kind {
				records = append(records, seedingLine(t, r, start))
			}
		}
	})
	return records, replies
}

// seedingLine is r as the time since start, the account, the seed time
// and the flags that are set.
func seedingLine(t *testing.T, r events.Record, start time.Time) string {
	t.Helper()
	var h tracker.History
	if err := json.Unmarshal(r.Data, &h); err != nil {
		t.Fatal(err)
	}
	if h.Release != "orchard" || h.InfoHash != orchard.String() || r.Node != "seedbox" {
		t.Fatalf("seeding record %s %s, want orchard's from seedbox", r.Node, r.Data)
	}
	fields := []string{r.At.Sub(start).String(), h.Account, strconv.FormatInt(h.SeedSeconds, 10) + "s"}
	for _, flag := range []struct {
		name string
		set  bool
	}{{"seeder", h.Seeder}, {"active", h.Active}, {"owes", h.Owes}, {"prewarned", h.Prewarned}, {"hit-and-run", h.HitAndRun}} {
		if flag.set {
			fields = append(fields, flag.name)
		}
	}
	return strings.Join(fields, " ")
}

func expectRecords(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("seeding records =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestSeedTimeCountsOnlyGapsUnderNinetyMinutes(t *testing.T) {
	sends := []send{
		{0, "heron", seeder, "started", 0, 0, 0},
		{30 * time.Minute, "heron", seeder, "", 0, 0, 0},
		{time.Hour, "heron", seeder, "", 0, 0, 0},
		{90 * time.Minute, "heron", seeder, "", 0, 0, 0},
		{89 * time.Minute, "heron", seeder, "", 0, 0, 0},
	}
	got, _ := seedingRecords(t, catalog(), scenario.Tracker{}, sends, 0)
	expectRecords(t, got, []string{
		"0s heron 0s seeder active",
		"30m0s heron 1800s seeder active",
		"1h30m0s heron 5400s seeder active",
		"4h29m0s heron 10740s seeder active",
	})
}

func TestSeedTimeSkipsLeechingAndTheGapAcrossAStopAndStart(t *testing.T) {
	sends := []send{
		{0, "heron", leecher, "started", size, 0, 0},
		{10 * time.Minute, "heron", leecher, "completed", 0, size, 0},
		{10 * time.Minute, "heron", leecher, "", 0, size, 0},
		{10 * time.Minute, "heron", leecher, "stopped", 0, size, 0},
		{10 * time.Minute, "heron", leecher, "started", 0, 0, 0},
		{10 * time.Minute, "heron", leecher, "", 0, 0, 0},
	}
	got, _ := seedingRecords(t, catalog(), scenario.Tracker{}, sends, 0)
	expectRecords(t, got, []string{
		"0s heron 0s active",
		"10m0s heron 0s seeder active owes",
		"20m0s heron 600s seeder active owes",
		"30m0s heron 1200s seeder owes",
		"40m0s heron 1200s seeder active owes",
		"50m0s heron 1800s seeder active owes",
	})
}

func TestAPeerThatStopsAnnouncingWhileItOwesIsPrewarnedThenAHitAndRun(t *testing.T) {
	tr := scenario.Tracker{Prewarn: 2 * time.Minute, Grace: 5 * time.Minute, InactiveAfter: 90 * time.Second, Sites: map[string]scenario.SiteRule{
		"heron": {MinSeedTime: 3 * time.Minute}, "tamarack": {MinRatio: 1, MinSeedTime: 3 * time.Minute},
	}}
	uploadedItAll := func(account string) []send {
		return []send{
			{0, account, leecher, "started", size, 0, 0},
			{time.Minute, account, leecher, "completed", 0, size, 0},
			{time.Minute, account, leecher, "", 0, size, size},
		}
	}
	t.Run("heron, where ratio doesn't count", func(t *testing.T) {
		got, _ := seedingRecords(t, catalog(), tr, uploadedItAll("heron"), 10*time.Minute)
		expectRecords(t, got, []string{
			"0s heron 0s active",
			"1m0s heron 0s seeder active owes",
			"2m0s heron 60s seeder active owes",
			"3m30s heron 60s seeder owes",
			"4m0s heron 60s seeder owes prewarned",
			"7m0s heron 60s seeder owes prewarned hit-and-run",
		})
	})
	t.Run("tamarack, where 1:1 meets the rule", func(t *testing.T) {
		got, _ := seedingRecords(t, catalog(), tr, uploadedItAll("tamarack"), 10*time.Minute)
		expectRecords(t, got, []string{
			"0s tamarack 0s active",
			"1m0s tamarack 0s seeder active owes",
			"2m0s tamarack 60s seeder active",
			"3m30s tamarack 60s seeder",
		})
	})
}

func TestTheSeedersTimeLandsOnNoAccount(t *testing.T) {
	sends := []send{
		{0, releases.Seeders, seeder, "started", 0, 0, 0},
		{30 * time.Minute, releases.Seeders, seeder, "", 0, 0, 0},
		{0, "heron", leecher, "started", 0, 0, 0},
		{30 * time.Minute, "heron", leecher, "", 0, 0, 0},
		{0, "tamarack", "-LT2000-othertamarack", "started", 0, 0, 0},
	}
	got, _ := seedingRecords(t, catalog(), scenario.Tracker{}, sends, 0)
	expectRecords(t, got, []string{
		"30m0s heron 0s seeder active",
		"1h0m0s heron 1800s seeder active",
		"1h0m0s tamarack 0s seeder active",
	})
}

func TestRepliesCountTheSwarmTheTrackerSaw(t *testing.T) {
	cat := catalog()
	cat.Releases[0].Seeders = 3
	sends := []send{
		{0, releases.Seeders, seeder, "started", 0, 0, 0},
		{0, "heron", leecher, "started", size, 0, 0},
		{0, "heron", leecher, "completed", 0, size, 0},
		{0, "tamarack", "-LT2000-othertamarack", "started", size, 0, 0},
		{0, "tamarack", "-LT2000-othertamarack", "stopped", size, 0, 0},
		{61 * time.Second, "torznab", "-LT2000-latecomerlate", "started", size, 0, 0},
	}
	_, got := seedingRecords(t, cat, scenario.Tracker{Interval: 5 * time.Second, InactiveAfter: time.Minute}, sends, 0)
	want := []string{
		replied(4, 0, 0, "5", "5", ""),
		replied(4, 1, 0, "5", "5", ""),
		replied(5, 0, 1, "5", "5", ""),
		replied(5, 1, 1, "5", "5", ""),
		replied(5, 0, 1, "5", "5", ""),
		replied(3, 1, 1, "5", "5", ""),
	}
	if !slices.Equal(got, want) {
		t.Errorf("replies =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestTheUnregisterCueUnregistersASeedingTorrent(t *testing.T) {
	s := scenario.Scenario{Seed: "tracker", Tracker: scenario.Tracker{Interval: 5 * time.Second}}
	synctest.Test(t, func(t *testing.T) {
		f := faketest.InMemory(t, tracker.Fake, s, opts...)
		a := send{0, "heron", seeder, "started", 0, 0, 0}
		if got, want := get(t, f, a.query()), replied(1, 0, 0, "5", "5", ""); got != want {
			t.Fatalf("the seeder's announce = %q, want %q", got, want)
		}
		for _, was := range []bool{false, true} {
			if got := fmt.Sprint(f.Cue(t, tracker.Unregister.Name, tracker.UnregisterArgs{Release: "orchard"})); got != fmt.Sprintf("{%t}", was) {
				t.Fatalf("the cue answered %s, want was %t", got, was)
			}
		}
		a.event = ""
		if got, want := get(t, f, a.query()), failed(tracker.FailureUnregistered, "5", "5"); got != want {
			t.Fatalf("the seeder's next announce = %q, want %q", got, want)
		}
	})
}
