package indexers_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// site is a Node whose every job answers outcome, with report, at once.
type site struct {
	mu      sync.Mutex
	outcome error
	report  *searching.Report
	jobs    int
}

func (s *site) answer(outcome error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outcome = outcome
}

func (s *site) Run(context.Context, searching.Job) (searching.Poll, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs++
	now := time.Now()
	return searching.Poll{Started: now, Answer: &searching.Answer{Started: now, Outcome: s.outcome, Report: s.report}}, nil
}

type healthWorld struct {
	x      *indexers.Indexers
	lib    *storage.DB[storage.Library]
	site   *site
	roster indexerstest.Roster
	box    role.NodeID
	id     indexers.ID
}

// newHealthWorld saves Fixture Site, a Torznab Indexer homed on seedbox,
// which holds Searching beside home.
func newHealthWorld(t *testing.T) *healthWorld {
	t.Helper()
	sealer := secrettest.Sealer(t)
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), works.Schema(), indexers.Schema())
	home, box := role.NodeID{UUID: storage.NewUUID()}, role.NodeID{UUID: storage.NewUUID()}
	w := &healthWorld{
		lib: lib, site: &site{}, box: box, id: indexers.ID{UUID: storage.NewUUID()},
		roster: indexerstest.Roster{Controller: home, Holders: []role.NodeID{home, box}, Absent: map[role.NodeID]nodes.Absence{}, Names: map[role.NodeID]string{home: "home", box: "seedbox"}},
	}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := nodestest.Insert(ctx, tx, home, "home"); err != nil {
			return err
		}
		if err := nodestest.Insert(ctx, tx, box, "seedbox"); err != nil {
			return err
		}
		return indexerstest.Torznab(ctx, tx, sealer, w.id, "Fixture Site", box, "https://site.test/torznab", "k")
	})
	if err != nil {
		t.Fatal(err)
	}
	w.x = indexers.New(indexers.Config{
		Library: lib, Sealer: sealer, Roster: w.roster,
		Searching: func(role.NodeID) searching.Remote { return w.site },
		Types:     map[works.TypeID]indexers.Facet{indexerstest.Type: indexerstest.Facet{}},
		Changed:   discard[indexers.Changed]{}, Status: discard[indexers.StatusChanged]{}, Settings: discard[indexers.SettingsChanged]{},
		Timings: indexers.Timings{Job: 2 * time.Minute, Poll: time.Second, Limited: time.Minute, Ladder: []time.Duration{time.Minute, time.Hour}},
		Zone:    time.UTC, Log: slog.New(slog.DiscardHandler),
	})
	return w
}

// test runs Fixture Site's Test with the site answering outcome.
func (w *healthWorld) test(t *testing.T, outcome error) {
	t.Helper()
	w.site.answer(outcome)
	_, err := w.x.Test(t.Context(), identity.Admin{}, w.id)
	if failed := new(indexers.TestFailedError); (outcome == nil) != (err == nil) || (err != nil && !errors.As(err, &failed)) {
		t.Fatalf("Test with the site answering %v = %v", outcome, err)
	}
}

