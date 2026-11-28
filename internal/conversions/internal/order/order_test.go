package order

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// TestEveryInsertSortsWhereItWasPut inserts at random places in a lane,
// always through Before, Between or After, and checks the keys sort in
// the lane's order with none repeated.
func TestEveryInsertSortsWhereItWasPut(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a fixed draw the test replays
	var lane []Key
	for range 2000 {
		at := r.IntN(len(lane) + 1)
		var k Key
		switch {
		case len(lane) == 0:
			k = After("")
		case at == 0:
			k = Before(lane[0])
		case at == len(lane):
			k = After(lane[len(lane)-1])
		default:
			k = Between(lane[at-1], lane[at])
		}
		lane = slices.Insert(lane, at, k)
	}
	if !slices.IsSorted(lane) {
		t.Fatal("the keys don't sort in the lane's order")
	}
	for i := 1; i < len(lane); i++ {
		if lane[i] == lane[i-1] {
			t.Fatalf("key %q repeats", lane[i])
		}
	}
}
