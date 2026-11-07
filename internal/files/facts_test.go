package files_test

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestItemFactsReadReleaseOnTheCalendarAndAnyFillingFile(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	uhd := l.target(t, "UHD", []release.Quality{release.WEBDL2160p})
	profile := l.profile(t, "Both", hd, uhd)
	met, upgradable := l.work(t, "w1", false, &profile), l.work(t, "w2", false, &profile)
	l.place(t, met.Items[0].ID, met.ID, "Shelf w1/Scarf - HD.mkv", release.Bluray1080p, hd.ID)
	l.place(t, upgradable.Items[0].ID, upgradable.ID, "Shelf w2/Scarf - HD.mkv", release.WEBDL1080p, hd.ID)
	l.place(t, upgradable.Items[1].ID, upgradable.ID, "Shelf w2/Mittens - kept.mkv", release.WEBDL1080p)
	tomorrow := today.AddDate(0, 0, 1)
	l.fx.SetTiming(met.Items[1].ID, targets.Timing{Released: release.Known(works.Date{Year: tomorrow.Year(), Month: int(tomorrow.Month()), Day: tomorrow.Day()})})
	l.fx.SetTiming(upgradable.Items[1].ID, targets.Timing{})

	var got map[works.WorkID]works.Facts
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = l.files(grabbing{}).ItemFacts(ctx, r, works.Only([]works.WorkID{met.ID, upgradable.ID}))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[works.WorkID]works.Facts{
		met.ID:        {met.Items[0].ID: {Released: true, Filled: true}, met.Items[1].ID: {}},
		upgradable.ID: {upgradable.Items[0].ID: {Released: true, Filled: true}, upgradable.Items[1].ID: {}},
	}
	for id, facts := range want {
		for item, f := range facts {
			if got[id][item] != f {
				t.Errorf("Item %s of Work %s has facts %+v, want %+v", item, id, got[id][item], f)
			}
		}
	}
}