func (w *healthWorld) answer(t *testing.T) health.Answer {
	t.Helper()
	a, err := w.x.HealthSource().Answer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func (w *healthWorld) key(kind string) health.Key { return health.KeyOf(kind, w.id.String()) }

func TestEachStopRaisesOneFailThatNamesItsFixAndLinksToTheIndexer(t *testing.T) {
	cases := []struct {
		stop *searching.Stopped
		want string
	}{
		{&searching.Stopped{Cause: searching.CredentialsRejected}, "Fixture Site rejected the username or password. Update them on the Indexer and test it."},
		{&searching.Stopped{Cause: searching.CaptchaNeeded}, "Fixture Site needs a captcha to sign in again. Open the Indexer and type the code."},
		{&searching.Stopped{Cause: searching.ChallengeUnsolved}, "Fixture Site is behind a browser challenge, and seedbox has no Challenge solver. Set one on seedbox, or paste a browser cookie on the Indexer."},
		{&searching.Stopped{Cause: searching.ChallengeUnsolved, SolverTried: true}, "seedbox's Challenge solver couldn't clear Fixture Site's challenge. Paste a browser cookie on the Indexer."},
		{&searching.Stopped{Cause: searching.CookieExpired}, "Fixture Site no longer accepts the browser cookie. Paste a new one on the Indexer."},
		{&searching.Stopped{Cause: searching.DefinitionUnloadable}, "Fixture Site's definition doesn't load. Replace the custom definition, or wait for a catalog fix."},
		{&searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"}, "Fixture Site refused the API key with 'Account suspended'. Check the account, then test the Indexer."},
		{&searching.Stopped{Cause: searching.RequestRejected, Text: "Incorrect parameter"}, "Fixture Site refused a search with 'Incorrect parameter'. Test the Indexer after the feed is fixed."},
	}
	for _, c := range cases {
		t.Run(string(c.stop.Cause), func(t *testing.T) {
			w := newHealthWorld(t)
			w.test(t, c.stop)
			a := w.answer(t)
			if len(a.Items) != 1 || len(a.Unsure) != 0 {
				t.Fatalf("Health after a stop = %+v, want one item", a)
			}
			it := a.Items[0]
			if it.Key != w.key("stopped") || it.Level != health.Fail || it.Subject != "Fixture Site" || it.Text != c.want ||
				it.Action.Kind() != health.ActionLink || it.Action.Href() != "/settings/indexers/"+w.id.String() {
				t.Errorf("the stop's item = %+v, want a Fail %q linking to the Indexer", it, c.want)
			}
			w.test(t, nil)
			if a := w.answer(t); len(a.Items) != 0 {
				t.Errorf("Health after a passing Test = %+v, want nothing", a.Items)
			}
		})
	}
}

func TestAnIndexerThatIsOffRaisesNothing(t *testing.T) {
	w := newHealthWorld(t)
	w.test(t, &searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"})
	off := false
	if _, err := w.x.Update(t.Context(), identity.Admin{}, w.id, indexers.Edit{On: &off}); err != nil {
		t.Fatal(err)
	}
	if a := w.answer(t); len(a.Items) != 0 || len(a.Unsure) != 0 {
		t.Errorf("Health of a stopped Indexer that is off = %+v, want nothing", a)
	}
}

func TestFailingPastTheLongBackoffWarnsAtItsEdge(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newHealthWorld(t)
		began := time.Now()
		w.test(t, &searching.Failing{Err: errors.New("the Indexer responded with status 500")})
		a := w.answer(t)
		if len(a.Items) != 0 || !a.Next.Equal(began.Add(6*time.Hour)) {
			t.Fatalf("Health right after a failure = %+v, want nothing until %s", a, began.Add(6*time.Hour))
		}

		time.Sleep(6 * time.Hour)
		a = w.answer(t)
		want := "Every request to Fixture Site has failed since " + began.UTC().Format("15:04") + ". SlipStream retries automatically."
		if len(a.Items) != 1 || a.Items[0].Key != w.key("failing") || a.Items[0].Level != health.Warn || a.Items[0].Text != want || !a.Next.IsZero() {
			t.Fatalf("Health 6h into a failure = %+v, want one Warn %q and no Next", a, want)
		}

		w.test(t, &searching.Failing{Err: errors.New("the Indexer responded with status 500")})
		retry := time.Now().Add(time.Hour)
		a = w.answer(t)
		want = "Every request to Fixture Site has failed since " + began.UTC().Format("15:04") + ". SlipStream retries at " + retry.UTC().Format("15:04") + "."
		if len(a.Items) != 1 || a.Items[0].Text != want || !a.Next.Equal(retry) {
			t.Fatalf("Health after another failure = %+v, want %q with Next at the retry %s", a, want, retry)
		}

		w.test(t, nil)
		if a := w.answer(t); len(a.Items) != 0 || !a.Next.IsZero() {
			t.Errorf("Health after a passing Test = %+v, want nothing", a)
		}
	})
}

func TestTheLongBackoffIsASetting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newHealthWorld(t)
		w.test(t, &searching.Failing{Err: errors.New("the Indexer responded with status 500")})
		time.Sleep(time.Hour)
		if a := w.answer(t); len(a.Items) != 0 {
			t.Fatalf("Health 1h into a failure = %+v, want nothing", a.Items)
		}
		if err := w.x.Settings().Apply(t.Context(), setting.Patch{indexers.LongBackoff.Key: []byte("3600")}); err != nil {
			t.Fatal(err)
		}
		if a := w.answer(t); len(a.Items) != 1 || a.Items[0].Key != w.key("failing") {
			t.Errorf("Health 1h into a failure with a 1h long backoff = %+v, want the failing Warn", a.Items)
		}
	})
}

