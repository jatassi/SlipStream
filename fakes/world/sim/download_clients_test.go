package sim_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/deluge"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/qbittorrent"
	"github.com/jatassi/slipstream/fakes/rtorrent"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/transmission"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	delugeclient "github.com/jatassi/slipstream/internal/torrent/clients/deluge"
)

const importDelay = 5 * time.Minute

func TestTransmission(t *testing.T) {
	t.Parallel()
	for _, v := range transmission.Versions {
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			clientSuite(t, transmission.Fake, v)
		})
	}
}

func TestQBittorrent(t *testing.T) {
	t.Parallel()
	for _, v := range qbittorrent.Versions {
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			clientSuite(t, qbittorrent.Fake, v)
		})
	}
}

func TestDeluge(t *testing.T) {
	t.Parallel()
	for _, v := range deluge.Versions {
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			clientSuite(t, deluge.Fake, v, scenarioTest{"heals a label that never arrived", suite.healsALabelThatNeverArrived})
		})
	}
}

func TestRTorrent(t *testing.T) {
	t.Parallel()
	for _, v := range rtorrent.Versions {
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			clientSuite(t, rtorrent.Fake, v)
		})
	}
}

type scenarioTest struct {
	name string
	test func(suite, *testing.T)
}

// clientSuite runs the spec's Testing Decisions for one kind at one
// version, and the kind's own tests, each in a Fake world over net.Pipe
// under synctest, on the external-clients Scenario with the client of
// spec's kind in place of its Transmission.
func clientSuite(t *testing.T, spec fake.Spec, version string, own ...scenarioTest) {
	c := suite{kind: spec.Name, version: version}
	for _, sc := range append([]scenarioTest{
		{"tests, imports, shares a pause and removes with its files", suite.testsImportsSharesAPauseAndRemoves},
		{"imports through a Client folder after the Import delay", suite.importsThroughAClientFolderAfterTheDelay},
	}, own...) {
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) { sc.test(c, t) })
		})
	}
}

type suite struct{ kind, version string }

// foreignFolders is the client's own name for its download folder, unlike
// its Node's, by kind: rTorrent runs on Unix only.
var foreignFolders = map[string]string{"rtorrent": "/srv/torrents"}

func (c suite) foreignFolder() string {
	if f, ok := foreignFolders[c.kind]; ok {
		return f
	}
	return `D:\Torrents`
}

// launch launches the world with the suite's client, after edit. A kind
// that takes no username signs in with the password alone.
func (c suite) launch(t *testing.T, edit func(*scenario.DownloadClient)) (*sim.World, *clientRig) {
	t.Helper()
	s, err := scenario.Load("external-clients")
	if err != nil {
		t.Fatal(err)
	}
	dc := &s.Nodes[0].DownloadClients[0]
	dc.Kind, dc.Version = c.kind, c.version
	if clients.Kind(c.kind).Facts().SignIn.Username == clients.NotTaken {
		dc.Username = ""
	}
	edit(dc)
	w := sim.Launch(t, s, sim.WithWalk())
	return w, &clientRig{t: t, w: w, kind: c.kind, client: *dc}
}

