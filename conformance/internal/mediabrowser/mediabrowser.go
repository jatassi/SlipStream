// Package mediabrowser is what the Jellyfin and Emby conformance suites
// share: their Target, and the server that drives either product through
// the one wire client the way the kinds do. A change is a path report, a
// move is the carry the kinds run (snapshot every user's data and the
// item's collections, report the old paths Deleted and wait until they
// are gone, report the new paths Created and wait until they exist, then
// write the data back), and a scan's end is the RefreshLibrary task read
// Idle with a LastExecutionResult.EndTimeUtc later than before the scan
// was asked for.
package mediabrowser

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/jellyfin"
)

// Target is the Jellyfin or Emby under test.
type Target struct {
	Address string
	// Key is an API key, as SlipStream holds.
	Key  string
	HTTP *http.Client
	// Library names a library SlipStream may fill and empty, and Folder is
	// one of its folders as the server sees it.
	Library string
	Folder  link.ServerPath
	Disk    mediaservers.Disk
	// Fresh is a library never scanned with content, by name.
	Fresh *mediaservers.Fresh
	// FreshRoot is a folder of Library never scanned with content, beside
	// Folder, nil when the target has none.
	FreshRoot *Root
	// Admin signs in as an administrator for the cases that need a user,
	// nil when the target gives none.
	Admin *Admin
	Play  func(t *testing.T, p link.ServerPath, d link.Decision) (session Played, stop func())
	// Fake runs the cases that need the fake's cues.
	Fake          bool
	Settle, Bound time.Duration
	Log           *slog.Logger
}

// Root is a library folder and the local folder it is.
type Root struct {
	Folder link.ServerPath
	Disk   mediaservers.Disk
}

// Admin is an administrator's name and password.
type Admin struct{ User, Password string }

// Played is a playback a Target's Play started: its session and item, as
// a player reports them.
type Played struct {
	Session string
	Item    string
}

// App is how the suite names itself to the server.
var App = jellyfin.App{Client: "SlipStream", Device: "SlipStream conformance", DeviceID: "slipstream-conformance", Version: "conformance"}

// Client is a wire client to tg with token, the key when "".
func (tg Target) Client(t *testing.T, f jellyfin.Flavor, token string) *jellyfin.Client {
	t.Helper()
	base, err := url.Parse(tg.Address)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		token = tg.Key
	}
	return jellyfin.New(base, f, token, App, tg.HTTP)
}

// Server drives one product through the wire client.
type Server struct {
	C    *jellyfin.Client
	f    jellyfin.Flavor
	tg   Target
	user string
	mark time.Time
}

var _ mediaservers.Server = (*Server)(nil)

