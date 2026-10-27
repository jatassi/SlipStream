package searching

import (
	"context"
	"fmt"
	"time"

	searchv1 "github.com/jatassi/slipstream/internal/pb/slipstream/search/v1"
	"github.com/jatassi/slipstream/internal/rpc"
)

// Run starts the job req names, or joins its run, and answers when it ends
// or with Running once Timings.Hold passed. The job runs on whatever the
// call does: a call cut short leaves it running for the next to join.
func (s *Service) Run(ctx context.Context, req *searchv1.RunRequest) (*searchv1.RunResponse, error) {
	job, site, err := decodeJob(req.GetJob())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	if s.drivers[job.Kind] == nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "this Node runs no %s Indexer", job.Kind)
	}
	if (job.Kind == Catalog) != (job.Definition != nil) {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "a catalog Indexer's job carries its definition, and no other job carries one")
	}
	r := s.jobs.join(job.ID, func(r *run) Answer { return s.do(job, site, r) })
	hold := time.NewTimer(s.cfg.Timings.Hold)
	defer hold.Stop()
	select {
	case <-r.done:
		return &searchv1.RunResponse{Answer: &searchv1.RunResponse_Done{Done: encodeAnswer(r.answer)}}, nil
	case <-hold.C:
		return &searchv1.RunResponse{Answer: &searchv1.RunResponse_Running{Running: &searchv1.Running{StartedUnixMs: unixMs(r.startedAt())}}}, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for the job: %w", ctx.Err())
	}
}

// ReachSolver lands with the Challenge solver.
func (s *Service) ReachSolver(context.Context, *searchv1.ReachSolverRequest) (*searchv1.ReachSolverResponse, error) {
	return nil, rpc.Errorf(rpc.CodeUnimplemented, "this Node reaches no Challenge solver yet")
}

// do runs job on its own context, bounded by Timings.JobCap.
func (s *Service) do(job Job, site *Site, r *run) Answer {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.Timings.JobCap)
	defer cancel()
	site.svc, site.run, site.slot = s, r, s.sessions.slot(job)
	d := s.drivers[job.Kind]
	var a Answer
	if job.Kind == Catalog {
		def, err := s.defs.load(*job.Definition)
		if err != nil {
			a.Outcome = &Stopped{Cause: DefinitionUnloadable, Text: err.Error()}
			return a
		}
		site.def, site.delay = def, def.Summary().RequestDelay
	}
	var err error
	switch c := job.Call.(type) {
	case SearchCall:
		a.Releases, err = d.Search(ctx, site, c.Query)
	case RecentCall:
		a.Releases, err = d.Recent(ctx, site)
	case FetchCall:
		a.Got, err = d.Fetch(ctx, site, c.Link)
	case TestCall:
		var report Report
		if report, err = d.Test(ctx, site, c.Captcha); err == nil {
			a.Report = &report
		}
	case CaptchaCall:
		var captcha Captcha
		if captcha, err = d.Captcha(ctx, site); err == nil {
			a.Captcha = &captcha
		}
	}
	a.Outcome = outcomeOf(ctx, err, s.cfg.Timings.JobCap)
	a.Session = site.slot.newer(job.Session)
	return a
}
