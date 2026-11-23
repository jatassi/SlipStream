package indexers_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

var now = time.Date(2026, 11, 14, 13, 40, 0, 0, time.UTC)

// used is n reservations in the window: the oldest at oldest, the rest a
// minute ago.
func used(n int, oldest time.Time) []time.Time {
	out := []time.Time{oldest}
	for range n - 1 {
		out = append(out, now.Add(-time.Minute))
	}
	return out
}

// torrentLeech is 50 searches an hour with 36 used, the oldest leaving in
// 20 minutes, two at once and RSS sync every 15 minutes whose last fetch
// used one page.
func torrentLeech() indexers.Facts {
	return indexers.Facts{
		AtOnce: 2, InFlight: map[indexers.Claim]int{}, Limit: indexers.Limit{Count: 50, Per: indexers.Hour},
		Window: used(36, now.Add(-40*time.Minute)),
		RSS:    indexers.RSSPlan{On: true, Every: 15 * time.Minute, LastPages: 1},
	}
}

// take is how many lines of c FitOf reserves from f at now, each answered
// before the next once Searches at once is in flight, and the Fit that
// stopped them.
func take(f indexers.Facts, c indexers.Claim) (taken, waves int, last indexers.Fit) {
	for {
		fit := indexers.FitOf(f, c, now)
		switch {
		case fit.Reserve:
			taken++
			f.InFlight[c]++
			f.Window = append(f.Window, now)
		case fit.Wait != nil && fit.Wait.Busy:
			waves++
			f.InFlight[c] = 0
		default:
			return taken, waves, fit
		}
	}
}

func TestFitOf(t *testing.T) {
	full := func(f indexers.Facts) indexers.Facts {
		f.Window = used(50, now.Add(-40*time.Minute))
		return f
	}
	tight := func(count int) indexers.Facts {
		f := torrentLeech()
		f.Limit.Count, f.Window = count, nil
		return f
	}
	off := torrentLeech()
	off.RSS.On = false
	busy := torrentLeech()
	busy.InFlight = map[indexers.Claim]int{indexers.ClaimBackground: 2}
	stuck := busy
	stuck.Stuck = true
	manual := torrentLeech()
	manual.InFlight = map[indexers.Claim]int{indexers.ClaimPress: 3}
	lastSlot := torrentLeech()
	lastSlot.Window = used(49, now.Add(-40*time.Minute))
	at := now.Add(20 * time.Minute)
	for _, tc := range []struct {
		name string
		f    indexers.Facts
		c    indexers.Claim
		want indexers.Fit
	}{
		{"a press takes the window's last slot", lastSlot, indexers.ClaimPress, indexers.Fit{Reserve: true}},
		{"a press at a full window is skipped with its time", full(torrentLeech()), indexers.ClaimPress, indexers.Fit{Skip: &indexers.Outcome{Kind: indexers.LimitReached, Until: at}}},
		{"a background line at a full window waits until it frees", full(off), indexers.ClaimBackground, indexers.Fit{Wait: &indexers.RoomError{Until: at}}},
		{"a limit of 2 an hour keeps one slot for a background line", tight(2), indexers.ClaimBackground, indexers.Fit{Reserve: true}},
		{"a limit of 1 an hour under RSS sync is Starved", tight(1), indexers.ClaimBackground, indexers.Fit{Wait: &indexers.RoomError{Starved: true}}},
		{"a limit of 1 an hour still takes a press", tight(1), indexers.ClaimPress, indexers.Fit{Reserve: true}},
		{"busy waits", busy, indexers.ClaimBackground, indexers.Fit{Wait: &indexers.RoomError{Busy: true}}},
		{"a press goes one past Searches at once", busy, indexers.ClaimPress, indexers.Fit{Reserve: true}},
		{"busy with every query past its wait skips late", stuck, indexers.ClaimBackground, indexers.Fit{Skip: &indexers.Outcome{Kind: indexers.Late}}},
		{"Manual queries count toward the press cap", manual, indexers.ClaimPress, indexers.Fit{Wait: &indexers.RoomError{Busy: true}}},
		{"Manual queries leave background lines their own cap", manual, indexers.ClaimBackground, indexers.Fit{Reserve: true}},
		{"no limit", indexers.Facts{AtOnce: 1, InFlight: map[indexers.Claim]int{}}, indexers.ClaimBackground, indexers.Fit{Reserve: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := indexers.FitOf(tc.f, tc.c, now)
			if !sameFit(got, tc.want) {
				t.Fatalf("FitOf = %s, want %s", fitWords(got), fitWords(tc.want))
			}
		})
	}
	t.Run("a limit of 2 an hour waits once its one background slot is used", func(t *testing.T) {
		f := tight(2)
		f.Window = []time.Time{now.Add(-10 * time.Minute)}
		if got := indexers.FitOf(f, indexers.ClaimBackground, now); got.Wait == nil || got.Wait.Starved || !got.Wait.Until.Equal(now.Add(50*time.Minute)) {
			t.Fatalf("FitOf = %s, want a wait until %s", fitWords(got), now.Add(50*time.Minute))
		}
	})
	t.Run("TorrentLeech keeps three for RSS sync and takes eleven background lines two at a time", func(t *testing.T) {
		taken, waves, last := take(torrentLeech(), indexers.ClaimBackground)
		if taken != 11 || waves != 5 {
			t.Fatalf("background lines taken = %d in %d full waves, want 11 two at a time", taken, waves)
		}
		if last.Wait == nil || !last.Wait.Until.Equal(now.Add(20*time.Minute)) {
			t.Fatalf("the twelfth line = %+v, want a wait until the oldest search leaves at %s", last, now.Add(20*time.Minute))
		}
	})
	t.Run("RSS off reserves nothing", func(t *testing.T) {
		if taken, _, _ := take(off, indexers.ClaimBackground); taken != 14 {
			t.Fatalf("background lines taken with RSS off = %d, want all 14 free slots", taken)
		}
	})
}

