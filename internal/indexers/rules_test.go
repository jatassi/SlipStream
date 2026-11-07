package indexers

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

var (
	home   = role.NodeID{UUID: storage.NewUUID()}
	seed   = role.NodeID{UUID: storage.NewUUID()}
	t0     = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	ladder = Timings{Limited: time.Minute, Ladder: []time.Duration{time.Minute, 5 * time.Minute, time.Hour}}
)

func ok(u Use, node role.NodeID) answer { return answer{use: u, node: node, revision: 1} }

func failed(node role.NodeID, err error) answer {
	return answer{use: UseSearch, node: node, revision: 1, outcome: err}
}

func TestNext(t *testing.T) {
	failing := &searching.Failing{Err: errors.New("HTTP 500")}
	captcha := &searching.Stopped{Cause: searching.CaptchaNeeded, Text: "captcha"}
	refused := &searching.Stopped{Cause: searching.AccountRefused, Text: "banned"}
	later := t0.Add(time.Minute)
	for _, tc := range []struct {
		name string
		from Status
		a    answer
		want Status
	}{
		{
			"a failure climbs the first rung", Status{State: OK}, failed(home, failing),
			Status{State: Failing, Rung: 1, Since: t0, Until: t0.Add(time.Minute), Last: &Failure{At: t0, Text: "HTTP 500"}, Revision: 1},
		},
		{
			"another failure climbs and keeps since", Status{State: Failing, Rung: 1, Since: t0.Add(-time.Hour), Until: t0}, failed(home, failing),
			Status{State: Failing, Rung: 2, Since: t0.Add(-time.Hour), Until: t0.Add(5 * time.Minute), Last: &Failure{At: t0, Text: "HTTP 500"}, Revision: 1},
		},
		{
			"the top rung holds", Status{State: Failing, Rung: 3, Since: t0, Until: t0}, failed(home, failing),
			Status{State: Failing, Rung: 3, Since: t0, Until: t0.Add(time.Hour), Last: &Failure{At: t0, Text: "HTTP 500"}, Revision: 1},
		},
		{
			"a success steps down and clears", Status{State: Failing, Rung: 2, Since: t0, Until: later, Last: &Failure{At: t0}}, ok(UseSearch, home),
			Status{State: OK, Rung: 1, Success: t0, Last: &Failure{At: t0}, Revision: 1},
		},
		{
			"limited waits Retry-After", Status{State: OK}, failed(home, &searching.Limited{RetryAfter: 5 * time.Minute}),
			Status{State: Limited, Until: t0.Add(5 * time.Minute), Revision: 1},
		},
		{
			"limited without a time waits Timings.Limited", Status{State: OK}, failed(home, &searching.Limited{}),
			Status{State: Limited, Until: t0.Add(time.Minute), Revision: 1},
		},
		{
			"a node-bound stop off home waits for home", Status{State: OK}, failed(seed, captcha),
			Status{State: HomeOnly, Since: t0, Cause: searching.CaptchaNeeded, Text: "captcha", Node: seed, Last: &Failure{At: t0, Text: captcha.Error()}, Revision: 1},
		},
		{
			"a node-bound stop on home stops", Status{State: OK}, failed(home, captcha),
			Status{State: Stopped, Since: t0, Cause: searching.CaptchaNeeded, Text: "captcha", Node: home, Last: &Failure{At: t0, Text: captcha.Error()}, Revision: 1},
		},
		{
			"a stop isn't softened to a wait for home", Status{State: Stopped, Since: t0, Cause: searching.AccountRefused, Node: home}, failed(seed, captcha),
			Status{State: Stopped, Since: t0, Cause: searching.AccountRefused, Node: home, Last: &Failure{At: t0, Text: captcha.Error()}, Revision: 1},
		},
		{
			"a search's success leaves a stop", Status{State: Stopped, Since: t0, Cause: searching.AccountRefused, Node: home}, ok(UseSearch, home),
			Status{State: Stopped, Since: t0, Cause: searching.AccountRefused, Node: home, Success: t0, Revision: 1},
		},
		{
			"a passing Test clears a stop", Status{State: Stopped, Since: t0, Cause: searching.AccountRefused, Node: home}, ok(UseTest, home),
			Status{State: OK, Success: t0, Revision: 1},
		},
		{
			"a limit leaves a stop", Status{State: Stopped, Since: t0, Cause: searching.AccountRefused}, failed(home, &searching.Limited{}),
			Status{State: Stopped, Since: t0, Cause: searching.AccountRefused, Revision: 1},
		},
		{
			"any answer from home ends a wait for home", Status{State: HomeOnly, Since: t0, Cause: searching.CaptchaNeeded, Node: seed}, failed(home, &searching.Limited{}),
			Status{State: Limited, Until: t0.Add(time.Minute), Revision: 1},
		},
		{
			"home's own stop replaces a wait for home", Status{State: HomeOnly, Since: t0, Cause: searching.CaptchaNeeded, Node: seed}, failed(home, refused),
			Status{State: Stopped, Since: t0, Cause: searching.AccountRefused, Text: "banned", Node: home, Last: &Failure{At: t0, Text: refused.Error()}, Revision: 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := next(tc.from, tc.a, home, t0, ladder); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("next = %+v\nwant   %+v", got, tc.want)
			}
		})
	}
}

