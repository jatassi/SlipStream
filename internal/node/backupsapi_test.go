package node

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/backups/archive"
)

// backupsController is a running Controller claimed on a fresh data dir,
// with its Admin session and its API key.
// backupsController is a claimed Controller whose start's catch-up runs
// have ended. Its first scheduled Backup is one of them, and it waits
// behind the other Maintenance window tasks, so left running it can land
// in the middle of a test's own Backups.
func backupsController(t *testing.T) (n *Node, c client, session, apiKey string) {
	t.Helper()
	env, _ := testEnv(t)
	n, c = startInMemory(t, env)
	claimed := c.claim(n.Ready().SetupCode, "1234")
	session, apiKey = sessionOf(t, claimed), keyOf(t, c, claimed)
	synctest.Wait()
	return n, c, session, apiKey
}

func asKey(apiKey string) map[string]string { return map[string]string{"X-Api-Key": apiKey} }

type backupsListed struct {
	Backups []struct {
		ID      string `json:"id"`
		Trigger string `json:"trigger"`
		State   string `json:"state"`
	} `json:"backups"`
	Passphrase struct {
		SavedAt *time.Time `json:"savedAt"`
	} `json:"passphrase"`
	Sheet struct {
		DownloadedAt *time.Time `json:"downloadedAt"`
		Unlisted     []string   `json:"unlisted"`
	} `json:"sheet"`
}

func listBackups(t *testing.T, c client, session string) backupsListed {
	t.Helper()
	return decode[backupsListed](t, c.do(call{method: http.MethodGet, path: "/api/v1/backups", session: session}), http.StatusOK)
}

func TestThePassphraseNeedsAReconfirmedSession(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n, c, session, apiKey := backupsController(t)
		got := decode[struct {
			Passphrase string `json:"passphrase"`
		}](t, c.do(call{method: http.MethodGet, path: "/api/v1/backups/passphrase", session: session}), http.StatusOK)
		if got.Passphrase != currentWords(t, n) {
			t.Fatalf("the passphrase shown is %q, want the Vault's current one", got.Passphrase)
		}
		needsAccount := problem(403, "needs_account", "The API key can't do this. Sign in to an Account to do it.")
		for _, k := range []call{
			{method: http.MethodGet, path: "/api/v1/backups/passphrase"},
			{method: http.MethodPost, path: "/api/v1/backups/passphrase", body: strings.NewReader(`{"deleteOlder":true}`)},
			{method: http.MethodGet, path: "/api/v1/backups/recovery-sheet"},
			{method: http.MethodPost, path: "/api/v1/backups/recovery-sheet/ticket"},
			{method: http.MethodPost, path: "/api/v1/backups/retire"},
		} {
			k.headers = asKey(apiKey)
			r := c.do(k)
			if r.status != http.StatusForbidden || r.body != needsAccount {
				t.Errorf("%s %s with the API key = %d %s, want 403 needs_account", k.method, k.path, r.status, r.body)
			}
		}
		for _, k := range []call{
			{method: http.MethodPost, path: "/api/v1/backups/passphrase/saved"},
			{method: http.MethodPost, path: "/api/v1/backups/passphrase/check", body: strings.NewReader(`{"passphrase":"` + currentWords(t, n) + `"}`)},
		} {
			k.headers = asKey(apiKey)
			expect(t, c.do(k), 409, problem(409, "passphrase_needs_session", "Only an Admin signed in to an Account can confirm the Backup passphrase"))
		}
		if l := listBackups(t, c, session); l.Passphrase.SavedAt != nil {
			t.Fatal("a refused call saved the passphrase")
		}
	})
}

func TestBackUpNowAnswers202(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n, c, session, apiKey := backupsController(t)
		r := c.do(call{method: http.MethodPost, path: "/api/v1/backups", headers: asKey(apiKey)})
		if r.status != http.StatusAccepted {
			t.Fatalf("Back up now = %d %s, want 202", r.status, r.body)
		}
		takeBackup(t, n)
		manual := 0
		for _, b := range listBackups(t, c, session).Backups {
			if b.Trigger == string(archive.Manual) && b.State == "verified" {
				manual++
			}
		}
		if manual != 2 {
			t.Fatalf("listed %d checked manual Backups, want Back up now's and the one taken after it", manual)
		}
	})
}

func TestDeletingTheNewestVerifiedIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n, c, session, _ := backupsController(t)
		takeBackup(t, n)
		takeBackup(t, n)
		l := listBackups(t, c, session)
		newest, older := l.Backups[0].ID, l.Backups[1].ID
		expect(t, c.do(call{method: http.MethodDelete, path: "/api/v1/backups/" + newest, session: session}), 409,
			`{"status":409,"code":"newest_verified","title":"The newest good Backup can't be deleted","detail":"This is your newest good Backup, so it can't be deleted. Select Back up now, then delete it."}`+"\n")
		left := decode[backupsListed](t, c.do(call{method: http.MethodDelete, path: "/api/v1/backups/" + older, session: session}), http.StatusOK)
		ids := make([]string, len(left.Backups))
		for i, b := range left.Backups {
			ids[i] = b.ID
		}
		if len(ids) != len(l.Backups)-1 || ids[0] != newest || slices.Contains(ids, older) {
			t.Fatalf("after deleting %s the list is %v, want every other Backup, %s first", older, ids, newest)
		}
	})
}

func TestTheSheetListsEveryDestinationAndRecordsTheDownload(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n, c, session, _ := backupsController(t)
		folder := filepath.Join(shortTempDir(t), "copies")
		if err := os.Mkdir(folder, 0o700); err != nil {
			t.Fatal(err)
		}
		self := n.half.Load().nodes.Self().String()
		added := decode[struct {
			Destination struct {
				ID string `json:"id"`
			} `json:"destination"`
		}](t, c.do(call{method: http.MethodPost, path: "/api/v1/backup-destinations", session: session, body: strings.NewReader(`{"nodeId":"` + self + `","folder":` + quoted(folder) + `}`)}), http.StatusCreated)
		if l := listBackups(t, c, session); !slices.Equal(l.Sheet.Unlisted, []string{added.Destination.ID}) || l.Sheet.DownloadedAt != nil {
			t.Fatalf("before a download the sheet reads %+v, want the new destination unlisted", l.Sheet)
		}
		if r := c.do(call{method: http.MethodPost, path: "/api/v1/backups/recovery-sheet/ticket", session: session}); r.status != http.StatusNoContent {
			t.Fatalf("the sheet's ticket = %d %s, want 204", r.status, r.body)
		}
		r := c.do(call{method: http.MethodGet, path: "/api/v1/backups/recovery-sheet", session: session})
		if r.status != http.StatusOK || !strings.Contains(r.header.Get("Content-Disposition"), "slipstream-recovery-sheet.txt") {
			t.Fatalf("the sheet's download = %d %v", r.status, r.header)
		}
		for _, want := range []string{"Backup passphrase: " + currentWords(t, n) + "\n", ": " + folder + "\n", "Restore a Backup"} {
			if !strings.Contains(r.body, want) {
				t.Errorf("the sheet lacks %q:\n%s", want, r.body)
			}
		}
		if l := listBackups(t, c, session); len(l.Sheet.Unlisted) != 0 || l.Sheet.DownloadedAt == nil {
			t.Fatalf("after the download the sheet reads %+v, want it recorded with every destination", l.Sheet)
		}
	})
}

func TestTheBackupStepIsDoneOnceSaved(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, c, session, _ := backupsController(t)
		state := func() string {
			v := decode[struct {
				Steps []struct {
					ID    string `json:"id"`
					State string `json:"state"`
				} `json:"steps"`
			}](t, c.do(call{method: http.MethodGet, path: "/api/v1/first-run", session: session}), http.StatusOK)
			for _, s := range v.Steps {
				if s.ID == "backup" {
					return s.State
				}
			}
			return ""
		}
		if got := state(); got != "todo" {
			t.Fatalf("the backup step reads %q before the passphrase is saved, want todo", got)
		}
		wrong := decode[struct {
			Matches bool `json:"matches"`
		}](t, c.do(call{method: http.MethodPost, path: "/api/v1/backups/passphrase/check", session: session, body: strings.NewReader(`{"passphrase":"not the words"}`)}), http.StatusOK)
		if wrong.Matches || state() != "todo" {
			t.Fatal("a wrong saved copy matched or finished the step")
		}
		if r := c.do(call{method: http.MethodPost, path: "/api/v1/backups/passphrase/saved", session: session}); r.status != http.StatusNoContent {
			t.Fatalf("I saved it = %d %s, want 204", r.status, r.body)
		}
		if got := state(); got != "done" {
			t.Fatalf("the backup step reads %q once the passphrase is saved, want done", got)
		}
	})
}

func quoted(s string) string { return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"` }
