package series

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func airDateOn() setting.Patch { return setting.Patch{"nameByAirDate": json.RawMessage("true")} }

func toOrder(o string) setting.Patch {
	return setting.Patch{"order": json.RawMessage(`"` + o + `"`)}
}

// namedByAirDate is the Name by air date switch among fields.
func namedByAirDate(t *testing.T, fields []setting.Field) bool {
	t.Helper()
	for _, f := range fields {
		if s, ok := f.(setting.SwitchField); ok && s.Key == "nameByAirDate" {
			return s.Value
		}
	}
	t.Fatalf("the options %+v have no Name by air date switch", fields)
	return false
}

func (sh shelf) namedByAirDate(t *testing.T, id works.WorkID) bool {
	t.Helper()
	var fields []setting.Field
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		fields, err = sh.s.OptionFields(ctx, r, id)
		return err
	})
	return namedByAirDate(t, fields)
}

func (sh shelf) setOptions(t *testing.T, id works.WorkID, p setting.Patch, preview bool) works.OptionsChange {
	t.Helper()
	c, err := sh.works.SetOptions(t.Context(), identity.Admin{}, id, works.Optioning{Type: p}, preview)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNameByAirDateStoresWithoutAFetch(t *testing.T) {
	sh := open(t)
	tree := sh.add(t, newShow(7801, "Late Night", 0, 3))
	fetches, published := sh.src.fetched(), sh.sent.n.Load()

	c := sh.setOptions(t, tree.ID, airDateOn(), true)
	if c.Moves || !namedByAirDate(t, c.Fields) {
		t.Fatalf("a preview of Name by air date answered moves %t and %+v, want no move and the switch on", c.Moves, c.Fields)
	}
	if sh.namedByAirDate(t, tree.ID) {
		t.Fatal("a preview turned Name by air date on")
	}

	c = sh.setOptions(t, tree.ID, airDateOn(), false)
	if c.Moves || !namedByAirDate(t, c.Fields) || !sh.namedByAirDate(t, tree.ID) {
		t.Fatalf("turning Name by air date on answered moves %t and %+v, and stored %t", c.Moves, c.Fields, sh.namedByAirDate(t, tree.ID))
	}
	if got := sh.src.fetched(); got != fetches {
		t.Fatalf("turning Name by air date on fetched the series %d times, want none", got-fetches)
	}
	if got := sh.sent.n.Load() - published; got != 1 {
		t.Fatalf("turning Name by air date on published %d works.changed, want 1", got)
	}

	sh.setOptions(t, tree.ID, airDateOn(), false)
	if got := sh.sent.n.Load() - published; got != 1 {
		t.Fatalf("setting Name by air date to what it was published %d works.changed in all, want the first alone", got)
	}
	sh.refresh(t, tree.ID, newShow(7801, "Late Night", 0, 4))
	if !sh.namedByAirDate(t, tree.ID) {
		t.Fatal("a refresh turned Name by air date off")
	}
}

func TestAbsoluteOrderAndNameByAirDateExcludeEachOther(t *testing.T) {
	sh := open(t)
	aired := newShow(7901, "Two Orders", 0, 3)
	aired.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderAbsolute}
	tree := sh.add(t, aired)
	sh.src.put(absoluteOf(aired, db.SeasonID(7901*100+50)), aired)
	sh.setOptions(t, tree.ID, airDateOn(), false)

	refused := func(p setting.Patch) *setting.RangeError {
		t.Helper()
		_, err := sh.works.SetOptions(t.Context(), identity.Admin{}, tree.ID, works.Optioning{Type: p}, true)
		r, ok := errors.AsType[*setting.RangeError](err)
		if !ok {
			t.Fatalf("SetOptions(%s) = %v, want a RangeError", p, err)
		}
		return r
	}
	if r := refused(toOrder("absolute")); r.Key != "order" || r.Sentence != "Absolute order names episodes by number. Turn off Name by air date to use it." {
		t.Fatalf("Absolute order with Name by air date on was refused with %+v", r)
	}

	fetches := sh.src.fetched()
	both := toOrder("absolute")
	both["nameByAirDate"] = json.RawMessage("false")
	if c := sh.setOptions(t, tree.ID, both, false); !c.Moves || sh.namedByAirDate(t, tree.ID) {
		t.Fatalf("Absolute order with Name by air date off answered moves %t, and the switch reads %t", c.Moves, sh.namedByAirDate(t, tree.ID))
	}
	if got := sh.src.fetched() - fetches; got != 1 {
		t.Fatalf("changing to Absolute order fetched %d times, want 1", got)
	}
	if r := refused(airDateOn()); r.Key != "nameByAirDate" || r.Sentence != "Name by air date doesn't work with Absolute order. Change the episode order to Aired or DVD first." {
		t.Fatalf("Name by air date in Absolute order was refused with %+v", r)
	}
}

func TestANewSeriesTakesTheAddDefault(t *testing.T) {
	sh := open(t)
	before := sh.add(t, newShow(8001, "Before", 0, 2))
	fields, err := sh.works.SetSettings(t.Context(), identity.Admin{}, Def.ID(), setting.Patch{"series.add.nameByAirDate": json.RawMessage("true")})
	if err != nil {
		t.Fatal(err)
	}
	var shown bool
	for _, f := range fields {
		if s, ok := f.(setting.SwitchField); ok && s.Key == "series.add.nameByAirDate" {
			shown = s.Value && s.Label == "Name by air date"
		}
	}
	if !shown {
		t.Fatalf("the series settings read %+v, want Name by air date on", fields)
	}

	after := sh.add(t, newShow(8002, "After", 0, 2))
	if !sh.namedByAirDate(t, after.ID) {
		t.Fatal("a series added with the default on isn't named by air date")
	}
	if sh.namedByAirDate(t, before.ID) {
		t.Fatal("changing the add default turned Name by air date on for a series already added")
	}
}
