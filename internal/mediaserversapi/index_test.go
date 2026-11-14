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

// TestThePathIndexFollowsPlex lands an episode on Living room Plex. The
// listing a day after the first learns its item and its show. Once the
// file is gone,
// Plex's timeline marks the entry stale, a lookup finds no item there,
// and the index forgets it the same minute.
func TestThePathIndexFollowsPlex(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, px := newFresh(t)
		rel := "Voyage/Season 01/Voyage - S01E01.mkv"
		episode, err := link.ParsePath(`C:\Media\Shows\Voyage\Season 01\Voyage - S01E01.mkv`)
		if err != nil {
			t.Fatal(err)
		}
		place(t, px.Server.Media, "Shows/"+rel)
		ids := fileIDs{}
		s.tell(t, at(t, files.PathAdded, ids.of("e1"), rel))
		settle(time.Minute)
		if _, ok := s.w.Item(t, s.id, episode); ok {
			t.Fatal("the index holds the episode before a listing read it")
		}
		settle(24 * time.Hour)
		it, ok := s.w.Item(t, s.id, episode)
		if !ok || it.Key == "" || it.Top == "" || it.Top == it.Key || it.Stale {
			t.Fatalf("after the next day's listing the index holds %+v, %t; want the episode's item and its show", it, ok)
		}
		if err := os.Remove(filepath.Join(px.Server.Media, "Shows", filepath.FromSlash(rel))); err != nil {
			t.Fatal(err)
		}
		s.tell(t, at(t, files.PathRemoved, ids.of("e1"), rel))
		settle(time.Minute)
		if it, ok := s.w.Item(t, s.id, episode); ok {
			t.Errorf("once Plex removed the episode the index still holds %+v", it)
		}
	})
}
