package contenttypetest_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

func TestTheFixtureConformsToCalendar(t *testing.T) {
	f := contenttypetest.New()
	lib := storagetest.Library(t, node.LibrarySchemas(contenttype.NewSet(f.Def()))...)
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: f}, Source: f.Source(), Artwork: f, Timings: works.DefaultTimings(),
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	f.Put("shelf", contenttypetest.Record{Tree: works.Fetched{Label: "Shelf", Children: []works.Entry{
		{Key: "box", Label: "Box", Children: []works.Entry{{Key: "cup", Label: "Cup"}, {Key: "jar", Label: "Jar"}, {Key: "pot", Label: "Pot"}}},
	}}})
	tree, _, err := w.Add(t.Context(), identity.Admin{}, contenttypetest.ID, "shelf", works.Adding{Monitor: true})
	if err != nil {
		t.Fatal(err)
	}
	from := calendar.NewDay(2026, time.November, 1)
	span, err := calendar.NewSpan(from, from.Add(30))
	if err != nil {
		t.Fatal(err)
	}
	cup, jar, pot := tree.Items[0].ID, tree.Items[1].ID, tree.Items[2].ID
	f.SetDated(tree.ID, cup, contenttypetest.Shipped.Key, from)
	f.SetDated(tree.ID, jar, contenttypetest.Shipped.Key, from.Add(-1))
	f.SetDated(tree.ID, pot, contenttypetest.Shipped.Key, from.Add(30))
	f.SetDated(tree.ID, jar, contenttypetest.Shipped.Key, from.Add(29))
	contenttypetest.Calendar(t, contenttypetest.CalendarCase{Library: lib, Type: f, Span: span, Want: []calendar.Dated{
		{Work: tree.ID, Item: cup, Kind: contenttypetest.Shipped.Key, Day: from},
		{Work: tree.ID, Item: jar, Kind: contenttypetest.Shipped.Key, Day: from.Add(29)},
	}})
}
