package music_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/works"
)

func TestTheMusicConformsToCalendarOnFullReleaseDays(t *testing.T) {
	const artist = "a1000000-0000-4000-8000-000000000007"
	album := func(n int, released string, tracks int) scenario.MusicAlbum {
		a := scenario.MusicAlbum{
			ID: fmt.Sprintf("b7000000-0000-4000-8000-%012d", n), Release: fmt.Sprintf("c7000000-0000-4000-8000-%012d", n),
			Title: fmt.Sprintf("Album %d", n), Kind: "Album", FirstRelease: released,
		}
		for i := 1; i <= tracks; i++ {
			a.Tracks = append(a.Tracks, scenario.MusicTrack{Recording: fmt.Sprintf("d7000000-0000-4000-8000-%06d%06d", n, i), Title: fmt.Sprintf("Track %d", i), Disc: 1})
		}
		return a
	}
	s := catalog(t)
	s.Music.Artists = append(s.Music.Artists, scenario.MusicArtist{ID: artist, Revisions: []scenario.MusicRevision{{Name: "Dated", SortName: "Dated", Albums: []scenario.MusicAlbum{
		album(1, "2026-11-20", 3), album(2, "2003", 1), album(3, "2001-02", 1), album(4, "2026-12", 1),
		album(5, "2026-12-15", 1), album(6, "2026-11-14", 1), album(7, "", 1),
	}}}})
	m, lib, tree := added(t, s, artist)
	from := calendar.NewDay(2026, time.November, 15)
	span, err := calendar.NewSpan(from, from.Add(30))
	if err != nil {
		t.Fatal(err)
	}
	first := slices.IndexFunc(tree.Levels, func(l works.Level) bool { return l.Key == "b7000000-0000-4000-8000-000000000001" })
	var want []calendar.Dated
	for _, it := range tree.Items {
		if tree.Levels[first].ID == it.LevelID {
			want = append(want, calendar.Dated{Work: tree.ID, Item: it.ID, Kind: "release", Day: calendar.NewDay(2026, time.November, 20)})
		}
	}
	if len(want) != 3 {
		t.Fatalf("Album 1 holds %d tracks, want 3", len(want))
	}
	contenttypetest.Calendar(t, contenttypetest.CalendarCase{Library: lib, Type: m, Span: span, Index: "music_album_released", Want: want})
}
