package sim_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/transmission"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

const importDelay = 5 * time.Minute

func TestTransmission(t *testing.T) {
	for _, v := range transmission.Versions {
		t.Run(v, func(t *testing.T) { clientSuite(t, transmission.Fake, v) })
	}
}

// clientSuite runs the spec's Testing Decisions for one kind at one
// version, each in a Fake world over net.Pipe under synctest, on the
// external-clients Scenario with the client of spec's kind in place of its
// Transmission.
func clientSuite(t *testing.T, spec fake.Spec, version string) {
	c := suite{kind: spec.Name, version: version}
	for _, sc := range []struct {
		name string
		test func(*testing.T)
	}{
		{"tests, imports and shares a pause", c.testsImportsAndSharesAPause},
		{"imports through a Client folder after the Import delay", c.importsThroughAClientFolderAfterTheDelay},
	} {
		t.Run(sc.name, func(t *testing.T) { synctest.Test(t, sc.test) })
	}
}

type suite struct{ kind, version string }

// launch launches the world with the suite's client, after edit.
func (c suite) launch(t *testing.T, edit func(*scenario.DownloadClient)) (*sim.World, *clientRig) {
	t.Helper()
	s, err := scenario.Load("external-clients")
	if err != nil {
		t.Fatal(err)
	}
	dc := &s.Nodes[0].DownloadClients[0]
	dc.Kind, dc.Version = c.kind, c.version
	edit(dc)
	w := sim.Launch(t, s, sim.WithWalk())
	return w, &clientRig{t: t, w: w, kind: c.kind, client: *dc}
}

// testsImportsAndSharesAPause adds and Tests the client as the world did,
// sees a wrong password and a folder on a library root fail the Test,
// imports a Grab from it, then reads a pause made in the client and
// resumes it through SlipStream.
func (c suite) testsImportsAndSharesAPause(t *testing.T) {
	w, r := c.launch(t, func(*scenario.DownloadClient) {})
	time.Sleep(time.Minute)
	saved := r.saved()
	if saved.Reach != "responding" || !strings.HasPrefix(saved.ProductVersion, c.version+".") {
		t.Fatalf("the added client reads %s at %q, want responding at %s", saved.Reach, saved.ProductVersion, c.version)
	}
	passed := decode[struct {
		Product string `json:"product"`
		Folder  string `json:"nodeFolder"`
	}](t, "testDownloadClient", r.test(r.draft(nil)))
	if passed.Product != saved.Name || passed.Folder != w.ClientFolder("home", c.kind) {
		t.Fatalf("the Test passed for %s with %s, want %s with %s", passed.Product, passed.Folder, saved.Name, w.ClientFolder("home", c.kind))
	}
	r.expectFailed("a wrong password", r.draft(func(d map[string]any) { d["password"] = "not-" + r.client.Password }),
		saved.Name+" rejected the username or password. Check them and test again.", "username", "password")
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
	r.act("resume", row.ID)
	eventually(t, "the Torrent seeds again", func() bool { return r.torrent().State == "seeding" })
	time.Sleep(time.Minute)
	if got := r.torrent(); got.State != "seeding" || got.Pending != "" {
		t.Fatalf("a minute after the resume the Torrent reads %s with %q pending, want seeding", got.State, got.Pending)
	}
}

// importsThroughAClientFolderAfterTheDelay sets grabsTo through the API
// on a client that calls its folder D:\Torrents, and sees the Grab's
// Torrent in the client, held through the Import delay, then imported as a
// hard link found through the Client folder.
func (c suite) importsThroughAClientFolderAfterTheDelay(t *testing.T) {
	w, r := c.launch(t, func(dc *scenario.DownloadClient) {
		dc.Folder, dc.GrabsTo, dc.ImportDelay = `D:\Torrents`, false, importDelay
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

func (r *clientRig) act(action, id string) {
	r.t.Helper()
	reply := r.w.API().Admin("home", http.MethodPost, "/api/v1/torrents/actions", `{"action":"`+action+`","ids":["`+id+`"]}`)
	if reply.Status != http.StatusOK {
		r.t.Fatalf("actOnTorrents %s = %d %s, want 200", action, reply.Status, reply.Body)
	}
}

// expectLinked checks the placed file is the client's download, hard
// linked rather than copied.
func (r *clientRig) expectLinked() {
	r.t.Helper()
	var downloaded os.FileInfo
	err := filepath.WalkDir(r.w.ClientFolder("home", r.kind), func(path string, d os.DirEntry, err error) error {
		if err == nil && d.Name() == release+".mkv" {
			downloaded, err = os.Stat(path)
		}
		return err //nolint:wrapcheck // the test reports it as is
	})
	if err != nil || downloaded == nil {
		r.t.Fatalf("the client's folder holds no %s.mkv: %v", release, err)
	}
	linked, err := os.Stat(filepath.Join(r.w.Manifest().MediaDir, filepath.FromSlash(placed)))
	if err != nil {
		r.t.Fatal(err)
	}
	if !os.SameFile(downloaded, linked) {
		r.t.Fatal("the placed file isn't the client's download hard linked: the import copied it")
	}
}
