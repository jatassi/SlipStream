package indexers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

type quiet[T any] struct{}

func (quiet[T]) Publish(context.Context, T) {}

// passing is a Node whose every job passes at once.
type passing struct{}

func (passing) Run(context.Context, searching.Job) (searching.Poll, error) {
	now := time.Now()
	return searching.Poll{Started: now, Answer: &searching.Answer{Started: now}}, nil
}

// onTwo is saved's Fixture Site with seed holding Searching beside home,
// and a session on each pasted for it.
func onTwo(t *testing.T) (*Indexers, *storage.DB[file], Indexer) {
	t.Helper()
	x, lib, idx := saved(t)
	x.cfg.Roster = roster{holders: []role.NodeID{home, seed}}
	x.cfg.Searching = func(role.NodeID) searching.Remote { return passing{} }
	x.cfg.Changed, x.cfg.Status = quiet[Changed]{}, quiet[StatusChanged]{}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'seedbox', 0)", seed); err != nil {
			return err
		}
		for _, node := range []role.NodeID{home, seed} {
			if err := x.offer(ctx, tx, idx.ID, node, searching.Session{UserAgent: "Firefox", Pasted: true, Earned: t0}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return x, lib, idx
}

func sessionsOn(t *testing.T, x *Indexers, lib *storage.DB[file], id ID) map[role.NodeID]bool {
	t.Helper()
	out := map[role.NodeID]bool{}
	for _, node := range []role.NodeID{home, seed} {
		s, err := x.session(t.Context(), lib, id, node)
		if err != nil {
			t.Fatal(err)
		}
		out[node] = s != nil
	}
	return out
}

func TestLettingSearchingGoDeletesTheNodesSessionsAndMovesItsIndexers(t *testing.T) {
	x, lib, idx := onTwo(t)
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
		return rehome(ctx, tx, home, seed)
	})
	if err != nil {
		t.Fatal(err)
	}
	moved, err := Get(t.Context(), lib, idx.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := sessionsOn(t, x, lib, idx.ID); moved.Home != seed || got[home] || !got[seed] {
		t.Fatalf("after home let Searching go the Indexer is homed on %s with sessions %v, want seedbox with only seedbox's", moved.Home, got)
	}
}

func TestPruneDeletesOldQueriesAndTheSessionsOfNodesThatNoLongerSearch(t *testing.T) {
	x, lib, idx := onTwo(t)
	x.cfg.Roster = roster{holders: []role.NodeID{home}}
	x.cfg.Timings.KeepQueries, x.cfg.Timings.Unanswered = 30*24*time.Hour, time.Hour
	now := time.Now()
	rows := map[string]time.Time{
		"old": now.Add(-31 * 24 * time.Hour), "abandoned": now.Add(-2 * time.Hour), "running": now.Add(-time.Minute), "answered": now.Add(-2 * time.Hour),
	}
	ids := map[string]QueryID{}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
		for name, at := range rows {
			ids[name] = QueryID{UUID: storage.NewUUID()}
			if err := insertQuery(ctx, tx, ids[name], idx, UseSearch, storage.FromTime(at)); err != nil {
				return err
			}
		}
		return complete(ctx, tx, ids["answered"], answer{use: UseSearch, node: home, revision: idx.Revision}, storage.FromTime(now))
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := x.Prune(t.Context()); err != nil {
		t.Fatal(err)
	}
	left, err := readQueries(t.Context(), lib, idx.ID, Page{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	outcomes := map[QueryID]string{}
	for _, q := range left {
		outcomes[q.ID] = q.Outcome
	}
	want := map[QueryID]string{ids["abandoned"]: "lost", ids["running"]: "", ids["answered"]: "ok"}
	if len(outcomes) != len(want) || outcomes[ids["abandoned"]] != "lost" || outcomes[ids["running"]] != "" || outcomes[ids["answered"]] != "ok" {
		t.Errorf("query rows after Prune = %v, want %v with the old one deleted", outcomes, want)
	}
	if got := sessionsOn(t, x, lib, idx.ID); !got[home] || got[seed] {
		t.Errorf("sessions after Prune = %v, want home's kept and seedbox's deleted, since seedbox no longer holds Searching", got)
	}
}

func TestAPastedCookieMakesAnIndexerHoldASession(t *testing.T) {
	_, lib, idx := saved(t)
	if got, err := Get(t.Context(), lib, idx.ID); err != nil || holdsSession(got) {
		t.Fatalf("a Torznab feed with no pasted cookie holds a session: %t, %v; want it stateless", holdsSession(got), err)
	}
	_, lib, idx = onTwo(t)
	if got, err := Get(t.Context(), lib, idx.ID); err != nil || !holdsSession(got) {
		t.Fatalf("a feed with a pasted cookie holds a session: %t, %v; want it to", holdsSession(got), err)
	}
}

func TestMovingAnIndexerDiscardsTheCaptchasShownForItsOldHome(t *testing.T) {
	x, _, idx := onTwo(t)
	shown := signInOf(idx, home)
	shown.expires = time.Now().Add(time.Minute)
	if _, err := x.Test(t.Context(), identity.Admin{}, idx.ID, &CaptchaAnswer{Attempt: x.attempts.keep(shown), Text: "x7k2"}); err != nil {
		t.Fatalf("an answer to a captcha shown on home = %v, want it taken", err)
	}
	attempt := x.attempts.keep(shown)
	move := func(to role.NodeID) {
		t.Helper()
		if _, err := x.Update(t.Context(), identity.Admin{}, idx.ID, Edit{Home: &to}, nil); err != nil {
			t.Fatalf("move to %s: %v", to, err)
		}
	}
	move(seed)
	move(home)
	if _, err := x.Test(t.Context(), identity.Admin{}, idx.ID, &CaptchaAnswer{Attempt: attempt, Text: "x7k2"}); !errors.Is(err, ErrCaptchaExpired) {
		t.Errorf("an answer to a captcha shown on home before a move away and back = %v, want it expired", err)
	}
}
