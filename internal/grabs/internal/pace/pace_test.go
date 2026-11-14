package pace_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/searches"
)

const (
	interval = 8 * time.Hour
	longest  = 7 * 24 * time.Hour
)

var t0 = time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)

func press(items ...string) pace.Entry {
	return pace.Entry{Lane: pace.Now, Trigger: searches.Search, Items: nilIfEmpty(items), Gone: true, ResetGap: true, At: t0}
}

func added(items ...string) pace.Entry {
	return pace.Entry{Lane: pace.Now, Trigger: searches.Add, Holds: true, Items: nilIfEmpty(items), ResetGap: true, At: t0.Add(time.Minute)}
}

func released(items ...string) pace.Entry {
	return pace.Entry{Lane: pace.Backlog, Trigger: searches.ReleaseDate, Holds: true, Items: nilIfEmpty(items), At: t0.Add(2 * time.Minute)}
}

func failed(items ...string) pace.Entry {
	return pace.Entry{Lane: pace.Now, Trigger: searches.FailedGrab, Holds: true, Items: nilIfEmpty(items), At: t0.Add(3 * time.Minute)}
}

func backlog(items ...string) pace.Entry {
	return pace.Entry{Lane: pace.Backlog, Trigger: searches.Backlog, Holds: true, Items: nilIfEmpty(items), At: t0}
}

func searchAll() pace.Entry {
	return pace.Entry{Lane: pace.Bulk, Trigger: searches.SearchAll, ResetGap: true, At: t0.Add(time.Minute)}
}

func nilIfEmpty(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	return items
}

// queued is the row Merge leaves from nothing for e.
func queued(t *testing.T, e pace.Entry) pace.Row {
	t.Helper()
	r, _, _ := pace.Merge(pace.Row{}, e, interval, time.Hour)
	return r
}

// queue is what a row holds in its queue: lane, trigger, Hold, gone cells,
// Items and queued time.
type queue struct {
	Lane    pace.Lane
	Trigger searches.Trigger
	Holds   bool
	Gone    bool
	Items   []string
	Queued  time.Time
}

func queueOf(r pace.Row) queue {
	return queue{r.Lane, r.Trigger, r.Holds, r.Gone, r.Items, r.Queued}
}

func TestMergeLandingTwiceIsLandingOnce(t *testing.T) {
	open := pace.Row{Exists: true, Gap: 2 * interval, Due: t0, Searched: t0.Add(-interval),
		Searching: &pace.Run{Lane: pace.Backlog, Holds: true, Items: []string{"e1", "e2"}}}
	for _, c := range []struct {
		name string
		row  pace.Row
		e    pace.Entry
	}{
		{"a press on nothing", pace.Row{}, press("e1")},
		{"an add on a queued press", queued(t, press("e1")), added("e2")},
		{"a press on a queued add", queued(t, added("e2")), press("e1")},
		{"Search all on a queued backlog entry", queued(t, backlog("e3")), searchAll()},
		{"a press on an open backlog run", open, press("e1", "e3")},
		{"a gap reset alone", open, pace.Entry{Trigger: searches.Retarget, ResetGap: true, At: t0}},
	} {
		once, _, _ := pace.Merge(c.row, c.e, interval, time.Hour)
		twice, _, promote := pace.Merge(once, c.e, interval, time.Hour)
		if !reflect.DeepEqual(once, twice) || promote {
			t.Errorf("%s: landing again moved\n%+v\nto\n%+v (promote %v)", c.name, once, twice, promote)
		}
	}
}

func TestTheAdminsPressOutranksAutomaticTriggersInTheQueue(t *testing.T) {
	for _, c := range []struct {
		name  string
		row   pace.Row
		e     pace.Entry
		want  queue
		state pace.State
	}{
		{"a press on a queued add takes its trigger and opens gone cells", queued(t, added("e2")), press("e1"),
			queue{pace.Now, searches.Search, false, true, []string{"e1", "e2"}, t0}, pace.Queued},
		{"an add on a queued press leaves its trigger and gone cells", queued(t, press("e1")), added("e2"),
			queue{pace.Now, searches.Search, false, true, []string{"e1", "e2"}, t0}, pace.Queued},
		{"a release date on a queued press leaves its trigger and gone cells", queued(t, press("e1")), released("e1"),
			queue{pace.Now, searches.Search, false, true, []string{"e1"}, t0}, pace.Merged},
		{"a failure on a queued press leaves its trigger and gone cells", queued(t, press()), failed("e4"),
			queue{pace.Now, searches.Search, false, true, nil, t0}, pace.Merged},
		{"Search all on a queued backlog entry takes it", queued(t, backlog("e3")), searchAll(),
			queue{pace.Bulk, searches.SearchAll, false, false, nil, t0}, pace.Queued},
		{"a backlog entry on queued Search all leaves it", queued(t, searchAll()), backlog("e3"),
			queue{pace.Bulk, searches.SearchAll, false, false, nil, t0}, pace.Merged},
	} {
		got, state, _ := pace.Merge(c.row, c.e, interval, time.Hour)
		if !reflect.DeepEqual(queueOf(got), c.want) || state != c.state {
			t.Errorf("%s: %+v %v, want %+v %v", c.name, queueOf(got), state, c.want, c.state)
		}
	}
}

