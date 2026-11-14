// Package jellyfin is the Jellyfin conformance suite: the cases every kind
// shares, then Jellyfin's own, from #71's probes and recorded run
// 2026-11-14 filed under recorded/. It runs against fake-jellyfin on
// every PR and against a real Jellyfin behind the online tag.
package jellyfin

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/mediabrowser"
	"github.com/jatassi/slipstream/conformance/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/jellyfin"
)

// The shared suite's types, for the tests outside conformance that build
// a Target.
type (
	Target  = mediabrowser.Target
	Root    = mediabrowser.Root
	Admin   = mediabrowser.Admin
	Played  = mediabrowser.Played
	Disk    = mediaservers.Disk
	DirDisk = mediaservers.DirDisk
	Fresh   = mediaservers.Fresh
)

// Run runs every case against tg.
func Run(t *testing.T, tg Target) {
	shared, s := mediabrowser.Shared(t, "jellyfin", jellyfin.Jellyfin, tg)
	mediaservers.Shared(t, shared)
	jf := []struct {
		name string
		run  func(*testing.T, Target, mediaservers.Target, *mediabrowser.Server)
	}{
		{"MonitorDelayIsConfigured", MonitorDelayIsConfigured},
		{"FreshRootNeedsScanWithContent", FreshRootNeedsScanWithContent},
		{"ReportDroppedByScanIsSentAgain", ReportDroppedByScanIsSentAgain},
		{"TwoReportsInOneDelay", TwoReportsInOneDelay},
		{"VersionsOnOneItem", VersionsOnOneItem},
		{"CollectionAddIsIdempotent", CollectionAddIsIdempotent},
		{"EmbyFormsRefused", EmbyFormsRefused},
		{"KeyCreatedWithNoBody", KeyCreatedWithNoBody},
		{"DeviceIDRevokes", DeviceIDRevokes},
		{"SocketPushesSessions", func(t *testing.T, _ Target, _ mediaservers.Target, s *mediabrowser.Server) { s.SocketPushesSessions(t) }},
	}
	var cases []mediaservers.Case
	for _, c := range jf {
		cases = append(cases, mediaservers.Case{Name: c.name, Run: func(t *testing.T) { c.run(t, tg, shared, s) }})
	}
	mediaservers.Each(t, cases)
}

