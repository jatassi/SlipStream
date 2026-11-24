// Package emby is the Emby conformance suite: the cases every kind
// shares, then Emby's own, from #71's and #101's probes. It runs against
// fake-emby on every PR and against a real Emby behind the online tag.
//
// The scan-end signal is decided here. SlipStream's full scan on Emby is
// POST /Library/Refresh, and it has ended when the RefreshLibrary task is
// Idle with a LastExecutionResult.EndTimeUtc later than before the scan
// was asked for, the rule Jellyfin follows (graft 3). A library's
// RefreshStatus is no end signal: it carries no end time, and on 4.10 it
// never read Idle after a recursive refresh (online run 2026-11-14).
package emby

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/mediabrowser"
	"github.com/jatassi/slipstream/conformance/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/jellyfin"
	"github.com/jatassi/slipstream/internal/storing"
)

// The shared suite's types, for the tests outside conformance that build
// a Target.
type (
	Target  = mediabrowser.Target
	Admin   = mediabrowser.Admin
	Played  = mediabrowser.Played
	Disk    = mediaservers.Disk
	DirDisk = mediaservers.DirDisk
	Library = mediaservers.Library
)

// Run runs every case against tg.
func Run(t *testing.T, tg Target) {
	shared, s := mediabrowser.Shared(t, "emby", jellyfin.Emby, tg)
	mediaservers.Shared(t, shared)
	emby := []struct {
		name string
		run  func(*testing.T, Target, mediaservers.Target, *mediabrowser.Server)
	}{
		{"PathLookup", PathLookup},
		{"VersionsAreItems", VersionsAreItems},
		{"RecursiveRefreshFindsFile", RecursiveRefreshFindsFile},
		{"UserDataDropsFavorite", UserDataDropsFavorite},
		{"DeletedWaitsTheDelay", DeletedWaitsTheDelay},
		{"TranscodeAfterReport", TranscodeAfterReport},
		{"KeyCreated", func(t *testing.T, _ Target, _ mediaservers.Target, s *mediabrowser.Server) { s.KeyCreated(t) }},
		{"SocketPushesSessions", func(t *testing.T, _ Target, _ mediaservers.Target, s *mediabrowser.Server) { s.SocketPushesSessions(t) }},
	}
	var cases []mediaservers.Case
	for _, c := range emby {
		cases = append(cases, mediaservers.Case{Name: c.name, Run: func(t *testing.T) { c.run(t, tg, shared, s) }})
	}
	mediaservers.Each(t, cases)
}

// PathLookup pins /Items?Path=: the file's exact path finds its item and
// only that. Emby 4.10 also answered one item for the file's folder
// (online run 2026-11-14), so the case logs what the folder, a prefix of
// the name and, on a POSIX server, the name in other case find, for the
// next run to record.
func PathLookup(t *testing.T, _ Target, shared mediaservers.Target, s *mediabrowser.Server) {
	f := shared.Case(t)
	defer f.Clear()
	r := mediaservers.Rel(t, "Looked Up", ".mkv")
	f.Place(r)
	f.Tell(r)
	it := f.Found(r)
	p := f.At(r).String()
	items, err := s.C.ItemsAt(t.Context(), s.User(), p)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != it.Key {
		t.Fatalf("?Path=%s found %d items, want item %s alone", p, len(items), it.Key)
	}
	others := map[string]string{"its folder": f.At(r).Dir().String(), "a name prefix": p[:len(p)-len(".mkv")]}
	if !f.At(r).Windows() {
		others["its name in other case"] = p[:len(p)-len(".mkv")] + ".MKV"
	}
	for what, q := range others {
		items, err := s.C.ItemsAt(t.Context(), s.User(), q)
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range items {
			t.Logf("?Path= with %s (%s) finds %s item %s at %q, the file's item %v", what, q, x.Type, x.ID, x.Path, x.ID == it.Key)
		}
		if len(items) == 0 {
			t.Logf("?Path= with %s (%s) finds no item", what, q)
		}
	}
}

// VersionsAreItems places two versions of one film in its folder. Each
// file is its own item, and the two share a PresentationUniqueKey.
func VersionsAreItems(t *testing.T, _ Target, shared mediaservers.Target, s *mediabrowser.Server) {
	f := shared.Case(t)
	defer f.Clear()
	a, b := mediaservers.Rel(t, "Versioned", " - 1080p.mkv"), mediaservers.Rel(t, "Versioned", " - 2160p.mkv")
	f.Place(a, b)
	f.Tell(a, b)
	if x, y := f.Found(a), f.Found(b); x.Key == y.Key {
		t.Fatalf("both versions are item %s", x.Key)
	}
	var keys []string
	for _, r := range []storing.Rel{a, b} {
		items, err := s.C.ItemsAt(t.Context(), s.User(), f.At(r).String())
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 {
			t.Fatalf("?Path=%s found %d items", f.At(r), len(items))
		}
		keys = append(keys, items[0].PresentationUniqueKey)
	}
	if keys[0] == "" || keys[0] != keys[1] {
		t.Fatalf("the versions' PresentationUniqueKeys are %q", keys)
	}
}