// testsImportsSharesAPauseAndRemoves adds and Tests the client as the
// world did, sees a wrong password and a folder on a library root fail
// the Test, imports a Grab from it, reads a pause made in the client and
// resumes it through SlipStream, then removes it with its files, which
// leaves the imported hard link.
func (c suite) testsImportsSharesAPauseAndRemoves(t *testing.T) {
	w, r := c.launch(t, func(*scenario.DownloadClient) {})
	time.Sleep(time.Minute)
	saved := r.saved()
	if saved.Reach != "responding" || (saved.ProductVersion != c.version && !strings.HasPrefix(saved.ProductVersion, c.version+".")) {
		t.Fatalf("the added client reads %s at %q, want responding at %s", saved.Reach, saved.ProductVersion, c.version)
	}
	passed := decode[struct {
		Product string `json:"product"`
		Folder  string `json:"nodeFolder"`
	}](t, "testDownloadClient", r.test(r.draft(nil)))
	if passed.Product != saved.Name || passed.Folder != w.ClientFolder("home", c.kind) {
		t.Fatalf("the Test passed for %s with %s, want %s with %s", passed.Product, passed.Folder, saved.Name, w.ClientFolder("home", c.kind))
	}
	if r.client.Username == "" {
		r.expectFailed("a wrong password", r.draft(func(d map[string]any) { d["password"] = "not-" + r.client.Password }),
			saved.Name+" rejected the password. Check it and test again.", "password")
	} else {
		r.expectFailed("a wrong password", r.draft(func(d map[string]any) { d["password"] = "not-" + r.client.Password }),
			saved.Name+" rejected the username or password. Check them and test again.", "username", "password")
	}
	root := filepath.Join(w.Manifest().MediaDir, "movies")
	r.expectFailed("a folder inside a library root", r.draft(func(d map[string]any) {
		d["clientFolders"] = []map[string]string{{"client": r.client.Folder, "node": root}}
	}), saved.Name+" saves to "+root+", which overlaps a library root on home. Choose a download folder outside every library root and test again.", "folder")

	search(t, w, addOrchard(t, w))
	expectImportedFrom(t, await(t, w))
	expectPlaced(t, w)
	r.expectLinked()

	w.Cue(c.kind, c.kind+"-pause", map[string]string{"node": "home", "release": release})
	eventually(t, "the Torrent reads paused", func() bool { return r.torrent().State == "paused" })
	time.Sleep(time.Minute)
	row := r.torrent()
	if row.State != "paused" {
		t.Fatalf("a minute after the pause the Torrent reads %s, want paused", row.State)
	}
	r.act(`{"action":"resume","ids":["` + row.ID + `"]}`)
	eventually(t, "the Torrent seeds again", func() bool { return r.torrent().State == "seeding" })
	time.Sleep(time.Minute)
	if got := r.torrent(); got.State != "seeding" || got.Pending != "" {
		t.Fatalf("a minute after the resume the Torrent reads %s with %q pending, want seeding", got.State, got.Pending)
	}

	r.act(`{"action":"remove","ids":["` + row.ID + `"],"deleteFiles":true}`)
	eventually(t, "the Torrent is gone", func() bool { return r.torrent().ID == "" })
	eventually(t, "its files are gone from the client's folder", func() bool { return r.downloaded() == nil })
	if _, err := os.Stat(filepath.Join(w.Manifest().MediaDir, filepath.FromSlash(placed))); err != nil {
		t.Fatalf("after the remove the imported file: %v, want it kept", err)
	}
}

// healsALabelThatNeverArrived has Deluge list the Grab's torrent only
// after SlipStream labeled it, so the label never arrived, and sees the
// torrent labeled once the Node lists it.
func (c suite) healsALabelThatNeverArrived(t *testing.T) {
	w, r := c.launch(t, func(*scenario.DownloadClient) {})
	w.Cue(c.kind, c.kind+"-late-add", map[string]string{"node": "home", "after": "30s"})
	search(t, w, addOrchard(t, w))
	expectImportedFrom(t, await(t, w))
	if got := r.delugeLabel(w.InfoHash(release)); got != clients.DefaultLabel {
		t.Fatalf("the torrent's label in Deluge reads %q, want %q", got, clients.DefaultLabel)
	}
}

// importsThroughAClientFolderAfterTheDelay sets grabsTo through the API
// on a client that names its folder unlike its Node, and sees the Grab's
// Torrent in the client, held through the Import delay, then imported as a
// hard link found through the Client folder.
func (c suite) importsThroughAClientFolderAfterTheDelay(t *testing.T) {
	w, r := c.launch(t, func(dc *scenario.DownloadClient) {
		dc.Folder, dc.GrabsTo, dc.ImportDelay = c.foreignFolder(), false, importDelay
	})
	saved := r.saved()
	r.grabsTo(saved.ID)
	search(t, w, addOrchard(t, w))

	var row torrentRow
	eventually(t, "the Torrent seeds in "+saved.Name, func() bool {
		row = r.torrent()
		return row.State == "seeding"
	})
	if row.ClientID != saved.ID || row.ClientName != saved.Name || row.ClientKind != c.kind {
		t.Fatalf("the Torrent is in %s %s (%s), want %s", row.ClientKind, row.ClientName, row.ClientID, saved.Name)
	}
	finished := time.Now()
	time.Sleep(importDelay - time.Minute)
	if gs := grabsListed(t, w); len(gs) != 1 || gs[0].Stage != "downloading" {
		t.Fatalf("within the Import delay the Grabs are %+v, want one downloading", gs)
	}

	g := await(t, w)
	if waited := time.Since(finished); waited < importDelay-time.Minute {
		t.Fatalf("the Grab imported %s after the client finished, want the Import delay of %s", waited, importDelay)
	}
	expectImportedFrom(t, g)
	expectPlaced(t, w)
	r.expectLinked()
}

