package searching

import (
	"context"
	"errors"
	"fmt"
	"time"

	searchv1 "github.com/jatassi/slipstream/internal/pb/slipstream/search/v1"
	"github.com/jatassi/slipstream/internal/release"
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
	if (job.Kind == Native) != (job.Port != "") {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "a native Indexer's job names its port, and no other job names one")
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

// ReachSolver sends GET {address}/health to a Challenge solver from this
// Node, within the solver's timeout, and answers the status. A solver the
// Node can't reach fails the call with FailedPrecondition and the cause.
func (s *Service) ReachSolver(ctx context.Context, req *searchv1.ReachSolverRequest) (*searchv1.ReachSolverResponse, error) {
	if _, err := httpURL(req.GetAddress()); err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "the Challenge solver's address: %v", err)
	}
	status, err := reachSolver(ctx, s.client, req.GetAddress(), time.Duration(req.GetTimeoutMs())*time.Millisecond)
	if u, ok := errors.AsType[*SolverUnreachable](err); ok {
		return nil, rpc.Errorf(rpc.CodeFailedPrecondition, "%s", u.Cause)
	}
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	return &searchv1.ReachSolverResponse{Status: int32(status)}, nil //nolint:gosec // an HTTP status fits
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
		site.def = def
		site.Space(def.Summary().RequestDelay)
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
	for i := range a.Releases {
		a.Releases[i].Flags = withFactorFlags(a.Releases[i])
	}
	a.Outcome = outcomeOf(ctx, err, s.cfg.Timings.JobCap)
	a.Session = site.slot.newer(job.Session)
	return a
}

// withFactorFlags are r's flags and the ones its factors mean, as Radarr's
// Torznab parser reads them: download 0 is freeleech, 0.25 freeleech75,
// 0.5 halfleech and 0.75 freeleech25, and upload 2 is doubleupload. Every
// kind's Releases come through here, so no driver sets these flags.
func withFactorFlags(r Release) []Flag {
	flags := r.Flags
	switch r.Factors.Download {
	case 0:
		flags = append(flags, release.Freeleech)
	case 0.25:
		flags = append(flags, release.Freeleech75)
	case 0.5:
		flags = append(flags, release.Halfleech)
	case 0.75:
		flags = append(flags, release.Freeleech25)
	}
	if r.Factors.Upload == 2 {
		flags = append(flags, release.DoubleUpload)
	}
	return flags
}
