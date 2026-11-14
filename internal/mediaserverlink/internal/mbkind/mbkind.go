// Package mbkind is the Jellyfin and Emby kinds: one family of steps over
// the one wire client, with the few places the products differ held as
// data on the family. Jellyfin drops pending path reports when a full scan
// starts, so its reports wait out a running scan and settle one monitor
// delay later. Emby takes a Deleted report at once.
package mbkind

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/jellyfin"
)

// family is one product of the Jellyfin and Emby family.
type family struct {
	about  link.About
	flavor jellyfin.Flavor
	// scanDropsReports is Jellyfin's: a full scan or a folder change drops
	// the reports still inside their monitor delay.
	scanDropsReports bool
	// deletesAtOnce is Emby's: a Deleted report removes its item without
	// waiting out the monitor delay (#101).
	deletesAtOnce bool
}

var (
	Jellyfin link.Kind = family{
		about:            link.About{Name: "jellyfin", Label: "Jellyfin", Form: link.SignInKeyOrAdmin, Browser: true, Floor: link.MustVersion("12.2")},
		flavor:           jellyfin.Jellyfin,
		scanDropsReports: true,
	}
	Emby link.Kind = family{
		about:         link.About{Name: "emby", Label: "Emby", Form: link.SignInKeyOrAdmin, Browser: true, Floor: link.MustVersion("4.10")},
		flavor:        jellyfin.Emby,
		deletesAtOnce: true,
	}
)

func (f family) About() link.About { return f.about }

// keyName is the name of the API key SignIn creates.
const keyName = "SlipStream"

func (f family) Dial(d link.Dialed) link.Conn {
	return &conn{f: f, d: d, c: f.client(d, d.Credential.APIKey)}
}

func (f family) client(d link.Dialed, token string) *jellyfin.Client {
	base, err := url.Parse(d.Address)
	if err != nil {
		base = &url.URL{Host: d.Address}
	}
	app := jellyfin.App{Client: keyName, Device: keyName, DeviceID: d.DeviceID, Version: "2"}
	return jellyfin.New(base, f.flavor, token, app, d.HTTP)
}

// conn is one Jellyfin or Emby server as one Node reaches it.
type conn struct {
	f family
	d link.Dialed
	c *jellyfin.Client
}

// problem is err in the link's words: a server's own failure as a
// *link.ServerError, and the caller's canceled or expired context as it
// is, the link's.
func problem(ctx context.Context, err error) error {
	if err == nil || ctx.Err() != nil {
		return err
	}
	if _, ok := errors.AsType[*link.ServerError](err); ok {
		return err
	}
	if e, ok := errors.AsType[*jellyfin.Error](err); ok {
		p := link.Problem{Reason: link.ReasonFailed, Status: e.Status, Detail: e.Detail}
		switch {
		case jellyfin.Refused(err) || e.Status == 403:
			p.Reason = link.ReasonRefused
		case jellyfin.NotFound(err):
			p.Reason = link.ReasonUnsupported
		}
		return &link.ServerError{Problem: p}
	}
	if _, ok := errors.AsType[*url.Error](err); ok {
		return &link.ServerError{Problem: link.Problem{Reason: link.ReasonUnreachable, Detail: err.Error()}}
	}
	return &link.ServerError{Problem: link.Problem{Reason: link.ReasonFailed, Detail: err.Error()}}
}

func failed(detail string) error {
	return &link.ServerError{Problem: link.Problem{Reason: link.ReasonFailed, Detail: detail}}
}

func (c *conn) Reach(ctx context.Context) (link.Identity, error) {
	id, err := c.identity(ctx)
	return id, problem(ctx, err)
}

func (c *conn) identity(ctx context.Context) (link.Identity, error) {
	info, err := c.c.PublicInfo(ctx)
	if err != nil {
		return link.Identity{}, err
	}
	v, err := link.ParseVersion(info.Version)
	if err != nil {
		return link.Identity{}, failed(err.Error())
	}
	return link.Identity{ID: info.ID, Name: info.ServerName, Version: v}, nil
}

// same is the identity at the address when it is the saved server's, and
// ReasonOtherServer when another server responds there. A draft has no
// saved identity, so any server is its own.
func (c *conn) same(ctx context.Context) (link.Identity, error) {
	id, err := c.identity(ctx)
	if err != nil {
		return id, err
	}
	if want := c.d.Server.Identity; want != "" && id.ID != want {
		return id, &link.ServerError{Problem: link.Problem{Reason: link.ReasonOtherServer, Found: id.ID}}
	}
	return id, nil
}

