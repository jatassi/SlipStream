package contenttypetest

import (
	"cmp"
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// CalendarCase is one type's world for the shared Calendar suite.
type CalendarCase struct {
	Library *storage.DB[storage.Library]
	Type    contenttype.Library
	Span    calendar.Span
	// Want is what Dated answers for Span. The type's test seeds dates
	// inside the span, before it, on its To day, and partial or empty ones.
	Want []calendar.Dated
	// Index is the index the type's span read, its first query, searches;
	// empty for a type with no tables.
	Index string
}

// Calendar runs a Facet's contract: exactly Want; every Day inside the
// span; every Kind one the type declares; every Item one its Work has;
// and the span read through Index.
func Calendar(t *testing.T, c CalendarCase) {
	t.Helper()
	var got []calendar.Dated
	var plans []storagetest.QueryPlan
	err := c.Library.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		plans = storagetest.Plan(t, r, func(r storage.Reader[storage.Library]) error {
			var err error
			got, err = c.Type.Dated(ctx, r, c.Span)
			return err
		})
		for _, d := range got {
			w, _, err := works.ItemOf(ctx, r, d.Item)
			if err != nil {
				t.Errorf("Dated answered Item %s: %v", d.Item, err)
			} else if w.ID != d.Work {
				t.Errorf("Dated answered Item %s of Work %s, which is %s's", d.Item, d.Work, w.ID)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	declared := c.Type.Describe().Dated
	for _, d := range got {
		if !c.Span.Contains(d.Day) {
			t.Errorf("Dated answered %s, outside %s to %s", d.Day, c.Span.From(), c.Span.To())
		}
		if !slices.ContainsFunc(declared, func(k works.DatedKind) bool { return k.Key == d.Kind }) {
			t.Errorf("Dated answered kind %s, which the type doesn't declare", d.Kind)
		}
	}
	if got, want := sorted(got), sorted(c.Want); !reflect.DeepEqual(got, want) {
		t.Errorf("Dated = %+v, want %+v", got, want)
	}
	if c.Index != "" && (len(plans) == 0 || !plans[0].Uses(c.Index)) {
		t.Errorf("the span read's plan = %+v, want it through %s", plans, c.Index)
	}
}

func sorted(ds []calendar.Dated) []calendar.Dated {
	return slices.SortedFunc(slices.Values(ds), func(a, b calendar.Dated) int {
		return cmp.Or(a.Day.Compare(b.Day), a.Item.Compare(b.Item.UUID), cmp.Compare(a.Kind, b.Kind))
	})
}
