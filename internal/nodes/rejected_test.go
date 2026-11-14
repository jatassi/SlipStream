package nodes_test

import (
	"context"
	"errors"
	"math/rand/v2"
	"reflect"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/role/roletest"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// TestCheckNowForgetsTheReportsSkippedFromANode has home hold the probe,
// whose Controller half fails home's two reports, so it skips both: the
// Nodes category warns about them, and Check now on that item clears it
// and still reprobes home.
func TestCheckNowForgetsTheReportsSkippedFromANode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		var reprobed int
		served := roletest.Serve()
		c := startController(t, func(cfg *nodes.Config) {
			cfg.Served = append(cfg.Served, served)
			cfg.Always = []role.Name{roletest.Name}
			cfg.Probe = func(context.Context, nodes.ID) ([]role.ProbeResult, error) {
				mu.Lock()
				defer mu.Unlock()
				reprobed++
				return nil, nil
			}
		})
		probe := holdProbe(t, c.svc.Controls())
		synctest.Wait()
		home := c.svc.Self().String()
		published := c.pub.count(home)
		for _, n := range []uint64{1, 2} {
			served.FailAt(n, errors.New("the result names no Item"))
			if err := probe.Report(t.Context(), n); err != nil {
				t.Fatal(err)
			}
			synctest.Wait()
		}

		if got := c.pub.count(home) - published; got != 2 {
			t.Errorf("nodes.changed was published %d times for home as its reports were skipped, want once for each", got)
		}
		src := c.svc.HealthSource()
		key := health.KeyOf(home, "rejected")
		a, err := src.Answer(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		want := health.Item{Key: key, Level: health.Warn, Says: health.Says{
			Subject: "home",
			Text:    "SlipStream can't apply 2 reports from home and skipped them. The log has an error for each one. Read them, then Check now.",
			Detail:  []health.Fact{{Label: "Last skipped", At: time.Now().UTC().Truncate(time.Microsecond)}},
			Action:  health.CheckAction(),
		}}
		if len(a.Items) != 1 || !reflect.DeepEqual(a.Items[0], want) {
			t.Fatalf("with two reports skipped the items are %+v, want %+v", a.Items, want)
		}

		if _, err := src.Check(t.Context(), key); err != nil {
			t.Fatal(err)
		}
		if a, err := src.Answer(t.Context()); err != nil || len(a.Items) != 0 {
			t.Errorf("after Check now the items are %+v (%v), want none", a.Items, err)
		}
		mu.Lock()
		defer mu.Unlock()
		if reprobed != 1 {
			t.Errorf("Check now reprobed home %d times, want once", reprobed)
		}
	})
}

// holdProbe runs the Controller's own holder half, holding the probe,
// until the test ends.
func holdProbe(t *testing.T, controls *role.Controls) *roletest.Probe {
	t.Helper()
	local := storagetest.Local(t, role.LocalSchema())
	probe := roletest.NewProbe(local)
	h, err := role.NewHolding(t.Context(), role.HoldingConfig{
		Local:    local,
		Upstream: role.InProcess(controls),
		Held:     []role.Held{probe.Held()},
		Timings:  role.Timings{RestartMin: time.Second, RestartMax: time.Minute, Beat: 15 * time.Second},
		Rand:     rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Log:      discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { _ = h.Run(ctx) })
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	return probe
}
