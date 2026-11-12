package sim_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim/simtest"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
	"github.com/jatassi/slipstream/internal/works"
)

// retryPieces is each release's size in pieces.
const retryPieces = 5

type retryRow struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	Hold      string `json:"hold"`
	Size      int64  `json:"size"`
	Done      int64  `json:"done"`
	Retryable bool   `json:"retryable"`
	Retry     *struct {
		Found *int64 `json:"found"`
	} `json:"retry"`
}

// retryRows is home's Torrents by id, as GET /torrents lists them.
func retryRows(t *testing.T, w *sim.World) map[string]retryRow {
	t.Helper()
	rows := simtest.Decode[struct {
		Rows []retryRow `json:"rows"`
	}](t, "listTorrents", w.API().Admin("home", http.MethodGet, "/api/v1/torrents?tab=all", "")).Rows
	out := map[string]retryRow{}
	for _, r := range rows {
		out[r.ID] = r
	}
	return out
}

// attachRelease has the Controller hand box a release of name announcing
// to idleTracker, whose payload swarm holds when seeded.
func attachRelease(t *testing.T, lib *storage.DB[storage.Library], swarm *enginetest.World, box role.NodeID, name string, seeded bool) (downloadclients.ID, bencode.InfoHash) {
	t.Helper()
	payload := make([]byte, retryPieces*releases.PieceLength)
	copy(payload, name)
	for i := len(name); i < len(payload); i++ {
		payload[i] = byte(i % 251)
	}
	raw, hash, err := releases.Metainfo(name, nil, payload, idleTracker, false)
	if err != nil {
		t.Fatal(err)
	}
	if seeded {
		swarm.Payload(hash, payload)
	}
	m, err := bencode.DecodeMetainfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	var id downloadclients.ID
	err = lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		want := downloadclients.Want{Target: downloadclients.Target{Node: box}, Metainfo: m, Raw: raw, ContentType: works.TypeID("movie"), WrittenOff: func(role.NodeID) bool { return false }}
		id, _, err = downloadclients.Attach(ctx, tx, want, storage.Now())
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id, hash
}

// TestRetryChecksAFailedSeedAgain fails four torrents on seedbox as a
// read of their files fails: a seed, a seed the Admin paused, a seed that
// has since lost three of its five pieces, and a download. Retry refuses
// the download as not_retryable. The two seeds whose data is all there
// get back the wanted state they had: one seeds again and the paused one
// stays paused. The one that lost pieces stays paused, and its row says
// its check found two of its five pieces.
func TestRetryChecksAFailedSeedAgain(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w, swarm := launchSwarm(t, enginetest.TrackerRules{Interval: 30 * time.Minute}, &engines{})
		u, err := storage.ParseUUID(seedbox(t, w.API()).ID)
		if err != nil {
			t.Fatal(err)
		}
		boxID := role.NodeID{UUID: u}
		controller := w.Node("home").Half()
		whole, wholeHash := attachRelease(t, controller.Library, swarm, boxID, "Hold.Fast.mkv", true)
		paused, pausedHash := attachRelease(t, controller.Library, swarm, boxID, "Kept.Still.mkv", true)
		lost, lostHash := attachRelease(t, controller.Library, swarm, boxID, "Lost.Half.mkv", true)
		download, downloadHash := attachRelease(t, controller.Library, swarm, boxID, "Never.Came.mkv", false)
		controller.Clients.Touch(boxID)
		seeds := []downloadclients.ID{whole, paused, lost}
		eventuallyIn(t, w, "seeding", seeds...)
		act(t, w, downloadclients.Action{Kind: downloadclients.Pause, IDs: []downloadclients.ID{paused}})
		simtest.Eventually(t, "the paused seed is paused", func() bool { return retryRows(t, w)[paused.String()].Hold == "wanted" })

		readError := &engine.Error{Category: engine.CategorySystem, Code: 2, Text: "No such file or directory"}
		for _, h := range []bencode.InfoHash{wholeHash, pausedHash, lostHash, downloadHash} {
			swarm.Fail(h, readError)
		}
		if err := swarm.Lose(lostHash, 3*releases.PieceLength); err != nil {
			t.Fatal(err)
		}
		eventuallyIn(t, w, "failed", whole, paused, lost, download)
		rows := retryRows(t, w)
		for _, id := range append(seeds, download) {
			if r := rows[id.String()]; r.Retryable != (id != download) {
				t.Fatalf("the failed torrent %s reads %+v, want only the seeds retryable", id, r)
			}
		}

		retryAll(t, w, whole, paused, lost, download)
		simtest.Eventually(t, "every Retry is settled and the seed that was started seeds again", func() bool {
			rows := retryRows(t, w)
			return found(rows[whole.String()]) != nil && found(rows[paused.String()]) != nil && found(rows[lost.String()]) != nil &&
				rows[whole.String()].State == "seeding" && rows[whole.String()].Hold == ""
		})
		time.Sleep(time.Hour)
		wantSettled(t, retryRows(t, w), whole, paused, lost, download)
	})
}

