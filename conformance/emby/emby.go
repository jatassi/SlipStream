// Package emby is the Emby conformance suite: the cases every kind
// shares, then Emby's own, from #71's and #101's probes. It runs against
// fake-emby on every PR and against a real Emby behind the online tag.
//
// The scan-end signal is decided here. SlipStream's full scan on Emby is
// POST /Library/Refresh, and it has ended when the RefreshLibrary task is
// Idle with a LastExecutionResult.EndTimeUtc later than before the scan
// was asked for, the rule Jellyfin follows (graft 3). A library's
// RefreshStatus is no end signal: it carries no end time, so a scan that
// went Active and back to Idle between two polls reads as one not yet
// started. RefreshStatusReturnsToIdle records what it does.
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
	Played  = mediabrowser.Played
	Disk    = mediaservers.Disk
	DirDisk = mediaservers.DirDisk
	Fresh   = mediaservers.Fresh
)

// Run runs every case against tg.
func Run(t *testing.T, tg Target) {
	shared, s := mediabrowser.Shared(t, "emby", jellyfin.Emby, tg)
	mediaservers.Shared(t, shared)
	emby := []struct {
		name string
		run  func(*testing.T, Target, mediaservers.Target, *mediabrowser.Server)
	}{
		{"PathLookupExact", PathLookupExact},
		{"VersionsAreItems", VersionsAreItems},
		{"RefreshStatusReturnsToIdle", RefreshStatusReturnsToIdle},
		{"UserDataKeepsPlayCount", UserDataKeepsPlayCount},
		{"DeletedLeavesAtOnce", DeletedLeavesAtOnce},
		{"TranscodeAfterReport", TranscodeAfterReport},
		{"SocketPushesSessions", func(t *testing.T, _ Target, _ mediaservers.Target, s *mediabrowser.Server) { s.SocketPushesSessions(t) }},
	}
	var cases []mediaservers.Case
	for _, c := range emby {
		cases = append(cases, mediaservers.Case{Name: c.name, Run: func(t *testing.T) { c.run(t, tg, shared, s) }})
	}
	mediaservers.Each(t, cases)
}

// PathLookupExact pins /Items?Path=: the file's exact path finds its item
// and only that, while its folder, a prefix of its name and, on a POSIX
// server, its name in other case find nothing.
func PathLookupExact(t *testing.T, _ Target, shared mediaservers.Target, s *mediabrowser.Server) {
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
	misses := map[string]string{"its folder": f.At(r).Dir().String(), "a name prefix": p[:len(p)-len(".mkv")]}
	if !f.At(r).Windows() {
		misses["its name in other case"] = p[:len(p)-len(".mkv")] + ".MKV"
	}
	for what, q := range misses {
		items, err := s.C.ItemsAt(t.Context(), s.User(), q)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 0 {
			t.Fatalf("?Path= with %s (%s) found %d items, want none", what, q, len(items))
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

// RefreshStatusReturnsToIdle refreshes the library's item recursively
// with a file placed and not reported. The library's RefreshStatus reads
// Idle again and the file has an item.
func RefreshStatusReturnsToIdle(t *testing.T, _ Target, shared mediaservers.Target, s *mediabrowser.Server) {
	f := shared.Case(t)
	defer f.Clear()
	r := mediaservers.Rel(t, "Refreshed", ".mkv")
	f.Place(r)
	if err := s.C.RefreshLibrary(t.Context(), shared.Library); err != nil {
		t.Fatal(err)
	}
	f.Found(r)
	mediaservers.Eventually(t, shared, "RefreshStatus Idle", func() (bool, error) {
		return refreshStatus(t, s, shared.Library) == "Idle", nil
	})
}

// UserDataKeepsPlayCount is #101's finding: a UserData write answers
// success and sets played, position and favorite, and leaves the play
// count and last played date as they were.
func UserDataKeepsPlayCount(t *testing.T, _ Target, shared mediaservers.Target, s *mediabrowser.Server) {
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
	played := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	want := jellyfin.UserData{Played: true, PlayCount: before.PlayCount + 3, PlaybackPositionTicks: 600_000_000, IsFavorite: true, LastPlayedDate: &played}
	if err := s.C.SetUserData(t.Context(), s.User(), it.Key, want); err != nil {
		t.Fatal(err)
	}
	got, err := s.C.UserData(t.Context(), s.User(), it.Key)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Played || !got.IsFavorite || got.PlaybackPositionTicks != want.PlaybackPositionTicks {
		t.Fatalf("after the write the data is %+v, want played, favorite and the position set", got)
	}
	if got.PlayCount != before.PlayCount || (got.LastPlayedDate != nil && got.LastPlayedDate.Equal(played)) {
		t.Fatalf("after the write the play count is %d (was %d) and last played %v", got.PlayCount, before.PlayCount, got.LastPlayedDate)
	}
}

// DeletedLeavesAtOnce is #101's delete timing: a Deleted report removes
// the item without the monitor delay.
func DeletedLeavesAtOnce(t *testing.T, _ Target, shared mediaservers.Target, s *mediabrowser.Server) {
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
	quick := shared
	quick.Bound = cfg.MonitorDelay / 2
	mediaservers.Eventually(t, quick, "the deleted file's item gone inside half the monitor delay", func() (bool, error) {
		_, ok, err := shared.Server.ItemAt(t.Context(), shared.Library, f.At(r))
		return !ok, err
	})
}

// TranscodeAfterReport plays a transcode whose player hasn't reported its
// playback. The session has no transcode info until the report.
func TranscodeAfterReport(t *testing.T, tg Target, shared mediaservers.Target, s *mediabrowser.Server) {
	if tg.Play == nil {
		t.Skip("a real server's sessions need a player")
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
