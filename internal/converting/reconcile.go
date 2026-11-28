package converting

import cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"

// Have is a job this holder took up: its conversion id, as a string of its
// bytes, and its attempt.
type Have struct {
	ID      string
	Attempt uint32
	Paused  bool
	// Ended is a run that exited, failed to start or was reported Lost.
	Ended bool
}

// Plan is what one intent asks of the holder.
type Plan struct {
	// Drop is every job no longer in the intent at its attempt: a cancel, a
	// put-back or an older attempt. A live run is killed.
	Drop []string
	// Lost is a job the intent says runs that this holder never took up:
	// a restart.
	Lost   []*cv1.Job
	Pause  []*cv1.Job
	Resume []*cv1.Job
	Start  []*cv1.Job
}

// reconcile plans one intent against the jobs the holder has. A ready job
// starts while the Node is inside its hours, the snapshot may start work,
// no Media Server holds the Node, and fewer than at_once runs are live.
// Every live run pauses under hold and resumes without it.
func reconcile(have []Have, in *cv1.Intent, mayStart bool) Plan {
	jobs := map[string]*cv1.Job{}
	for _, j := range in.GetJobs() {
		jobs[string(j.GetConversionId())] = j
	}
	var p Plan
	kept := map[string]Have{}
	for _, r := range have {
		if j, ok := jobs[r.ID]; !ok || j.GetAttempt() != r.Attempt {
			p.Drop = append(p.Drop, r.ID)
			continue
		}
		kept[r.ID] = r
	}
	live := 0
	for _, r := range kept {
		if !r.Ended {
			live++
		}
	}
	hold := in.GetHold()
	for _, j := range in.GetJobs() {
		r, ok := kept[string(j.GetConversionId())]
		switch {
		case ok && r.Ended:
		case ok && hold && !r.Paused:
			p.Pause = append(p.Pause, j)
		case ok && !hold && r.Paused:
			p.Resume = append(p.Resume, j)
		case ok:
		case j.GetStage() == cv1.Stage_STAGE_RUNNING || j.GetStage() == cv1.Stage_STAGE_PAUSED:
			p.Lost = append(p.Lost, j)
		case j.GetStart() && mayStart && !hold && live < int(in.GetAtOnce()):
			p.Start = append(p.Start, j)
			live++
		}
	}
	return p
}
