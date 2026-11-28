package converting

import (
	"slices"
	"testing"

	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
)

func job(id string, attempt uint32, stage cv1.Stage, start bool) *cv1.Job {
	return &cv1.Job{ConversionId: []byte(id), Attempt: attempt, Stage: stage, Start: start}
}

func ready(id string) *cv1.Job  { return job(id, 1, cv1.Stage_STAGE_READY, true) }
func runs(id string) *cv1.Job   { return job(id, 1, cv1.Stage_STAGE_RUNNING, true) }
func paused(id string) *cv1.Job { return job(id, 1, cv1.Stage_STAGE_PAUSED, true) }
func later(id string) *cv1.Job  { return job(id, 1, cv1.Stage_STAGE_READY, false) }
func live(id string) Have       { return Have{ID: id, Attempt: 1} }
func pausedRun(id string) Have  { return Have{ID: id, Attempt: 1, Paused: true} }
func endedRun(id string) Have   { return Have{ID: id, Attempt: 1, Ended: true} }
func intent(atOnce uint32, hold bool, jobs ...*cv1.Job) *cv1.Intent {
	return &cv1.Intent{Jobs: jobs, Hold: hold, AtOnce: atOnce}
}

type planned struct{ drop, lost, pause, resume, start []string }

func ids(jobs []*cv1.Job) []string {
	var out []string
	for _, j := range jobs {
		out = append(out, string(j.GetConversionId()))
	}
	return out
}

func TestReconcile(t *testing.T) {
	cases := []struct {
		name     string
		have     []Have
		in       *cv1.Intent
		mayStart bool
		want     planned
	}{
		{"ready jobs start in order up to at_once", nil, intent(2, false, ready("a"), ready("b"), ready("c")), true, planned{start: []string{"a", "b"}}},
		{"a live run counts against at_once", []Have{live("b")}, intent(2, false, ready("a"), runs("b"), ready("c")), true, planned{start: []string{"a"}}},
		{"a paused run counts against at_once", []Have{pausedRun("a")}, intent(1, false, paused("a"), ready("b")), true, planned{resume: []string{"a"}}},
		{"an ended run doesn't count and isn't started again", []Have{endedRun("a")}, intent(1, false, runs("a"), ready("b")), true, planned{start: []string{"b"}}},
		{"nothing starts while the snapshot may not start", nil, intent(2, false, ready("a")), false, planned{}},
		{"nothing starts outside the Node's hours", nil, intent(2, false, later("a")), true, planned{}},
		{"hold pauses every live run and starts nothing", []Have{live("a"), pausedRun("b")}, intent(3, true, runs("a"), paused("b"), ready("c")), true, planned{pause: []string{"a"}}},
		{"no hold resumes a paused run", []Have{pausedRun("a")}, intent(1, false, paused("a")), true, planned{resume: []string{"a"}}},
		{"a run outside its hours keeps running", []Have{live("a")}, intent(1, false, job("a", 1, cv1.Stage_STAGE_RUNNING, false)), false, planned{}},
		{"a canceled job's run is killed", []Have{live("a"), endedRun("b")}, intent(1, false), true, planned{drop: []string{"a", "b"}}},
		{"an older attempt is killed and the new one starts", []Have{live("a")}, intent(1, false, job("a", 2, cv1.Stage_STAGE_READY, true)), true, planned{drop: []string{"a"}, start: []string{"a"}}},
		{"a running or paused job with no run here is lost", nil, intent(1, false, runs("b"), paused("a"), ready("c")), true, planned{lost: []string{"b", "a"}, start: []string{"c"}}},
	}
	for _, c := range cases {
		p := reconcile(c.have, c.in, c.mayStart)
		got := planned{p.Drop, ids(p.Lost), ids(p.Pause), ids(p.Resume), ids(p.Start)}
		for _, f := range []struct {
			what      string
			got, want []string
		}{{"drop", got.drop, c.want.drop}, {"lost", got.lost, c.want.lost}, {"pause", got.pause, c.want.pause}, {"resume", got.resume, c.want.resume}, {"start", got.start, c.want.start}} {
			if !slices.Equal(f.got, f.want) {
				t.Errorf("%s: %s = %q, want %q", c.name, f.what, f.got, f.want)
			}
		}
	}
}
