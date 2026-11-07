package port

import (
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/searching"
)

// The framework. Nothing here is exported: a port can't reach the loop,
// the session or the Site, so it can't get them wrong.

// driver is searching.Driver for kind native. It dispatches by site.Port.
type driver struct{ r Registry }

// runner is a Port's Impl with S erased.
type runner interface {
	search(ctx context.Context, s *searching.Site, e Entry, q searching.Query) ([]searching.Release, error)
	recent(ctx context.Context, s *searching.Site, e Entry) ([]searching.Release, error)
	fetch(ctx context.Context, s *searching.Site, e Entry, link searching.Link) (searching.Got, error)
	test(ctx context.Context, s *searching.Site, e Entry) (searching.Report, error)
}

// typed is Impl[S] behind runner, with S's settings.
type typed[S any] struct {
	impl     Impl[S]
	settings []setting
}

// port is the job's port, or the outcome a retired or unknown id
// answers. A retired id answers port_retired with the record's reason,
// so the Controller writes the stop as for any cause.
func (d driver) port(s *searching.Site) (Port, error) {
	if p, ok := d.r.byID[ID(s.Port)]; ok {
		s.Space(p.entry.RateLimit)
		return p, nil
	}
	if x, ok := d.r.retired[ID(s.Port)]; ok {
		return Port{}, &searching.Stopped{Cause: searching.PortRetired, Text: x.Reason}
	}
	return Port{}, &searching.Failing{Err: fmt.Errorf("this Node carries no port %q", s.Port)}
}

func (d driver) Search(ctx context.Context, s *searching.Site, q searching.Query) ([]searching.Release, error) {
	p, err := d.port(s)
	if err != nil {
		return nil, err
	}
	return p.run.search(ctx, s, p.entry, q)
}

func (d driver) Recent(ctx context.Context, s *searching.Site) ([]searching.Release, error) {
	p, err := d.port(s)
	if err != nil {
		return nil, err
	}
	return p.run.recent(ctx, s, p.entry)
}

func (d driver) Fetch(ctx context.Context, s *searching.Site, link searching.Link) (searching.Got, error) {
	p, err := d.port(s)
	if err != nil {
		return nil, err
	}
	return p.run.fetch(ctx, s, p.entry, link)
}

// Test signs in when the port holds a session, then runs Recent, or a
// Basic search for "test" when the site lists no new uploads. Any
// well-formed answer passes, zero items included. A CodeLogin reads the
// code the Admin typed from s.Code.
func (d driver) Test(ctx context.Context, s *searching.Site, _ *searching.CaptchaAnswer) (searching.Report, error) {
	p, err := d.port(s)
	if err != nil {
		return searching.Report{}, err
	}
	return p.run.test(ctx, s, p.entry)
}

// Captcha refuses: no port has a captcha step.
func (driver) Captcha(context.Context, *searching.Site) (searching.Captcha, error) {
	return searching.Captcha{}, &searching.Failing{Err: errors.New("a ported site has no captcha step")}
}

// search is HttpIndexerBase.Fetch for one Query.
func (t *typed[S]) search(context.Context, *searching.Site, Entry, searching.Query) ([]searching.Release, error) {
	panic("port: search is not implemented")
}

func (t *typed[S]) recent(context.Context, *searching.Site, Entry) ([]searching.Release, error) {
	panic("port: recent is not implemented")
}

func (t *typed[S]) fetch(context.Context, *searching.Site, Entry, searching.Link) (searching.Got, error) {
	panic("port: fetch is not implemented")
}

func (t *typed[S]) test(context.Context, *searching.Site, Entry) (searching.Report, error) {
	panic("port: test is not implemented")
}
