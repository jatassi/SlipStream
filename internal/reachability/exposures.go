package reachability

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/reachability/internal/db"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// Exposure is something the Admin wants usable away from home: the UI and
// portal, or one Media Server.
type Exposure struct {
	ID     judge.ExposureID
	Kind   judge.Kind
	Server link.ServerID
	// Public is a Media Server's public address. The UI and portal's is
	// Address, read on every run.
	Public   *url.URL
	WantedAt time.Time
}

const unitPrefix = "exposure:"

func (e Exposure) Unit() tasks.UnitKey { return tasks.UnitKey(unitPrefix + e.ID.String()) }

func exposureOfUnit(k tasks.UnitKey) (judge.ExposureID, error) {
	u, err := storage.ParseUUID(strings.TrimPrefix(string(k), unitPrefix))
	return judge.ExposureID{UUID: u}, err
}

var kindNames = [...]string{judge.UIPortal: "ui-portal", judge.MediaServer: "media-server"}

func KindName(k judge.Kind) string { return kindNames[k] }

func ParseKind(s string) (judge.Kind, bool) {
	i := slices.Index(kindNames[:], s)
	if i < 0 {
		return 0, false
	}
	return judge.Kind(i), true //nolint:gosec // G115: an index of kindNames fits
}

// Wanted is an Admin's ask to set something up for use away from home.
type Wanted struct {
	Kind   judge.Kind
	Server link.ServerID
	Public string
}

func parsePublic(s string) (*url.URL, error) {
	if strings.TrimSpace(s) == "" {
		return nil, ErrPublicRequired
	}
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return nil, ErrPublicNotURL
	}
	return u, nil
}

func exposureOf(row db.ReachabilityExposure) Exposure {
	k, _ := ParseKind(row.Kind)
	e := Exposure{ID: row.ID, Kind: k, Server: row.ServerID, WantedAt: row.WantedAt.Time()}
	if row.Public != nil {
		e.Public, _ = url.Parse(*row.Public)
	}
	return e
}

// wantedIn is every wanted Exposure, by id.
func wantedIn(ctx context.Context, rd storage.Reader[storage.Library]) ([]Exposure, error) {
	rows, err := db.New(rd).ListExposures(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Exposure, len(rows))
	for i, row := range rows {
		out[i] = exposureOf(row)
	}
	return out, nil
}

// digest is what a Media Server record change must touch to rerun its
// Exposure: its address, reaching and runs-on Nodes, kind and identity.
func digest(rec mediaservers.Record) []byte {
	h := sha256.New()
	for _, s := range []string{rec.Address, rec.Reaching.String(), rec.RunsOn.String(), string(rec.Kind), rec.Identity} {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	return h.Sum(nil)
}

// Named is an Exposure with the name the page shows it by.
type Named struct {
	Exposure
	Subject string
}

// Want sets w up for use away from home and starts its first check.
func (r *Reachability) Want(ctx context.Context, _ identity.Admin, w Wanted) (Named, error) {
	e := Exposure{ID: judge.ExposureID{UUID: storage.NewUUID()}, Kind: w.Kind, WantedAt: r.now()}
	if w.Kind == judge.MediaServer {
		u, err := parsePublic(w.Public)
		if err != nil {
			return Named{}, err
		}
		e.Server, e.Public = w.Server, u
	}
	subject := uiSubject
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		wanted, err := wantedIn(ctx, tx)
		if err != nil {
			return err
		}
		if slices.ContainsFunc(wanted, func(x Exposure) bool { return x.Kind == e.Kind && x.Server == e.Server }) {
			return ErrExists
		}
		p := db.InsertExposureParams{ID: e.ID, Kind: KindName(e.Kind), WantedAt: storage.FromTime(e.WantedAt)}
		if e.Kind == judge.UIPortal {
			if err := q.InsertExposure(ctx, p); err != nil {
				return err
			}
			return q.DeleteDismissalUI(ctx)
		}
		subject, err = insertServer(ctx, tx, e, p)
		return err
	})
	if err != nil {
		return Named{}, err
	}
	r.touch(ctx)
	r.changed(ctx, e.ID.UUID)
	r.start(ctx, e.Unit())
	return Named{Exposure: e, Subject: subject}, nil
}

