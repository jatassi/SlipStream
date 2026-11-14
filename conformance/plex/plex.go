// Package plex is the Plex Media Server conformance suite: the cases every
// kind shares, then Plex's own, from #71's probes and the recorded runs of
// 2026-11-14 filed under recorded/. It runs against fake-plex on every PR
// and against a real Plex behind the online tag.
package plex

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// Disk, DirDisk and Fresh are the shared suite's, for the tests outside
// conformance that build a Target.
type (
	Disk    = mediaservers.Disk
	DirDisk = mediaservers.DirDisk
	Fresh   = mediaservers.Fresh
)

// Target is the Plex under test.
type Target struct {
	Address string
	// Token is the server's own token, from plex.tv's resources.
	Token string
	HTTP  *http.Client
	// Section is a section SlipStream may fill and empty, and Folder one of
	// its Locations.
	Section string
	Folder  link.ServerPath
	Disk    Disk
	Fresh   *Fresh
	Play    func(t *testing.T, p link.ServerPath, d link.Decision) (stop func())
	// Fake runs the cases that would destroy a real server's watch data.
	Fake          bool
	Settle, Bound time.Duration
	Log           *slog.Logger
}

var headers = plex.Headers{ClientID: "slipstream-conformance", Product: "SlipStream", Version: "conformance", Device: "SlipStream"}

func (tg Target) client(t *testing.T, token string) *plex.Client {
	t.Helper()
	base, err := url.Parse(tg.Address)
	if err != nil {
		t.Fatal(err)
	}
	return plex.New(base, token, headers, tg.HTTP)
}

// Run runs every case against tg.
func Run(t *testing.T, tg Target) {
	c := tg.client(t, tg.Token)
	shared := mediaservers.Target{
		Kind: "plex", Server: &server{c: c, tg: tg, scans: map[string]scanMark{}},
		Library: tg.Section, Folder: tg.Folder, Disk: tg.Disk, Fresh: tg.Fresh, Play: tg.Play,
		Settle: tg.Settle, Bound: tg.Bound, Log: tg.Log,
	}
	mediaservers.Shared(t, shared)
	plexCases := []struct {
		name string
		run  func(*testing.T, Target, mediaservers.Target)
	}{
		{"IdentityWithoutToken", IdentityWithoutToken},
		{"RenameInFolderKeepsRatingKey", RenameInFolderKeepsRatingKey},
		{"MovedDestinationThenSource", MovedDestinationThenSource},
		{"SourceFirstPurges", SourceFirstPurges},
		{"BackslashPaths", BackslashPaths},
		{"EditionsOnlyUnderPlexPass", EditionsOnlyUnderPlexPass},
		{"TimelineOnAdd", TimelineOnAdd},
	}
	var cases []mediaservers.Case
	for _, pc := range plexCases {
		cases = append(cases, mediaservers.Case{Name: pc.name, Run: func(t *testing.T) { pc.run(t, tg, shared) }})
	}
	mediaservers.Each(t, cases)
}

// IdentityWithoutToken is the call Reach relies on: /identity answers
// with no token while every other route answers 401 (design correction 4;
// recorded run 2026-11-14 setup, where /identity answered 200 for a
// revoked token).
func IdentityWithoutToken(t *testing.T, tg Target, _ mediaservers.Target) {
	bare := tg.client(t, "")
	id, err := bare.Identity(t.Context())
	if err != nil {
		t.Fatalf("/identity with no token: %v", err)
	}
	if id.MachineIdentifier == "" {
		t.Fatal("/identity names no machineIdentifier")
	}
	if _, err := bare.Sections(t.Context()); !plex.Refused(err) {
		t.Fatalf("/library/sections with no token answered %v, want a 401", err)
	}
	if _, err := tg.client(t, "not-the-server-token").Sections(t.Context()); !plex.Refused(err) {
		t.Fatalf("/library/sections with another token answered %v, want a 401", err)
	}
}

