package mediaserverlink_test

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	fakeemby "github.com/jatassi/slipstream/fakes/emby"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	fakejellyfin "github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/fakes/scenario"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	wire "github.com/jatassi/slipstream/internal/providers/jellyfin"
	"github.com/jatassi/slipstream/internal/storage"
)

var at = time.Date(2026, 11, 14, 14, 2, 3, 456_000_000, time.UTC)

func path(t *testing.T, s string) link.ServerPath {
	t.Helper()
	p, err := link.ParsePath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func server() link.ServerRef {
	return link.ServerRef{ID: link.ServerID{UUID: storage.NewUUID()}, Kind: "jellyfin", Identity: "f4c1", SettingsVersion: 3}
}

func change() link.ChangeID { return link.ChangeID{UUID: storage.NewUUID()} }

func snapshot() *link.Snapshot {
	return &link.Snapshot{
		Item: link.ItemKey{Item: "a1", Top: "s1"},
		Users: []link.UserData{{User: "Sam", Played: true, PlayCount: 2, Position: 754 * time.Second,
			Favorite: true, LastPlayed: at, Rating: 8.5}},
		Collections: []string{"Noir"},
	}
}

func works(t *testing.T) []link.Work {
	return []link.Work{
		link.Refresh{Changes: []link.JobChange{
			{ID: change(), Kind: link.ChangeAdded, Library: "1", Path: path(t, `D:\Movies\Heat (1995)\Heat (1995).mkv`)},
			{ID: change(), Kind: link.ChangeSidecarRemoved, Library: "1", Path: path(t, "/media/movies/Ran/Ran.en.srt")},
		}},
		link.Carry{Changes: []link.JobChange{{ID: change(), Kind: link.ChangeMoved, Library: "2",
			Path: path(t, "/media/shows/Show/S01E01.mkv"), From: path(t, "/media/shows/Show/Show - 1x01.mkv"), Snapshot: snapshot()}}},
		link.Fresh{Library: "3", Folder: path(t, `\\nas\media\music`), Requested: at, Started: at.Add(time.Second), Changes: []link.ChangeID{change()}},
		link.Rescan{Libraries: []string{"1", "2"}, Changes: []link.ChangeID{change()}},
		link.Lookup{Paths: []link.ServerPath{path(t, "/media/movies/Heat.mkv")}},
		link.Index{Library: "1", Cursor: "500"},
		link.Libraries{},
	}
}

func TestAJobCrossesTheWireUnchanged(t *testing.T) {
	s := server()
	for _, w := range works(t) {
		j := link.Job{Server: s, Work: w, Step: link.StepOldRemoved, At: at}
		got, err := mediaserverlink.JobOf(s, mediaserverlink.JobOnWire(j))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, j) {
			t.Errorf("%T crossed as %+v, want %+v", w, got, j)
		}
	}
}

func TestAnAnswerCrossesTheWireUnchanged(t *testing.T) {
	s := server()
	a := link.Advanced{
		Job:  link.Job{Server: s, Work: works(t)[1], Step: link.StepRestored, At: at},
		Done: true,
		Wait: at.Add(time.Minute),
		Outcomes: []link.Outcome{
			{Change: change(), Result: link.ResultAccepted, Lost: []link.Loss{link.LostPlayCount, link.LostLastPlayed}},
			{Change: change(), Result: link.ResultConverted},
		},
		Learned:   []link.IndexEntry{{Path: path(t, `D:\Movies\Heat.mkv`), Library: "1", Item: link.ItemKey{Item: "7", Top: "7"}, Gone: true}},
		Libraries: []link.Library{{Key: "1", Name: "Movies", Folders: []link.ServerPath{path(t, `D:\Movies`), path(t, `E:\Movies`)}}},
	}
	got, err := mediaserverlink.AdvancedOf(s, mediaserverlink.AdvancedOnWire(a))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, a) {
		t.Errorf("answer crossed as %+v, want %+v", got, a)
	}
}

