package regex

import (
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestATimedMatchInASynctestBubbleStartsNoGoroutineThere(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		if ok, err := MustCompile(`remux`, opts).Match(TextOf("REMUX")); !ok || err != nil {
			t.Fatalf("Match = %v, %v; want true", ok, err)
		}
	})
}

func TestACatastrophicMatchInASynctestBubbleTimesOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		re := MustCompile(`(x+x+)+y`, Options{Timeout: 50 * time.Millisecond})
		var te *TimeoutError
		if _, err := re.Match(TextOf(strings.Repeat("x", 40))); !errors.As(err, &te) {
			t.Fatalf("Match = %v; want a *TimeoutError", err)
		}
	})
}