func watched(t *testing.T, tg mediaservers.Target, f *mediaservers.Files, title string) (mediaservers.Item, link.ServerPath) {
	t.Helper()
	r := mediaservers.Rel(t, title, ".mkv")
	f.Place(r)
	f.Tell(r)
	it := f.Found(r)
	if err := tg.Server.Watched(t.Context(), it.Key, "SlipStream conformance"); err != nil {
		t.Fatal(err)
	}
	return it, f.At(r)
}

func kept(t *testing.T, it mediaservers.Item, want int) {
	t.Helper()
	collected := len(it.Collections) == 1 && it.Collections[0] == "SlipStream conformance"
	if it.PlayCount != want || collected != (want > 0) {
		t.Fatalf("item %s has play count %d and collections %v, want %d and the collection kept as much", it.Key, it.PlayCount, it.Collections, want)
	}
}

// RenameInFolderKeepsRatingKey is case B: one refresh of the folder, the
// same ratingKey at the new name, the old part gone, the play count and
// collection kept.
func RenameInFolderKeepsRatingKey(t *testing.T, tg Target, shared mediaservers.Target) {
	f := shared.Case(t)
	defer f.Clear()
	old, from := watched(t, shared, f, "Renamed")
	r := mediaservers.Rel(t, "Renamed", ".mkv")
	to := mediaservers.Rel(t, "Renamed", " Bluray-1080p.mkv")
	f.Move(r, to)
	if err := shared.Server.Moved(t.Context(), tg.Section, []link.ServerPath{from}, []link.ServerPath{f.At(to)}); err != nil {
		t.Fatal(err)
	}
	now := f.Found(to)
	if now.Key != old.Key || len(now.Files) != 1 {
		t.Fatalf("after the rename the item is %+v, want ratingKey %s with one file", now, old.Key)
	}
	kept(t, now, 1)
}

// MovedDestinationThenSource is cases C and D: the destination refresh
// alone moves the file to a new item that carries the play count and
// collection, the old ratingKey is a 404 at once, and the source refresh
// after it changes nothing.
func MovedDestinationThenSource(t *testing.T, tg Target, shared mediaservers.Target) {
	f := shared.Case(t)
	defer f.Clear()
	old, from := watched(t, shared, f, "Moved")
	r := mediaservers.Rel(t, "Moved", ".mkv")
	to := mediaservers.Elsewhere(t, r)
	f.Move(r, to)
	c := tg.client(t, tg.Token)
	if err := c.RefreshPath(t.Context(), plex.ID(tg.Section), f.At(to).Dir().String()); err != nil {
		t.Fatal(err)
	}
	now := f.Found(to)
	kept(t, now, 1)
	mediaservers.Eventually(t, shared, "the old ratingKey gone", func() (bool, error) {
		_, ok, err := shared.Server.Item(t.Context(), old.Key)
		return !ok, err
	})
	if err := c.RefreshPath(t.Context(), plex.ID(tg.Section), from.Dir().String()); err != nil {
		t.Fatal(err)
	}
	after := f.Found(to)
	if after.Key != now.Key {
		t.Fatalf("the source refresh moved the file from item %s to %s", now.Key, after.Key)
	}
	kept(t, after, 1)
}

// SourceFirstPurges is case E, which a real server's watch data can't
// afford: with autoEmptyTrash on, a refresh of the vacated folder first
// deletes the item, and the destination's new item has no play count and
// no collection.
func SourceFirstPurges(t *testing.T, tg Target, shared mediaservers.Target) {
	if !tg.Fake {
		t.Skip("source first destroys watch data; recorded run 2026-11-14 case E is the real server's result")
	}
	f := shared.Case(t)
	defer f.Clear()
	old, from := watched(t, shared, f, "Purged")
	r := mediaservers.Rel(t, "Purged", ".mkv")
	to := mediaservers.Elsewhere(t, r)
	f.Move(r, to)
	c := tg.client(t, tg.Token)
	if err := c.RefreshPath(t.Context(), plex.ID(tg.Section), from.Dir().String()); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := shared.Server.Item(t.Context(), old.Key); err != nil || ok {
		t.Fatalf("the source refresh kept item %s (err %v)", old.Key, err)
	}
	if err := c.RefreshPath(t.Context(), plex.ID(tg.Section), f.At(to).Dir().String()); err != nil {
		t.Fatal(err)
	}
	kept(t, f.Found(to), 0)
}

