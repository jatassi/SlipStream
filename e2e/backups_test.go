package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type torrentRow struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	NodeID string `json:"nodeId"`
}

type reconcilingView struct {
	Restore *struct {
		NewMachine bool `json:"newMachine"`
		Rehome     bool `json:"rehome"`
	} `json:"restore"`
	Nodes []reconcilingNode `json:"nodes"`
	Roots []struct {
		ID string `json:"id"`
	} `json:"roots"`
	Review []struct {
		Reason string `json:"reason"`
		Found  string `json:"found"`
		Count  int    `json:"count"`
	} `json:"review"`
}

type reconcilingNode struct {
	Name    string  `json:"name"`
	Adopted bool    `json:"adopted"`
	Granted *string `json:"granted"`
}

type backupRow struct {
	Name    string `json:"name"`
	Trigger string `json:"trigger"`
}

// restoring is a world whose Admin takes Backups on home and restores
// them.
type restoring struct {
	run *world.Run
	m   world.Manifest
	a   *worldtest.API
}

func launchRestoring(t *testing.T, name string) restoring {
	t.Helper()
	run := worldtest.Launch(t, name)
	return restoring{run: run, m: run.Manifest(), a: api(t, run)}
}

// slipstream runs the host command with args and answers its output.
func (w restoring) slipstream(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(worldtest.Context(t), w.m.Binary.Path, args...)
	cmd.Env = []string{"SystemRoot=" + os.Getenv("SystemRoot")}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (w restoring) must(t *testing.T, args ...string) string {
	t.Helper()
	out, err := w.slipstream(t, args...)
	if err != nil {
		t.Fatalf("slipstream %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

func (w restoring) dataDir(name string) string { return w.a.Node(name).DataDir }

// seed adds and searches each movie on home, then waits until home lists
// one torrent seeding per movie.
func (w restoring) seed(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		w.search(t, w.add(t, key))
	}
	w.seeding(t, len(keys))
}

func (w restoring) add(t *testing.T, key string) string {
	t.Helper()
	return w.addOn(t, "home", key)
}

func (w restoring) addOn(t *testing.T, node, key string) string {
	t.Helper()
	var added struct {
		Work struct {
			ID string `json:"id"`
		} `json:"work"`
	}
	decode(t, "addWork", w.a.Admin(node, http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"`+key+`"}`), &added)
	return added.Work.ID
}

func (w restoring) search(t *testing.T, work string) {
	t.Helper()
	if r := w.a.Admin("home", http.MethodPost, "/api/v1/works/"+work+"/search", ""); r.Status != http.StatusAccepted {
		t.Fatalf("searchWork = %d %s, want 202", r.Status, r.Body)
	}
}

func (w restoring) torrents(t *testing.T, node string) []torrentRow {
	t.Helper()
	var out struct {
		Rows []torrentRow `json:"rows"`
	}
	decode(t, "listTorrents", w.a.Admin(node, http.MethodGet, "/api/v1/torrents", ""), &out)
	return out.Rows
}

func (w restoring) seeding(t *testing.T, n int) []torrentRow {
	t.Helper()
	var rows []torrentRow
	worldtest.Until(t, "home lists the torrents seeding", func() (bool, error) {
		rows = w.torrents(t, "home")
		return len(rows) == n && !slices.ContainsFunc(rows, func(r torrentRow) bool { return r.State != "seeding" }), nil
	})
	return rows
}

// backUp takes a Backup on the running home with slipstream backup and
// answers its path and the passphrase an Admin session reads.
func (w restoring) backUp(t *testing.T) (path, passphrase string) {
	t.Helper()
	out := w.must(t, "backup", "--data-dir", w.dataDir("home"))
	path = out[strings.LastIndex(out, ": ")+2:]
	if !strings.HasPrefix(out, "Took a ") || !strings.HasSuffix(path, ".tar.zst.age") {
		t.Fatalf("slipstream backup printed %q, want the size and the path", out)
	}
	var shown struct {
		Passphrase string `json:"passphrase"`
	}
	decode(t, "getBackupPassphrase", w.a.Signed("home", w.a.SignIn("home"), http.MethodGet, "/api/v1/backups/passphrase", ""), &shown)
	return path, shown.Passphrase
}

func (w restoring) backups(t *testing.T, node string) []backupRow {
	t.Helper()
	var out struct {
		Backups []backupRow `json:"backups"`
	}
	decode(t, "listBackups", w.a.Admin(node, http.MethodGet, "/api/v1/backups", ""), &out)
	return out.Backups
}

func (w restoring) reconciling(t *testing.T, node string) reconcilingView {
	t.Helper()
	var v reconcilingView
	decode(t, "getReconciling", w.a.Admin(node, http.MethodGet, "/api/v1/reconciling", ""), &v)
	return v
}

func (w restoring) superseded(t *testing.T, node string) bool {
	t.Helper()
	var s struct {
		Superseded bool `json:"superseded"`
	}
	decode(t, "getStatus", w.a.Get(node, "/api/v1/status"), &s)
	return s.Superseded
}

func (w restoring) nodeID(t *testing.T, on, name string) string {
	t.Helper()
	rows := w.a.Nodes(on)
	i := slices.IndexFunc(rows, func(r worldtest.Row) bool { return r.Name == name })
	if i < 0 {
		t.Fatalf("listNodes on %s has no row %s: %+v", on, name, rows)
	}
	return rows[i].ID
}

// saveAll saves every root of the re-home step at its old path.
func (w restoring) saveAll(t *testing.T, node string) {
	t.Helper()
	var roots []string
	for _, r := range w.reconciling(t, node).Roots {
		roots = append(roots, `{"rootId":"`+r.ID+`"}`)
	}
	var out struct {
		Roots []struct {
			Refused *string `json:"refused"`
		} `json:"roots"`
	}
	r := w.a.Admin(node, http.MethodPost, "/api/v1/reconciling/rehome", `{"roots":[`+strings.Join(roots, ",")+`],"done":true}`)
	decode(t, "saveRehome", r, &out)
	for _, saved := range out.Roots {
		if saved.Refused != nil {
			t.Fatalf("saveRehome = %s, want every root saved", r.Body)
		}
	}
}

// join re-pairs the running Node name live with a Join Code controller
// mints.
func (w restoring) join(t *testing.T, controller, name string) {
	t.Helper()
	code := w.a.MintJoinCode(controller).Code
	if out := w.must(t, "join", "--data-dir", w.dataDir(name), code); !strings.HasPrefix(out[strings.LastIndex(out, "\n")+1:], "joined ") {
		t.Fatalf("slipstream join printed %q, want the joined line", out)
	}
}

func (w restoring) ended(t *testing.T, node string) {
	t.Helper()
	worldtest.Until(t, "Reconciling ends on "+node, func() (bool, error) {
		return w.reconciling(t, node).Restore == nil, nil
	})
}

func (w restoring) history(t *testing.T, node string) []string {
	t.Helper()
	var out struct {
		Entries []struct {
			Kind string `json:"kind"`
		} `json:"entries"`
	}
	decode(t, "listHistory", w.a.Admin(node, http.MethodGet, "/api/v1/history?limit=50", ""), &out)
	var kinds []string
	for _, e := range out.Entries {
		kinds = append(kinds, e.Kind)
	}
	return kinds
}

// uploadRestore restores the Backup at path on the Unclaimed Node name
// through its first-run page's calls: the upload, the passphrase, and the
// confirm that the old Controller stays off.
func (w restoring) uploadRestore(t *testing.T, name, path, passphrase string) {
	t.Helper()
	code := worldtest.SetupCode(t, w.run, name)
	file, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(worldtest.Context(t), http.MethodPost, w.a.Node(name).URL+"/api/v1/setup/restore", bytes.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Setup-Code", code)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var upload struct {
		ID string `json:"id"`
	}
	err = json.NewDecoder(resp.Body).Decode(&upload)
	_ = resp.Body.Close()
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("uploadRestore = %d, %v, want 201 with an id", resp.StatusCode, err)
	}
	open, err := json.Marshal(map[string]string{"passphrase": passphrase})
	if err != nil {
		t.Fatal(err)
	}
	var summary struct {
		Controller string `json:"controller"`
		NewMachine bool   `json:"newMachine"`
		Nodes      []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	}
	decode(t, "openRestore", w.a.Setup(name, code, http.MethodPost, "/api/v1/setup/restore/"+upload.ID+"/open", string(open)), &summary)
	if summary.Controller != "home" || !summary.NewMachine || len(summary.Nodes) != 2 {
		t.Fatalf("openRestore = %+v, want a Backup of home from another machine listing 2 Nodes", summary)
	}
	confirm := "/api/v1/setup/restore/" + upload.ID + "/confirm"
	if r := w.a.Setup(name, code, http.MethodPost, confirm, `{"oldControllerOff":false}`); r.Status != http.StatusUnprocessableEntity {
		t.Fatalf("confirmRestore with the old Controller on = %d %s, want 422", r.Status, r.Body)
	}
	if r := w.a.Setup(name, code, http.MethodPost, confirm, `{"oldControllerOff":true}`); r.Status != http.StatusAccepted {
		t.Fatalf("confirmRestore = %d %s, want 202", r.Status, r.Body)
	}
	w.controller(t, name)
}

func (w restoring) controller(t *testing.T, name string) {
	t.Helper()
	status := w.a.Node(name).URL + "/api/v1/status"
	worldtest.Until(t, name+" starts as the Controller", func() (bool, error) {
		var s struct {
			Claim string `json:"claim"`
		}
		if resp, err := http.Get(status); err == nil { //nolint:gosec,noctx // the Node's address from the manifest; Until bounds the wait
			_ = json.NewDecoder(resp.Body).Decode(&s)
			_ = resp.Body.Close()
		}
		return s.Claim == "controller", nil
	})
}

// TestADeadControllerComesBackOnANewMachine takes a Backup of home with
// both movies seeding on seedbox, kills home, and restores the copy on
// seedbox's Backup destination onto the Unclaimed spare through its
// first-run page's calls. spare starts as the Controller in Reconciling,
// which waits on the re-home step and on seedbox. Save all and a live
// re-pair of seedbox end it with both torrents kept, and History records
// the restore, the new Controller and the end.
func TestADeadControllerComesBackOnANewMachine(t *testing.T) {
	t.Parallel()
	w := launchRestoring(t, "dead-controller")
	w.seed(t, "9300001", "9300002")
	destination := filepath.Join(w.dataDir("seedbox"), "destination")
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	seedbox := w.nodeID(t, "home", "seedbox")
	if r := w.a.Admin("home", http.MethodPost, "/api/v1/backup-destinations", `{"nodeId":"`+seedbox+`","folder":`+quote(destination)+`}`); r.Status != http.StatusCreated {
		t.Fatalf("addBackupDestination = %d %s, want 201", r.Status, r.Body)
	}
	path, passphrase := w.backUp(t)
	copied := filepath.Join(destination, filepath.Base(path))
	worldtest.Until(t, "seedbox's destination holds the Backup", func() (bool, error) {
		_, err := os.Stat(copied)
		return err == nil, nil
	})
	act(t, w.run, "home", world.KillNode)

	w.uploadRestore(t, "spare", copied, passphrase)
	v := w.reconciling(t, "spare")
	if v.Restore == nil || !v.Restore.NewMachine || !v.Restore.Rehome || len(v.Roots) != 1 {
		t.Fatalf("getReconciling after the restore = %+v, want a new machine's restore waiting on the re-home step and one root", v)
	}
	if r := w.a.Admin("spare", http.MethodPost, "/api/v1/works/"+w.addOn(t, "spare", "9300001")+"/search", ""); r.Status != http.StatusConflict || !strings.Contains(r.Body, `"code":"reconciling"`) {
		t.Fatalf("searchWork while Reconciling = %d %s, want 409 reconciling", r.Status, r.Body)
	}
	w.saveAll(t, "spare")
	w.join(t, "spare", "seedbox")
	w.ended(t, "spare")
	if rows := w.seedingOrSpread(t, "spare", 2); rows[0].NodeID != seedbox || rows[1].NodeID != seedbox {
		t.Fatalf("listTorrents on spare = %+v, want both on seedbox", rows)
	}
	got := w.history(t, "spare")
	for _, kind := range []string{"backup_restored", "controller_replaced", "reconciling_finished"} {
		if !slices.Contains(got, kind) {
			t.Fatalf("listHistory on spare = %v, want %s", got, kind)
		}
	}
}

// seedingOrSpread is node's n torrents once none is removed or failed.
// After a re-pair a private torrent waits out the start spread as paused
// with the window hold, then seeds.
func (w restoring) seedingOrSpread(t *testing.T, node string, n int) []torrentRow {
	t.Helper()
	var rows []torrentRow
	worldtest.Until(t, node+" lists every torrent kept", func() (bool, error) {
		rows = w.torrents(t, node)
		return len(rows) == n && !slices.ContainsFunc(rows, func(r torrentRow) bool { return r.State != "seeding" && r.State != "paused" }), nil
	})
	return rows
}

func quote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// TestTheOldControllerIsFencedThenJoinsAsAWorker gives home Searching and
// Downloading for The Quiet Harbor, so its own client seeds it, and
// restores home's Backup offline on spare while home keeps running. Once
// seedbox re-pairs with spare it carries the proof, and home, dialing
// seedbox, is Superseded: every Admin read refuses. Wipe and join makes
// home a new Worker of spare, home-2, whose Holdings take Harbor off the
// review list and give it the Downloading Role.
func TestTheOldControllerIsFencedThenJoinsAsAWorker(t *testing.T) {
	t.Parallel()
	w := launchRestoring(t, "dead-controller")
	w.search(t, w.add(t, "9300001"))
	w.seeding(t, 1)
	home, seedbox := w.nodeID(t, "home", "home"), w.nodeID(t, "home", "seedbox")
	w.a.SetRoles("home", home, "storage", "converting", "subtitles", "metadata", "media-server-link", "notifying", "serving", "searching", "downloading")
	w.a.SetRoles("home", seedbox, "downloading")
	w.search(t, w.add(t, "9300002"))
	rows := w.seeding(t, 2)
	if !slices.ContainsFunc(rows, func(r torrentRow) bool { return r.NodeID == home }) {
		t.Fatalf("listTorrents on home = %+v, want one on home", rows)
	}
	path, passphrase := w.backUp(t)
	pass := filepath.Join(t.TempDir(), "pass")
	if err := os.WriteFile(pass, []byte(passphrase), 0o600); err != nil {
		t.Fatal(err)
	}

	act(t, w.run, "spare", world.StopNode)
	if out := w.must(t, "restore", "--data-dir", w.dataDir("spare"), "--passphrase-file", pass, "--old-controller-off", path); !strings.HasPrefix(out, "Prepared the restore of the Backup from ") {
		t.Fatalf("slipstream restore offline printed %q, want the next step", out)
	}
	act(t, w.run, "spare", world.StartNode)
	w.controller(t, "spare")
	// spare lists Harbor for review, then looks for its files: the row is
	// written before the look asks Harbor's Node.
	var v reconcilingView
	worldtest.Until(t, "spare lists Harbor for review and looks for its files", func() (bool, error) {
		v = w.reconciling(t, "spare")
		return len(v.Review) > 0 && v.Review[0].Found != "not_looked", nil
	})
	if len(v.Review) != 1 || v.Review[0].Reason != "not_held" || v.Review[0].Found != "files" {
		t.Fatalf("getReconciling's review list = %+v, want Harbor not held with its files found", v.Review)
	}
	w.join(t, "spare", "seedbox")
	// seedbox tells home itself once it reaches it, so home may already
	// be Superseded and refuse the dial.
	address := w.must(t, "node-address", "--data-dir", w.dataDir("seedbox"))
	if r := w.a.Admin("home", http.MethodPut, "/api/v1/nodes/"+seedbox+"/dial", `{"nodeAddress":`+quote(address[strings.LastIndex(address, "\n")+1:])+`}`); r.Status != http.StatusOK && !strings.Contains(r.Body, `"code":"superseded"`) {
		t.Fatalf("setDial on home = %d %s, want 200, or 409 superseded", r.Status, r.Body)
	}
	worldtest.Until(t, "home is Superseded", func() (bool, error) {
		return w.superseded(t, "home"), nil
	})
	worldtest.Expect(t, "listNodes on the Superseded home", w.a.Admin("home", http.MethodGet, "/api/v1/nodes", ""), worldtest.Reply{Status: http.StatusConflict,
		Body: `{"status":409,"code":"superseded","title":"This Controller was replaced"}` + "\n"})
	var fence struct {
		State  string `json:"state"`
		Worker string `json:"worker"`
	}
	decode(t, "getSuperseded", w.a.Admin("home", http.MethodGet, "/api/v1/superseded", ""), &fence)
	if fence.State != "proven" || fence.Worker != "seedbox" {
		t.Fatalf("getSuperseded = %+v, want proven by seedbox", fence)
	}

	code := w.a.MintJoinCode("spare").Code
	if r := w.a.Signed("home", w.a.SignIn("home"), http.MethodPost, "/api/v1/superseded/wipe", `{"joinCode":"`+code+`"}`); r.Status != http.StatusAccepted {
		t.Fatalf("wipeSuperseded = %d %s, want 202", r.Status, r.Body)
	}
	if granted := w.granted(t, "spare", "home-2"); granted != "home-2 now has the Downloading Role for the 1 torrent it reported." {
		t.Fatalf("the Reconciling page reads %q for home-2", granted)
	}
	if v := w.reconciling(t, "spare"); len(v.Review) != 0 {
		t.Fatalf("the review list after home-2 reported = %+v, want it empty", v.Review)
	}
	homeTwo := w.nodeID(t, "spare", "home-2")
	if rows := w.seedingOrSpread(t, "spare", 2); !slices.ContainsFunc(rows, func(r torrentRow) bool { return r.NodeID == homeTwo }) {
		t.Fatalf("listTorrents on spare = %+v, want Harbor on home-2", rows)
	}
	for _, f := range []string{"library.db", "library.key", "backup.key", "backups.json"} {
		if _, err := os.Stat(filepath.Join(w.dataDir("home"), f)); !os.IsNotExist(err) {
			t.Fatalf("home's data dir still holds %s after Wipe and join: %v", f, err)
		}
	}
}

// granted waits until node's Holdings are adopted on controller with the
// Downloading Role, and answers the Reconciling page's sentence for it.
func (w restoring) granted(t *testing.T, controller, node string) string {
	t.Helper()
	var granted *string
	worldtest.Until(t, node+"'s Holdings are adopted with the Downloading Role", func() (bool, error) {
		nodes := w.reconciling(t, controller).Nodes
		if i := slices.IndexFunc(nodes, func(n reconcilingNode) bool { return n.Name == node && n.Adopted }); i >= 0 {
			granted = nodes[i].Granted
		}
		return granted != nil, nil
	})
	return *granted
}

// TestAHostileWorkersReplyRaisesOnlyHealth claims office as a second
// Controller and joins seedbox to it. home, dialing seedbox, hears that
// office replaced it with no proof: it keeps running and raises one Nodes
// Health Error.
func TestAHostileWorkersReplyRaisesOnlyHealth(t *testing.T) {
	t.Parallel()
	w := launchRestoring(t, "hostile-worker")
	seedbox := w.nodeID(t, "home", "seedbox")
	code := worldtest.SetupCode(t, w.run, "office")
	if r := w.a.Setup("office", code, http.MethodPost, "/api/v1/setup/controller", `{"email":"olu@example.test","displayName":"Olu Bello","pin":"1357"}`); r.Status != http.StatusOK && r.Status != http.StatusCreated {
		t.Fatalf("claimController on office = %d %s", r.Status, r.Body)
	}
	session := w.a.SignInAs("office", "olu@example.test", "1357")
	var minted struct {
		Code string `json:"code"`
	}
	mint := w.a.Signed("office", session, http.MethodPost, "/api/v1/join-codes", "")
	if err := json.Unmarshal([]byte(mint.Body), &minted); err != nil || mint.Status != http.StatusCreated {
		t.Fatalf("mintJoinCode on office = %d %s, want 201", mint.Status, mint.Body)
	}
	if out := w.must(t, "join", "--data-dir", w.dataDir("seedbox"), minted.Code); !strings.Contains(out, "joined office as seedbox") {
		t.Fatalf("slipstream join printed %q, want seedbox joined to office", out)
	}
	// seedbox tells home itself once it reaches it, so home may already
	// be Superseded and refuse the dial.
	address := w.must(t, "node-address", "--data-dir", w.dataDir("seedbox"))
	if r := w.a.Admin("home", http.MethodPut, "/api/v1/nodes/"+seedbox+"/dial", `{"nodeAddress":`+quote(address[strings.LastIndex(address, "\n")+1:])+`}`); r.Status != http.StatusOK && !strings.Contains(r.Body, `"code":"superseded"`) {
		t.Fatalf("setDial on home = %d %s, want 200, or 409 superseded", r.Status, r.Body)
	}
	var fence struct {
		State  string `json:"state"`
		Worker string `json:"worker"`
	}
	worldtest.Until(t, "home hears seedbox's reply", func() (bool, error) {
		decode(t, "getSuperseded", w.a.Admin("home", http.MethodGet, "/api/v1/superseded", ""), &fence)
		return fence.State != "none", nil
	})
	if fence.State != "unproven" || fence.Worker != "seedbox" {
		t.Fatalf("getSuperseded = %+v, want unproven from seedbox", fence)
	}
	if w.superseded(t, "home") {
		t.Fatal("getStatus on home reads superseded after a reply with no proof")
	}
	worldtest.Until(t, "home's Health lists the Nodes Error for seedbox", func() (bool, error) {
		health := w.a.Admin("home", http.MethodGet, "/api/v1/system/health", "")
		return health.Status == http.StatusOK && strings.Contains(health.Body, `"subject":"seedbox is paired with another Controller"`), nil
	})
}

// TestARestoreInPlaceAfterAGrab takes a Backup on home while a Grab waits
// at fetching, lets the Grab finish on seedbox, and changes the Admin's
// PIN. slipstream restore on the running home restarts it Reconciling:
// the new PIN signs in, every Backup taken since is listed, and seedbox's
// torrent, which the Backup doesn't know, is adopted and taken by the
// Grab. Then slipstream restore offline on the stopped home does the same
// at the next start.
func TestARestoreInPlaceAfterAGrab(t *testing.T) {
	t.Parallel()
	w := launchRestoring(t, "restore-in-place-after-grab")
	setFault(t, w.run, fault.Rule{ID: "hold-fetch", Target: "fixture-site", Route: "torznab.download", Times: 1}, "delay=8s")
	work := w.add(t, "9300001")
	w.search(t, work)
	worldtest.Until(t, "the Grab waits at fetching", func() (bool, error) {
		return strings.Contains(w.a.Admin("home", http.MethodGet, "/api/v1/grabs", "").Body, `"stage":"fetching"`), nil
	})
	path, passphrase := w.backUp(t)
	w.seeding(t, 1)
	session := w.a.SignIn("home")
	if r := w.a.Signed("home", session, http.MethodPost, "/api/v1/account/pin", `{"current":"2468","new":"1357"}`); r.Status != http.StatusNoContent {
		t.Fatalf("changePin = %d %s", r.Status, r.Body)
	}
	pass := filepath.Join(t.TempDir(), "pass")
	if err := os.WriteFile(pass, []byte(passphrase), 0o600); err != nil {
		t.Fatal(err)
	}
	before := w.backups(t, "home")

	restored := func(out string) {
		t.Helper()
		w.controller(t, "home")
		w.ended(t, "home")
		w.a.SignInAs("home", "ada@example.test", "1357")
		after := w.backups(t, "home")
		for _, b := range before {
			if !slices.ContainsFunc(after, func(a backupRow) bool { return a.Name == b.Name }) {
				t.Fatalf("listBackups after the restore = %+v, want %s still listed", after, b.Name)
			}
		}
		if !slices.ContainsFunc(after, func(a backupRow) bool { return a.Trigger == "before-restore" }) {
			t.Fatalf("listBackups after the restore = %+v, want the before-restore Backup", after)
		}
		before = after
		w.seedingOrSpread(t, "home", 1)
		var grab struct {
			Grabs []grabView `json:"grabs"`
		}
		worldtest.Until(t, "the Grab at fetching is decided", func() (bool, error) {
			decode(t, "listGrabs", w.a.Admin("home", http.MethodGet, "/api/v1/grabs", ""), &grab)
			return len(grab.Grabs) == 1 && grab.Grabs[0].Stage != "fetching", nil
		})
		if g := grab.Grabs[0]; g.Failure != nil || g.InfoHash == "" {
			t.Fatalf("after %q the Grab reads %+v, want it past fetching with the adopted torrent", out, g)
		}
	}
	out := w.must(t, "restore", "--data-dir", w.dataDir("home"), "--passphrase-file", pass, path)
	if !strings.HasSuffix(out, "SlipStream is restarting to finish it.") {
		t.Fatalf("slipstream restore online printed %q", out)
	}
	restored(out)

	act(t, w.run, "home", world.StopNode)
	out = w.must(t, "restore", "--data-dir", w.dataDir("home"), "--passphrase-file", pass, path)
	if !strings.HasSuffix(out, "Start SlipStream to finish it.") {
		t.Fatalf("slipstream restore offline printed %q", out)
	}
	act(t, w.run, "home", world.StartNode)
	restored(out)
}