// insertServer wants a Media Server Exposure with its record's digest and
// answers the server's name.
func insertServer(ctx context.Context, tx *storage.Tx[storage.Library], e Exposure, p db.InsertExposureParams) (string, error) {
	q := db.New(tx)
	servers, err := mediaservers.ServersIn(ctx, tx)
	if err != nil {
		return "", err
	}
	i := slices.IndexFunc(servers, func(s mediaservers.Record) bool { return s.ID == e.Server })
	if i < 0 {
		return "", ErrServerNotFound
	}
	p.ServerID, p.Public = e.Server, new(e.Public.String())
	if err := q.InsertExposure(ctx, p); err != nil {
		return "", err
	}
	if err := q.SetServerDigest(ctx, db.SetServerDigestParams{ID: e.ID, Digest: digest(servers[i])}); err != nil {
		return "", err
	}
	return servers[i].Name, q.DeleteDismissalServer(ctx, e.Server)
}

func subjectIn(ctx context.Context, rd storage.Reader[storage.Library], e Exposure) (string, error) {
	if e.Kind == judge.UIPortal {
		return uiSubject, nil
	}
	servers, err := mediaservers.ServersIn(ctx, rd)
	if err != nil {
		return "", err
	}
	i := slices.IndexFunc(servers, func(s mediaservers.Record) bool { return s.ID == e.Server })
	if i < 0 {
		return "", ErrServerNotFound
	}
	return servers[i].Name, nil
}

// SetPublic changes a Media Server Exposure's public address and checks
// it again.
func (r *Reachability) SetPublic(ctx context.Context, _ identity.Admin, id judge.ExposureID, public string) (Named, error) {
	u, err := parsePublic(public)
	if err != nil {
		return Named{}, err
	}
	var (
		e       Exposure
		subject string
	)
	err = r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		n, err := q.SetPublic(ctx, db.SetPublicParams{ID: id, Public: new(u.String())})
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrNotFound
		}
		row, err := q.GetExposure(ctx, id)
		if err != nil {
			return err
		}
		e = exposureOf(row)
		subject, err = subjectIn(ctx, tx, e)
		return err
	})
	if err != nil {
		return Named{}, err
	}
	r.changed(ctx, id.UUID)
	r.start(ctx, e.Unit())
	return Named{Exposure: e, Subject: subject}, nil
}

// Unwant stops checking an Exposure and forgets its result.
func (r *Reachability) Unwant(ctx context.Context, _ identity.Admin, id judge.ExposureID) error {
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		n, err := db.New(tx).DeleteExposure(ctx, id)
		if err == nil && n == 0 {
			return ErrNotFound
		}
		return err
	})
	if err != nil {
		return err
	}
	r.touch(ctx)
	r.changed(ctx, id.UUID)
	return nil
}

// Dismiss hides a suggestion: the UI and portal's when server is zero,
// else that Media Server's.
func (r *Reachability) Dismiss(ctx context.Context, _ identity.Admin, server link.ServerID) error {
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		at := storage.FromTime(r.now())
		if server.IsZero() {
			return q.DismissUI(ctx, at)
		}
		servers, err := mediaservers.ServersIn(ctx, tx)
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(servers, func(s mediaservers.Record) bool { return s.ID == server }) {
			return ErrServerNotFound
		}
		return q.DismissServer(ctx, db.DismissServerParams{ServerID: server, At: at})
	})
	if err != nil {
		return err
	}
	r.changed(ctx)
	return nil
}

// start queues units after the Write that wanted them committed, so the
// task lists them. A Start that finds the task gone lost a race with the
// last Exposure's removal: nothing is left to check.
func (r *Reachability) start(ctx context.Context, units ...tasks.UnitKey) {
	if len(units) == 0 {
		return
	}
	_, err := r.cfg.Tasks.Start(ctx, tasks.Start{Task: TaskID, Units: units, Trigger: tasks.ByEvent})
	switch {
	case errors.Is(err, tasks.ErrUnknownTask):
		r.cfg.Log.DebugContext(ctx, "the reachability check lost its last route before it started")
	case err != nil:
		r.cfg.Log.WarnContext(ctx, "start the reachability check", "err", err)
	}
}
