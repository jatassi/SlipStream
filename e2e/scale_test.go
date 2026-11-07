package e2e

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func TestTheWorldAddsEveryGeneratedMovie(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("scale")
	if err != nil {
		t.Fatal(err)
	}
	s.Name = "scale-small"
	g := scenario.TMDBGenerated{Count: 40, Seed: "scale-small"}
	s.TMDB.Generated = &g
	run := worldtest.LaunchE2EScenario(t, s)
	a := api(t, run)

	rec, err := events.Wait(worldtest.Context(t), run.Manifest().RunDir, events.Query{Kind: events.WorksAdded})
	if err != nil {
		t.Fatalf("events.jsonl has no %s record: %v", events.WorksAdded, err)
	}
	var journaled struct {
		ContentType string  `json:"contentType"`
		Count       int     `json:"count"`
		Seconds     float64 `json:"seconds"`
	}
	if err := json.Unmarshal(rec.Data, &journaled); err != nil || journaled.ContentType != "movie" || journaled.Count != 40 || journaled.Seconds <= 0 {
		t.Fatalf("%s = %s, %v; want 40 movies and the time they took", events.WorksAdded, rec.Data, err)
	}

	var roots struct {
		Roots []struct {
			Works int `json:"works"`
		} `json:"roots"`
	}
	decode(t, "listRoots", a.Admin("home", http.MethodGet, "/api/v1/roots", ""), &roots)
	if len(roots.Roots) != 1 || roots.Roots[0].Works != 40 {
		t.Fatalf("listRoots = %+v, want the one movie root holding 40 Works", roots)
	}

	for _, i := range []int{0, 10, 39} {
		m := g.Movie(i)
		again := decode(t, "addWork again", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"`+strconv.FormatInt(m.ID, 10)+`"}`), &added{})
		var tr tree
		if err := json.Unmarshal(again.Work, &tr); err != nil || again.Created || tr.Label != m.Title || len(tr.Items) != 1 || !tr.Items[0].Monitored {
			t.Fatalf("addWork of generated movie %d = created %t, work %s: %v; want %q added already and monitored", i, again.Created, again.Work, err, m.Title)
		}
	}
}