func TestAPressOnAnOpenRunPromotesItAndQueuesOnlyWhatItDoesntCover(t *testing.T) {
	run := pace.Run{Lane: pace.Backlog, Holds: true, Items: []string{"e1", "e2"}}
	open := pace.Row{Exists: true, Gap: interval, Due: t0, Searching: &run}

	got, state, promote := pace.Merge(open, press("e2"), interval, time.Hour)
	if state != pace.Merged || !promote || got.Lane != 0 || got.Again() {
		t.Errorf("a covered press reads %v promote=%v lane=%v again=%v, want Merged, promoted and nothing queued", state, promote, got.Lane, got.Again())
	}
	if want := (pace.Run{Lane: pace.Now, Items: []string{"e1", "e2"}}); !reflect.DeepEqual(*got.Searching, want) {
		t.Errorf("the promoted run reads %+v, want %+v", *got.Searching, want)
	}

	got, state, promote = pace.Merge(open, press("e2", "e3"), interval, time.Hour)
	if state != pace.Queued || !promote || !got.Again() || !reflect.DeepEqual(got.Items, []string{"e3"}) {
		t.Errorf("a press past the run reads %v promote=%v again=%v items=%v, want Queued, promoted, again, only e3", state, promote, got.Again(), got.Items)
	}

	got, state, promote = pace.Merge(open, released("e1"), interval, time.Hour)
	if state != pace.Merged || promote || got.Searching.Lane != pace.Backlog {
		t.Errorf("a release date on its own run reads %v promote=%v, want Merged and no promotion", state, promote)
	}

	if queued(t, press("e1")).Again() {
		t.Error("a row queued with no open run reads again")
	}
}

func TestANewRowIsSpreadAndAResetBringsItsDueBackToOneInterval(t *testing.T) {
	spread := pace.Spread("work", "l:01", interval)
	if spread < 0 || spread >= interval || spread != pace.Spread("work", "l:01", interval) {
		t.Fatalf("Spread = %v, want one fixed offset inside %v", spread, interval)
	}
	r, _, _ := pace.Merge(pace.Row{}, backlog(), interval, spread)
	if r.Gap != interval || !r.Due.Equal(t0.Add(spread)) {
		t.Errorf("a new row has gap %v due %v, want %v due %v", r.Gap, r.Due, interval, t0.Add(spread))
	}
	searched := t0.Add(-time.Hour)
	long := pace.Row{Exists: true, Gap: longest, Due: searched.Add(longest), Searched: searched}
	r, _, _ = pace.Merge(long, pace.Entry{Trigger: searches.Retarget, ResetGap: true, At: t0}, interval, spread)
	if r.Gap != interval || !r.Due.Equal(searched.Add(interval)) || r.Lane != 0 {
		t.Errorf("a reset row has gap %v due %v lane %v, want %v due %v and nothing queued", r.Gap, r.Due, r.Lane, interval, searched.Add(interval))
	}
}

func TestGapsDoubleToTheLongestAndAnUnheardRunLeavesThemAlone(t *testing.T) {
	for _, c := range []struct {
		name   string
		prev   time.Duration
		r      pace.Result
		fresh  bool
		capped time.Duration
		want   time.Duration
	}{
		{"answered at the interval", interval, pace.Answered, false, 0, 2 * interval},
		{"answered near the longest", 5 * 24 * time.Hour, pace.Answered, false, 0, longest},
		{"answered at the longest", longest, pace.Answered, false, 0, longest},
		{"unheard", 4 * interval, pace.Unheard, false, 0, 4 * interval},
		{"took", 4 * interval, pace.Took, false, 0, interval},
		{"answered while fresh", 4 * interval, pace.Answered, true, 0, interval},
		{"answered under a cap", 4 * interval, pace.Answered, false, 3 * interval, 3 * interval},
	} {
		if got := pace.Gap(c.prev, interval, longest, c.r, c.fresh, c.capped); got != c.want {
			t.Errorf("%s: gap %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAPressStartsWithOneMoreWaitingLine(t *testing.T) {
	for _, c := range []struct {
		waiting         int
		background, now bool
	}{{0, true, true}, {1, true, true}, {2, false, true}, {3, false, false}} {
		r := pace.Room{AtOnce: 2, Waiting: c.waiting}
		if r.Starts(false) != c.background || r.Starts(true) != c.now {
			t.Errorf("%d waiting of 2 at once: background %v press %v, want %v %v", c.waiting, r.Starts(false), r.Starts(true), c.background, c.now)
		}
	}
}

func TestLanesAreServedInRankOrderUnderTheirWireWords(t *testing.T) {
	var words []string
	for i, l := range pace.Lanes {
		if i > 0 && l <= pace.Lanes[i-1] {
			t.Errorf("lane %v comes after a lane it outranks", l)
		}
		words = append(words, l.String())
	}
	if !reflect.DeepEqual(words, []string{"now", "bulk", "backlog"}) || !pace.Now.Press() || pace.Bulk.Press() || pace.Backlog.Press() {
		t.Errorf("lanes read %v, want now, bulk, backlog with only now a press", words)
	}
}