// roster holds Searching on holders, the first being the Controller, or
// the Controller is covering, holding none.
type roster struct {
	holders  []role.NodeID
	covering role.NodeID
	away     map[role.NodeID]nodes.Absence
}

func (r roster) Self() role.NodeID {
	if !r.covering.IsZero() {
		return r.covering
	}
	return r.holders[0]
}

func (r roster) Holding(role.Name) []role.NodeID { return r.holders }
func (r roster) Name(n role.NodeID) string       { return n.String() }
func (r roster) Away(_ role.Name, n role.NodeID) (nodes.Absence, bool) {
	a, ok := r.away[n]
	return a, ok
}

func (r roster) may(n role.NodeID) bool {
	_, away := r.away[n]
	return (slices.Contains(r.holders, n) || n == r.covering && !n.IsZero()) && !away
}

func (r roster) Pick(name role.Name, prefer role.NodeID) (role.NodeID, bool) {
	if r.may(prefer) {
		return prefer, true
	}
	return r.PickOther(name, prefer)
}

func (r roster) PickOther(_ role.Name, besides role.NodeID) (role.NodeID, bool) {
	for _, n := range append([]role.NodeID{r.covering}, r.holders...) {
		if n != besides && r.may(n) {
			return n, true
		}
	}
	return role.NodeID{}, false
}

