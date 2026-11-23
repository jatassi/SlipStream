package node

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/lifecycle"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/system"
)

// taken is a Backup a running Controller took, with the passphrase that
// opens it.
type taken struct {
	file, words string
	at          time.Time
}

func takeBackup(t *testing.T, n *Node) taken {
	t.Helper()
	h := n.half.Load()
	run, err := h.backups.Take(t.Context(), backups.Request{Trigger: archive.Manual})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := run.Wait(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return taken{file: h.backups.Path(entry), words: currentWords(t, n), at: entry.TakenAt}
}

func currentWords(t *testing.T, n *Node) string {
	t.Helper()
	v, err := n.backups.Vault(n.lib.Load().keys)
	if err != nil {
		t.Fatal(err)
	}
	_, p, err := v.Current(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return p.Reveal()
}

func fenceOf(t *testing.T, n *Node) backups.Fence {
	t.Helper()
	v, err := n.backups.Vault(n.lib.Load().keys)
	if err != nil {
		t.Fatal(err)
	}
	return v.Fence()
}

func phaseOf(t *testing.T, n *Node) reconciling.Phase {
	t.Helper()
	var phase reconciling.Phase
	err := n.lib.Load().db.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		phase, err = reconciling.State(ctx, r)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return phase
}

func stop(n *Node) {
	n.Stop()
	<-n.Done()
}

func triggers(l backups.Ledger) []archive.Trigger {
	var out []archive.Trigger
	for _, e := range l.Backups {
		out = append(out, e.Trigger)
	}
	return out
}

func count[T comparable](s []T, v T) int {
	n := 0
	for _, x := range s {
		if x == v {
			n++
		}
	}
	return n
}

func TestARestoreInPlaceKeepsTheBeforeRestoreBackupNewerBackupsAndCredentials(t *testing.T) {
	env, _ := testEnv(t)
	synctest.Test(t, func(t *testing.T) {
		n, c := startInMemory(t, env)
		claimed := c.claim(n.Ready().SetupCode, "1234")
		revoked := keyOf(t, c, claimed)
		first := takeBackup(t, n)
		kept := keyOf(t, c, claimed)
		takeBackup(t, n)
		fence := fenceOf(t, n)
		stop(n)

		line, err := RestoreOffline(t.Context(), env, restoreFlags{file: first.file}, first.words)
		if err != nil {
			t.Fatal(err)
		}
		if want := "Prepared the restore of the Backup from " + first.at.Local().Format("2 Jan, 15:04") + ". Start SlipStream to finish it."; line != want {
			t.Fatalf("restore printed %q, want %q", line, want)
		}
		again, ac := startInMemory(t, env)
		if fenceOf(t, again) != fence {
			t.Fatal("the restore in place replaced the fence secret")
		}
		if r := ac.admin(kept, http.MethodGet, "/api/v1/nodes", ""); r.status != http.StatusOK {
			t.Fatalf("the API key made after the Backup = %d %s, want it kept", r.status, r.body)
		}
		if r := ac.admin(revoked, http.MethodGet, "/api/v1/nodes", ""); r.status != http.StatusUnauthorized {
			t.Fatalf("the API key the Backup holds = %d, want it gone with its replacement", r.status)
		}
		expect(t, ac.getSession(sessionOf(t, claimed)), 401, signedOut)
		got := triggers(ledgerOf(t, env))
		if count(got, archive.Manual) != 2 || count(got, archive.BeforeRestore) != 1 {
			t.Fatalf("the Backups index lists %v, want both manual Backups and one before-restore", got)
		}
	})
}

// homeAndBox is a Controller home with box paired as its Worker, both
// stopped, and a Backup home took while box was paired.
func homeAndBox(t *testing.T) (net *fault.Network, homeEnv, boxEnv Env, apiKey string, b taken) {
	t.Helper()
	net = fault.NewNetwork()
	homeEnv, homeHost := machine(t, net, "127.0.0.2", "home")
	boxEnv, boxHost := machine(t, net, "127.0.0.3", "box")
	home, hc := startOn(t, homeEnv, homeHost)
	apiKey, code := claimedHome(t, home, hc)
	box, bc := startOn(t, boxEnv, boxHost)
	joinBox(t, box, bc, boxEnv.DataDir, code, apiKey)
	synctest.Wait()
	b = takeBackup(t, home)
	stop(home)
	stop(box)
	return net, homeEnv, boxEnv, apiKey, b
}

// wipe leaves only node.key in env's data dir, as a reinstall that kept
// the machine's key would.
func wipe(t *testing.T, env Env) {
	t.Helper()
	entries, err := os.ReadDir(env.DataDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "node.key" {
			if err := os.RemoveAll(filepath.Join(env.DataDir.Path(), e.Name())); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func hostOf(t *testing.T, env Env) *fault.Host {
	t.Helper()
	h, ok := env.Net.(*fault.Host)
	if !ok {
		t.Fatalf("%T isn't an in-memory host", env.Net)
	}
	return h
}

func selfRows(rows []nodeRow) int {
	n := 0
	for _, r := range rows {
		if r.Self {
			n++
		}
	}
	return n
}

func nodesOn(t *testing.T, c client, apiKey string) []nodeRow {
	t.Helper()
	return decode[struct {
		Nodes []nodeRow `json:"nodes"`
	}](t, c.admin(apiKey, http.MethodGet, "/api/v1/nodes", ""), http.StatusOK).Nodes
}

func TestANewMachineKeepsOneControllerRow(t *testing.T) {
	t.Run("a fresh machine", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			net, _, _, apiKey, b := homeAndBox(t)
			spareEnv, spareHost := machine(t, net, "127.0.0.4", "spare")
			if _, err := RestoreOffline(t.Context(), spareEnv, restoreFlags{file: b.file, oldOff: true}, b.words); err != nil {
				t.Fatal(err)
			}
			spare, sc := startOn(t, spareEnv, spareHost)
			rows := nodesOn(t, sc, apiKey)
			if len(rows) != 2 || selfRows(rows) != 1 || spare.Ready().Claim != system.Controller {
				t.Fatalf("listNodes on the new machine = %+v as %s, want the Controller as self and box", rows, spare.Ready().Claim)
			}
		})
	})
	t.Run("the Worker the Backup lists", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			_, _, boxEnv, apiKey, b := homeAndBox(t)
			wipe(t, boxEnv)
			if _, err := RestoreOffline(t.Context(), boxEnv, restoreFlags{file: b.file, oldOff: true}, b.words); err != nil {
				t.Fatal(err)
			}
			_, bc := startOn(t, boxEnv, hostOf(t, boxEnv))
			rows := nodesOn(t, bc, apiKey)
			if len(rows) != 1 || !rows[0].Self || rows[0].Name != "home" {
				t.Fatalf("listNodes on box = %+v, want only the Controller home, as self", rows)
			}
		})
	})
}

func TestTheRestoresWriteRunsOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, _, boxEnv, apiKey, b := homeAndBox(t)
		wipe(t, boxEnv)
		if _, err := RestoreOffline(t.Context(), boxEnv, restoreFlags{file: b.file, oldOff: true}, b.words); err != nil {
			t.Fatal(err)
		}
		boxEnv.Test.BeforeRestoreStage = func(s RestoreStep) error {
			if s.Stage == RestoreWritten {
				return errCrash
			}
			return nil
		}
		if _, err := Start(t.Context(), boxEnv); !errors.Is(err, errCrash) {
			t.Fatalf("the start stopped after the Write = %v", err)
		}
		boxEnv.Test.BeforeRestoreStage = nil
		_, bc := startOn(t, boxEnv, hostOf(t, boxEnv))
		if rows := nodesOn(t, bc, apiKey); len(rows) != 1 || !rows[0].Self {
			t.Fatalf("listNodes after the second start = %+v, want only the Controller, as self", rows)
		}
	})
}

func TestTheCLIRefusesANewMachineWithoutOldControllerOff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		net, _, _, _, b := homeAndBox(t)
		spareEnv, _ := machine(t, net, "127.0.0.4", "spare")
		_, err := RestoreOffline(t.Context(), spareEnv, restoreFlags{file: b.file}, b.words)
		if want := "this Backup is of home, another machine. Make sure home stays off, then run slipstream restore again with --old-controller-off to confirm it"; err == nil || err.Error() != want {
			t.Fatalf("restore = %v, want %q", err, want)
		}
		if holds := dataDirHolds(t, spareEnv.DataDir); slices.ContainsFunc(holds, func(name string) bool {
			return strings.HasSuffix(name, lifecycle.Restored) || name == lifecycle.FileName
		}) {
			t.Fatalf("the refused restore left %v", holds)
		}
	})
}

