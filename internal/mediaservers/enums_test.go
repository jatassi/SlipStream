package mediaservers_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
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

func TestTheWireEnumsNameEveryDomainValueInOrder(t *testing.T) {
	for _, c := range []struct{ name, got, want string }{
		{"ChangeView.Kind", enumOf[mediaservers.ChangeView](t, "Kind"), joined(link.ChangeKinds())},
		{"ChangeView.Step", enumOf[mediaservers.ChangeView](t, "Step"), joined(link.Steps())},
		{"ChangeView.Result", enumOf[mediaservers.ChangeView](t, "Result"), joined(link.Results())},
		{"ChangeView.Lost", enumOf[mediaservers.ChangeView](t, "Lost"), joined(link.Losses())},
		{"ServerView.Contact", enumOf[mediaservers.ServerView](t, "Contact"), joined(link.Contacts())},
		{"SessionView.Decision", enumOf[mediaservers.SessionView](t, "Decision"), joined(link.Decisions())},
		{"KindView.SignIn", enumOf[mediaservers.KindView](t, "SignIn"), joined([]link.SignIn{link.SignInPlexPIN, link.SignInKeyOrAdmin})},
	} {
		if c.got != c.want {
			t.Errorf("%s enum = %q, want %q", c.name, c.got, c.want)
		}
	}
}
