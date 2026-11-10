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