func TestRoute(t *testing.T) {
	inGrace := nodes.Absence{Since: t0}
	lapsed := nodes.Absence{Since: t0, Lapsed: true}
	feed := Indexer{Kind: Torznab, Home: home}
	catalog := Indexer{Kind: Catalog, Home: home, Source: Definition{Login: cardigann.SessionLogin}}
	failover := Indexer{Kind: Catalog, Home: home, Source: Definition{Login: cardigann.SessionLogin}, AllowFailover: true}
	keyed := Indexer{Kind: Catalog, Home: home, Source: Definition{Login: cardigann.StatelessLogin}}
	cookie := Indexer{Kind: RSS, Home: home, Source: Feed{}, Pasted: true}
	cookieFailover := Indexer{Kind: RSS, Home: home, Source: Feed{}, Pasted: true, AllowFailover: true}
	both := []role.NodeID{seed, home}
	covered := roster{holders: []role.NodeID{home}, covering: seed, away: map[role.NodeID]nodes.Absence{home: lapsed}}
	for _, tc := range []struct {
		name string
		r    roster
		idx  Indexer
		st   Status
		to   role.NodeID
		wait *Outcome
	}{
		{"home while it may", roster{holders: both}, catalog, Status{}, home, nil},
		{"wait inside the grace", roster{holders: both, away: map[role.NodeID]nodes.Absence{home: inGrace}}, feed, Status{}, role.NodeID{}, &Outcome{Kind: Waiting, Node: home, Since: t0}},
		{"a feed moves past the grace", roster{holders: both, away: map[role.NodeID]nodes.Absence{home: lapsed}}, feed, Status{}, seed, nil},
		{"a session stays past the grace", roster{holders: both, away: map[role.NodeID]nodes.Absence{home: lapsed}}, catalog, Status{}, role.NodeID{}, &Outcome{Kind: Waiting, Node: home, Since: t0, Lapsed: true}},
		{"a catalog Indexer that keeps no session moves past the grace", roster{holders: both, away: map[role.NodeID]nodes.Absence{home: lapsed}}, keyed, Status{}, seed, nil},
		{"failover moves a session", roster{holders: both, away: map[role.NodeID]nodes.Absence{home: lapsed}}, failover, Status{}, seed, nil},
		{"a wait for home never moves", roster{holders: both, away: map[role.NodeID]nodes.Absence{home: lapsed}}, feed, Status{State: HomeOnly}, role.NodeID{}, &Outcome{Kind: Waiting, Node: home, Since: t0, Lapsed: true}},
		{"no other holder", roster{holders: []role.NodeID{home}, away: map[role.NodeID]nodes.Absence{home: lapsed}}, feed, Status{}, role.NodeID{}, &Outcome{Kind: Waiting, Node: home, Since: t0, Lapsed: true}},
		{"a pasted cookie stays past the grace", roster{holders: both, away: map[role.NodeID]nodes.Absence{home: lapsed}}, cookie, Status{}, role.NodeID{}, &Outcome{Kind: Waiting, Node: home, Since: t0, Lapsed: true}},
		{"failover moves a pasted cookie", roster{holders: both, away: map[role.NodeID]nodes.Absence{home: lapsed}}, cookieFailover, Status{}, seed, nil},
		{"the Controller covering takes a feed", covered, feed, Status{}, seed, nil},
		{"the Controller covering takes a catalog Indexer that keeps no session", covered, keyed, Status{}, seed, nil},
		{"a session waits while the Controller covers", covered, catalog, Status{}, role.NodeID{}, &Outcome{Kind: Waiting, Node: home, Since: t0, Lapsed: true}},
		{"the Controller covering takes a session with failover", covered, failover, Status{}, seed, nil},
		{"a wait for home stays while the Controller covers", covered, failover, Status{State: HomeOnly}, role.NodeID{}, &Outcome{Kind: Waiting, Node: home, Since: t0, Lapsed: true}},
		{"home back in step ends the move", roster{holders: both}, failover, Status{State: HomeOnly}, home, nil},
		{"a feed homed off the holders goes to one", roster{holders: []role.NodeID{seed}}, feed, Status{}, seed, nil},
		{"a session homed off the holders waits", roster{holders: []role.NodeID{seed}}, catalog, Status{}, role.NodeID{}, &Outcome{Kind: Waiting, Node: home}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			to, wait := route(tc.r, tc.idx, tc.st)
			if to != tc.to || !reflect.DeepEqual(wait, tc.wait) {
				t.Fatalf("route = %s, %+v; want %s, %+v", to, wait, tc.to, tc.wait)
			}
		})
	}
}