// Shared is tg as the shared cases take it, with its server and its
// library keys read from the server.
func Shared(t *testing.T, kind link.KindName, f jellyfin.Flavor, tg Target) (mediaservers.Target, *Server) {
	t.Helper()
	s := &Server{C: tg.Client(t, f, ""), f: f, tg: tg}
	users, err := s.C.Users(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(users, func(u jellyfin.User) bool { return u.Policy.IsAdministrator })
	if i < 0 {
		t.Fatal("the server has no administrator to read watch data as")
	}
	s.user = users[i].ID
	shared := mediaservers.Target{
		Kind: kind, Server: s, Library: s.LibraryKey(t, tg.Library), Folder: tg.Folder, Disk: tg.Disk,
		FreshTakesChange: f == jellyfin.Emby, Settle: tg.Settle, Bound: tg.Bound, Log: tg.Log,
	}
	if tg.Fresh != nil {
		fresh := *tg.Fresh
		fresh.Library = s.LibraryKey(t, fresh.Library)
		shared.Fresh = &fresh
	}
	if tg.Play != nil {
		shared.Play = func(t *testing.T, p link.ServerPath, d link.Decision) func() {
			played, stop := tg.Play(t, p, d)
			if f == jellyfin.Emby {
				s.Report(t, played)
			}
			return stop
		}
	}
	return shared, s
}

// User is the administrator whose data the cases read.
func (s *Server) User() string { return s.user }

// Report is the player reporting the playback, after which Emby shows
// its transcode.
func (s *Server) Report(t *testing.T, p Played) {
	t.Helper()
	if err := s.C.ReportPlaying(t.Context(), jellyfin.Playing{ItemID: p.Item, PlaySessionID: p.Session, PlayMethod: jellyfin.Transcode}); err != nil {
		t.Fatal(err)
	}
}

// LibraryKey is the ItemId of the library named name.
func (s *Server) LibraryKey(t *testing.T, name string) string {
	t.Helper()
	libs, err := s.C.Libraries(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(libs, func(l jellyfin.VirtualFolder) bool { return l.Name == name })
	if i < 0 {
		t.Fatalf("no library %q", name)
	}
	return libs[i].ItemID
}

func (s *Server) Reach(ctx context.Context) (link.Identity, error) {
	info, err := s.C.PublicInfo(ctx)
	if err != nil {
		return link.Identity{}, err
	}
	v, err := link.ParseVersion(info.Version)
	if err != nil {
		return link.Identity{}, err
	}
	return link.Identity{ID: info.ID, Name: info.ServerName, Version: v}, nil
}

func (s *Server) Libraries(ctx context.Context) ([]link.Library, error) {
	vfs, err := s.C.Libraries(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]link.Library, len(vfs))
	for i, vf := range vfs {
		out[i] = link.Library{Key: vf.ItemID, Name: vf.Name}
		for _, l := range vf.Locations {
			p, err := link.ParsePath(l)
			if err != nil {
				return nil, err
			}
			out[i].Folders = append(out[i].Folders, p)
		}
	}
	return out, nil
}

// Tell reports each path Created, or Deleted when it is gone.
func (s *Server) Tell(ctx context.Context, _ string, told []mediaservers.Told) error {
	updates := make([]jellyfin.Update, len(told))
	for i, x := range told {
		updates[i] = jellyfin.Update{Path: x.Path.String(), UpdateType: jellyfin.Created}
		if x.Gone {
			updates[i].UpdateType = jellyfin.Deleted
		}
	}
	return s.C.ReportUpdated(ctx, updates)
}

func (s *Server) report(ctx context.Context, typ jellyfin.UpdateType, paths []link.ServerPath) error {
	updates := make([]jellyfin.Update, len(paths))
	for i, p := range paths {
		updates[i] = jellyfin.Update{Path: p.String(), UpdateType: typ}
	}
	return s.C.ReportUpdated(ctx, updates)
}

// snapshot is one moved item's data: every user's, and its collections.
type snapshot struct {
	data        map[string]jellyfin.UserData
	collections []string
}

// Moved runs the carry: snapshotted, old_removed, new_found, restored.
func (s *Server) Moved(ctx context.Context, library string, from, to []link.ServerPath) error {
	snaps, err := s.snapshot(ctx, library, from)
	if err != nil {
		return err
	}
	if err := s.report(ctx, jellyfin.Deleted, from); err != nil {
		return err
	}
	for _, p := range from {
		if err := s.wait(ctx, library, p, false); err != nil {
			return err
		}
	}
	if err := s.report(ctx, jellyfin.Created, to); err != nil {
		return err
	}
	for i, p := range to {
		if err := s.wait(ctx, library, p, true); err != nil {
			return err
		}
		it, _, err := s.find(ctx, library, p)
		if err != nil {
			return err
		}
		if err := s.restore(ctx, it.ID, snaps[i]); err != nil {
			return err
		}
	}
	return nil
}

// snapshot is every user's data and the collections of the item at each
// path.
func (s *Server) snapshot(ctx context.Context, library string, paths []link.ServerPath) ([]snapshot, error) {
	users, err := s.C.Users(ctx)
	if err != nil {
		return nil, err
	}
	snaps := make([]snapshot, len(paths))
	for i, p := range paths {
		it, ok, err := s.find(ctx, library, p)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("no item at %s to snapshot", p)
		}
		snaps[i].data = map[string]jellyfin.UserData{}
		for _, u := range users {
			d, err := s.C.UserData(ctx, u.ID, it.ID)
			if err != nil {
				return nil, err
			}
			snaps[i].data[u.ID] = d
		}
		if snaps[i].collections, err = s.collectionsOf(ctx, it.ID); err != nil {
			return nil, err
		}
	}
	return snaps, nil
}

func (s *Server) restore(ctx context.Context, item string, snap snapshot) error {
	for user, d := range snap.data {
		if err := s.C.SetUserData(ctx, user, item, d); err != nil {
			return err
		}
	}
	for _, c := range snap.collections {
		if err := s.collect(ctx, c, item); err != nil {
			return err
		}
	}
	return nil
}