// fromClient is a Grab's walk on one Node that downloads, probes and
// imports there.
var fromClient = []string{"fetching", "downloading", "probing", "importing", "imported"}

func expectImportedFrom(t *testing.T, g grabView) {
	t.Helper()
	if g.Stage != "imported" || !slices.Equal(g.stages(), fromClient) {
		t.Fatalf("the Grab ended %s with %+v after %v, want imported after %v", g.Stage, g.Failure, g.stages(), fromClient)
	}
}

// clientRig is the Scenario's one client on home, as an Admin reaches it.
type clientRig struct {
	t      *testing.T
	w      *sim.World
	kind   string
	client scenario.DownloadClient
}

type savedClient struct {
	ID             string `json:"id"`
	NodeID         string `json:"nodeId"`
	Name           string `json:"name"`
	Reach          string `json:"reach"`
	ProductVersion string `json:"productVersion"`
}

type torrentRow struct {
	ID         string `json:"id"`
	ClientID   string `json:"clientId"`
	ClientName string `json:"clientName"`
	ClientKind string `json:"clientKind"`
	State      string `json:"state"`
	Pending    string `json:"pending"`
}

// saved is the client as listDownloadClients shows it on home's card.
func (r *clientRig) saved() savedClient {
	r.t.Helper()
	list := decode[struct {
		Cards []struct {
			NodeName  string        `json:"nodeName"`
			Externals []savedClient `json:"externals"`
		} `json:"cards"`
	}](r.t, "listDownloadClients", r.w.API().Admin("home", http.MethodGet, "/api/v1/download-clients", ""))
	for _, c := range list.Cards {
		if c.NodeName == "home" && len(c.Externals) == 1 {
			return c.Externals[0]
		}
	}
	r.t.Fatalf("listDownloadClients = %+v, want home's card with one external client", list.Cards)
	return savedClient{}
}

func (r *clientRig) grabsTo(id string) {
	r.t.Helper()
	reply := r.w.API().Admin("home", http.MethodPatch, "/api/v1/download-clients/"+r.saved().NodeID+"/settings", `{"grabsTo":"`+id+`"}`)
	if reply.Status != http.StatusOK {
		r.t.Fatalf("setDownloadClientSettings = %d %s, want 200", reply.Status, reply.Body)
	}
}

// draft is the client's settings as an Admin types them in the form,
// after edit.
func (r *clientRig) draft(edit func(map[string]any)) map[string]any {
	d := map[string]any{
		"nodeId": r.saved().NodeID, "kind": r.kind, "address": r.w.ClientURL("home", r.kind),
		"username": r.client.Username, "password": r.client.Password,
		"clientFolders": []map[string]string{{"client": r.client.Folder, "node": r.w.ClientFolder("home", r.kind)}},
	}
	if edit != nil {
		edit(d)
	}
	return d
}

func (r *clientRig) test(draft map[string]any) worldtest.Reply {
	r.t.Helper()
	body, err := json.Marshal(draft)
	if err != nil {
		r.t.Fatal(err)
	}
	return r.w.API().Admin("home", http.MethodPost, "/api/v1/external-download-clients/test", string(body))
}

func (r *clientRig) expectFailed(what string, draft map[string]any, detail string, fields ...string) {
	r.t.Helper()
	reply := r.test(draft)
	var p struct {
		Code   string   `json:"code"`
		Detail string   `json:"detail"`
		Fields []string `json:"fields"`
	}
	if err := json.Unmarshal([]byte(reply.Body), &p); err != nil || reply.Status != http.StatusUnprocessableEntity ||
		p.Code != "download_client_test_failed" || p.Detail != detail || !slices.Equal(p.Fields, fields) {
		r.t.Fatalf("the Test with %s = %d %s, want 422 download_client_test_failed %q on %v", what, reply.Status, reply.Body, detail, fields)
	}
}