func TestGate(t *testing.T) {
	r := roster{holders: []role.NodeID{home}}
	idx := Indexer{Kind: Torznab, Home: home}
	later := t0.Add(time.Minute)
	for _, tc := range []struct {
		name string
		st   Status
		used Usage
		u    Use
		to   role.NodeID
		wait *Outcome
	}{
		{"ok routes", Status{State: OK}, Usage{}, UseSearch, home, nil},
		{"stopped", Status{State: Stopped, Cause: searching.AccountRefused, Text: "banned"}, Usage{}, UseSearch, role.NodeID{}, &Outcome{Kind: IsStopped, Cause: searching.AccountRefused, Text: "banned"}},
		{"limited inside its wait", Status{State: Limited, Until: later}, Usage{}, UseSearch, role.NodeID{}, &Outcome{Kind: IsLimited, Until: later}},
		{"limited past its wait", Status{State: Limited, Until: t0}, Usage{}, UseSearch, home, nil},
		{"failing inside its backoff", Status{State: Failing, Until: later}, Usage{}, UseFetch, role.NodeID{}, &Outcome{Kind: IsFailing, Until: later}},
		{"searches at their limit", Status{}, Usage{SearchesFree: later}, UseRecent, role.NodeID{}, &Outcome{Kind: LimitReached, Until: later}},
		{"grabs at their limit leave searches", Status{}, Usage{GrabsFree: later}, UseSearch, home, nil},
		{"a Test counts against no limit", Status{}, Usage{SearchesFree: later, GrabsFree: later}, UseTest, home, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			to, wait := gate(idx, tc.st, tc.used, r, tc.u, t0)
			if to != tc.to || !reflect.DeepEqual(wait, tc.wait) {
				t.Fatalf("gate = %s, %+v; want %s, %+v", to, wait, tc.to, tc.wait)
			}
		})
	}
}

func TestAFetchDropsOnlyWhenItWouldWaitForTheAdmin(t *testing.T) {
	lapsed := roster{holders: []role.NodeID{home, seed}, away: map[role.NodeID]nodes.Absence{home: {Since: t0, Lapsed: true}}}
	inGrace := roster{holders: []role.NodeID{home, seed}, away: map[role.NodeID]nodes.Absence{home: {Since: t0}}}
	later := t0.Add(time.Minute)
	feed := Indexer{Kind: Torznab, Home: home}
	catalog := Indexer{Kind: Catalog, Home: home, Source: Definition{Login: cardigann.SessionLogin}}
	for _, tc := range []struct {
		name string
		r    roster
		idx  Indexer
		st   Status
		want error
	}{
		{"fetches while ok", roster{holders: []role.NodeID{home}}, feed, Status{State: OK}, nil},
		{"drops once stopped", roster{holders: []role.NodeID{home}}, feed, Status{State: Stopped, Cause: searching.AccountRefused}, &DropError{Why: DropStopped}},
		{"waits while limited", roster{holders: []role.NodeID{home}}, feed, Status{State: Limited, Until: later}, &WaitError{Outcome: Outcome{Kind: IsLimited, Until: later}}},
		{"waits while failing", roster{holders: []role.NodeID{home}}, feed, Status{State: Failing, Until: later}, &WaitError{Outcome: Outcome{Kind: IsFailing, Until: later}}},
		{"waits for home inside the grace", inGrace, catalog, Status{}, &WaitError{Outcome: Outcome{Kind: Waiting, Node: home, Since: t0}}},
		{"drops a session past the grace", lapsed, catalog, Status{}, &DropError{Why: DropHomeOffline}},
		{"a feed moves past the grace", lapsed, feed, Status{}, nil},
		{"drops a wait for home past the grace", lapsed, feed, Status{State: HomeOnly}, &DropError{Why: DropHomeOffline}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := &Indexers{cfg: Config{Roster: tc.r}}
			var got any = x.fetchGate(tc.idx, tc.st, t0)
			if want := any(tc.want); !reflect.DeepEqual(got, want) {
				t.Fatalf("fetchGate = %#v, want %#v", got, want)
			}
		})
	}
}

func TestFree(t *testing.T) {
	three := Limit{Count: 3, Per: Hour}
	at := func(ago ...time.Duration) []time.Time {
		out := make([]time.Time, len(ago))
		for i, d := range ago {
			out[i] = t0.Add(-d)
		}
		return out
	}
	for _, tc := range []struct {
		name     string
		l        Limit
		reserved []time.Time
		want     time.Time
	}{
		{"no limit", Limit{}, at(1, 2, 3, 4), time.Time{}},
		{"under the limit", three, at(time.Minute, 2*time.Minute), time.Time{}},
		{"old calls left the window", three, at(time.Minute, 61*time.Minute, 2*time.Hour), time.Time{}},
		{"at the limit, the oldest leaves first", three, at(10*time.Minute, 40*time.Minute, 20*time.Minute), t0.Add(20 * time.Minute)},
		{"past the limit, enough must leave", three, at(10*time.Minute, 40*time.Minute, 20*time.Minute, 30*time.Minute), t0.Add(30 * time.Minute)},
		{"a day window", Limit{Count: 1, Per: Day}, at(5 * time.Hour), t0.Add(19 * time.Hour)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := free(tc.l, tc.reserved, t0); !got.Equal(tc.want) {
				t.Fatalf("free = %s, want %s", got, tc.want)
			}
		})
	}
}