func sameFit(a, b indexers.Fit) bool {
	switch {
	case a.Reserve || b.Reserve:
		return a.Reserve == b.Reserve && a.Skip == nil && a.Wait == nil
	case a.Skip != nil && b.Skip != nil:
		return a.Skip.Kind == b.Skip.Kind && a.Skip.Until.Equal(b.Skip.Until)
	case a.Wait != nil && b.Wait != nil:
		return a.Wait.Busy == b.Wait.Busy && a.Wait.Starved == b.Wait.Starved && a.Wait.Until.Equal(b.Wait.Until)
	}
	return false
}

func fitWords(f indexers.Fit) string {
	switch {
	case f.Reserve:
		return "reserve"
	case f.Skip != nil:
		return "skip " + string(f.Skip.Kind) + " until " + f.Skip.Until.String()
	case f.Wait != nil:
		return "wait: " + f.Wait.Error()
	}
	return "nothing"
}

// insertQueries saves n search queries of Indexer id reserved at at, of
// claim, completed unless open.
func (p *planned) insertQueries(id indexers.ID, n int, at time.Time, claim string, open bool) {
	p.t.Helper()
	err := p.lib.Write(p.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for range n {
			var completed, outcome any
			if !open {
				completed, outcome = storage.FromTime(at), "ok"
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO indexers_query (id, indexer_id, use, claim, reserved_at, completed_at, outcome, revision)
				VALUES (?, ?, 'search', ?, ?, ?, ?, 1)`, storage.NewUUID(), id, claim, storage.FromTime(at), completed, outcome)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		p.t.Fatal(err)
	}
}

func (p *planned) reserve(name string, c indexers.Claim) (indexers.QueryID, error) {
	p.t.Helper()
	var q indexers.QueryID
	err := p.lib.Write(p.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		q, err = p.x.Reserve(ctx, tx, p.ids[name], indexerstest.Type, c)
		return err
	})
	return q, err
}

// leech is TorrentLeech's Room now.
func (p *planned) leech() indexers.Room {
	p.t.Helper()
	var rooms map[indexers.ID]indexers.Room
	err := p.lib.Read(p.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		rooms, err = p.x.Room(ctx, r)
		return err
	})
	if err != nil {
		p.t.Fatal(err)
	}
	return rooms[p.ids["TorrentLeech"]]
}

// reserveAll is how many background lines Reserve takes on Indexer name,
// each in-flight one answered once it is busy, and the wait that stopped
// it. Room has none free each time Reserve is busy.
func (p *planned) reserveAll(name string) (taken int, stopped *indexers.RoomError) {
	p.t.Helper()
	for {
		_, err := p.reserve(name, indexers.ClaimBackground)
		if errors.As(err, &stopped) && stopped.Busy {
			if free := p.leech().Background.Free; free != 0 {
				p.t.Fatalf("Room has %d background lines free while Reserve is busy", free)
			}
			p.answerAll()
			continue
		}
		if err != nil {
			return taken, stopped
		}
		taken++
	}
}

// answerAll completes every open query, as their jobs answering would.
func (p *planned) answerAll() {
	p.t.Helper()
	err := p.lib.Write(p.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE indexers_query SET completed_at = reserved_at, outcome = 'ok' WHERE completed_at IS NULL")
		return err
	})
	if err != nil {
		p.t.Fatal(err)
	}
}

func TestReserveWritesTheClaimAndRoomAgreesWithIt(t *testing.T) {
	p := plannedWith(t, "TorrentLeech")
	p.rssEvery = 15 * time.Minute
	p.edit("TorrentLeech", indexers.Edit{Searches: &indexers.Limit{Count: 50, Per: indexers.Hour}})
	oldest := storage.FromTime(time.Now().Add(-40 * time.Minute)).Time()
	p.insertQueries(p.ids["TorrentLeech"], 1, oldest, "press", false)
	p.insertQueries(p.ids["TorrentLeech"], 35, time.Now().Add(-time.Minute), "background", false)

	if r := p.leech(); r.Background.Free != 2 || r.Press.Free != 3 {
		t.Fatalf("Room = %+v, want 2 background lines and 3 presses free", r)
	}
	taken, room := p.reserveAll("TorrentLeech")
	if taken != 11 || room == nil || !room.Until.Equal(oldest.Add(time.Hour)) {
		t.Fatalf("Reserve took %d background lines and stopped with %v, want 11 and a wait until %s", taken, room, oldest.Add(time.Hour))
	}
	if r := p.leech().Background; r.Free != 0 || !r.Until.Equal(room.Until) {
		t.Fatalf("Room's background = %+v, want none free until %s as Reserve said", r, room.Until)
	}
	var background int
	if err := p.lib.QueryRowContext(t.Context(), "SELECT count(*) FROM indexers_query WHERE claim = 'background'").Scan(&background); err != nil || background != 35+11 {
		t.Fatalf("background query rows = %d, %v; want 46", background, err)
	}
	p.answerAll()
	if r := p.leech().Press; r.Free != 3 {
		t.Fatalf("Room's press = %+v, want the reserve's 3 slots free for presses", r)
	}
	for range 3 {
		if _, err := p.reserve("TorrentLeech", indexers.ClaimPress); err != nil {
			t.Fatalf("a press into RSS sync's reserve: %v", err)
		}
	}
	p.answerAll()
	var wait *indexers.WaitError
	if _, err := p.reserve("TorrentLeech", indexers.ClaimPress); !errors.As(err, &wait) || wait.Outcome.Kind != indexers.LimitReached || !wait.Outcome.Until.Equal(oldest.Add(time.Hour)) {
		t.Fatalf("a press at the full window = %v, want skipped limit_reached until %s", err, oldest.Add(time.Hour))
	}
	if r := p.leech().Press; r.Free != 0 || !r.Until.IsZero() || r.Starved {
		t.Fatalf("Room's press at the full window = %+v, want nothing holding it: a press skips the Indexer", r)
	}
}

func TestAnInFlightQueryOlderThanJobCapNoLongerCounts(t *testing.T) {
	p := plannedWith(t, "Alder")
	p.insertQueries(p.ids["Alder"], 2, time.Now().Add(-searching.DefaultTimings().JobCap-time.Second), "background", true)
	if _, err := p.reserve("Alder", indexers.ClaimBackground); err != nil {
		t.Fatalf("a background line beside two queries past JobCap: %v", err)
	}
	var room *indexers.RoomError
	p.insertQueries(p.ids["Alder"], 1, time.Now(), "background", true)
	if _, err := p.reserve("Alder", indexers.ClaimBackground); !errors.As(err, &room) || !room.Busy {
		t.Fatalf("a background line beside two fresh queries = %v, want busy", err)
	}
}

func TestABusyIndexerWhoseQueriesAreAllPastTheirWaitSkipsLate(t *testing.T) {
	p := plannedWith(t, "Alder")
	p.insertQueries(p.ids["Alder"], 2, time.Now().Add(-5*time.Minute), "background", true)
	var wait *indexers.WaitError
	if _, err := p.reserve("Alder", indexers.ClaimBackground); !errors.As(err, &wait) || wait.Outcome.Kind != indexers.Late {
		t.Fatalf("a background line behind two stuck queries = %v, want skipped late", err)
	}
}

func TestManualSearchCountsTowardThePressCap(t *testing.T) {
	p := plannedWith(t, "Alder")
	for range 3 {
		p.plan()
	}
	var room *indexers.RoomError
	if _, err := p.reserve("Alder", indexers.ClaimPress); !errors.As(err, &room) || !room.Busy {
		t.Fatalf("a press beside three Manual searches = %v, want busy", err)
	}
	if _, err := p.reserve("Alder", indexers.ClaimBackground); err != nil {
		t.Fatalf("a background line beside three Manual searches: %v", err)
	}
}

func TestSearchOneAskedAgainAfterARestartJoinsTheSameJob(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := plannedWith(t, "Alder")
		p.node.Answer(p.ids["Alder"], 90*time.Second, searching.Answer{Releases: []searching.Release{{Title: "Orchard.2026.1080p"}}})
		q, err := p.reserve("Alder", indexers.ClaimBackground)
		if err != nil {
			t.Fatal(err)
		}
		line := indexers.Line{Indexer: p.ids["Alder"], Name: "Alder", Query: q}
		s := indexers.Search{Type: indexerstest.Type, Terms: indexers.Terms{Mode: "search", Query: "orchard"}}
		deadline := time.Now().Add(2 * time.Minute)

		stopped, stop := context.WithCancel(t.Context())
		time.AfterFunc(30*time.Second, stop)
		if _, err := p.x.SearchOne(stopped, line, s, deadline); !errors.Is(err, context.Canceled) {
			t.Fatalf("SearchOne stopped by its Controller = %v, want canceled", err)
		}
		synctest.Wait()
		var room *indexers.RoomError
		p.insertQueries(p.ids["Alder"], 1, time.Now(), "background", true)
		if _, err := p.reserve("Alder", indexers.ClaimBackground); !errors.As(err, &room) || !room.Busy {
			t.Fatalf("Reserve beside the stopped line = %v, want busy: its query is still in flight", err)
		}

		a, err := p.x.SearchOne(t.Context(), line, s, deadline)
		if err != nil {
			t.Fatal(err)
		}
		if a.Outcome.Kind != indexers.Found || len(p.node.Jobs()) != 1 {
			t.Fatalf("SearchOne asked again = %s with %d jobs run, want found from the one job", a.Outcome.Kind, len(p.node.Jobs()))
		}
		if elapsed := time.Since(deadline.Add(-2 * time.Minute)); elapsed != 90*time.Second {
			t.Fatalf("the joined answer came %s after the first ask, want 1m30s", elapsed)
		}

		time.Sleep(time.Until(deadline))
		late, err := p.x.SearchOne(t.Context(), indexers.Line{Indexer: p.ids["Alder"], Query: q}, s, deadline)
		if err != nil || late.Outcome.Kind != indexers.Late || len(p.node.Jobs()) != 1 {
			t.Fatalf("SearchOne past its deadline = %s, %v with %d jobs, want late without asking", late.Outcome.Kind, err, len(p.node.Jobs()))
		}
	})
}