func (c *conn) Check(ctx context.Context) (link.Checked, error) {
	out, err := c.check(ctx)
	return out, problem(ctx, err)
}

func (c *conn) check(ctx context.Context) (link.Checked, error) {
	id, err := c.same(ctx)
	if err != nil {
		return link.Checked{}, err
	}
	cfg, err := c.c.Configuration(ctx)
	if err != nil {
		return link.Checked{}, err
	}
	libs, err := c.libraries(ctx)
	if err != nil {
		return link.Checked{}, err
	}
	return link.Checked{Identity: id, MonitorDelay: cfg.MonitorDelay, Libraries: libs}, nil
}

func (c *conn) libraries(ctx context.Context) ([]link.Library, error) {
	vfs, err := c.c.Libraries(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]link.Library, len(vfs))
	for i, vf := range vfs {
		out[i] = link.Library{Key: vf.ItemID, Name: vf.Name}
		for _, l := range vf.Locations {
			p, err := link.ParsePath(l)
			if err != nil {
				return nil, failed(err.Error())
			}
			out[i].Folders = append(out[i].Folders, p)
		}
	}
	return out, nil
}

// SignIn signs in as the administrator with the server's DeviceId,
// creates the SlipStream key, reads it back from the key list, since
// Jellyfin's create answers no body, and signs that session out.
func (c *conn) SignIn(ctx context.Context, user, password string) (link.Credential, error) {
	cred, err := c.signIn(ctx, user, password)
	return cred, problem(ctx, err)
}

func (c *conn) signIn(ctx context.Context, user, password string) (link.Credential, error) {
	auth, err := c.c.AuthenticateByName(ctx, user, password)
	if err != nil {
		return link.Credential{}, err
	}
	admin := c.f.client(c.d, auth.AccessToken)
	defer func() { _ = admin.Logout(context.WithoutCancel(ctx)) }()
	if !auth.User.Policy.IsAdministrator {
		return link.Credential{}, &link.ServerError{Problem: link.Problem{Reason: link.ReasonRefused, Detail: user + " isn't an administrator"}}
	}
	if err := admin.CreateKey(ctx, keyName); err != nil {
		return link.Credential{}, err
	}
	keys, err := admin.Keys(ctx)
	if err != nil {
		return link.Credential{}, err
	}
	var newest jellyfin.Key
	for _, k := range keys {
		if k.AppName == keyName && !k.DateCreated.Before(newest.DateCreated) {
			newest = k
		}
	}
	if newest.AccessToken == "" {
		return link.Credential{}, failed("the key list has no " + keyName + " key after creating one")
	}
	return link.Credential{APIKey: newest.AccessToken}, nil
}

// Watch polls the server's sessions every every until ctx ends.
func (c *conn) Watch(ctx context.Context, every time.Duration, seen func(link.Seen)) error {
	if _, err := c.same(ctx); err != nil {
		return problem(ctx, err)
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		s := link.Seen{At: time.Now(), Polled: true}
		sessions, err := c.sessions(ctx)
		if err := problem(ctx, err); err != nil {
			p, ok := link.ServerProblem(err)
			if !ok {
				return err
			}
			s.Problem = &p
		}
		s.Sessions = sessions
		seen(s)
		select {
		case <-ctx.Done():
			return fmt.Errorf("the watch of %s ended: %w", c.f.about.Label, ctx.Err())
		case <-tick.C:
		}
	}
}

func (c *conn) sessions(ctx context.Context) ([]link.Session, error) {
	all, err := c.c.Sessions(ctx)
	if err != nil {
		return nil, err
	}
	var out []link.Session
	for _, s := range all {
		if s.NowPlayingItem == nil || len(out) == c.d.Timings.SessionCap {
			continue
		}
		p, err := link.ParsePath(s.NowPlayingItem.Path)
		if err != nil {
			continue
		}
		ls := link.Session{Key: s.ID, Path: p, User: s.UserName, Player: s.DeviceName, Product: s.Client, Decision: link.DecisionDirectPlay}
		if ti := s.TranscodingInfo; ti != nil && s.PlayState != nil && s.PlayState.PlayMethod == jellyfin.Transcode {
			ls.Decision, ls.Reasons = link.DecisionTranscode, slices.Clone(ti.TranscodeReasons)
			if ti.IsVideoDirect {
				ls.Decision = link.DecisionCopy
			}
			ls.Hardware = ti.VideoEncoderIsHardware || (ti.HardwareAccelerationType != "" && ti.HardwareAccelerationType != "none")
		}
		out = append(out, ls)
	}
	return out, nil
}
