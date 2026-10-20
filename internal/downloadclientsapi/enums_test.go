package downloadclientsapi

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
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
	words := make([]string, len(downloadclients.Tabs))
	for i, tab := range downloadclients.Tabs {
		words[i] = tabWords[tab]
	}
	states, tabs := joined(downloadclients.States), joined(words)
	for _, c := range []struct{ name, got, want string }{
		{"TorrentRow.State", enumOf[TorrentRow](t, "State"), states},
		{"stateCount.State", enumOf[stateCount](t, "State"), states},
		{"TorrentRow.Tabs", enumOf[TorrentRow](t, "Tabs"), tabs},
		{"tabCount.Tab", enumOf[tabCount](t, "Tab"), tabs},
		{"listInput.Tab", enumOf[listInput](t, "Tab"), tabs},
	} {
		if c.got != c.want {
			t.Errorf("%s enum = %q, want %q", c.name, c.got, c.want)
		}
	}
	body, _ := reflect.TypeFor[actInput]().FieldByName("Body")
	action, _ := body.Type.FieldByName("Action")
	if got, want := action.Tag.Get("enum"), joined(downloadclients.ActionKinds); got != want {
		t.Errorf("actInput.Body.Action enum = %q, want %q", got, want)
	}
}
