package grabsapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

func TestAFailureStaysCurrentUntilALaterGrabHoldsItsCell(t *testing.T) {
	f := searchable(t, site{})
	uhd := targets.TargetID{UUID: storage.NewUUID()}
	err := f.lib.Write(f.ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO targets_target (id, name, created_at) VALUES (?, '2160p', 0)`, uhd)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	current := func() map[string]bool {
		t.Helper()
		code, body := f.s.Call(t, http.MethodGet, "/api/v1/grabs", "")
		var out struct {
			Grabs []struct {
				ID      string `json:"id"`
				Failure *struct {
					Current bool `json:"current"`
				} `json:"failure"`
			} `json:"grabs"`
		}
		if err := json.Unmarshal([]byte(body), &out); code != http.StatusOK || err != nil {
			t.Fatalf("listGrabs = %d %s", code, body)
		}
		got := map[string]bool{}
		for _, g := range out.Grabs {
			if g.Failure != nil {
				got[g.ID] = g.Failure.Current
			}
		}
		return got
	}

	failed := f.grabAt(t, "failed", "magnet_invalid", f.hd.ID)
	f.grabAt(t, "failed", "canceled", uhd)
	if got := current(); !got[failed] {
		t.Fatalf("with a later Grab for 2160p the HD failure reads current %v, want true", got)
	}
	f.grabAt(t, "failed", "canceled", f.hd.ID)
	if got := current(); got[failed] {
		t.Fatalf("with a later Grab for HD the HD failure reads current %v, want false", got)
	}
}