// wait polls until the server holds an item at p, or none when present
// is false, up to the target's Bound.
func (s *Server) wait(ctx context.Context, library string, p link.ServerPath, present bool) error {
	deadline := time.Now().Add(s.tg.Bound)
	for {
		_, ok, err := s.find(ctx, library, p)
		if err != nil {
			return err
		}
		if ok == present {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("the item at %s didn't %s within %v", p, map[bool]string{true: "appear", false: "go"}[present], s.tg.Bound)
		}
		time.Sleep(mediaservers.Poll)
	}
}

// SidecarRemoved reports the Sidecar's path Deleted and the video's
// Modified.
func (s *Server) SidecarRemoved(ctx context.Context, _ string, video, sidecar link.ServerPath) error {
	return s.C.ReportUpdated(ctx, []jellyfin.Update{
		{Path: sidecar.String(), UpdateType: jellyfin.Deleted}, {Path: video.String(), UpdateType: jellyfin.Modified},
	})
}

func (s *Server) task(ctx context.Context) (jellyfin.Task, error) {
	t, ok, err := s.C.Task(ctx, jellyfin.RefreshLibraryTask)
	if err != nil {
		return jellyfin.Task{}, err
	}
	if !ok {
		return jellyfin.Task{}, fmt.Errorf("the server has no %s task", jellyfin.RefreshLibraryTask)
	}
	return t, nil
}

func lastEnd(t jellyfin.Task) time.Time {
	if t.LastExecutionResult == nil {
		return time.Time{}
	}
	return t.LastExecutionResult.EndTimeUtc
}

// Scan starts the RefreshLibrary task, a scan of every library, after
// marking the task's last end.
func (s *Server) Scan(ctx context.Context, _ string) error {
	t, err := s.task(ctx)
	if err != nil {
		return err
	}
	s.mark = lastEnd(t)
	return s.C.RefreshAll(ctx)
}

// ScanEnded is graft 3's rule: the task is Idle and ended after the mark.
// Reading the server's own last end before the request keeps the rule
// free of the two clocks' skew.
func (s *Server) ScanEnded(ctx context.Context, _ string) (bool, error) {
	t, err := s.task(ctx)
	if err != nil {
		return false, err
	}
	return t.State == jellyfin.TaskIdle && lastEnd(t).After(s.mark), nil
}

func has(it jellyfin.Item, p link.ServerPath) bool {
	paths := []string{it.Path}
	for _, ms := range it.MediaSources {
		paths = append(paths, ms.Path)
	}
	return slices.ContainsFunc(paths, func(x string) bool {
		f, err := link.ParsePath(x)
		return err == nil && f.Key() == p.Key()
	})
}

// find is the item with a file at p: Emby's ?Path= lookup, or on
// Jellyfin, which has none, a page-by-page listing of the library.
func (s *Server) find(ctx context.Context, library string, p link.ServerPath) (jellyfin.Item, bool, error) {
	if s.f == jellyfin.Emby {
		items, err := s.C.ItemsAt(ctx, s.user, p.String())
		if err != nil || len(items) == 0 {
			return jellyfin.Item{}, false, err
		}
		return items[0], true, nil
	}
	const size = 100
	for start := 0; ; start += size {
		page, err := s.C.Items(ctx, jellyfin.Query{
			User: s.user, Parent: library, Recursive: true, Types: []string{jellyfin.TypeMovie, jellyfin.TypeEpisode}, Start: start, Limit: size,
		})
		if err != nil {
			return jellyfin.Item{}, false, err
		}
		for _, it := range page.Items {
			if has(it, p) {
				return it, true, nil
			}
		}
		if start+size >= page.Total {
			return jellyfin.Item{}, false, nil
		}
	}
}

func (s *Server) item(ctx context.Context, it jellyfin.Item) (mediaservers.Item, error) {
	out := mediaservers.Item{Key: it.ID}
	for _, ms := range it.MediaSources {
		f, err := link.ParsePath(ms.Path)
		if err != nil {
			return mediaservers.Item{}, err
		}
		out.Files = append(out.Files, f)
		for _, st := range ms.MediaStreams {
			if st.IsExternal && st.Type == "Subtitle" {
				out.Sidecars++
			}
		}
	}
	if d := it.UserData; d != nil {
		out.Played, out.PlayCount = d.Played, d.PlayCount
	}
	var err error
	out.Collections, err = s.collectionsOf(ctx, it.ID)
	return out, err
}

func (s *Server) ItemAt(ctx context.Context, library string, p link.ServerPath) (mediaservers.Item, bool, error) {
	it, ok, err := s.find(ctx, library, p)
	if err != nil || !ok {
		return mediaservers.Item{}, false, err
	}
	out, err := s.item(ctx, it)
	return out, err == nil, err
}

