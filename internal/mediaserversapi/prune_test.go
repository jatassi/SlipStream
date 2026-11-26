package mediaserversapi_test

import (
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/housekeeping"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/tasks"
)

// TestThePruneMovesPlaysSinceAndDeletesInBatches ends five of six plays on
// Den and lets 31 days pass. One Housekeeping run, two plays a batch,
// deletes the five, keeps the open one and moves plays_since to 30 days
// ago, the default Keep play history for. Raising the setting to 60 days
// and pruning again leaves plays_since where it was.
func TestThePruneMovesPlaysSinceAndDeletesInBatches(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWatched(t)
		server := serverOf(t, w.id)
		w.w.Stop(w.w.Seedbox)
		start := time.Now()
		for i := range 6 {
			s := session(start)
			if err := w.w.Apply(t.Context(), w.w.Seedbox, &msv1.Report{ServerId: server.Bytes(), Report: &msv1.Report_PlayStarted{PlayStarted: &msv1.PlayStarted{Session: s}}}); err != nil {
				t.Fatal(err)
			}
			if i < 5 {
				w.w.EndPlay(t, w.w.Seedbox, server, playID(t, s), start.Add(time.Duration(i+1)*10*time.Second))
			}
		}
		wait(31 * day)

		before := time.Now()
		prune(t, w)
		after := time.Now()
		left := clip + ` direct_play (direct direct none) [] by ada on Living room, open`
		expectPlays(t, w.plays(t), left)
		since := w.w.PlaysSince(t, server)
		if since.Before(before.Add(-30*day)) || since.After(after.Add(-30*day)) {
			t.Errorf("plays_since is %v, want 30 days before the prune at %v", since, before)
		}
		if !strings.Contains(w.w.Logged(), `msg="pruned Media Server plays" plays=5`) {
			t.Errorf("the prune didn't log the 5 plays it deleted:\n%s", w.w.Logged())
		}

		if status, body := w.o.Call(t, http.MethodPatch, servers+"/settings", `{"values":{"mediaServers.keepPlays":5184000}}`); status != http.StatusOK {
			t.Fatalf("raise Keep play history for = %d %s", status, body)
		}
		prune(t, w)
		if got := w.w.PlaysSince(t, server); !got.Equal(since) {
			t.Errorf("after raising Keep play history for, plays_since is %v, want %v", got, since)
		}
		expectPlays(t, w.plays(t), left)
	})
}

func prune(t *testing.T, w watched) {
	t.Helper()
	step := w.w.Moves().Housekeeping()
	task := housekeeping.Task(housekeeping.Config{Steps: []housekeeping.Step{step}})
	if err := task.Run(t.Context(), tasks.Unit{Key: step.Key}, &tasks.Progress{}); err != nil {
		t.Fatal(err)
	}
}