func delay(t *testing.T, s *mediabrowser.Server) time.Duration {
	t.Helper()
	cfg, err := s.C.Configuration(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return cfg.MonitorDelay
}

// MonitorDelayIsConfigured reads LibraryMonitorDelay, which the sender
// waits out and never changes.
func MonitorDelayIsConfigured(t *testing.T, tg Target, _ mediaservers.Target, s *mediabrowser.Server) {
	if d := delay(t, s); d <= 0 || d > tg.Settle {
		t.Fatalf("the monitor delay is %v, want more than 0 and within the target's settle %v", d, tg.Settle)
	}
}

func scanned(t *testing.T, shared mediaservers.Target) {
	t.Helper()
	if err := shared.Server.Scan(t.Context(), shared.Library); err != nil {
		t.Fatal(err)
	}
	mediaservers.Eventually(t, shared, "the full scan's end", func() (bool, error) { return shared.Server.ScanEnded(t.Context(), shared.Library) })
}

func absent(t *testing.T, shared mediaservers.Target, what string, p link.ServerPath) {
	t.Helper()
	if _, ok, err := shared.Server.ItemAt(t.Context(), shared.Library, p); err != nil || ok {
		t.Fatalf("%s made an item (err %v)", what, err)
	}
}

// FreshRootNeedsScanWithContent is recorded run 2026-11-14: a library
// with a seeded folder and a fresh one. A report into the fresh folder is
// ignored, even after a full scan that ran while it was empty. A full
// scan with the file on disk picks it up, later reports count, and the
// folder keeps counting them once it is empty again.
func FreshRootNeedsScanWithContent(t *testing.T, tg Target, shared mediaservers.Target, _ *mediabrowser.Server) {
	if tg.FreshRoot == nil {
		t.Skip("a real server's fresh root is recorded run 2026-11-14 under recorded/, since SlipStream adds no library folder")
	}
	root := shared
	root.Folder, root.Disk = tg.FreshRoot.Folder, tg.FreshRoot.Disk
	f := root.Case(t)
	defer f.Clear()
	bravo, charlie, delta := mediaservers.Rel(t, "Bravo", ".mkv"), mediaservers.Rel(t, "Charlie", ".mkv"), mediaservers.Rel(t, "Delta", ".mkv")
	scanned(t, shared)
	f.Place(bravo)
	f.Tell(bravo)
	time.Sleep(tg.Settle)
	absent(t, shared, "a report into a root scanned only while empty", f.At(bravo))
	scanned(t, shared)
	f.Found(bravo)
	f.Place(charlie)
	f.Tell(charlie)
	f.Found(charlie)
	f.Remove(bravo)
	f.Remove(charlie)
	f.Tell(bravo, charlie)
	f.Gone(bravo)
	f.Gone(charlie)
	f.Place(delta)
	f.Tell(delta)
	f.Found(delta)
}

// ReportDroppedByScanIsSentAgain is Jellyfin dropping every pending
// report when a full scan starts: a report sent before the scan, for a
// file that lands after the scan read its folder, does nothing. The same
// report sent again after the scan ends counts.
func ReportDroppedByScanIsSentAgain(t *testing.T, tg Target, shared mediaservers.Target, _ *mediabrowser.Server) {
	f := shared.Case(t)
	defer f.Clear()
	r := mediaservers.Rel(t, "Dropped", ".mkv")
	if err := shared.Server.Tell(t.Context(), shared.Library, []mediaservers.Told{{Path: f.At(r)}}); err != nil {
		t.Fatal(err)
	}
	scanned(t, shared)
	f.Place(r)
	time.Sleep(tg.Settle)
	absent(t, shared, "a report the full scan dropped", f.At(r))
	f.Tell(r)
	f.Found(r)
}

// TwoReportsInOneDelay pins what a repeat report does: it restarts that
// path's timer, so the item shows a monitor delay after the second report
// and not after the first.
func TwoReportsInOneDelay(t *testing.T, _ Target, shared mediaservers.Target, s *mediabrowser.Server) {
	d := delay(t, s)
	f := shared.Case(t)
	defer f.Clear()
	r := mediaservers.Rel(t, "Repeated", ".mkv")
	f.Place(r)
	f.Tell(r)
	time.Sleep(d / 2)
	f.Tell(r)
	time.Sleep(d/2 + d/4)
	absent(t, shared, "a path reported twice, a monitor delay after the first report,", f.At(r))
	f.Found(r)
}

// VersionsOnOneItem places two versions of one film in its folder. Both
// are MediaSources of one item (#71 delta 9).
func VersionsOnOneItem(t *testing.T, _ Target, shared mediaservers.Target, _ *mediabrowser.Server) {
	f := shared.Case(t)
	defer f.Clear()
	a, b := mediaservers.Rel(t, "Versioned", " - 1080p.mkv"), mediaservers.Rel(t, "Versioned", " - 2160p.mkv")
	f.Place(a, b)
	f.Tell(a, b)
	x, y := f.Found(a), f.Found(b)
	if x.Key != y.Key || len(x.Files) != 2 {
		t.Fatalf("the versions are items %s and %s, the first with files %v", x.Key, y.Key, x.Files)
	}
}

// CollectionAddIsIdempotent adds an item to a collection it is already
// in. The server answers success and lists it once.
func CollectionAddIsIdempotent(t *testing.T, _ Target, shared mediaservers.Target, s *mediabrowser.Server) {
	f := shared.Case(t)
	defer f.Clear()
	r := mediaservers.Rel(t, "Collected", ".mkv")
	f.Place(r)
	f.Tell(r)
	it := f.Found(r)
	name := "SlipStream conformance twice"
	id, err := s.C.CreateCollection(t.Context(), name, []string{it.Key})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.C.AddToCollection(t.Context(), id, []string{it.Key}); err != nil {
		t.Fatalf("adding an item to a collection it is in: %v", err)
	}
	all, err := s.Collections(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got := all[name].Members; len(got) != 1 || got[0] != it.Key {
		t.Fatalf("the collection holds %v, want item %s once", got, it.Key)
	}
}

func call(t *testing.T, tg Target, path string, q url.Values, header http.Header) int {
	t.Helper()
	u, err := url.Parse(tg.Address)
	if err != nil {
		t.Fatal(err)
	}
	u = u.JoinPath(path)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range header {
		req.Header[k] = v
	}
	resp, err := tg.HTTP.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

// EmbyFormsRefused is 12.x's legacy authorization being off: the key
// works in Jellyfin's Authorization header and ?ApiKey=, and answers 401
// in X-Emby-Token, X-Emby-Authorization and ?api_key=.
func EmbyFormsRefused(t *testing.T, tg Target, _ mediaservers.Target, _ *mediabrowser.Server) {
	if got := call(t, tg, "/Users", url.Values{"ApiKey": {tg.Key}}, nil); got != http.StatusOK {
		t.Fatalf("?ApiKey= answered %d", got)
	}
	for what, c := range map[string]struct {
		q url.Values
		h http.Header
	}{
		"X-Emby-Token":         {h: http.Header{"X-Emby-Token": {tg.Key}}},
		"X-Emby-Authorization": {h: http.Header{"X-Emby-Authorization": {`Emby Client="SlipStream", Device="d", DeviceId="d", Version="1", Token="` + tg.Key + `"`}}},
		"?api_key=":            {q: url.Values{"api_key": {tg.Key}}},
	} {
		if got := call(t, tg, "/Users", c.q, c.h); got != http.StatusUnauthorized {
			t.Fatalf("the key in %s answered %d, want 401", what, got)
		}
	}
}

// KeyCreatedWithNoBody is the administrator sign-in's key: POST
// /Auth/Keys answers 204 with no body, the key list holds it, it works,
// and revoked it answers 401.
func KeyCreatedWithNoBody(t *testing.T, tg Target, _ mediaservers.Target, _ *mediabrowser.Server) {
	if tg.Admin == nil {
		t.Skip("the target gives no administrator to sign in as")
	}
	admin := signIn(t, tg, mediabrowser.App)
	defer func() { _ = admin.Logout(context.WithoutCancel(t.Context())) }()
	app := "SlipStream conformance " + time.Now().Format("150405.000")
	if err := admin.CreateKey(t.Context(), app); err != nil {
		t.Fatal(err)
	}
	keys, err := admin.Keys(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(keys, func(k jellyfin.Key) bool { return k.AppName == app })
	if i < 0 {
		t.Fatalf("no key named %q in %d keys", app, len(keys))
	}
	key := tg.Client(t, jellyfin.Jellyfin, keys[i].AccessToken)
	if _, err := key.Libraries(t.Context()); err != nil {
		t.Fatalf("the new key: %v", err)
	}
	if err := admin.RevokeKey(t.Context(), keys[i].AccessToken); err != nil {
		t.Fatal(err)
	}
	if _, err := key.Libraries(t.Context()); !jellyfin.Refused(err) {
		t.Fatalf("the revoked key answered %v, want a 401", err)
	}
}

func signIn(t *testing.T, tg Target, app jellyfin.App) *jellyfin.Client {
	t.Helper()
	base, err := url.Parse(tg.Address)
	if err != nil {
		t.Fatal(err)
	}
	auth, err := jellyfin.New(base, jellyfin.Jellyfin, "", app, tg.HTTP).AuthenticateByName(t.Context(), tg.Admin.User, tg.Admin.Password)
	if err != nil {
		t.Fatal(err)
	}
	return jellyfin.New(base, jellyfin.Jellyfin, auth.AccessToken, app, tg.HTTP)
}

// DeviceIDRevokes signs in twice with one DeviceId. The first token then
// answers 401 (#71 delta 13).
func DeviceIDRevokes(t *testing.T, tg Target, _ mediaservers.Target, _ *mediabrowser.Server) {
	if tg.Admin == nil {
		t.Skip("the target gives no administrator to sign in as")
	}
	app := mediabrowser.App
	app.DeviceID = "slipstream-conformance-revoke"
	first := signIn(t, tg, app)
	second := signIn(t, tg, app)
	defer func() { _ = second.Logout(context.WithoutCancel(t.Context())) }()
	if _, err := first.Sessions(t.Context()); !jellyfin.Refused(err) {
		t.Fatalf("the first token answered %v after a second sign-in on its DeviceId, want a 401", err)
	}
	if _, err := second.Sessions(t.Context()); err != nil {
		t.Fatalf("the second token: %v", err)
	}
}
