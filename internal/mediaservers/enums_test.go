package mediaservers_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
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

var (
	accesses = []link.Access{link.AccessPlexTV, link.AccessUsers, link.AccessUsersLinked}
	problems = []mediaservers.ProblemCode{
		mediaservers.ProblemPlexRefused, mediaservers.ProblemPlexLibrariesShort, mediaservers.ProblemUsernameTaken,
		mediaservers.ProblemServerRefused, mediaservers.ProblemMediaServerUnreached, mediaservers.ProblemFailed,
	}
)

// startedBy is every event the access table lets by start, sorted.
func startedBy(by access.By) string {
	var events []access.Event
	for _, t := range access.Table {
		if t.By&by != 0 && !slices.Contains(events, t.On) {
			events = append(events, t.On)
		}
	}
	slices.Sort(events)
	return joined(events)
}

func sorted(enum string) string {
	values := strings.Split(enum, ",")
	slices.Sort(values)
	return strings.Join(values, ",")
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
		{"AccessView.State", enumOf[mediaservers.AccessView](t, "State"), joined(access.States())},
		{"AccessView.Actions", sorted(enumOf[mediaservers.AccessView](t, "Actions")), startedBy(access.ByAdmin)},
		{"AccessServer.Access", enumOf[mediaservers.AccessServer](t, "Access"), joined(accesses)},
		{"AccessProblem.Code", enumOf[mediaservers.AccessProblem](t, "Code"), joined(problems)},
		{"MyAccessRow.State", enumOf[mediaservers.MyAccessRow](t, "State"), joined(slices.DeleteFunc(access.States(), func(s access.State) bool { return !s.MemberSees() }))},
		{"MyAccessRow.Actions", sorted(enumOf[mediaservers.MyAccessRow](t, "Actions")), startedBy(access.ByMember)},
		{"MyServer.Access", enumOf[mediaservers.MyServer](t, "Access"), joined(accesses)},
		{"MemberPinState.State", enumOf[mediaservers.MemberPinState](t, "State"), joined([]mediaservers.PinState{mediaservers.PinWaiting, mediaservers.PinExpired, mediaservers.PinSignedIn, mediaservers.PinRefused})},
		{"MemberPinState.Refusal", enumOf[mediaservers.MemberPinState](t, "Refusal"), joined([]mediaservers.PinRefusal{mediaservers.PlexAccountTaken, mediaservers.PlexAccountOwner, mediaservers.PlexAccountOther})},
	} {
		if c.got != c.want {
			t.Errorf("%s enum = %q, want %q", c.name, c.got, c.want)
		}
	}
}