func TestAnIndexerWaitingForItsHomeIsUnsure(t *testing.T) {
	w := newHealthWorld(t)
	w.test(t, &searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"})
	since := time.Date(2026, 10, 27, 8, 0, 0, 0, time.UTC)
	w.roster.Absent[w.box] = nodes.Absence{Since: since}
	a := w.answer(t)
	want := "Fixture Site is waiting for seedbox, which has been offline since 08:00."
	if len(a.Items) != 0 || len(a.Unsure) != 1 || a.Unsure[0].Key != w.key("stopped") || a.Unsure[0].Text != want {
		t.Errorf("Health with seedbox away = %+v, want the stop unsure with %q", a, want)
	}
}

func TestTheHealthCheckTestsEachIndexerItMayRetry(t *testing.T) {
	check := func(t *testing.T, w *healthWorld) tasks.Task {
		t.Helper()
		i := slices.IndexFunc(w.x.Tasks(), func(task tasks.Task) bool { return task.ID == indexers.HealthCheckTaskID })
		if i < 0 {
			t.Fatal("the Indexers declare no health check")
		}
		return w.x.Tasks()[i]
	}
	units := func(t *testing.T, w *healthWorld, task tasks.Task) []string {
		t.Helper()
		var out []tasks.Unit
		err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			out, err = task.Units(ctx, r)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		labels := []string{}
		for _, u := range out {
			labels = append(labels, u.Label)
		}
		return labels
	}
	for _, c := range []struct {
		stop    *searching.Stopped
		retried bool
	}{
		{&searching.Stopped{Cause: searching.CredentialsRejected}, false},
		{&searching.Stopped{Cause: searching.CaptchaNeeded}, false},
		{&searching.Stopped{Cause: searching.CookieExpired}, false},
		{&searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"}, false},
		{&searching.Stopped{Cause: searching.ChallengeUnsolved}, true},
		{&searching.Stopped{Cause: searching.DefinitionUnloadable}, true},
		{&searching.Stopped{Cause: searching.RequestRejected, Text: "Incorrect parameter"}, true},
	} {
		t.Run(string(c.stop.Cause), func(t *testing.T) {
			w := newHealthWorld(t)
			task := check(t, w)
			w.test(t, c.stop)
			got := units(t, w, task)
			if want := map[bool][]string{true: {"Fixture Site"}, false: {}}[c.retried]; !slices.Equal(got, want) {
				t.Fatalf("the health check's units after %s = %q, want %q", c.stop.Cause, got, want)
			}
			var shown bool
			err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
				var err error
				shown, err = task.Shown(ctx, r)
				return err
			})
			if err != nil || !shown {
				t.Fatalf("the health check is shown %t, %v; want it shown while an Indexer exists", shown, err)
			}
			jobs := w.site.jobs
			if err := task.Run(t.Context(), tasks.Unit{Key: tasks.UnitKey(w.id.String())}, nil); err != nil {
				t.Fatalf("the health check's unit = %v, want it to succeed whatever the Test finds", err)
			}
			if ran := w.site.jobs - jobs; ran != map[bool]int{true: 1, false: 0}[c.retried] {
				t.Errorf("the health check's unit sent %d jobs for a stop of %s", ran, c.stop.Cause)
			}
		})
	}
}

func TestTheHealthCheckOfAFeedItCantUseSucceedsAndItsTestSaysWhy(t *testing.T) {
	w := newHealthWorld(t)
	w.test(t, &searching.Stopped{Cause: searching.RequestRejected, Text: "Incorrect parameter"})
	w.site.answer(nil)
	w.site.report = &searching.Report{Refusal: searching.RSSRepeatedGUID}
	i := slices.IndexFunc(w.x.Tasks(), func(task tasks.Task) bool { return task.ID == indexers.HealthCheckTaskID })
	if err := w.x.Tasks()[i].Run(t.Context(), tasks.Unit{Key: tasks.UnitKey(w.id.String())}, nil); err != nil {
		t.Fatalf("the health check's unit = %v, want it to succeed for a feed SlipStream can't use", err)
	}
	if a := w.answer(t); len(a.Items) != 0 {
		t.Errorf("Health after a Test the site answered = %+v, want the stop cleared", a.Items)
	}
	_, err := w.x.Test(t.Context(), identity.Admin{}, w.id)
	var refused *indexers.FeedRefusedError
	if want := "Some items repeat the same GUID."; !errors.As(err, &refused) || refused.Sentence != want {
		t.Errorf("Test of a feed SlipStream can't use = %v, want %q", err, want)
	}
}