// wantSettled checks, an hour after their Retries, that the seed that was
// started seeds, the paused one stays paused, both found all their data,
// the one that lost three pieces stays paused with two found, and the
// download was never retried.
func wantSettled(t *testing.T, rows map[string]retryRow, whole, paused, lost, download downloadclients.ID) {
	t.Helper()
	for _, id := range []downloadclients.ID{whole, paused} {
		if r := rows[id.String()]; *found(r) != r.Size {
			t.Errorf("the seed %s whose data is all there found %d of %d bytes, want all", id, *found(r), r.Size)
		}
	}
	if r := rows[whole.String()]; r.State != "seeding" {
		t.Errorf("an hour after its Retry, the seed that was started reads %+v, want seeding", r)
	}
	if r := rows[paused.String()]; r.State != "paused" || r.Hold != "wanted" {
		t.Errorf("an hour after its Retry, the seed the Admin paused reads %+v, want paused as it was", r)
	}
	r := rows[lost.String()]
	if want := int64(2 * releases.PieceLength); r.State != "paused" || r.Hold != "wanted" || *found(r) != want || r.Done != want || r.Retryable {
		t.Errorf("an hour after its Retry, the seed that lost three of its five pieces reads %+v, found %d; want paused with the %d bytes left found", r, *found(r), want)
	}
	if r := rows[download.String()]; r.State != "failed" || r.Retry != nil {
		t.Errorf("the failed download reads %+v, want still failed and never retried", r)
	}
}

// eventuallyIn waits until each of ids reads state.
func eventuallyIn(t *testing.T, w *sim.World, state string, ids ...downloadclients.ID) {
	t.Helper()
	simtest.Eventually(t, "the torrents read "+state, func() bool {
		rows := retryRows(t, w)
		for _, id := range ids {
			if rows[id.String()].State != state {
				return false
			}
		}
		return true
	})
}

// retryAll has the Admin retry three failed seeds and a failed download:
// the seeds are retried and the download is refused as not_retryable.
func retryAll(t *testing.T, w *sim.World, whole, paused, lost, download downloadclients.ID) {
	t.Helper()
	body := fmt.Sprintf(`{"action":"retry","ids":[%q,%q,%q,%q]}`, whole, paused, lost, download)
	got := simtest.Decode[struct {
		Results []struct {
			Done    bool `json:"done"`
			Refusal *struct {
				Code string `json:"code"`
			} `json:"refusal"`
		} `json:"results"`
	}](t, "actOnTorrents", w.API().Admin("home", http.MethodPost, "/api/v1/torrents/actions", body)).Results
	if len(got) != 4 || !got[0].Done || !got[1].Done || !got[2].Done || got[3].Refusal == nil || got[3].Refusal.Code != "not_retryable" {
		t.Fatalf("Retry of the three seeds and the download answered %+v, want the seeds retried and the download not_retryable", got)
	}
}

func found(r retryRow) *int64 {
	if r.Retry == nil {
		return nil
	}
	return r.Retry.Found
}