func TestRestoringASupersededBackupStartsUnfenced(t *testing.T) {
	env, _ := testEnv(t)
	synctest.Test(t, func(t *testing.T) {
		n, c := startInMemory(t, env)
		c.claim(n.Ready().SetupCode, "1234")
		err := n.lib.Load().db.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `INSERT INTO nodes_fence (id, state, by_key, by_node, addresses, fenced_at) VALUES (1, 'retired', NULL, NULL, '[]', 0)`)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		b := takeBackup(t, n)
		before := fenceOf(t, n)
		stop(n)
		if _, err := RestoreOffline(t.Context(), env, restoreFlags{file: b.file}, b.words); err != nil {
			t.Fatal(err)
		}
		again, _ := startInMemory(t, env)
		var fenced bool
		err = again.lib.Load().db.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			return r.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM nodes_fence)`).Scan(&fenced)
		})
		if err != nil || fenced {
			t.Fatalf("the restored library is fenced: %v, %v", fenced, err)
		}
		if fenceOf(t, again) == before {
			t.Fatal("the restored superseded library kept the fence secret its Workers' proofs name")
		}
	})
}

func TestANewMachineRegeneratesTheFenceSecret(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		net, homeEnv, _, _, b := homeAndBox(t)
		home, _ := startOn(t, homeEnv, hostOf(t, homeEnv))
		old := fenceOf(t, home)
		stop(home)
		spareEnv, spareHost := machine(t, net, "127.0.0.4", "spare")
		if _, err := RestoreOffline(t.Context(), spareEnv, restoreFlags{file: b.file, oldOff: true}, b.words); err != nil {
			t.Fatal(err)
		}
		spare, _ := startOn(t, spareEnv, spareHost)
		if fenceOf(t, spare) == old {
			t.Fatal("the new machine kept the old Controller's fence secret")
		}
	})
}

func TestDockerVariablesRestoreInTheSameStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		net, _, _, apiKey, b := homeAndBox(t)
		spareEnv, spareHost := machine(t, net, "127.0.0.4", "spare")
		words := filepath.Join(shortTempDir(t), "passphrase")
		if err := os.WriteFile(words, []byte(b.words+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		spareEnv.Restore = &EnvRestore{File: b.file, PassphraseFile: words, OldControllerOff: true}
		spare, sc := startOn(t, spareEnv, spareHost)
		if spare.Ready().Claim != system.Controller || phaseOf(t, spare) == reconciling.None {
			t.Fatalf("the start with the restore variables is %s, want a Controller Reconciling", spare.Ready().Claim)
		}
		if rows := nodesOn(t, sc, apiKey); selfRows(rows) != 1 {
			t.Fatalf("listNodes = %+v, want the Controller as self", rows)
		}
	})
}

func TestDockerVariablesIgnoreAClaimedDataDir(t *testing.T) {
	env, stderr := testEnv(t)
	synctest.Test(t, func(t *testing.T) {
		n, c := startInMemory(t, env)
		c.claim(n.Ready().SetupCode, "1234")
		b := takeBackup(t, n)
		stop(n)
		env.Restore = &EnvRestore{File: b.file, PassphraseFile: filepath.Join(env.DataDir.Path(), "absent")}
		again, _ := startInMemory(t, env)
		if phaseOf(t, again) != reconciling.None {
			t.Fatal("the restore variables restored over a claimed data dir")
		}
		if !strings.Contains(stderr.String(), "SlipStream ignores SLIPSTREAM_RESTORE because this data dir is already set up") {
			t.Fatalf("the log holds no line on the ignored variables:\n%s", stderr)
		}
	})
}

func TestEnvRestoreNeedsBothVariables(t *testing.T) {
	if _, _, err := envRestore([]string{"SLIPSTREAM_RESTORE=/b.tar.zst.age"}); err == nil {
		t.Fatal("SLIPSTREAM_RESTORE alone was taken")
	}
	rest, r, err := envRestore([]string{"A=1", "SLIPSTREAM_RESTORE=/b", "SLIPSTREAM_RESTORE_PASSPHRASE_FILE=/p", "SLIPSTREAM_RESTORE_OLD_CONTROLLER_OFF=1"})
	if err != nil || !slices.Equal(rest, []string{"A=1"}) || *r != (EnvRestore{File: "/b", PassphraseFile: "/p", OldControllerOff: true}) {
		t.Fatalf("envRestore = %v, %+v, %v", rest, r, err)
	}
}

// slowly is a Backup sent in two halves, pause apart.
type slowly struct {
	b     []byte
	pause time.Duration
	sent  int
}

func (s *slowly) Read(p []byte) (int, error) {
	if s.sent == len(s.b) {
		return 0, io.EOF
	}
	if s.sent > 0 && s.sent == len(s.b)/2 {
		time.Sleep(s.pause)
	}
	end := min(len(s.b), s.sent+len(p))
	if s.sent < len(s.b)/2 {
		end = min(end, len(s.b)/2)
	}
	n := copy(p, s.b[s.sent:end])
	s.sent += n
	return n, nil
}

func upload(c client, setupCode string, body io.Reader, contentType string) reply {
	c.t.Helper()
	return c.do(call{method: http.MethodPost, path: "/api/v1/setup/restore", body: body, headers: map[string]string{"X-Setup-Code": setupCode, "Content-Type": contentType}})
}

func TestAnUploadIsNotCutAtFiveSeconds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		net, _, _, apiKey, b := homeAndBox(t)
		spareEnv, spareHost := machine(t, net, "127.0.0.4", "spare")
		spareEnv.Restart = func() error { return nil }
		spare, sc := startOn(t, spareEnv, spareHost)
		code := spare.Ready().SetupCode
		file, err := os.ReadFile(b.file)
		if err != nil {
			t.Fatal(err)
		}
		up := upload(sc, code, &slowly{b: file, pause: 8 * time.Second}, "application/octet-stream")
		id := decode[struct {
			ID string `json:"id"`
		}](t, up, http.StatusCreated).ID

		open := func(words string) reply {
			return sc.do(call{method: http.MethodPost, path: "/api/v1/setup/restore/" + id + "/open", body: strings.NewReader(`{"passphrase":"` + words + `"}`), headers: map[string]string{"X-Setup-Code": code}})
		}
		summary := decode[struct {
			Controller string `json:"controller"`
			NewMachine bool   `json:"newMachine"`
			Nodes      []struct {
				Name string `json:"name"`
			} `json:"nodes"`
		}](t, open(b.words), http.StatusOK)
		if summary.Controller != "home" || !summary.NewMachine || len(summary.Nodes) != 2 || summary.Nodes[1].Name != "box" {
			t.Fatalf("the summary is %+v, want home on another machine, then box", summary)
		}
		confirm := func(body string) reply {
			return sc.do(call{method: http.MethodPost, path: "/api/v1/setup/restore/" + id + "/confirm", body: strings.NewReader(body), headers: map[string]string{"X-Setup-Code": code}})
		}
		expect(t, confirm(`{"oldControllerOff":false}`), 422, `{"status":422,"code":"old_controller_on","title":"Confirm that the old Controller stays off","detail":"This Backup is of another machine. Confirm that its Controller stays off."}`+"\n")
		if r := confirm(`{"oldControllerOff":true}`); r.status != http.StatusAccepted {
			t.Fatalf("confirmRestore = %d %s, want 202", r.status, r.body)
		}
		<-spare.Done()

		again, ac := startOn(t, spareEnv, spareHost)
		if again.Ready().Claim != system.Controller || phaseOf(t, again) == reconciling.None {
			t.Fatalf("the restarted Node is %s, want a Controller Reconciling", again.Ready().Claim)
		}
		if rows := nodesOn(t, ac, apiKey); len(rows) != 2 || selfRows(rows) != 1 {
			t.Fatalf("listNodes = %+v, want the Controller as self and box", rows)
		}
		restored := ledgerOf(t, spareEnv).Backups
		if len(restored) != 1 || restored[0].Trigger != archive.Restored || !restored[0].Local {
			t.Fatalf("the Backups index lists %+v, want the uploaded Backup, kept", restored)
		}
	})
}

func TestAnUploadPastTheSpaceCapIsRefused(t *testing.T) {
	env, _ := testEnv(t)
	withDisk(&env, 100, nil)
	synctest.Test(t, func(t *testing.T) {
		n, c := startInMemory(t, env)
		r := upload(c, n.Ready().SetupCode, strings.NewReader(strings.Repeat("x", 101)), "application/octet-stream")
		expect(t, r, 422, `{"status":422,"code":"backup_no_space","title":"Not enough space","detail":"Restoring needs 101 bytes free on the volume of this Node's data folder, and it has 100 bytes free. Free some space, then try again."}`+"\n")
		if slices.Contains(dataDirHolds(t, env.DataDir), backups.UploadFile) {
			t.Fatal("the refused upload stayed in the data dir")
		}
	})
}

// controllerWithBackup is a Controller claimed on env, stopped after one
// Backup, with the session its claim set and its API key.
func controllerWithBackup(t *testing.T, env Env) (b taken, session, apiKey string) {
	t.Helper()
	n, c := startInMemory(t, env)
	claimed := c.claim(n.Ready().SetupCode, "1234")
	apiKey = keyOf(t, c, claimed)
	b = takeBackup(t, n)
	stop(n)
	return b, sessionOf(t, claimed), apiKey
}

func TestEachStagingErrorHasItsMessage(t *testing.T) {
	home, _ := testEnv(t)
	home.Test.Library = []storage.Schema[storage.Library]{markerAt(false)}
	var b taken
	synctest.Test(t, func(t *testing.T) { b, _, _ = controllerWithBackup(t, home) })
	{
		dir := shortTempDir(t)
		write := func(name string, data []byte) string {
			path := filepath.Join(dir, name)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			return path
		}
		file, err := os.ReadFile(b.file)
		if err != nil {
			t.Fatal(err)
		}
		other := archive.NewPassphrase(rand.Reader).Reveal()
		fresh := func(t *testing.T) Env {
			env, _ := testEnv(t)
			env.Test.Library = []storage.Schema[storage.Library]{markerAt(false)}
			return env
		}
		older := func(t *testing.T) Env {
			env, _ := testEnv(t)
			env.Test.Library = []storage.Schema[storage.Library]{storage.NewSchema[storage.Library]("marker", fstest.MapFS{"schema/00001_marker.sql": {Data: []byte(markerV1)}})}
			return env
		}
		full := func(t *testing.T) Env {
			env := fresh(t)
			withDisk(&env, 1, nil)
			return env
		}
		for _, c := range []struct {
			name  string
			env   func(*testing.T) Env
			file  string
			words string
			code  backups.StageCode
		}{
			{"a passphrase of another Backup", fresh, b.file, other, backups.WrongPassphrase},
			{"words off the list", fresh, b.file, "not a passphrase", backups.WrongPassphrase},
			{"another file", fresh, write("notes.txt", []byte("shopping list")), b.words, backups.NotABackup},
			{"half a Backup", fresh, write("half.tar.zst.age", file[:len(file)/2]), b.words, backups.Damaged},
			{"a newer library", older, b.file, b.words, backups.TooNew},
		} {
			t.Run(c.name, func(t *testing.T) {
				env := c.env(t)
				_, err := RestoreOffline(t.Context(), env, restoreFlags{file: c.file}, c.words)
				stage, ok := errors.AsType[*backups.StageError](err)
				if !ok || stage.Code != c.code {
					t.Fatalf("restore = %v, want %s", err, c.code)
				}
				if holds := dataDirHolds(t, env.DataDir); slices.ContainsFunc(holds, func(name string) bool { return strings.HasSuffix(name, lifecycle.Restored) }) {
					t.Fatalf("the refused restore left %v", holds)
				}
			})
		}
		t.Run("a full disk", func(t *testing.T) {
			env := full(t)
			_, err := RestoreOffline(t.Context(), env, restoreFlags{file: b.file}, b.words)
			stage, ok := errors.AsType[*backups.StageError](err)
			if need := backups.Size(int64(len(file))); !ok || stage.Code != backups.NoSpace || !strings.Contains(err.Error(), need) || !strings.Contains(err.Error(), env.DataDir.Path()) {
				t.Fatalf("restore = %v, want no space naming the %s it needs in %s", err, need, env.DataDir.Path())
			}
		})
	}
}

func TestAManualRollbackReplacesTheAPIKey(t *testing.T) {
	env, _ := testEnv(t)
	synctest.Test(t, func(t *testing.T) {
		b, _, apiKey := controllerWithBackup(t, env)
		if _, err := restoreOffline(t.Context(), env, restoreFlags{file: b.file}, b.words, lifecycle.ManualRollback); err != nil {
			t.Fatal(err)
		}
		_, c := startInMemory(t, env)
		if r := c.admin(apiKey, http.MethodGet, "/api/v1/nodes", ""); r.status != http.StatusUnauthorized {
			t.Fatalf("the API key the rolled back library held = %d, want it replaced", r.status)
		}
	})
}

func TestAnAutomaticRollbackOnlySwapsAndOpens(t *testing.T) {
	env, _ := testEnv(t)
	synctest.Test(t, func(t *testing.T) {
		b, session, apiKey := controllerWithBackup(t, env)
		if _, err := restoreOffline(t.Context(), env, restoreFlags{file: b.file}, b.words, lifecycle.AutomaticRollback); err != nil {
			t.Fatal(err)
		}
		n, c := startInMemory(t, env)
		if phaseOf(t, n) != reconciling.None {
			t.Fatal("an automatic rollback began Reconciling")
		}
		if r := c.admin(apiKey, http.MethodGet, "/api/v1/nodes", ""); r.status != http.StatusOK {
			t.Fatalf("the API key = %d %s, want it kept", r.status, r.body)
		}
		if r := c.getSession(session); r.status != http.StatusOK {
			t.Fatalf("the session = %d %s, want it kept", r.status, r.body)
		}
	})
}
