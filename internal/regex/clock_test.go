package regex

import (
	"testing"
	"testing/synctest"
)

func TestATimedMatchInASynctestBubbleStartsNoGoroutineThere(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		if ok, err := MustCompile(`remux`, opts).Match(TextOf("REMUX")); !ok || err != nil {
			t.Fatalf("Match = %v, %v; want true", ok, err)
		}
	})
}