func refreshStatus(t *testing.T, s *mediabrowser.Server, library string) string {
	t.Helper()
	libs, err := s.C.Libraries(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range libs {
		if l.ItemID == library {
			return l.RefreshStatus
		}
	}
	t.Fatalf("no library %s", library)
	return ""
}

// RecursiveRefreshFindsFile refreshes the library's item recursively, a
// Rescan's call, with a file placed and not reported. The file gets an
// item. The case logs the library's RefreshStatus after: Emby 4.10's never
// read Idle in five minutes (online run 2026-11-14), and what it reads is
// unrecorded.
func RecursiveRefreshFindsFile(t *testing.T, _ Target, shared mediaservers.Target, s *mediabrowser.Server) {
	f := shared.Case(t)
	defer f.Clear()
	r := mediaservers.Rel(t, "Refreshed", ".mkv")
	f.Place(r)
	if err := s.C.RefreshLibrary(t.Context(), shared.Library); err != nil {
		t.Fatal(err)
	}
	f.Found(r)
	t.Logf("after the refresh the library's RefreshStatus reads %q", refreshStatus(t, s, shared.Library))
}

// UserDataDropsFavorite is Emby 4.10's UserData write: it answers success
// and sets played, the position, the play count and the last played date,
// and keeps the favorite the item had (online run 2026-11-14).
// FavoriteItems sets the favorite.
func UserDataDropsFavorite(t *testing.T, _ Target, shared mediaservers.Target, s *mediabrowser.Server) {
	f := shared.Case(t)
	defer f.Clear()
	r := mediaservers.Rel(t, "Restored", ".mkv")
	f.Place(r)
	f.Tell(r)
	it := f.Found(r)
	before, err := s.C.UserData(t.Context(), s.User(), it.Key)
	if err != nil {
		t.Fatal(err)
	}
	if before.IsFavorite {
		t.Fatal("the new item is a favorite before any write")
	}
	played := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	want := jellyfin.UserData{Played: true, PlayCount: before.PlayCount + 3, PlaybackPositionTicks: 600_000_000, IsFavorite: true, LastPlayedDate: &played}
	if err := s.C.SetUserData(t.Context(), s.User(), it.Key, want); err != nil {
		t.Fatal(err)
	}
	got, err := s.C.UserData(t.Context(), s.User(), it.Key)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Played || got.PlaybackPositionTicks != want.PlaybackPositionTicks || got.PlayCount != want.PlayCount || got.LastPlayedDate == nil || !got.LastPlayedDate.Equal(played) {
		t.Fatalf("after the write the data is %+v, want played, the position, the play count and the last played date set", got)
	}
	if got.IsFavorite {
		t.Fatalf("after the write the data is %+v, want the favorite kept unset", got)
	}
	if err := s.C.SetFavorite(t.Context(), s.User(), it.Key, true); err != nil {
		t.Fatal(err)
	}
	if got, err = s.C.UserData(t.Context(), s.User(), it.Key); err != nil || !got.IsFavorite {
		t.Fatalf("after FavoriteItems the data is %+v (err %v), want a favorite", got, err)
	}
}

// DeletedWaitsTheDelay is a Deleted report on Emby 4.10: the item is
// still there half the monitor delay after it, as after any report
// (online run 2026-11-14), and gone once the delay has passed.
func DeletedWaitsTheDelay(t *testing.T, _ Target, shared mediaservers.Target, s *mediabrowser.Server) {
	cfg, err := s.C.Configuration(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	f := shared.Case(t)
	defer f.Clear()
	r := mediaservers.Rel(t, "Deleted", ".mkv")
	f.Place(r)
	f.Tell(r)
	f.Found(r)
	f.Remove(r)
	f.Tell(r)
	time.Sleep(cfg.MonitorDelay / 2)
	if _, ok, err := shared.Server.ItemAt(t.Context(), shared.Library, f.At(r)); err != nil || !ok {
		t.Fatalf("the deleted file's item went inside half the monitor delay (err %v)", err)
	}
	f.Gone(r)
}

// TranscodeAfterReport plays a transcode whose player hasn't reported its
// playback. The session has no transcode info until the report.
func TranscodeAfterReport(t *testing.T, tg Target, shared mediaservers.Target, s *mediabrowser.Server) {
	if tg.Play == nil {
		mediaservers.Skip(t, "a real server's sessions need a player")
	}
	f := shared.Case(t)
	defer f.Clear()
	r := mediaservers.Rel(t, "Transcoded", ".mkv")
	f.Place(r)
	f.Tell(r)
	f.Found(r)
	played, stop := tg.Play(t, f.At(r), link.DecisionTranscode)
	defer stop()
	transcoding := func() bool {
		t.Helper()
		sessions, err := s.C.Sessions(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		for _, sess := range sessions {
			if sess.ID == played.Session {
				return sess.TranscodingInfo != nil
			}
		}
		t.Fatalf("no session %s", played.Session)
		return false
	}
	if transcoding() {
		t.Fatal("the session has transcode info before its player reported")
	}
	s.Report(t, played)
	if !transcoding() {
		t.Fatal("the session has no transcode info after its player reported")
	}
}
