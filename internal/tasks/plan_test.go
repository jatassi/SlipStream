package tasks

import (
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

var (
	everyEight = setting.Duration{Key: "tasks.sync.cadence", Label: "Sync", Default: 8 * time.Hour, Min: 10 * time.Minute, Max: day, Step: time.Minute}
	daily      = setting.Duration{Key: "tasks.nightly.cadence", Label: "Nightly", Default: day, Min: day, Max: 7 * day, Step: day}

	syncTask     = Task{ID: "sync", Name: "Sync", Cadence: Every{Interval: everyEight}}
	updateCheck  = Task{ID: "update-check", Name: "Update check", Cadence: Every{Interval: everyEight, AtStart: true}}
	backup       = Task{ID: "backup", Name: "Backup", Cadence: InWindow{Days: daily}}
	housekeeping = Task{ID: "housekeeping", Name: "Housekeeping", Cadence: InWindow{Days: daily}, Overdue: Wait}

	// catalogRefresh is shaped as the Indexer catalog refresh: daily, no
	// faster than hourly, and at each start.
	hourlyToWeekly = setting.Duration{Key: "tasks.catalog.cadence", Label: "Catalog", Default: day, Min: time.Hour, Max: 7 * day, Step: time.Hour}
	catalogRefresh = Task{ID: "indexer-catalog-refresh", Name: "Indexer catalog refresh", Cadence: Every{Interval: hourlyToWeekly, AtStart: true}}
)

func baseKnobs(started string) knobs {
	return knobs{
		zone:          time.UTC,
		window:        setting.Span{Start: 4 * 60, End: 6 * 60},
		jitterPercent: 10,
		jitterCap:     5 * time.Minute,
		overdue:       CatchUp,
		started:       at(started),
	}
}

func overdue(o Overdue) *Overdue { return &o }

func TestPlan(t *testing.T) {
	waitGlobally := baseKnobs("2026-10-16T00:00:00Z")
	waitGlobally.overdue = Wait
	downThreeDays := state{task: syncTask, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-13T00:00:00Z"), lastEnded: at("2026-10-13T00:05:00Z"), jitter: 0.5}
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	inLondon := baseKnobs("2026-03-01T00:00:00Z")
	inLondon.zone = london
	for _, tc := range []struct {
		name   string
		now    string
		knobs  knobs
		states []state
		want   Plan
	}{
		{
			name:   "a never-run task opens at once under Catch up",
			now:    "2026-10-16T00:00:00Z",
			knobs:  baseKnobs("2026-10-16T00:00:00Z"),
			states: []state{{task: syncTask, firstSeen: at("2026-10-16T00:00:00Z")}},
			want:   Plan{Open: []Opening{{"sync", ByCatchUp}}, Next: map[TaskID]time.Time{}},
		},
		{
			name:   "a never-run task waits one interval under Wait",
			now:    "2026-10-16T00:00:00Z",
			knobs:  waitGlobally,
			states: []state{{task: syncTask, firstSeen: at("2026-10-16T00:00:00Z")}},
			want: Plan{
				Pass: []Passed{{"sync", at("2026-10-16T00:00:00Z")}},
				Next: map[TaskID]time.Time{"sync": at("2026-10-16T08:00:00Z")},
				Wake: at("2026-10-16T08:00:00Z"),
			},
		},
		{
			name:   "three days down catch up once",
			now:    "2026-10-16T00:00:00Z",
			knobs:  baseKnobs("2026-10-16T00:00:00Z"),
			states: []state{downThreeDays},
			want:   Plan{Open: []Opening{{"sync", ByCatchUp}}, Next: map[TaskID]time.Time{}},
		},
		{
			name:   "three days down under Wait pass the latest missed slot, as a null override follows the global rule",
			now:    "2026-10-16T00:00:00Z",
			knobs:  waitGlobally,
			states: []state{downThreeDays},
			want: Plan{
				Pass: []Passed{{"sync", at("2026-10-15T16:02:30Z")}},
				Next: map[TaskID]time.Time{"sync": at("2026-10-16T00:05:00Z")},
				Wake: at("2026-10-16T00:05:00Z"),
			},
		},
		{
			name:   "a task override of Wait beats a global Catch up",
			now:    "2026-10-16T00:00:00Z",
			knobs:  baseKnobs("2026-10-16T00:00:00Z"),
			states: []state{func() state { s := downThreeDays; s.override = overdue(Wait); return s }()},
			want: Plan{
				Pass: []Passed{{"sync", at("2026-10-15T16:02:30Z")}},
				Next: map[TaskID]time.Time{"sync": at("2026-10-16T00:05:00Z")},
				Wake: at("2026-10-16T00:05:00Z"),
			},
		},
		{
			name:   "a jittered slot",
			now:    "2026-10-16T01:00:00Z",
			knobs:  baseKnobs("2026-10-15T00:00:00Z"),
			states: []state{{task: syncTask, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-16T00:00:00Z"), lastEnded: at("2026-10-16T00:05:00Z"), jitter: 0.5}},
			want: Plan{
				Next: map[TaskID]time.Time{"sync": at("2026-10-16T08:02:30Z")},
				Wake: at("2026-10-16T08:02:30Z"),
			},
		},
		{
			name:   "the jittered slot opens a schedule run",
			now:    "2026-10-16T08:02:30Z",
			knobs:  baseKnobs("2026-10-15T00:00:00Z"),
			states: []state{{task: syncTask, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-16T00:00:00Z"), lastEnded: at("2026-10-16T00:05:00Z"), jitter: 0.5}},
			want:   Plan{Open: []Opening{{"sync", BySchedule}}, Next: map[TaskID]time.Time{}},
		},
		{
			name:   "a slot that fell due mid-run is skipped",
			now:    "2026-10-16T08:10:00Z",
			knobs:  baseKnobs("2026-10-15T00:00:00Z"),
			states: []state{{task: syncTask, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-16T00:00:00Z"), lastEnded: at("2026-10-16T08:10:00Z"), jitter: 0.5}},
			want: Plan{
				Next: map[TaskID]time.Time{"sync": at("2026-10-16T16:02:30Z")},
				Wake: at("2026-10-16T16:02:30Z"),
			},
		},
		{
			name:   "Run now leaves the next slot where it was",
			now:    "2026-10-16T10:05:00Z",
			knobs:  baseKnobs("2026-10-15T00:00:00Z"),
			states: []state{{task: syncTask, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-16T08:00:00Z"), lastEnded: at("2026-10-16T10:05:00Z")}},
			want: Plan{
				Next: map[TaskID]time.Time{"sync": at("2026-10-16T16:00:00Z")},
				Wake: at("2026-10-16T16:00:00Z"),
			},
		},
		{
			name:  "inside the window Backup opens before Housekeeping",
			now:   "2026-10-16T04:00:00Z",
			knobs: baseKnobs("2026-10-15T00:00:00Z"),
			states: []state{
				{task: backup, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-15T04:00:00Z"), lastEnded: at("2026-10-15T04:20:00Z")},
				{task: housekeeping, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-15T04:20:00Z"), lastEnded: at("2026-10-15T04:21:00Z")},
			},
			want: Plan{
				Open: []Opening{{"backup", BySchedule}},
				Next: map[TaskID]time.Time{"housekeeping": at("2026-10-16T04:00:00Z")},
			},
		},
		{
			name:  "Housekeeping waits while Backup is open",
			now:   "2026-10-16T04:30:00Z",
			knobs: baseKnobs("2026-10-15T00:00:00Z"),
			states: []state{
				{task: backup, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-15T04:00:00Z"), open: true},
				{task: housekeeping, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-15T04:20:00Z")},
			},
			want: Plan{Next: map[TaskID]time.Time{"housekeeping": at("2026-10-16T04:00:00Z")}},
		},
		{
			name:  "a window task due after the window closed waits for the next one",
			now:   "2026-10-16T06:30:00Z",
			knobs: baseKnobs("2026-10-15T00:00:00Z"),
			states: []state{
				{task: housekeeping, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-15T04:20:00Z")},
			},
			want: Plan{
				Next: map[TaskID]time.Time{"housekeeping": at("2026-10-17T04:00:00Z")},
				Wake: at("2026-10-17T04:00:00Z"),
			},
		},
		{
			name:   "a window that passed during downtime waits for tomorrow under Wait",
			now:    "2026-10-16T08:00:00Z",
			knobs:  baseKnobs("2026-10-16T08:00:00Z"),
			states: []state{{task: housekeeping, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-15T04:00:00Z")}},
			want: Plan{
				Pass: []Passed{{"housekeeping", at("2026-10-16T04:00:00Z")}},
				Next: map[TaskID]time.Time{"housekeeping": at("2026-10-17T04:00:00Z")},
				Wake: at("2026-10-17T04:00:00Z"),
			},
		},
		{
			name:   "a window that passed during downtime opens at clock start under Catch up",
			now:    "2026-10-16T08:00:00Z",
			knobs:  baseKnobs("2026-10-16T08:00:00Z"),
			states: []state{{task: backup, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-15T04:00:00Z")}},
			want:   Plan{Open: []Opening{{"backup", ByCatchUp}}, Next: map[TaskID]time.Time{}},
		},
		{
			name:   "a run cut short by a stop reruns at start under Wait too",
			now:    "2026-10-16T08:00:00Z",
			knobs:  baseKnobs("2026-10-16T08:00:00Z"),
			states: []state{{task: housekeeping, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-15T04:00:00Z"), lastEnded: at("2026-10-16T04:01:00Z"), cut: true}},
			want:   Plan{Open: []Opening{{"housekeeping", ByCatchUp}}, Next: map[TaskID]time.Time{}},
		},
		{
			name:   "AtStart opens at clock start",
			now:    "2026-10-16T01:00:00Z",
			knobs:  baseKnobs("2026-10-16T01:00:00Z"),
			states: []state{{task: updateCheck, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-16T00:00:00Z"), atStart: true}},
			want:   Plan{Open: []Opening{{"update-check", BySchedule}}, Next: map[TaskID]time.Time{}},
		},
		{
			name:   "a daily AtStart task opens at a fresh install's first start under Wait",
			now:    "2026-10-16T01:00:00Z",
			knobs:  waitGlobally,
			states: []state{{task: catalogRefresh, firstSeen: at("2026-10-16T01:00:00Z"), atStart: true}},
			want:   Plan{Open: []Opening{{"indexer-catalog-refresh", BySchedule}}, Next: map[TaskID]time.Time{}},
		},
		{
			name:   "a daily AtStart task that ran an hour ago opens again at a restart",
			now:    "2026-10-16T01:00:00Z",
			knobs:  baseKnobs("2026-10-16T01:00:00Z"),
			states: []state{{task: catalogRefresh, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-16T00:00:00Z"), lastEnded: at("2026-10-16T00:01:00Z"), atStart: true}},
			want:   Plan{Open: []Opening{{"indexer-catalog-refresh", BySchedule}}, Next: map[TaskID]time.Time{}},
		},
		{
			name:   "a daily AtStart task waits a day after its start run",
			now:    "2026-10-16T01:00:00Z",
			knobs:  baseKnobs("2026-10-16T01:00:00Z"),
			states: []state{{task: catalogRefresh, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-16T01:00:00Z"), lastEnded: at("2026-10-16T01:00:00Z")}},
			want: Plan{
				Next: map[TaskID]time.Time{"indexer-catalog-refresh": at("2026-10-17T01:00:00Z")},
				Wake: at("2026-10-17T01:00:00Z"),
			},
		},
		{
			name:   "a unit due on return opens a returned run",
			now:    "2026-10-16T01:00:00Z",
			knobs:  baseKnobs("2026-10-15T00:00:00Z"),
			states: []state{{task: syncTask, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-16T00:00:00Z"), returned: true}},
			want:   Plan{Open: []Opening{{"sync", ByReturn}}, Next: map[TaskID]time.Time{}},
		},
		{
			name:   "the 2026-03-29 Europe/London DST day keeps 04:00",
			now:    "2026-03-29T00:00:00Z",
			knobs:  inLondon,
			states: []state{{task: housekeeping, firstSeen: at("2026-03-01T00:00:00Z"), slot: at("2026-03-28T04:00:00Z")}},
			want: Plan{
				Next: map[TaskID]time.Time{"housekeeping": at("2026-03-29T03:00:00Z")},
				Wake: at("2026-03-29T03:00:00Z"),
			},
		},
		{
			name:   "04:00 BST opens the window",
			now:    "2026-03-29T03:00:00Z",
			knobs:  inLondon,
			states: []state{{task: housekeeping, firstSeen: at("2026-03-01T00:00:00Z"), slot: at("2026-03-28T04:00:00Z")}},
			want:   Plan{Open: []Opening{{"housekeeping", BySchedule}}, Next: map[TaskID]time.Time{}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := inUTC(plan(at(tc.now), tc.knobs, tc.states))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("plan =\n%+v\nwant\n%+v", got, tc.want)
			}
		})
	}
}

func inUTC(p Plan) Plan {
	for i := range p.Pass {
		p.Pass[i].Slot = p.Pass[i].Slot.UTC()
	}
	for id, next := range p.Next {
		p.Next[id] = next.UTC()
	}
	if !p.Wake.IsZero() {
		p.Wake = p.Wake.UTC()
	}
	return p
}

func install(s string) role.NodeID {
	id, err := storage.ParseUUID(s)
	if err != nil {
		panic(err)
	}
	return role.NodeID{UUID: id}
}

var (
	here  = install("019a0000-0000-7000-8000-000000000001")
	there = install("019a0000-0000-7000-8000-000000000002")

	spread = Task{ID: "refresh", Name: "Refresh", Cadence: InWindow{Days: daily, Spread: 30 * time.Minute}, Overdue: Wait}
)

// startOf is when a spread task last run at slot starts next, as a clock
// started at started on install sees it at now.
func startOf(t *testing.T, id role.NodeID, started, now, slot string) time.Time {
	t.Helper()
	k := baseKnobs(started)
	k.install = id
	p := plan(at(now), k, []state{{task: spread, firstSeen: at("2026-10-01T00:00:00Z"), slot: at(slot)}})
	next, ok := p.Next[spread.ID]
	if !ok {
		t.Fatalf("plan = %+v, want refresh waiting", p)
	}
	return next.UTC()
}

func TestTheSpreadIsFixedPerOpeningAndInstall(t *testing.T) {
	first := startOf(t, here, "2026-10-15T00:00:00Z", "2026-10-16T01:00:00Z", "2026-10-15T04:00:00Z")
	opening := at("2026-10-16T04:00:00Z")
	if first.Before(opening) || !first.Before(opening.Add(30*time.Minute)) {
		t.Fatalf("the start is %s, want it in the window's first 30 minutes", first)
	}
	if restarted := startOf(t, here, "2026-10-16T00:30:00Z", "2026-10-16T01:00:00Z", "2026-10-15T04:00:00Z"); !restarted.Equal(first) {
		t.Fatalf("after a restart the start is %s, want %s", restarted, first)
	}
	if later := startOf(t, here, "2026-10-15T00:00:00Z", "2026-10-16T03:59:00Z", "2026-10-15T04:00:00Z"); !later.Equal(first) {
		t.Fatalf("planned again later the start is %s, want %s", later, first)
	}
	if elsewhere := startOf(t, there, "2026-10-15T00:00:00Z", "2026-10-16T01:00:00Z", "2026-10-15T04:00:00Z"); elsewhere.Equal(first) {
		t.Fatalf("another install starts at %s too, want its own offset", first)
	}
	next := startOf(t, here, "2026-10-15T00:00:00Z", "2026-10-17T01:00:00Z", "2026-10-16T04:00:00Z")
	if next.Sub(first) == day {
		t.Fatalf("the next opening starts at %s, the same offset as %s, want its own", next, first)
	}
}

func TestTheSpreadKeepsCatalogOrderInTheWindow(t *testing.T) {
	k := baseKnobs("2026-10-15T00:00:00Z")
	k.install = here
	ranToday := func(task Task) state {
		return state{task: task, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-16T04:00:00Z"), lastEnded: at("2026-10-16T04:01:00Z")}
	}
	dueToday := func(task Task) state {
		return state{task: task, firstSeen: at("2026-10-01T00:00:00Z"), slot: at("2026-10-15T04:00:00Z")}
	}
	waiting := plan(at("2026-10-16T04:01:00Z"), k, []state{ranToday(backup), dueToday(spread), dueToday(housekeeping)})
	start := waiting.Next["refresh"]
	if len(waiting.Open) != 0 || !start.After(at("2026-10-16T04:01:00Z")) || !waiting.Wake.Equal(start) {
		t.Fatalf("at 04:01 plan = %+v, want refresh and housekeeping waiting for refresh's start", waiting)
	}
	if got := plan(start, k, []state{ranToday(backup), dueToday(spread), dueToday(housekeeping)}).Open; !reflect.DeepEqual(got, []Opening{{"refresh", BySchedule}}) {
		t.Fatalf("at refresh's start plan opens %v, want refresh alone", got)
	}
	open := dueToday(spread)
	open.open = true
	if got := plan(start, k, []state{ranToday(backup), open, dueToday(housekeeping)}).Open; len(got) != 0 {
		t.Fatalf("while refresh runs plan opens %v, want housekeeping waiting", got)
	}
	done := ranToday(spread)
	done.slot = start
	if got := plan(start.Add(time.Minute), k, []state{ranToday(backup), done, dueToday(housekeeping)}).Open; !reflect.DeepEqual(got, []Opening{{"housekeeping", BySchedule}}) {
		t.Fatalf("after refresh plan opens %v, want housekeeping", got)
	}
}
