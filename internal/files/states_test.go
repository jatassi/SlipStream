package files_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestItemStatesReadEveryItemOfEachWorkInTreeOrder(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	profile := l.profile(t, "HD", hd)
	met, thinned := l.work(t, "w1", true, &profile), l.work(t, "w2", true, &profile)
	l.place(t, met.Items[0].ID, met.ID, "Shelf w1/Scarf - HD.mkv", release.WEBDL1080p, hd.ID)
	if _, err := l.works.SetMonitored(t.Context(), admin, met.ID, works.Within{Items: []works.ItemID{met.Items[1].ID}}, false); err != nil {
		t.Fatal(err)
	}
	l.fx.Put("w2", contenttypetest.Record{Tree: works.Fetched{Label: "Shelf w2", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c2", Label: "Mittens"}}},
	}}})
	if _, err := l.works.Refresh(t.Context(), admin, thinned.ID); err != nil {
		t.Fatal(err)
	}

	var got map[works.WorkID]files.WorkStates
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = l.files(files.NoGrabs{}).ItemStates(ctx, r, works.Only([]works.WorkID{met.ID, thinned.ID, {UUID: storage.NewUUID()}}))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, w := range []works.Tree{met, thinned} {
		s := got[w.ID]
		var line strings.Builder
		fmt.Fprintf(&line, "%s levels=%d:", w.Label, len(s.Levels))
		for _, it := range s.Items {
			fmt.Fprintf(&line, " %s %v monitored=%t live=%t released=%t;", it.Item.Label, it.Status, it.Item.Monitored, it.Item.Live(), it.Released)
		}
		lines = append(lines, line.String())
	}
	want := "Shelf w1 levels=1: Scarf met monitored=true live=true released=true; Mittens missing monitored=false live=true released=true;\n" +
		"Shelf w2 levels=1: Mittens missing monitored=true live=true released=true; Scarf missing monitored=true live=false released=true;"
	if g := strings.Join(lines, "\n"); len(got) != 2 || g != want {
		t.Fatalf("ItemStates answers %d Works:\n%s\nwant 2:\n%s", len(got), g, want)
	}
}
