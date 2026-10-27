package e2e

import (
	"encoding/json"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type sseView struct {
	As    string `json:"as"`
	Node  string `json:"node"`
	Event string `json:"event"`
	Data  string `json:"data"`
}

func messages(t *testing.T, runDir string, after uint64) []sseView {
	t.Helper()
	recs, err := events.Select(runDir, events.Query{Kind: events.SSEMessage, Match: map[string]string{"as": world.AdminIdentity}, After: after})
	if err != nil {
		t.Fatal(err)
	}
	out := make([]sseView, len(recs))
	for i, r := range recs {
		if err := json.Unmarshal(r.Data, &out[i]); err != nil || r.Node != out[i].Node {
			t.Fatalf("sse.message %d = %s on %s: %v", r.Seq, r.Data, r.Node, err)
		}
	}
	return out
}

func TestSSEIsCapturedFromLaunch(t *testing.T) {
	t.Parallel()
	w := walk(t, "single-machine")
	ready, err := events.Select(w.m.RunDir, events.Query{Kind: events.RunReady})
	if err != nil || len(ready) != 1 {
		t.Fatalf("run.ready records %+v, %v; want one", ready, err)
	}
	opening, err := events.Wait(worldtest.Context(t), w.m.RunDir, events.Query{Kind: events.SSEMessage, Match: map[string]string{"as": world.AdminIdentity}})
	if err != nil {
		t.Fatal(err)
	}
	launched := messages(t, w.m.RunDir, 0)
	if want := (sseView{world.AdminIdentity, "home", "live.resync", "{}"}); opening.Seq > ready[0].Seq || launched[0] != want {
		t.Fatalf("the captured stream starts %+v at seq %d, want %+v first and before run.ready at seq %d", launched[0], opening.Seq, want, ready[0].Seq)
	}
	changed := sseView{world.AdminIdentity, "home", "grabs.changed", `{"grabId":"` + w.grab.ID + `","workId":"` + w.work + `"}`}
	found := false
	for _, m := range launched {
		found = found || m == changed
	}
	if !found {
		t.Fatalf("the captured stream %+v holds no %+v", launched, changed)
	}

	act(t, w.run, "home", world.KillNode, world.StartNode)
	restarted, err := events.Select(w.m.RunDir, events.Query{Kind: events.NodeReady, Node: "home"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := events.Wait(worldtest.Context(t), w.m.RunDir, events.Query{Kind: events.SSEMessage, Node: "home", After: restarted[len(restarted)-1].Seq}); err != nil {
		t.Fatalf("no sse.message after home restarted: %v", err)
	}
}
