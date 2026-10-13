package sim_test

import (
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func home(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("home")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestSimHome makes TestHome's assertions on the simulation tier.
func TestSimHome(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t))
		worldtest.CheckHome(t, w.API())
	})
}

// recorder is a testing.TB that keeps what a check reports.
type recorder struct {
	testing.TB
	errs []string
}

func (r *recorder) Errorf(format string, args ...any) {
	r.errs = append(r.errs, fmt.Sprintf(format, args...))
}
