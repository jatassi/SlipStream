package entry_test

import (
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/history/entry"
)

func panics(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		got, _ := recover().(string)
		if !strings.Contains(got, want) {
			t.Errorf("panicked with %q, want %q", got, want)
		}
	}()
	f()
}

func TestTheCatalogRefusesADuplicateKindARowWithNoGroupAndALateOwner(t *testing.T) {
	grabbed := entry.Row[noFacts]{Spec: entry.Spec{Kind: "grabbed", Label: "Grabbed", Group: entry.Grabs}}
	c := entry.NewCatalog()
	c.Register("grabs", []entry.Entry{grabbed})
	panics(t, "grabbed, which is already registered", func() { c.Register("other", []entry.Entry{grabbed}) })
	panics(t, "with no History group", func() {
		c.Register("other", []entry.Entry{entry.Row[noFacts]{Spec: entry.Spec{Kind: "test", Label: "Test", Group: "system"}}})
	})
	c.Seal()
	panics(t, "after the catalog was sealed", func() {
		c.Register("late", []entry.Entry{entry.Row[noFacts]{Spec: entry.Spec{Kind: "late", Label: "Late", Group: entry.Library}}})
	})
	if specs := c.Specs(); len(specs) != 1 || specs[0].Kind != "grabbed" {
		t.Fatalf("the catalog holds %+v, want Grabbed alone", specs)
	}
}
