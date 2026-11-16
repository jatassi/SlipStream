package worksapi

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/works"
)

func TestTheAddSearchEnumsNameEveryScopeInOrder(t *testing.T) {
	scopes := make([]string, len(works.SearchScopes))
	for i, s := range works.SearchScopes {
		scopes[i] = string(s)
	}
	want := strings.Join(scopes, ",")
	for _, c := range []struct {
		name  string
		typ   reflect.Type
		field string
	}{
		{"newWork.Search", reflect.TypeFor[newWork](), "Search"},
		{"searchChoice.Value", reflect.TypeFor[searchChoice](), "Value"},
		{"scopeCount.Scope", reflect.TypeFor[scopeCount](), "Scope"},
	} {
		f, _ := c.typ.FieldByName(c.field)
		if got := f.Tag.Get("enum"); got != want {
			t.Errorf("%s enum = %q, want every SearchScope in order: %q", c.name, got, want)
		}
	}
}