// saved opens a library holding one Torznab Indexer homed on home, with
// two searches an hour.
func saved(t *testing.T) (*Indexers, *storage.DB[file], Indexer) {
	t.Helper()
	sealer := secrettest.Sealer(t)
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), Schema())
	idx := Indexer{
		ID: ID{UUID: storage.NewUUID()}, Name: "Fixture Site", Kind: Torznab, Home: home, On: true, Priority: DefaultPriority,
		Searches: Limit{Count: 2, Per: Hour}, Grabs: Limit{Per: Hour}, Source: Feed{}, FeedHost: "site.test", Revision: 2, Created: t0,
	}
	x := New(Config{Library: lib, Sealer: sealer, Timings: ladder})
	sealed, err := x.seal(idx.ID, Secrets{Address: "https://site.test/torznab", APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	idx.Secrets = sealed
	err = lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)", home); err != nil {
			return err
		}
		p, err := row(idx, storage.Now())
		if err != nil {
			return err
		}
		return db.New(tx).InsertIndexer(ctx, p)
	})
	if err != nil {
		t.Fatal(err)
	}
	return x, lib, idx
}

func TestReserveRefusesPastTheLimitAndCountsRecentWithSearches(t *testing.T) {
	_, lib, idx := saved(t)
	take := func(u Use, at time.Time) error {
		return lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
			_, err := reserve(ctx, tx, idx, u, storage.FromTime(at))
			return err
		})
	}
	for _, u := range []Use{UseSearch, UseRecent} {
		if err := take(u, t0); err != nil {
			t.Fatalf("reserve %s under the limit: %v", u, err)
		}
	}
	var limit *LimitError
	if err := take(UseSearch, t0.Add(time.Minute)); !errors.As(err, &limit) || !limit.Until.Equal(t0.Add(time.Hour)) {
		t.Fatalf("a third search in the hour = %v, want *LimitError until %s", err, t0.Add(time.Hour))
	}
	for _, u := range []Use{UseFetch, UseTest} {
		if err := take(u, t0.Add(time.Minute)); err != nil {
			t.Errorf("reserve %s with searches used up: %v", u, err)
		}
	}
	if err := take(UseSearch, t0.Add(time.Hour)); err != nil {
		t.Errorf("a search once the first left the window: %v", err)
	}
	var n int
	if err := lib.QueryRowContext(t.Context(), "SELECT count(*) FROM indexers_query WHERE indexer_id = ?", idx.ID).Scan(&n); err != nil || n != 5 {
		t.Fatalf("query rows = %d, %v; want 5, the refused one not among them", n, err)
	}
}

