package downloadclients_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/live"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// levels is the hub as Clients watches it: ch wakes it as a change of the
// storage levels would.
type levels struct{ ch chan struct{} }

func (l levels) Watch(context.Context, ...live.Event) <-chan struct{} { return l.ch }

// guarded is one downloading Node whose client reports what facts holds,
// with what its intent and the Admins were told.
type guarded struct {
	lib    library
	node   role.NodeID
	c      *dc.Clients
	levels levels

	mu      sync.Mutex
	facts   *dlv1.Facts
	touched int
	cards   int
}

func newGuarded(t *testing.T) *guarded {
	t.Helper()
	g := &guarded{lib: open(t), levels: levels{ch: make(chan struct{})}}
	g.node = addNode(t, g.lib, "seedbox")
	g.c = dc.New(dc.Config{
		Library: g.lib, Changed: published{}, Roster: &roster{serving: []role.NodeID{g.node}}, Hub: g.levels,
		Touch: func(nodes ...role.NodeID) {
			g.mu.Lock()
			defer g.mu.Unlock()
			g.touched += len(nodes)
		},
		CardsChanged: g,
		Facts: func(role.NodeID) (*dlv1.Facts, bool) {
			g.mu.Lock()
			defer g.mu.Unlock()
			return g.facts, g.facts != nil
		},
		Live: dc.LiveConfig{Watched: func() []role.NodeID { return nil }, Event: &admin{}, Sample: liveSample},
		Log:  slog.New(slog.DiscardHandler),
	})
	return g
}

func (g *guarded) Publish(context.Context, dc.ClientsChanged) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cards++
}

// sample has the client report free bytes left of 1 TB on /data, sampled
// at at.
func (g *guarded) sample(free int64, at time.Time) {
	g.mu.Lock()
	g.facts = &dlv1.Facts{Bound: "192.0.2.4:6881", Downloads: &dlv1.Volume{
		Mount: "/data", Name: "/data", Total: 1000 * setting.GB, Free: free, SampledAtUnixMs: at.UnixMilli(),
	}}
	g.mu.Unlock()
	g.c.Arrived(g.node, string(role.Downloading))
}

func (g *guarded) told() (touched, cards int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.touched, g.cards
}

func (g *guarded) holds(t *testing.T) bool {
	t.Helper()
	return intent(t, g.lib, g.node).GetHoldDownloads()
}

// ownErrorLevel sets the error level Health resolves for node's mount, as
// an Admin does from Library roots.
func ownErrorLevel(t *testing.T, lib library, node role.NodeID, mount string, below int64) {
	t.Helper()
	exec(t, lib, "INSERT INTO health_volume (node_id, mount, error_amount, error_unit) VALUES (?, ?, ?, 'bytes')", node, mount, below)
}

func TestTheDiskGuardHoldsAtHealthsErrorLevel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newGuarded(t)
		ctx, stop := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = g.c.Run(ctx)
		}()
		defer func() {
			stop()
			<-done
		}()

		g.sample(30*setting.GB, time.Now())
		if g.holds(t) {
			t.Fatal("30 GB free holds new downloads, want none held above Health's 20 GB error level")
		}
		g.sample(19*setting.GB, time.Now().Add(time.Second))
		touched, cards := g.told()
		if !g.holds(t) || touched != 1 {
			t.Fatalf("19 GB free: held %v, intent touched %d times; want held below the 20 GB error level and the Node told once", g.holds(t), touched)
		}
		cardsBefore := cards

		ownErrorLevel(t, g.lib, g.node, "/other", 50*setting.GB)
		ownErrorLevel(t, g.lib, g.node, "/data", 10*setting.GB)
		g.levels.ch <- struct{}{}
		synctest.Wait()
		touched, cards = g.told()
		if g.holds(t) || touched != 2 || cards != cardsBefore+1 {
			t.Fatalf("with /data's own error level at 10 GB: held %v, touched %d, cards told %d more; want lifted, the Node and its card told once",
				g.holds(t), touched, cards-cardsBefore)
		}
		got, err := g.c.Cards(t.Context())
		if err != nil || len(got) != 1 || got[0].Guard != nil {
			t.Fatalf("cards = %+v, %v; want seedbox's with no guard", got, err)
		}
	})
}

func TestTheGuardLiftsOnTheFirstSampleAbove(t *testing.T) {
	g := newGuarded(t)
	start := time.Now()
	g.sample(5*setting.GB, start)
	cards, err := g.c.Cards(t.Context())
	if err != nil || len(cards) != 1 || cards[0].Guard == nil || cards[0].Guard.Volume.Free != 5*setting.GB || cards[0].Guard.ErrorBelow.Bytes != 20*setting.GB ||
		cards[0].Volume == nil || cards[0].Volume.Mount != "/data" || cards[0].Volume.Free != 5*setting.GB {
		t.Fatalf("cards = %+v, %v; want seedbox's /data volume with 5 GB free and its guard below 20 GB", cards, err)
	}
	g.sample(19*setting.GB, start.Add(time.Minute))
	if touched, _ := g.told(); !g.holds(t) || touched != 1 {
		t.Fatalf("a sample still below: held %v, touched %d; want still held and the Node told once", g.holds(t), touched)
	}
	g.sample(20*setting.GB, start.Add(2*time.Minute))
	if touched, _ := g.told(); g.holds(t) || touched != 2 {
		t.Fatalf("the first sample at the error level: held %v, touched %d; want lifted at once", g.holds(t), touched)
	}
	g.sample(1*setting.GB, start.Add(90*time.Second))
	if touched, _ := g.told(); g.holds(t) || touched != 2 {
		t.Fatalf("an older sample arriving late: held %v, touched %d; want the newer sample to stand", g.holds(t), touched)
	}
	in := intent(t, g.lib, g.node)
	if in.GetSampleEverySeconds() != 300 {
		t.Errorf("the intent samples every %ds, want Storage's 5 minutes", in.GetSampleEverySeconds())
	}
}

func intent(t *testing.T, lib library, node role.NodeID) *dlv1.Intent {
	t.Helper()
	var in *dlv1.Intent
	err := lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		in, err = dc.Intent(ctx, rd, node)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return in
}