func (s *Server) Item(ctx context.Context, key string) (mediaservers.Item, bool, error) {
	it, ok, err := s.C.Item(ctx, s.user, key)
	if err != nil || !ok {
		return mediaservers.Item{}, false, err
	}
	out, err := s.item(ctx, it)
	return out, err == nil, err
}

// Collection is one collection and its members' ids.
type Collection struct {
	ID      string
	Members []string
}

// Collections are the server's collections by name.
func (s *Server) Collections(ctx context.Context) (map[string]Collection, error) {
	page, err := s.C.Items(ctx, jellyfin.Query{User: s.user, Recursive: true, Types: []string{jellyfin.TypeBoxSet}})
	if err != nil {
		return nil, err
	}
	out := map[string]Collection{}
	for _, c := range page.Items {
		in, err := s.C.Items(ctx, jellyfin.Query{User: s.user, Parent: c.ID})
		if err != nil {
			return nil, err
		}
		col := Collection{ID: c.ID}
		for _, it := range in.Items {
			col.Members = append(col.Members, it.ID)
		}
		out[c.Name] = col
	}
	return out, nil
}

func (s *Server) collectionsOf(ctx context.Context, item string) ([]string, error) {
	all, err := s.Collections(ctx)
	if err != nil {
		return nil, err
	}
	var out []string
	for name, c := range all {
		if slices.Contains(c.Members, item) {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out, nil
}

// collect puts item in the collection name, making it when it doesn't
// exist.
func (s *Server) collect(ctx context.Context, name, item string) error {
	all, err := s.Collections(ctx)
	if err != nil {
		return err
	}
	if c, ok := all[name]; ok {
		return s.C.AddToCollection(ctx, c.ID, []string{item})
	}
	_, err = s.C.CreateCollection(ctx, name, []string{item})
	return err
}

// Watched marks item played for the administrator and puts it in the
// collection name, as a case seeds it. SlipStream itself marks nothing
// played.
func (s *Server) Watched(ctx context.Context, item, name string) error {
	if err := s.C.MarkPlayed(ctx, s.user, item); err != nil {
		return err
	}
	return s.collect(ctx, name, item)
}

var decisions = map[bool]link.Decision{true: link.DecisionCopy, false: link.DecisionTranscode}

func (s *Server) Sessions(ctx context.Context) ([]link.Session, error) {
	sessions, err := s.C.Sessions(ctx)
	if err != nil {
		return nil, err
	}
	var out []link.Session
	for _, sess := range sessions {
		if sess.NowPlayingItem == nil {
			continue
		}
		ls := link.Session{Key: sess.ID, User: sess.UserName, Player: sess.DeviceName, Product: sess.Client, Decision: link.DecisionDirectPlay}
		if ls.Path, err = link.ParsePath(sess.NowPlayingItem.Path); err != nil {
			return nil, err
		}
		if ti := sess.TranscodingInfo; ti != nil && sess.PlayState != nil && sess.PlayState.PlayMethod == jellyfin.Transcode {
			ls.Decision, ls.Reasons = decisions[ti.IsVideoDirect], ti.TranscodeReasons
			ls.Hardware = ti.VideoEncoderIsHardware || (ti.HardwareAccelerationType != "" && ti.HardwareAccelerationType != "none")
		}
		out = append(out, ls)
	}
	return out, nil
}

// SocketPushesSessions opens the server's socket with the key, each
// product's way, asks for sessions and reads a Sessions message.
func (s *Server) SocketPushesSessions(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	got, done := make(chan struct{}, 1), make(chan struct{})
	var listenErr error
	go func() {
		defer close(done)
		listenErr = s.C.Socket(ctx, time.Second, func() {}, func(m jellyfin.Message) {
			if m.MessageType != jellyfin.MessageSessions {
				return
			}
			if _, err := m.Sessions(); err != nil {
				return
			}
			select {
			case got <- struct{}{}:
			default:
			}
		})
	}()
	defer func() {
		cancel()
		<-done
		if !errors.Is(listenErr, context.Canceled) {
			t.Errorf("the socket ended with %v, want the canceled context", listenErr)
		}
	}()
	select {
	case <-got:
	case <-done:
		t.Fatalf("the socket closed: %v", listenErr)
	case <-time.After(s.tg.Bound):
		t.Fatalf("no Sessions message within %v", s.tg.Bound)
	}
}
