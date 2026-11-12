package grabsapi

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/grabs"
)

func enumOf[T any](t *testing.T, field string) string {
	t.Helper()
	f, ok := reflect.TypeFor[T]().FieldByName(field)
	if !ok {
		t.Fatalf("%s has no field %s", reflect.TypeFor[T](), field)
	}
	return f.Tag.Get("enum")
}

func joined[S ~string](values []S) string {
	words := make([]string, len(values))
	for i, v := range values {
		words[i] = string(v)
	}
	return strings.Join(words, ",")
}

func TestTheManualSearchEnumsNameEveryDomainValueInOrder(t *testing.T) {
	for _, c := range []struct{ name, got, want string }{
		{"pickProblem.Code", enumOf[pickProblem](t, "Code"), joined(grabs.PickCodes)},
		{"liftKey.Code", enumOf[liftKey](t, "Code"), joined(grabs.Liftable)},
		{"blockEntry.Cause", enumOf[blockEntry](t, "Cause"), joined(grabs.BlockCauses)},
	} {
		if c.got != c.want {
			t.Errorf("%s enum = %q, want %q", c.name, c.got, c.want)
		}
	}
}

func TestTheGrabEnumsNameEveryStageCodeAndStepInOrder(t *testing.T) {
	var inFlight []grabs.Stage
	for _, s := range grabs.Stages {
		if s.InFlight() {
			inFlight = append(inFlight, s)
		}
	}
	for _, c := range []struct{ name, got, want string }{
		{"grab.Stage", unreached(enumOf[grab](t, "Stage")), joined(grabs.Stages)},
		{"entered.Stage", unreached(enumOf[entered](t, "Stage")), joined(grabs.Stages)},
		{"failure.Stage", unreached(enumOf[failure](t, "Stage")), joined(inFlight)},
		{"heldCell.Stage", unreached(enumOf[heldCell](t, "Stage")), joined(inFlight)},
		{"failure.Code", enumOf[failure](t, "Code"), joined(grabs.Codes)},
		{"lostAt.Step", enumOf[lostAt](t, "Step"), joined(grabs.Steps)},
	} {
		if c.got != c.want {
			t.Errorf("%s enum = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// unreached is enum without probing and transferring, the stages S33
// (Import pipeline) moved into the Import and the wire still names.
func unreached(enum string) string {
	return strings.Replace(enum, "probing,transferring,", "", 1)
}
