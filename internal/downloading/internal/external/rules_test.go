package external

import (
	"errors"
	"math/rand/v2"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

var t0 = time.Date(2026, 11, 7, 14, 0, 0, 0, time.UTC)

func hash(b byte) bencode.InfoHash { return bencode.InfoHash{b} }

func listedOf(hashes ...byte) map[bencode.InfoHash]int {
	out := map[bencode.InfoHash]int{}
	for _, h := range hashes {
		out[hash(h)] = 0
	}
	return out
}

func gotOf(hashes ...byte) map[bencode.InfoHash]bool {
	out := map[bencode.InfoHash]bool{}
	for _, h := range hashes {
		out[hash(h)] = true
	}
	return out
}

func TestATorrentVanishesAfterTwoPollsWithoutIt(t *testing.T) {
	listed := listedOf(1)
	vanished, missed, mass := absence(listed, gotOf(), 50, 3)
	if len(vanished) != 0 || mass || missed[hash(1)] != 1 {
		t.Fatalf("after one miss: vanished %v, mass %v, missed %v; want one miss and nothing vanished", vanished, mass, missed)
	}
	vanished, missed, mass = absence(missed, gotOf(), 50, 3)
	if !slices.Equal(vanished, []bencode.InfoHash{hash(1)}) || mass {
		t.Fatalf("after two misses: vanished %v, mass %v; want the one torrent vanished", vanished, mass)
	}
	if _, ok := missed[hash(1)]; ok {
		t.Errorf("a vanished torrent is still counted: %v", missed)
	}
}

func TestATorrentListedAgainStartsItsCountAfresh(t *testing.T) {
	_, missed, _ := absence(listedOf(1, 2), gotOf(2), 50, 3)
	_, missed, _ = absence(missed, gotOf(1, 2), 50, 3)
	vanished, _, _ := absence(missed, gotOf(2), 50, 3)
	if len(vanished) != 0 {
		t.Errorf("a torrent missed, listed, then missed vanished: %v", vanished)
	}
}

func TestMostTorrentsMissingAtOnceMarksNothing(t *testing.T) {
	listed := map[bencode.InfoHash]int{hash(1): 1, hash(2): 0, hash(3): 0, hash(4): 0}
	vanished, missed, mass := absence(listed, gotOf(4), 50, 3)
	if !mass || len(vanished) != 0 {
		t.Fatalf("3 of 4 missing: vanished %v, mass %v; want mass and nothing vanished", vanished, mass)
	}
	if missed[hash(1)] != 1 || missed[hash(2)] != 0 {
		t.Errorf("mass absence changed the counts: %v", missed)
	}
}

func TestHalfMissingOrFewerThanTheFloorIsNoMassAbsence(t *testing.T) {
	cases := []struct {
		name   string
		listed map[bencode.InfoHash]int
		got    map[bencode.InfoHash]bool
	}{
		{"2 of 4, not more than half", listedOf(1, 2, 3, 4), gotOf(3, 4)},
		{"2 of 2, under the floor", listedOf(1, 2), gotOf()},
	}
	for _, c := range cases {
		if _, _, mass := absence(c.listed, c.got, 50, 3); mass {
			t.Errorf("%s: mass absence", c.name)
		}
	}
}

func TestAnUnlistedAddIsSentOnceMoreThenNotKept(t *testing.T) {
	wait := 2 * time.Minute
	cases := []struct {
		name    string
		resends int
		after   time.Duration
		want    Resend
	}{
		{"inside the wait", 0, time.Minute, ResendWait},
		{"after the wait", 0, wait, ResendNow},
		{"resent, inside the wait", 1, time.Minute, ResendWait},
		{"resent, after the wait", 1, wait, ResendNotKept},
	}
	for _, c := range cases {
		if got := resend(t0, c.resends, t0.Add(c.after), wait); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}

func TestTheImportDelayCountsFromTheLatestOfFinishAndMove(t *testing.T) {
	delay := 10 * time.Minute
	cases := []struct {
		name                         string
		clientFinished, sight, quiet time.Time
		want                         time.Time
	}{
		{"the client's finish time", t0, t0.Add(time.Minute), time.Time{}, t0.Add(delay)},
		{"first sight when the client gives none", time.Time{}, t0.Add(time.Minute), time.Time{}, t0.Add(time.Minute + delay)},
		{"a move that ended after the finish", t0, t0.Add(time.Minute), t0.Add(5 * time.Minute), t0.Add(5*time.Minute + delay)},
		{"a move before the finish", t0, t0, t0.Add(-time.Minute), t0.Add(delay)},
	}
	for _, c := range cases {
		if got := importAt(c.clientFinished, c.sight, c.quiet, delay); !got.Equal(c.want) {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

func TestWantedSync(t *testing.T) {
	started, paused := clients.Started, clients.Paused
	cases := []struct {
		name       string
		intent     clients.Run
		observed   clients.Run
		atLimit    bool
		w          Wanted
		wantSend   clients.Run
		wantChange *WantedChange
		wantNext   Wanted
	}{
		{"in step", started, started, false, Wanted{Applied: started}, 0, nil, Wanted{Applied: started}},
		{"an Admin's pause is sent", paused, started, false, Wanted{Applied: started}, paused, nil, Wanted{Applied: paused}},
		{
			"a pause in the client is reported", started, paused, false, Wanted{Applied: started},
			0, &WantedChange{started, paused}, Wanted{Applied: paused, Reported: &WantedChange{started, paused}},
		},
		{
			"a stale intent waits for the reported pause", started, paused, false, Wanted{Applied: paused, Reported: &WantedChange{started, paused}},
			0, nil, Wanted{Applied: paused, Reported: &WantedChange{started, paused}},
		},
		{"the intent shows the pause", paused, paused, false, Wanted{Applied: paused, Reported: &WantedChange{started, paused}}, 0, nil, Wanted{Applied: paused}},
		{"then an Admin's resume is sent", started, paused, false, Wanted{Applied: paused}, started, nil, Wanted{Applied: started}},
		{"a stop at the seeding limit is no pause", started, paused, true, Wanted{Applied: started}, 0, nil, Wanted{Applied: started}},
	}
	for _, c := range cases {
		send, change, next := wantedStep(c.intent, c.observed, c.atLimit, c.w)
		if send != c.wantSend || !sameChange(change, c.wantChange) || next.Applied != c.wantNext.Applied || !sameChange(next.Reported, c.wantNext.Reported) {
			t.Errorf("%s: send %d, change %v, next %+v; want send %d, change %v, next %+v", c.name, send, change, next, c.wantSend, c.wantChange, c.wantNext)
		}
	}
}

func sameChange(a, b *WantedChange) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func TestAStallNeedsARunningUnfinishedTorrentWithoutGrowth(t *testing.T) {
	after := 10 * time.Minute
	cases := []struct {
		name              string
		since             time.Duration
		running, finished bool
		want              bool
	}{
		{"grew recently", time.Minute, true, false, false},
		{"no growth for the stall time", after, true, false, true},
		{"paused", after, false, false, false},
		{"finished", after, true, true, false},
	}
	for _, c := range cases {
		if got := stalled(t0, t0.Add(c.since), after, c.running, c.finished); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	if stalled(t0, t0.Add(time.Hour), 0, true, false) {
		t.Error("a stall time of 0 stalled a torrent")
	}
}

func TestUnregisteredHoldsForTheGap(t *testing.T) {
	gap := 30 * time.Minute
	if unregistered(t0, t0.Add(time.Minute), gap) || !unregistered(t0, t0.Add(gap), gap) || unregistered(time.Time{}, t0, gap) {
		t.Error("unregistered isn't the message holding for the gap")
	}
}

func TestDeletableIsTheFilesInsideTheFolderAndTheirEmptyFolders(t *testing.T) {
	within := filepath.Join(string(filepath.Separator)+"downloads", "Movies")
	release := filepath.Join(within, "The.Attic.2024")
	paths, dirs, err := deletable([]string{
		filepath.Join(release, "Subs", "en.srt"), filepath.Join(release, "The.Attic.2024.mkv"), filepath.Join(within, "Orchard.mkv"),
	}, within)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Errorf("paths %v, want the three files", paths)
	}
	if want := []string{filepath.Join(release, "Subs"), release}; !slices.Equal(dirs, want) {
		t.Errorf("dirs %v, want %v, deepest first and never the download folder", dirs, want)
	}
	_, _, err = deletable([]string{filepath.Join(within, "a.mkv"), filepath.Join(within, "..", "Other", "b.mkv")}, within)
	if !errors.Is(err, errOutside) {
		t.Errorf("a file outside the folder: err %v, want errOutside", err)
	}
}

var noResponse = &clients.Problem{Cause: clients.NoResponse}

func TestAClientIsNotRespondingOnlyAfterTheGraceSinceItsFirstFailure(t *testing.T) {
	grace := time.Minute
	r := succeeded(reach{}, nil, t0)
	r = failed(r, noResponse, t0.Add(10*time.Second), grace, true)
	if r.state != clients.Responding {
		t.Fatalf("inside the grace: %s, want responding", r.state)
	}
	r = failed(r, noResponse, t0.Add(80*time.Second), grace, true)
	if r.state != clients.NotResponding || !r.since.Equal(t0.Add(10*time.Second)) {
		t.Fatalf("past the grace: %s since %s, want not responding since the first failure", r.state, r.since)
	}
	r = succeeded(r, nil, t0.Add(90*time.Second))
	if r.state != clients.Responding || r.failures != 0 || !r.since.Equal(t0.Add(90*time.Second)) {
		t.Errorf("after a success: %+v, want responding since then", r)
	}
}

func TestARefusedSignInStopsPollingOnlyForAKindWithALockout(t *testing.T) {
	refused := &clients.Problem{Cause: clients.WrongCredentials}
	if r := failed(reach{}, refused, t0, time.Minute, true); r.state != clients.SignInRefused || !r.stopped {
		t.Errorf("with a lockout: %+v, want refused and stopped", r)
	}
	if r := failed(reach{}, refused, t0, time.Minute, false); r.state != clients.SignInRefused || r.stopped {
		t.Errorf("without a lockout: %+v, want refused and polling", r)
	}
}

func TestNextPoll(t *testing.T) {
	tm := Timings{RetryMin: 5 * time.Second, RetryMax: 5 * time.Minute, RefusedRetry: 30 * time.Minute, LiveSample: time.Second, LiveFloor: 2 * time.Second}
	rnd := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // seeded on purpose so tests replay
	every := 10 * time.Second
	if got := nextPoll(t0, reach{}, every, false, tm, rnd); !got.Equal(t0.Add(every)) {
		t.Errorf("responding: %s, want Check every later", got)
	}
	if got := nextPoll(t0, reach{}, every, true, tm, rnd); !got.Equal(t0.Add(2 * time.Second)) {
		t.Errorf("watched: %s, want the live floor later", got)
	}
	if got := nextPoll(t0, reach{state: clients.SignInRefused}, every, false, tm, rnd); !got.Equal(t0.Add(30 * time.Minute)) {
		t.Errorf("refused without a lockout: %s, want the refused retry later", got)
	}
	if got := nextPoll(t0, reach{state: clients.SignInRefused, stopped: true}, every, true, tm, rnd); !got.IsZero() {
		t.Errorf("stopped: %s, want no poll", got)
	}
	var last time.Duration
	for n := 1; n <= 12; n++ {
		d := nextPoll(t0, reach{failures: n}, every, true, tm, rnd).Sub(t0)
		if d < tm.RetryMin/2 || d > tm.RetryMax {
			t.Fatalf("failure %d: %s, want within the backoff's bounds", n, d)
		}
		if n == 12 && d < last/2 {
			t.Errorf("failure %d: %s, want the backoff near its ceiling", n, d)
		}
		last = d
	}
}
