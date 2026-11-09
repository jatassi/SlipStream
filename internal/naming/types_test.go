package naming_test

import (
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestValuesOfAFileCrossingLevels(t *testing.T) {
	one, two := works.LevelID{UUID: storage.NewUUID()}, works.LevelID{UUID: storage.NewUUID()}
	last, first, extra := works.ItemID{UUID: storage.NewUUID()}, works.ItemID{UUID: storage.NewUUID()}, works.ItemID{UUID: storage.NewUUID()}
	work := works.Bare{Items: []works.Item{
		{ID: first, LevelID: one},
		{ID: last, LevelID: two},
		{ID: extra, LevelID: one, Aside: true},
	}}
	v := naming.Values{
		Work:   naming.Set{"Shelf": naming.Text("Oak")},
		Levels: map[works.LevelID]naming.Set{one: {"Box": naming.Number(1)}, two: {"Box": naming.Number(2)}},
		Items: map[works.ItemID]naming.ItemValues{
			first: {Set: naming.Set{"Thing": naming.Text("Lamp")}},
			last:  {Set: naming.Set{"Thing": naming.Text("Clock")}, Variant: "dated"},
			extra: {Set: naming.Set{"Thing": naming.Text("Spare")}},
		},
	}
	got := v.Of(work, []works.ItemID{last, first, extra})
	want := naming.Subject{Work: v.Work, Units: []naming.Unit{
		{Level: v.Levels[two], Item: v.Items[last]},
		{Level: v.Levels[one], Item: v.Items[first]},
		{Level: v.Levels[one], Item: v.Items[extra], Aside: true},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Of = %+v; want %+v", got, want)
	}
}
