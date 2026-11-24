package sim_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/movestest"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type previewReply struct {
	Token   string `json:"token"`
	Movable int    `json:"movable"`
	Units   []struct {
		Verdict string `json:"verdict"`
		To      struct {
			RootID   string `json:"rootId"`
			NodeName string `json:"nodeName"`
		} `json:"to"`
	} `json:"units"`
}

type batchReply struct {
	Batch struct {
		CanMoveBack bool `json:"canMoveBack"`
	} `json:"batch"`
	Units []struct {
		ID    string `json:"id"`
		Phase string `json:"phase"`
	} `json:"units"`
}

// problemCode is the code of a problem reply, failing unless it has status.
func problemCode(t *testing.T, what string, r worldtest.Reply, status int) string {
	t.Helper()
	var p struct {
		Code string `json:"code"`
	}
	if r.Status != status || json.Unmarshal([]byte(r.Body), &p) != nil {
		t.Fatalf("%s = %d %s, want %d", what, r.Status, r.Body, status)
	}
	return p.Code
}

func previewOver(t *testing.T, w movestest.World, folder string) previewReply {
	t.Helper()
	body := fmt.Sprintf(`{"origin":"work","works":[{"workId":%q,"rootId":%q}]}`, w.Work, w.Roots[folder].ID)
	return simtest.Decode[previewReply](t, "previewMoves", w.API().Admin("home", http.MethodPost, "/api/v1/moves/preview", body))
}

func startOver(w movestest.World, token string, confirm int) worldtest.Reply {
	return w.API().Admin("home", http.MethodPost, fmt.Sprintf("/api/v1/moves?confirm=%d", confirm), fmt.Sprintf(`{"token":%q}`, token))
}

// TestTheMovesAPIStartsOnceAndRefusesWhatItCannotDo previews and starts a
// Move over HTTP, sees a stale count and an unknown token refused and a
// replayed Start answer the same batch, then sees a finished unit refuse
// a cancel, a switch now and a retry, and offer its Move back until the
// Move back moved the Work again.
func TestTheMovesAPIStartsOnceAndRefusesWhatItCannotDo(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := movestest.Launch(t)
		plan := previewOver(t, w, "movies-f")
		if plan.Movable != 1 || len(plan.Units) != 1 || plan.Units[0].To.NodeName != "den" {
			t.Fatalf("the preview is %+v, want one Work that moves to den", plan)
		}
		if code := problemCode(t, "a Start with a stale count", startOver(w, plan.Token, 2), http.StatusConflict); code != "move_plan_changed" {
			t.Errorf("a Start with a stale count answered %s, want move_plan_changed", code)
		}
		if code := problemCode(t, "a Start of an unknown preview", startOver(w, "nope", 1), http.StatusGone); code != "move_plan_expired" {
			t.Errorf("a Start of an unknown preview answered %s, want move_plan_expired", code)
		}
		batch := startedOver(t, w, plan.Token)
		unit := doneOver(t, w, batch)
		refusals := []struct{ act, want string }{
			{"cancel", "unit_switched"},
			{"switch-now", "switch_not_waiting"},
			{"retry", "unit_not_failed"},
		}
		for _, r := range refusals {
			reply := w.API().Admin("home", http.MethodPost, "/api/v1/moves/units/"+unit+"/"+r.act, "")
			if code := problemCode(t, r.act, reply, http.StatusConflict); code != r.want {
				t.Errorf("%s of a done unit answered %s, want %s", r.act, code, r.want)
			}
		}
		back := simtest.Decode[previewReply](t, "previewMoveBack", w.API().Admin("home", http.MethodGet, "/api/v1/moves/"+batch+"/back", ""))
		if back.Movable != 1 || back.Units[0].To.RootID != w.Roots["movies-e"].ID {
			t.Errorf("the Move back is %+v, want one Work back to movies-e", back)
		}
		doneOver(t, w, startedOver(t, w, back.Token))
		first := simtest.Decode[batchReply](t, "getMove", w.API().Admin("home", http.MethodGet, "/api/v1/moves/"+batch, ""))
		if first.Batch.CanMoveBack {
			t.Errorf("the first batch reads %+v after its Move back, want it unable to move back", first.Batch)
		}
	})
}

// startedOver starts token's preview, then sends the Start again, and
// answers the batch both named.
func startedOver(t *testing.T, w movestest.World, token string) string {
	t.Helper()
	var batches [2]string
	for i := range batches {
		r := startOver(w, token, 1)
		var s struct {
			BatchID string `json:"batchId"`
		}
		if r.Status != http.StatusAccepted || json.Unmarshal([]byte(r.Body), &s) != nil {
			t.Fatalf("startMoves = %d %s, want 202", r.Status, r.Body)
		}
		batches[i] = s.BatchID
	}
	if batches[0] != batches[1] {
		t.Fatalf("a replayed Start answered batch %s, want %s", batches[1], batches[0])
	}
	return batches[0]
}

// doneOver waits until batch's one unit is done over HTTP, and answers
// its id.
func doneOver(t *testing.T, w movestest.World, batch string) string {
	t.Helper()
	var b batchReply
	simtest.Eventually(t, "the unit is done", func() bool {
		b = simtest.Decode[batchReply](t, "getMove", w.API().Admin("home", http.MethodGet, "/api/v1/moves/"+batch, ""))
		return len(b.Units) == 1 && b.Units[0].Phase == "done"
	})
	if !b.Batch.CanMoveBack {
		t.Errorf("the done batch reads %+v, want it able to move back", b)
	}
	return b.Units[0].ID
}

// TestTheMovesSettingsChangeByKeyAllOrNone reads the two Moves settings,
// changes the Plex bound, and sees a value out of range refused with
// nothing stored.
func TestTheMovesSettingsChangeByKeyAllOrNone(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := movestest.Scenario(t)
		w := sim.Launch(t, s)
		type settings struct {
			Settings []struct {
				Key   string `json:"key"`
				Value int64  `json:"value"`
			} `json:"settings"`
		}
		patch := func(body string) worldtest.Reply {
			return w.API().Admin("home", http.MethodPatch, "/api/v1/moves/settings", body)
		}
		changed := simtest.Decode[settings](t, "patchMoveSettings", patch(`{"values":{"moves.plexBound":600}}`))
		refused := patch(`{"values":{"moves.needsYouAfter":7200,"moves.plexBound":10}}`)
		if code := problemCode(t, "a Plex bound under a minute", refused, http.StatusUnprocessableEntity); code != "setting_out_of_range" {
			t.Errorf("a Plex bound under a minute answered %s, want setting_out_of_range", code)
		}
		got := simtest.Decode[settings](t, "getMoveSettings", w.API().Admin("home", http.MethodGet, "/api/v1/moves/settings", ""))
		want := map[string]int64{"moves.needsYouAfter": 3600, "moves.plexBound": 600}
		for _, read := range []settings{changed, got} {
			if len(read.Settings) != len(want) {
				t.Fatalf("the settings are %+v, want %v", read.Settings, want)
			}
			for _, f := range read.Settings {
				if f.Value != want[f.Key] {
					t.Errorf("%s is %d, want %d", f.Key, f.Value, want[f.Key])
				}
			}
		}
	})
}
