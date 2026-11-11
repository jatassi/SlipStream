package grabsapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

func (f *atticFixture) grabAt(t *testing.T, stage, failure string, target targets.TargetID) string {
	t.Helper()
	id := storage.NewUUID()
	now := storage.Now()
	err := f.lib.Write(f.ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, f.home); err != nil {
			return err
		}
		var code, msg, at *string
		if failure != "" {
			code, msg, at = &failure, new("The link names no info hash."), new("fetching")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO grabs_grab (id, work_id, item_id, edition, node_id, release_title, release_size, release_magnet,
			stage, failed_stage, failure_code, failure_message, created_at, updated_at)
			VALUES (?, ?, ?, 'any', ?, 'The.Attic.1080p.WEB-DL', 4096, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567', ?, ?, ?, ?, ?, ?)`,
			id, f.tree.ID, f.tree.Items[0].ID, f.home, stage, at, code, msg, now, now); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO grabs_target VALUES (?, 0, ?)`, id, target)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id.String()
}

func TestCancelThroughTheAPIKeyFailsTheGrabAndBlocklistsItsReleaseWithNoAccount(t *testing.T) {
	f := searchable(t, site{})
	id := f.grabAt(t, "fetching", "", f.hd.ID)

	for range 2 {
		if code, got := f.s.Call(t, http.MethodPost, "/api/v1/grabs/"+id+"/cancel", `{"blocklist":true}`); code != http.StatusNoContent {
			t.Fatalf("cancelGrab = %d %s, want 204", code, got)
		}
	}
	<-f.blocklisted

	_, got := f.s.Call(t, http.MethodGet, "/api/v1/grabs/"+id, "")
	for _, want := range []string{
		`"failure":{"stage":"fetching","code":"canceled","message":"This Grab was canceled through the API key.","needsYou":false,"current":true}`,
		`"actions":{"cancel":false,"blocklistable":false,"sendTo":false}`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("getGrab = %s, want %s", got, want)
		}
	}
	_, got = f.s.Call(t, http.MethodGet, "/api/v1/blocklist", "")
	for _, want := range []string{`"total":1`, `"cause":"by_hand"`, `"because":"Blocklisted through the API key when its Grab was canceled."`} {
		if !strings.Contains(got, want) {
			t.Fatalf("listBlocklist = %s, want %s", got, want)
		}
	}
}

func TestCancelAnswersEachProblem(t *testing.T) {
	f := searchable(t, site{})
	failed := f.grabAt(t, "failed", "magnet_invalid", f.hd.ID)

	f.s.Expect(t, http.MethodPost, "/api/v1/grabs/"+failed+"/cancel", `{"blocklist":false}`, http.StatusConflict,
		`{"status":409,"code":"grab_finished","title":"This Grab already finished, so there's nothing to cancel"}`)
	f.s.Expect(t, http.MethodPost, "/api/v1/grabs/"+storage.NewUUID().String()+"/cancel", `{"blocklist":false}`, http.StatusNotFound,
		`{"status":404,"code":"grab_not_found","title":"No Grab has that id"}`)
}
