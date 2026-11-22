package grabsapi

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/targets"
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

func TestEveryEnumNamesEveryDomainValueInOrder(t *testing.T) {
	var inFlight []grabs.Stage
	for _, s := range grabs.Stages {
		if s.InFlight() {
			inFlight = append(inFlight, s)
		}
	}
	queued := slices.DeleteFunc(searches.Triggers(), func(tr searches.Trigger) bool { return tr == searches.RSS })
	automatic := slices.DeleteFunc(slices.Clone(grabs.Outcomes), func(o grabs.Outcome) bool { return o == grabs.ManualListed })
	qualities := make([]string, 0, len(release.Qualities()))
	for _, q := range release.Qualities() {
		qualities = append(qualities, q.String())
	}
	for _, c := range []struct{ name, got, want string }{
		{"pickProblem.Code", enumOf[pickProblem](t, "Code"), joined(grabs.PickCodes)},
		{"liftKey.Code", enumOf[liftKey](t, "Code"), joined(grabs.Liftable)},
		{"blockEntry.Cause", enumOf[blockEntry](t, "Cause"), joined(grabs.BlockCauses)},
		{"grab.Stage", enumOf[Grab](t, "Stage"), joined(grabs.Stages)},
		{"entered.Stage", enumOf[entered](t, "Stage"), joined(grabs.Stages)},
		{"failure.Stage", enumOf[failure](t, "Stage"), joined(inFlight)},
		{"heldCell.Stage", enumOf[heldCell](t, "Stage"), joined(inFlight)},
		{"failure.Code", enumOf[failure](t, "Code"), joined(grabs.Codes)},
		{"lostAt.Step", enumOf[lostAt](t, "Step"), joined(grabs.Steps)},
		{"search.Outcome", enumOf[search](t, "Outcome"), joined(grabs.Outcomes)},
		{"grab.Source", enumOf[Grab](t, "Source"), joined(grabs.Sources)},
		{"grab.FoundBy", enumOf[Grab](t, "FoundBy"), joined(grabs.FoundBys)},
		{"search.Trigger", enumOf[search](t, "Trigger"), joined(searches.Triggers())},
		{"groupState.Trigger", enumOf[groupState](t, "Trigger"), joined(queued)},
		{"behind.AheadFrom", enumOf[behind](t, "AheadFrom"), joined(queued)},
		{"groupState.Last", enumOf[groupState](t, "Last"), joined(automatic)},
		{"wantedRow.Last", enumOf[wantedRow](t, "Last"), joined(automatic)},
		{"failureAt.Code", enumOf[failureAt](t, "Code"), joined(grabs.Codes)},
		{"releaseFacts.Quality", enumOf[releaseFacts](t, "Quality"), strings.Join(qualities, ",")},
		{"reason.Code", enumOf[reason](t, "Code"), joined(targets.ReasonCodes())},
	} {
		if c.got != c.want {
			t.Errorf("%s enum = %q, want %q", c.name, c.got, c.want)
		}
	}
}