// torrent is the world's one Torrent.
func (r *clientRig) torrent() torrentRow {
	r.t.Helper()
	rows := decode[struct {
		Rows []torrentRow `json:"rows"`
	}](r.t, "listTorrents", r.w.API().Admin("home", http.MethodGet, "/api/v1/torrents", "")).Rows
	if len(rows) != 1 {
		return torrentRow{}
	}
	return rows[0]
}

func (r *clientRig) act(body string) {
	r.t.Helper()
	reply := r.w.API().Admin("home", http.MethodPost, "/api/v1/torrents/actions", body)
	if reply.Status != http.StatusOK {
		r.t.Fatalf("actOnTorrents %s = %d %s, want 200", body, reply.Status, reply.Body)
	}
}

// delugeLabel is the label Deluge gives the torrent h, as its own web UI
// shows it.
func (r *clientRig) delugeLabel(h bencode.InfoHash) string {
	r.t.Helper()
	address, err := clients.ParseAddress(r.w.ClientURL("home", r.kind), clients.Deluge)
	if err != nil {
		r.t.Fatal(err)
	}
	c := delugeclient.Open(clients.Session{
		HTTP: r.w.Client(), Address: address, Username: r.client.Username,
		Secrets: clients.Secrets{Password: r.client.Password}, Call: time.Minute, Log: slog.New(slog.DiscardHandler),
	})
	defer func() { _ = c.Close(r.t.Context()) }()
	got, err := c.Poll(r.t.Context(), []bencode.InfoHash{h})
	if err != nil {
		r.t.Fatal(err)
	}
	return got[h].Label
}

// downloaded is the client's download of the release, nil when its folder
// holds none.
func (r *clientRig) downloaded() os.FileInfo {
	r.t.Helper()
	var downloaded os.FileInfo
	err := filepath.WalkDir(r.w.ClientFolder("home", r.kind), func(path string, d os.DirEntry, err error) error {
		if err == nil && d.Name() == release+".mkv" {
			downloaded, err = os.Stat(path)
		}
		return err //nolint:wrapcheck // the test reports it as is
	})
	if err != nil {
		r.t.Fatal(err)
	}
	return downloaded
}

// expectLinked checks the placed file is the client's download, hard
// linked rather than copied.
func (r *clientRig) expectLinked() {
	r.t.Helper()
	downloaded := r.downloaded()
	if downloaded == nil {
		r.t.Fatalf("the client's folder holds no %s.mkv", release)
	}
	linked, err := os.Stat(filepath.Join(r.w.Manifest().MediaDir, filepath.FromSlash(placed)))
	if err != nil {
		r.t.Fatal(err)
	}
	if !os.SameFile(downloaded, linked) {
		r.t.Fatal("the placed file isn't the client's download hard linked: the import copied it")
	}
}

// TestEveryKindOnOneNode launches the external-kinds Scenario, whose Node
// has a client of each kind, sees each respond at its version, and imports
// a Grab through qBittorrent, where its New Grabs go.
func TestEveryKindOnOneNode(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s, err := scenario.Load("external-kinds")
		if err != nil {
			t.Fatal(err)
		}
		w := sim.Launch(t, s, sim.WithWalk())
		time.Sleep(time.Minute)
		list := decode[struct {
			Cards []struct {
				Externals []savedClient `json:"externals"`
			} `json:"cards"`
		}](t, "listDownloadClients", w.API().Admin("home", http.MethodGet, "/api/v1/download-clients", ""))
		var got []string
		for _, c := range list.Cards {
			for _, e := range c.Externals {
				got = append(got, e.Name+" "+e.Reach+" "+e.ProductVersion)
			}
		}
		slices.Sort(got)
		want := []string{"Deluge responding 2.1.1", "Transmission responding 4.1.1", "qBittorrent responding 5.2.3", "rTorrent responding 0.16.27"}
		if !slices.Equal(got, want) {
			t.Fatalf("home's clients read %q, want %q", got, want)
		}

		search(t, w, addOrchard(t, w))
		expectImportedFrom(t, await(t, w))
		rows := decode[struct {
			Rows []torrentRow `json:"rows"`
		}](t, "listTorrents", w.API().Admin("home", http.MethodGet, "/api/v1/torrents", "")).Rows
		if len(rows) != 1 || rows[0].ClientKind != "qbittorrent" {
			t.Fatalf("the Torrents are %+v, want one in qBittorrent", rows)
		}
	})
}