func TestRecordWritesNothingForAnOlderRevisionButItsQuery(t *testing.T) {
	x, lib, idx := saved(t)
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
		q, err := reserve(ctx, tx, idx, UseSearch, storage.FromTime(t0))
		if err != nil {
			return err
		}
		st, err := x.recordIn(ctx, tx, q, idx.ID, answer{use: UseSearch, node: home, revision: 1, outcome: &searching.Failing{Err: errors.New("HTTP 500")}}, storage.FromTime(t0))
		if err != nil || st.State != "" {
			t.Errorf("record of a revision-1 answer to a revision-2 Indexer = %+v, %v; want no status", st, err)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var outcome string
	var statuses int
	if err := lib.QueryRowContext(t.Context(), "SELECT outcome, (SELECT count(*) FROM indexers_status) FROM indexers_query").Scan(&outcome, &statuses); err != nil {
		t.Fatal(err)
	}
	if outcome != "failing" || statuses != 0 {
		t.Fatalf("the stale answer left its query %q and %d statuses, want failing and none", outcome, statuses)
	}
}

func TestASessionReplayOrAnOlderOneChangesNothing(t *testing.T) {
	x, lib, idx := saved(t)
	offer := func(s searching.Session) {
		t.Helper()
		err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
			q, err := reserve(ctx, tx, idx, UseTest, storage.FromTime(t0))
			if err != nil {
				return err
			}
			_, err = x.recordIn(ctx, tx, q, idx.ID, answer{use: UseTest, node: home, revision: 2, session: &s}, storage.FromTime(t0))
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	cookie := func(v string) []searching.Cookie {
		return []searching.Cookie{{Name: "uid", Value: v, Domain: "site.test", Path: "/"}}
	}
	newest := searching.Session{Cookies: cookie("new"), UserAgent: "UA", Earned: t0}
	offer(newest)
	offer(newest)
	offer(searching.Session{Cookies: cookie("old"), UserAgent: "UA", Earned: t0.Add(-time.Hour)})
	got, err := x.session(t.Context(), lib, idx.ID, home)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || !reflect.DeepEqual(got.Cookies, newest.Cookies) || !got.Earned.Equal(t0) {
		t.Fatalf("session after a replay and an older answer = %+v, want the newest", got)
	}
	newer := searching.Session{Cookies: cookie("newer"), UserAgent: "UA", Earned: t0.Add(time.Minute)}
	offer(newer)
	if got, err = x.session(t.Context(), lib, idx.ID, home); err != nil || !reflect.DeepEqual(got.Cookies, newer.Cookies) {
		t.Fatalf("session after a newer answer = %+v, %v; want the newer", got, err)
	}
}

func TestQueryTakesEveryIDTheModeListsElseText(t *testing.T) {
	caps := Caps{Modes: map[string][]string{
		"search":   {"q"},
		"tvsearch": {"q", "season", "tvdbid", "imdbid"},
	}}
	ids := map[string]string{"imdbid": "tt0000901", "tvdbid": "901", "tmdbid": "77"}
	for _, tc := range []struct {
		name  string
		terms Terms
		want  searching.Query
	}{
		{"every id the caps list", Terms{Mode: "tvsearch", IDs: ids, Query: "Lantern Coast"}, searching.Query{Mode: "tvsearch", IDs: map[string]string{"tvdbid": "901", "imdbid": "tt0000901"}}},
		{"an id the type lacks is skipped", Terms{Mode: "tvsearch", IDs: map[string]string{"imdbid": "tt0000901"}, Query: "Lantern Coast"}, searching.Query{Mode: "tvsearch", IDs: map[string]string{"imdbid": "tt0000901"}}},
		{"no id the caps list", Terms{Mode: "tvsearch", IDs: map[string]string{"tmdbid": "901"}, Query: "Lantern Coast"}, searching.Query{Mode: "search", Text: "Lantern Coast"}},
		{"a mode the caps lack", Terms{Mode: "book", IDs: ids, Query: "Lantern Coast"}, searching.Query{Mode: "search", Text: "Lantern Coast"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := query(caps, tc.terms); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("query = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestServedIsTheIndexersCategoriesInTheTypesFamilies(t *testing.T) {
	movies := []newznab.ID{2000}
	for _, tc := range []struct {
		name string
		idx  Indexer
		want []newznab.ID
	}{
		{"every advertised category", Indexer{Caps: Caps{Categories: []newznab.ID{2000, 2040, 5000}}}, []newznab.ID{2000, 2040}},
		{"the categories set on it", Indexer{Categories: []newznab.ID{2040, 5040}, Caps: Caps{Categories: []newznab.ID{2000, 2040, 5000}}}, []newznab.ID{2040}},
		{"none of the type's", Indexer{Caps: Caps{Categories: []newznab.ID{5000, 5040}}}, nil},
		{"none advertised", Indexer{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := served(tc.idx, movies); !slices.Equal(got, tc.want) {
				t.Fatalf("served = %v, want %v", got, tc.want)
			}
		})
	}
}
