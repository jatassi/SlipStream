package mediaserversapi_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// TestThePathIndexFollowsPlex lands a film on Living room Plex. The
// listing a day after the first learns its item. Once the file is gone,
// Plex's timeline marks the entry stale, a lookup finds no item there,
// and the index forgets it the same minute.
func TestThePathIndexFollowsPlex(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, px := newFilms(t)
		rel := "Heat (1995)/Heat (1995).mkv"
		film, err := link.ParsePath(`C:\Media\Films\Heat (1995)\Heat (1995).mkv`)
		if err != nil {
			t.Fatal(err)
		}
		place(t, px.Server.Media, "Films/"+rel)
		ids := fileIDs{}
		s.tell(t, at(t, files.PathAdded, ids.of("heat"), rel))
		settle(time.Minute)
		if _, ok := s.w.Item(t, s.id, film); ok {
			t.Fatal("the index holds the film before a listing read it")
		}
		settle(24 * time.Hour)
		it, ok := s.w.Item(t, s.id, film)
		if !ok || it.Key == "" || it.Top != it.Key || it.Stale {
			t.Fatalf("after the next day's listing the index holds %+v, %t; want the film's item", it, ok)
		}
		if err := os.Remove(filepath.Join(px.Server.Media, "Films", filepath.FromSlash(rel))); err != nil {
			t.Fatal(err)
		}
		s.tell(t, at(t, files.PathRemoved, ids.of("heat"), rel))
		settle(time.Minute)
		if it, ok := s.w.Item(t, s.id, film); ok {
			t.Errorf("once Plex removed the film the index still holds %+v", it)
		}
	})
}
