// Package targets owns Targets, the encoding goals a file meets, and
// Profiles, the ordered Targets a Content Type's Works want.
package targets

import (
	"context"
	"embed"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("targets", schemaFiles) }

type (
	TargetID  = db.TargetID
	ProfileID = db.ProfileID
)

// Target is a named encoding goal. A file meets it when its resolution is
// in Resolutions, and an earlier one ranks higher.
type Target struct {
	ID          TargetID
	Name        string
	Resolutions []release.Resolution
	CreatedAt   storage.Time
}

// Profile is the Targets a Content Type's Works want, in rank order. The
// first Profile made for a Content Type is its default.
type Profile struct {
	ID          ProfileID
	Name        string
	ContentType works.TypeID
	Default     bool
	Targets     []TargetID
}

type Changed struct{}

var ChangedEvent = live.AdminEvent[Changed]("targets.changed")

type Publisher interface {
	Publish(ctx context.Context, v Changed)
}

var (
	ErrTargetNameTaken  = errors.New("a Target already has that name")
	ErrProfileNameTaken = errors.New("a Profile already has that name")
	ErrNoType           = errors.New("no Content Type has that id")
	ErrNoTarget         = errors.New("no Target has that id")
)

type Config struct {
	Library *storage.DB[file]
	// Types is every Content Type this binary registers.
	Types   []works.TypeID
	Changed Publisher
}

type Targets struct{ cfg Config }

func New(cfg Config) *Targets { return &Targets{cfg: cfg} }

func (t *Targets) CreateTarget(ctx context.Context, _ identity.Admin, name string, resolutions []release.Resolution) (Target, error) {
	target := Target{ID: TargetID{UUID: storage.NewUUID()}, Name: name, Resolutions: resolutions, CreatedAt: storage.Now()}
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		taken, err := q.TargetNameTaken(ctx, name)
		switch {
		case err != nil:
			return err
		case taken:
			return ErrTargetNameTaken
		}
		if err := q.InsertTarget(ctx, db.InsertTargetParams{ID: target.ID, Name: name, CreatedAt: target.CreatedAt}); err != nil {
			return err
		}
		for i, r := range resolutions {
			if err := q.InsertResolution(ctx, db.InsertResolutionParams{TargetID: target.ID, Position: int64(i + 1), Resolution: r}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Target{}, err
	}
	t.cfg.Changed.Publish(ctx, Changed{})
	return target, nil
}

// ListTargets lists every Target by name.
func (t *Targets) ListTargets(ctx context.Context, _ identity.Admin) ([]Target, error) {
	var out []Target
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		rows, err := q.ListTargets(ctx)
		if err != nil {
			return err
		}
		out, err = withResolutions(ctx, q, rows)
		return err
	})
	return out, err
}

func (t *Targets) CreateProfile(ctx context.Context, _ identity.Admin, name string, contentType works.TypeID, targets []TargetID) (Profile, error) {
	if !slices.Contains(t.cfg.Types, contentType) {
		return Profile{}, ErrNoType
	}
	p := Profile{ID: ProfileID{UUID: storage.NewUUID()}, Name: name, ContentType: contentType, Targets: targets}
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		taken, err := q.ProfileNameTaken(ctx, name)
		switch {
		case err != nil:
			return err
		case taken:
			return ErrProfileNameTaken
		}
		for _, id := range targets {
			switch has, err := q.HasTarget(ctx, id); {
			case err != nil:
				return err
			case !has:
				return ErrNoTarget
			}
		}
		hasDefault, err := q.HasDefaultProfile(ctx, contentType)
		if err != nil {
			return err
		}
		p.Default = !hasDefault
		if err := q.InsertProfile(ctx, db.InsertProfileParams{ID: p.ID, Name: name, ContentType: contentType, IsDefault: p.Default}); err != nil {
			return err
		}
		for i, id := range targets {
			if err := q.InsertProfileTarget(ctx, db.InsertProfileTargetParams{ProfileID: p.ID, Position: int64(i + 1), TargetID: id}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Profile{}, err
	}
	t.cfg.Changed.Publish(ctx, Changed{})
	return p, nil
}

// ListProfiles lists every Profile by Content Type, then name.
func (t *Targets) ListProfiles(ctx context.Context, _ identity.Admin) ([]Profile, error) {
	var out []Profile
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		rows, err := q.ListProfiles(ctx)
		if err != nil {
			return err
		}
		members, err := q.ListProfileTargets(ctx)
		if err != nil {
			return err
		}
		held := map[ProfileID][]TargetID{}
		for _, m := range members {
			held[m.ProfileID] = append(held[m.ProfileID], m.TargetID)
		}
		out = make([]Profile, len(rows))
		for i, row := range rows {
			out[i] = Profile{ID: row.ID, Name: row.Name, ContentType: row.ContentType, Default: row.IsDefault, Targets: held[row.ID]}
		}
		return nil
	})
	return out, err
}

// Wanted lists the Targets of t's default Profile in rank order, none when
// t has no Profile.
func Wanted(ctx context.Context, r storage.Reader[file], t works.TypeID) ([]Target, error) {
	q := db.New(r)
	rows, err := q.ListDefaultTargets(ctx, t)
	if err != nil {
		return nil, err
	}
	return withResolutions(ctx, q, rows)
}

func withResolutions(ctx context.Context, q *db.Queries, rows []db.TargetsTarget) ([]Target, error) {
	all, err := q.ListResolutions(ctx)
	if err != nil {
		return nil, err
	}
	of := map[TargetID][]release.Resolution{}
	for _, r := range all {
		of[r.TargetID] = append(of[r.TargetID], r.Resolution)
	}
	out := make([]Target, len(rows))
	for i, row := range rows {
		out[i] = Target{ID: row.ID, Name: row.Name, Resolutions: of[row.ID], CreatedAt: row.CreatedAt}
	}
	return out, nil
}
