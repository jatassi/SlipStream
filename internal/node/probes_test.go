package node

import (
	"maps"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/role"
)

// TestEveryProbeHasACheck holds the Node's checks to the Role table: each
// probe the table names has one, and none checks a probe it doesn't name.
func TestEveryProbeHasACheck(t *testing.T) {
	if got, want := slices.Sorted(maps.Keys(probeChecks)), slices.Sorted(slices.Values(role.Probes())); !slices.Equal(got, want) {
		t.Fatalf("the Node checks %v, the Role table names %v", got, want)
	}
}