// BackslashPaths is case A on a Windows server: a refresh path with
// forward slashes works, and Plex stores backslashes either way.
func BackslashPaths(t *testing.T, tg Target, shared mediaservers.Target) {
	if !tg.Folder.Windows() {
		t.Skip("backslash paths are a Windows Plex's; recorded run 2026-11-14 case A covers Jackson's server")
	}
	f := shared.Case(t)
	defer f.Clear()
	r := mediaservers.Rel(t, "Slashed", ".mkv")
	f.Place(r)
	slashed := strings.ReplaceAll(f.At(r).Dir().String(), `\`, "/")
	if err := tg.client(t, tg.Token).RefreshPath(t.Context(), plex.ID(tg.Section), slashed); err != nil {
		t.Fatal(err)
	}
	it := f.Found(r)
	if len(it.Files) != 1 || strings.Contains(it.Files[0].String(), "/") {
		t.Fatalf("the item's files are %v, want one stored with backslashes", it.Files)
	}
}

// EditionsOnlyUnderPlexPass places two editions in one folder. Plex Pass
// makes the tagged one its own item, and without it both are one item.
func EditionsOnlyUnderPlexPass(t *testing.T, tg Target, shared mediaservers.Target) {
	srv, err := tg.client(t, tg.Token).Server(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	f := shared.Case(t)
	defer f.Clear()
	plain := mediaservers.Rel(t, "Edition", ".mkv")
	cut := mediaservers.Rel(t, "Edition", " {edition-Director's Cut}.mkv")
	f.Place(plain, cut)
	f.Tell(plain)
	a, b := f.Found(plain), f.Found(cut)
	if (a.Key != b.Key) != srv.MyPlexSubscription {
		t.Fatalf("with Plex Pass %v the editions are items %s and %s", srv.MyPlexSubscription, a.Key, b.Key)
	}
}

// TimelineOnAdd listens on the notification socket while a file is added,
// and reads a timeline entry for its new item.
func TimelineOnAdd(t *testing.T, tg Target, shared mediaservers.Target) {
	ctx, cancel := context.WithCancel(t.Context())
	items, open, done := make(chan plex.ID, 64), make(chan struct{}), make(chan struct{})
	var listenErr error
	go func() {
		defer close(done)
		listenErr = tg.client(t, tg.Token).Notifications(ctx, func() { close(open) }, func(n plex.Notification) {
			for _, e := range n.Timeline {
				select {
				case items <- e.ItemID:
				case <-ctx.Done():
				}
			}
		})
	}()
	defer func() {
		cancel()
		<-done
		if !errors.Is(listenErr, context.Canceled) {
			t.Errorf("the notification socket ended with %v, want the canceled context", listenErr)
		}
	}()
	select {
	case <-open:
	case <-done:
		t.Fatalf("the notification socket didn't open: %v", listenErr)
	}
	f := shared.Case(t)
	defer f.Clear()
	r := mediaservers.Rel(t, "Announced", ".mkv")
	f.Place(r)
	f.Tell(r)
	it := f.Found(r)
	deadline := time.After(tg.Bound)
	for {
		select {
		case id := <-items:
			if string(id) == it.Key {
				return
			}
		case <-done:
			t.Fatalf("the notification socket closed: %v", listenErr)
		case <-deadline:
			t.Fatalf("no timeline entry for item %s within %v", it.Key, tg.Bound)
		}
	}
}
