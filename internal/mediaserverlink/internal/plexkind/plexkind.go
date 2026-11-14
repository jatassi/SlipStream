// Package plexkind is the Plex kind: how one Plex Media Server learns
// about library changes. Plan groups a server's ready rows into folder
// refreshes, section refreshes and carries. Advance runs one step of a
// Job through the wire client in internal/providers/plex. Plex is never
// written to: a carry reports in lost what Plex didn't carry by itself
// (recorded run 2026-11-14, cases B to E).
package plexkind

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// Kind is Plex.
var Kind link.Kind = kind{}

type kind struct{}

func (kind) About() link.About {
	return link.About{Name: "plex", Label: "Plex", Form: link.SignInPlexPIN, Floor: link.MustVersion("1.43")}
}

func (kind) Dial(d link.Dialed) link.Conn {
	base, err := url.Parse(d.Address)
	return &conn{
		server: d.Server, base: err, t: d.Timings,
		c: plex.New(base, d.Credential.Token, plex.Headers{ClientID: d.DeviceID, Product: "SlipStream"}, d.HTTP),
	}
}

// conn is one Plex at one address. base is the address's parse error,
// answered by every call, since Dial does no IO and returns no error.
type conn struct {
	server link.ServerRef
	base   error
	c      *plex.Client
	t      link.Timings
}

// fail is err in SlipStream's words: a *link.ServerError when Plex itself
// failed the call or didn't respond, and err as it is when ctx ended,
// which is the link's.
func fail(ctx context.Context, err error) error {
	if err == nil || ctx.Err() != nil {
		return err
	}
	if pe, ok := errors.AsType[*plex.Error](err); ok {
		reason := link.ReasonFailed
		if plex.Refused(err) {
			reason = link.ReasonRefused
		}
		return &link.ServerError{Problem: link.Problem{Reason: reason, Status: pe.Status, Detail: pe.Detail}}
	}
	if _, ok := errors.AsType[*url.Error](err); ok {
		return &link.ServerError{Problem: link.Problem{Reason: link.ReasonUnreachable, Detail: err.Error()}}
	}
	return &link.ServerError{Problem: link.Problem{Reason: link.ReasonFailed, Detail: err.Error()}}
}

// identity is who answered, refused as another server when it isn't the
// one SlipStream saved.
func (c *conn) identity(machine, name, version string) (link.Identity, error) {
	if c.server.Identity != "" && machine != c.server.Identity {
		return link.Identity{}, &link.ServerError{Problem: link.Problem{Reason: link.ReasonOtherServer, Found: machine}}
	}
	v, err := link.ParseVersion(version)
	if err != nil {
		return link.Identity{}, &link.ServerError{Problem: link.Problem{Reason: link.ReasonFailed, Detail: err.Error()}}
	}
	return link.Identity{ID: machine, Name: name, Version: v}, nil
}

// Reach reads /identity, which Plex answers without a token (recorded run
// 2026-11-14).
func (c *conn) Reach(ctx context.Context) (link.Identity, error) {
	if c.base != nil {
		return link.Identity{}, unreachable(c.base)
	}
	id, err := c.c.Identity(ctx)
	if err != nil {
		return link.Identity{}, fail(ctx, err)
	}
	return c.identity(id.MachineIdentifier, "", id.Version)
}

func (c *conn) Check(ctx context.Context) (link.Checked, error) {
	if c.base != nil {
		return link.Checked{}, unreachable(c.base)
	}
	s, err := c.c.Server(ctx)
	if err != nil {
		return link.Checked{}, fail(ctx, err)
	}
	id, err := c.identity(s.MachineIdentifier, s.FriendlyName, s.Version)
	if err != nil {
		return link.Checked{}, err
	}
	libs, err := c.libraries(ctx)
	return link.Checked{Identity: id, Libraries: libs}, err
}

// SignIn is for Jellyfin and Emby. Plex signs in through plex.tv on the
// Controller.
func (c *conn) SignIn(context.Context, string, string) (link.Credential, error) {
	return link.Credential{}, &link.ServerError{Problem: link.Problem{Reason: link.ReasonUnsupported, Detail: "Plex signs in with a plex.tv account"}}
}

func unreachable(err error) error {
	return &link.ServerError{Problem: link.Problem{Reason: link.ReasonUnreachable, Detail: fmt.Sprintf("the address: %v", err)}}
}

// Advance runs one step of j.
func (c *conn) Advance(ctx context.Context, j link.Job) (link.Advanced, error) {
	if c.base != nil {
		return link.Advanced{}, unreachable(c.base)
	}
	var a link.Advanced
	var err error
	switch w := j.Work.(type) {
	case link.Refresh:
		a, err = run(ctx, c, refresh, j, w)
	case link.Carry:
		a, err = run(ctx, c, carry, j, w)
	case link.Rescan:
		a, err = run(ctx, c, rescan, j, w)
	case link.Fresh:
		a, err = c.fresh(ctx, j, w)
	case link.Lookup:
		a, err = c.lookup(ctx, j, w)
	case link.Index:
		a, err = c.index(ctx, j, w)
	case link.Libraries:
		var libs []link.Library
		libs, err = c.libraries(ctx)
		a = link.Advanced{Job: j, Done: true, Libraries: libs}
	default:
		return link.Advanced{}, fmt.Errorf("plex has no step for %T", j.Work)
	}
	if err != nil {
		return link.Advanced{}, fail(ctx, err)
	}
	return a, nil
}

func (c *conn) libraries(ctx context.Context) ([]link.Library, error) {
	sections, err := c.c.Sections(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]link.Library, len(sections))
	for i, s := range sections {
		out[i] = link.Library{Key: string(s.Key), Name: s.Title, Folders: make([]link.ServerPath, 0, len(s.Locations))}
		for _, l := range s.Locations {
			p, err := link.ParsePath(l.Path)
			if err != nil {
				return nil, fmt.Errorf("section %s: %w", s.Title, err)
			}
			out[i].Folders = append(out[i].Folders, p)
		}
	}
	return out, nil
}

// later is now plus d, the time a waiting step asks to be called again.
func later(d time.Duration) time.Time { return time.Now().Add(d) }