func TestACheckCrossesTheWireUnchanged(t *testing.T) {
	c := link.Checked{
		Identity:     link.Identity{ID: "f4c1", Name: "Den Jellyfin", Version: link.MustServerVersion("12.2.0")},
		MonitorDelay: 60 * time.Second,
		Libraries:    []link.Library{{Key: "a", Name: "Shows", Folders: []link.ServerPath{path(t, "/media/shows")}}},
	}
	m, err := mediaserverlink.CheckedOnWire(c, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := mediaserverlink.CheckedOf(m)
	if err != nil || !reflect.DeepEqual(got, c) {
		t.Errorf("check crossed as %+v, %v, want %+v", got, err, c)
	}
	refused := &link.ServerError{Problem: link.Problem{Reason: link.ReasonRefused, Status: 401}}
	m, err = mediaserverlink.CheckedOnWire(link.Checked{}, refused)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mediaserverlink.CheckedOf(m); !problemIs(err, refused) {
		t.Errorf("a refused check crossed as %v, want %v", err, refused)
	}
}

// TestExposedCrossesTheWireUnchanged carries the recorded run's Plex, a
// server that reports nothing beyond its address, and a refusal.
func TestExposedCrossesTheWireUnchanged(t *testing.T) {
	for name, x := range map[string]link.Exposed{
		"Plex, as recorded": {
			Local: netip.MustParseAddrPort("10.0.0.5:32400"), Remote: link.Yes, MappingRoot: "mapped",
			Mapping: &link.PlexMapping{State: "mapped", Pending: true, Public: netip.MustParseAddrPort("203.0.113.9:32400")},
			Custom:  []string{"http://5.161.67.8:32400", "https://plex.atassi.org:443"}, IPv6: link.Yes,
		},
		"remote off, nothing mapped": {Remote: link.No, Mapping: &link.PlexMapping{State: "unknown"}, IPv6: link.No},
		"unmeasured Jellyfin":        {Local: netip.MustParseAddrPort("[fd72::5]:8096")},
	} {
		m, err := mediaserverlink.ExposedOnWire(x, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := mediaserverlink.ExposedOf(m); err != nil || !reflect.DeepEqual(got, x) {
			t.Errorf("%s crossed as %+v, %v, want %+v", name, got, err, x)
		}
	}
	refused := &link.ServerError{Problem: link.Problem{Reason: link.ReasonRefused, Status: 401}}
	m, err := mediaserverlink.ExposedOnWire(link.Exposed{}, refused)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mediaserverlink.ExposedOf(m); !problemIs(err, refused) {
		t.Errorf("a refused Exposed crossed as %v, want %v", err, refused)
	}
}

func TestACredentialCrossesTheWireUnchanged(t *testing.T) {
	c := link.Credential{Token: "token", APIKey: "key"}
	if got := mediaserverlink.CredentialOf(mediaserverlink.CredentialOnWire(c)); got != c {
		t.Errorf("credential crossed changed")
	}
}

// node plays the reaching Node's side of Advance over conn, as the holder's
// handler does.
type node struct {
	server link.ServerRef
	conn   func(link.Job) (link.Advanced, error)
}

func (n node) Advance(_ context.Context, req *msv1.AdvanceRequest) (*msv1.AdvanceResponse, error) {
	j, err := mediaserverlink.JobOf(n.server, req.GetJob())
	if err != nil {
		return nil, err
	}
	a, err := n.conn(j)
	if err != nil {
		p, err := mediaserverlink.ProblemOnWire(err)
		return &msv1.AdvanceResponse{Problem: p}, err
	}
	return mediaserverlink.AdvancedOnWire(a), nil
}

func TestRemoteAnswersAsTheReachingNodesConn(t *testing.T) {
	s := server()
	j := link.Job{Server: s, Work: works(t)[0], Step: link.StepIdle, At: at}
	want := link.Advanced{Job: link.Job{Server: s, Work: j.Work, Step: link.StepReported, At: at}, Wait: at.Add(time.Minute)}
	var got link.Job
	port := mediaserverlink.Remote(node{s, func(j link.Job) (link.Advanced, error) { got = j; return want, nil }})
	a, err := port.Advance(t.Context(), j)
	if err != nil || !reflect.DeepEqual(a, want) || !reflect.DeepEqual(got, j) {
		t.Fatalf("Remote ran %+v and answered %+v, %v; want %+v answered %+v", got, a, err, j, want)
	}
}

func TestRemoteKeepsAServersFailureApartFromTheLinks(t *testing.T) {
	s := server()
	failed := &link.ServerError{Problem: link.Problem{Reason: link.ReasonOtherServer, Found: "9a0b"}}
	_, err := mediaserverlink.Remote(node{s, func(link.Job) (link.Advanced, error) { return link.Advanced{}, failed }}).
		Advance(t.Context(), link.Job{Server: s, Work: link.Libraries{}, Step: link.StepNone})
	if !problemIs(err, failed) {
		t.Errorf("a server's failure crossed as %v, want %v", err, failed)
	}
	down := context.DeadlineExceeded
	_, err = mediaserverlink.Remote(node{s, func(link.Job) (link.Advanced, error) { return link.Advanced{}, down }}).
		Advance(t.Context(), link.Job{Server: s, Work: link.Libraries{}, Step: link.StepNone})
	if _, ok := link.ServerProblem(err); ok || !errors.Is(err, down) {
		t.Errorf("a link failure crossed as %v, want the deadline", err)
	}
}

func problemIs(err error, want *link.ServerError) bool {
	p, ok := link.ServerProblem(err)
	return ok && p == want.Problem
}

// worker plays the reaching Node's Members over users, as the holder's
// handler does.
type worker struct{ users link.Users }

func (w worker) Members(ctx context.Context, req *msv1.MembersRequest) (*msv1.MembersResponse, error) {
	return mediaserverlink.MembersOnWire(ctx, w.users, req)
}

const denServerID = "ddc0b5b7f2824180aae9f2f245dd00ca"

// served is Den on fake p with a Movies and a Shows library, and jackson
// its administrator.
func served(t *testing.T, p link.Kind) (faketest.Served, link.Conn, *wire.Client) {
	t.Helper()
	media := t.TempDir()
	for _, dir := range []string{"Movies", "Shows"} {
		if err := os.MkdirAll(filepath.Join(media, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	const key = "den-key"
	flavor, spec, version := wire.Jellyfin, fakejellyfin.Playing, "12.4.0"
	if p.About().Name == "emby" {
		flavor, spec, version = wire.Emby, fakeemby.Playing, "4.10.1.0"
	}
	f := faketest.InMemory(t, spec(fakejellyfin.Server{
		Name: "Den", Version: version, ID: denServerID, Keys: []string{key},
		Users: []fakejellyfin.User{{Name: "jackson", Password: "admin-password", Admin: true}},
		Root:  "/media",
		Libraries: []fakejellyfin.Library{
			{Name: "Movies", Type: "movies", Folders: []string{"/media/Movies"}},
			{Name: "Shows", Type: "tvshows", Folders: []string{"/media/Shows"}},
		},
	}), scenario.Scenario{Seed: "members"}, faketest.WithMedia(media), faketest.As("Den"))
	base, err := url.Parse(f.Address.URL(""))
	if err != nil {
		t.Fatal(err)
	}
	conn := p.Dial(link.Dialed{
		Server:  link.ServerRef{ID: link.ServerID{UUID: storage.NewUUID()}, Kind: p.About().Name, Identity: denServerID},
		Address: f.Address.URL(""), Credential: link.Credential{APIKey: key}, DeviceID: "den-device", HTTP: f.Client, Timings: link.DefaultTimings(),
	})
	return f, conn, wire.New(base, flavor, key, wire.App{Client: "test", Device: "test", DeviceID: "test", Version: "1"}, f.Client)
}

// held is the user id as the server holds it: its name, flags and
// folders, and whether pw signs it in, or gone.
func held(t *testing.T, admin *wire.Client, id string, pw link.Password) string {
	t.Helper()
	all, err := admin.Users(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(all, func(u wire.User) bool { return u.ID == id })
	if i < 0 {
		return "gone"
	}
	u := all[i]
	_, err = admin.AuthenticateByName(t.Context(), u.Name, pw.Reveal())
	p := u.Policy
	return fmt.Sprintf("%s admin=%t disabled=%t all=%t folders=%v signs-in=%t", u.Name, p.IsAdministrator, p.IsDisabled, p.EnableAllFolders, p.EnabledFolders, err == nil)
}

// TestUsersCrossTheWireToTheServer runs each users call through
// RemoteUsers and the Worker's MembersOnWire on Jellyfin and Emby, and
// reads after each what the server now holds.
func TestUsersCrossTheWireToTheServer(t *testing.T) {
	for _, name := range []link.KindName{"jellyfin", "emby"} {
		t.Run(string(name), func(t *testing.T) { synctest.Test(t, func(t *testing.T) { crossUsers(t, name) }) })
	}
}

// answered is whether err is what ok accepts, or nil with no ok.
func answered(err error, ok func(error) bool) bool {
	if ok == nil {
		return err == nil
	}
	return ok(err)
}

func takenName(err error) bool {
	r, ok := errors.AsType[*link.NameRefusedError](err)
	return ok && r.Taken
}

func crossUsers(t *testing.T, name link.KindName) {
	t.Helper()
	pw, err := link.NewPassword("sam-password")
	if err != nil {
		t.Fatal(err)
	}
	k, _ := mediaserverlink.Kinds.Of(name)
	_, conn, admin := served(t, k)
	direct, _ := mediaserverlink.UsersOf(conn)
	users := mediaserverlink.RemoteUsers(worker{direct}, link.ServerRef{ID: link.ServerID{UUID: storage.NewUUID()}})
	vfs, err := admin.Libraries(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	movies, folder := vfs[0].ItemID, map[bool]string{true: vfs[0].GUID, false: vfs[0].ItemID}[name == "emby"]
	sam, err := users.CreateUser(t.Context(), "slipstream-0123456789ab")
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		name string
		call func() error
		ok   func(error) bool
		want string
	}{
		{"restrict", func() error { return users.Restrict(t.Context(), sam, []string{movies}) }, nil,
			"slipstream-0123456789ab admin=false disabled=false all=false folders=[" + folder + "] signs-in=false"},
		{"set the password", func() error { return users.SetPassword(t.Context(), sam, pw) }, nil,
			"slipstream-0123456789ab admin=false disabled=false all=false folders=[" + folder + "] signs-in=true"},
		{"rename", func() error { return users.Rename(t.Context(), sam, "sam") }, nil,
			"sam admin=false disabled=false all=false folders=[" + folder + "] signs-in=true"},
		{"rename to a taken name", func() error { return users.Rename(t.Context(), sam, "Jackson") }, takenName,
			"sam admin=false disabled=false all=false folders=[" + folder + "] signs-in=true"},
		{"restrict a user the server lacks", func() error { return users.Restrict(t.Context(), "f00d", []string{movies}) }, link.Gone,
			"sam admin=false disabled=false all=false folders=[" + folder + "] signs-in=true"},
		{"disable", func() error { return users.Disable(t.Context(), sam) }, nil,
			"sam admin=false disabled=true all=false folders=[" + folder + "] signs-in=false"},
		{"delete", func() error { return users.DeleteUser(t.Context(), sam) }, nil, "gone"},
		{"delete again", func() error { return users.DeleteUser(t.Context(), sam) }, nil, "gone"},
	} {
		if err := step.call(); !answered(err, step.ok) {
			t.Errorf("%s answered %v", step.name, err)
		}
		if got := held(t, admin, sam, pw); got != step.want {
			t.Errorf("after %s the server holds %q, want %q", step.name, got, step.want)
		}
	}
	listed, err := users.ListUsers(t.Context())
	if err != nil || len(listed) != 1 || listed[0] != (link.User{ID: listed[0].ID, Name: "jackson", Administrator: true}) {
		t.Errorf("listed %+v, %v; want only jackson", listed, err)
	}
}
