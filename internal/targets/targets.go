// Package targets owns Targets, the encoding goals a file meets, and
// Profiles, the ordered Targets a Content Type's Works want. Its pure
// Judge answers "does this meet that Target, and how well" for every
// caller.
package targets

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
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

// InUseError is a Target that Profiles hold, By their names.
type InUseError struct{ By []string }

func (e *InUseError) Error() string {
	if len(e.By) == 1 {
		return e.By[0] + " uses it"
	}
	return strings.Join(e.By[:len(e.By)-1], ", ") + " and " + e.By[len(e.By)-1] + " use it"
}

type Config struct {
	Library *storage.DB[file]
	// Types is every Content Type this binary registers.
	Types   []works.TypeID
	Changed Publisher
}

type Targets struct{ cfg Config }

func New(cfg Config) *Targets { return &Targets{cfg: cfg} }

// CreateTarget checks s and creates its Target.
func (t *Targets) CreateTarget(ctx context.Context, _ identity.Admin, s Spec) (Target, error) {
	target, err := NewTarget(s)
	if err != nil {
		return Target{}, err
	}
	target.ID, target.CreatedAt = TargetID{UUID: storage.NewUUID()}, storage.Now()
	err = t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return insertTarget(ctx, db.New(tx), target)
	})
	if err != nil {
		return Target{}, err
	}
	t.cfg.Changed.Publish(ctx, Changed{})
	return target, nil
}

func insertTarget(ctx context.Context, q *db.Queries, t Target) error {
	switch taken, err := q.TargetNameTaken(ctx, t.Name); {
	case err != nil:
		return err
	case taken:
		return ErrTargetNameTaken
	}
	if err := q.InsertTarget(ctx, db.InsertTargetParams{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt, CutoffTier: int64(t.Cutoff)}); err != nil {
		return err
	}
	position := int64(0)
	for i, tier := range t.Tiers {
		for _, a := range tier {
			err := q.InsertQuality(ctx, db.InsertQualityParams{
				TargetID: t.ID, Quality: a.Quality, Tier: int64(i), Position: position, MinMibPerMinute: a.Size.Min, MaxMibPerMinute: a.Size.Max,
			})
			if err != nil {
				return err
			}
			position++
		}
	}
	for _, a := range Attributes() {
		mode, values := t.Filters.Of(a)
		if mode == Any {
			continue
		}
		if err := q.InsertFilter(ctx, db.InsertFilterParams{TargetID: t.ID, Attribute: a, Mode: mode}); err != nil {
			return err
		}
		for i, v := range values {
			if err := q.InsertFilterValue(ctx, db.InsertFilterValueParams{TargetID: t.ID, Attribute: a, Position: int64(i), Value: v}); err != nil {
				return err
			}
		}
	}
	return nil
}

// ListTargets lists every Target by name.
func (t *Targets) ListTargets(ctx context.Context, _ identity.Admin) ([]Target, error) {
	var out []Target
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		out, err = listTargets(ctx, db.New(r))
		return err
	})
	return out, err
}

// GetTarget reads Target id, ErrNoTarget when none has it.
func (t *Targets) GetTarget(ctx context.Context, by identity.Admin, id TargetID) (Target, error) {
	all, err := t.ListTargets(ctx, by)
	if err != nil {
		return Target{}, err
	}
	i := slices.IndexFunc(all, func(t Target) bool { return t.ID == id })
	if i < 0 {
		return Target{}, ErrNoTarget
	}
	return all[i], nil
}

// DuplicateTarget copies id's rules under the first free "<name> copy",
// then "<name> copy 2" and on.
func (t *Targets) DuplicateTarget(ctx context.Context, _ identity.Admin, id TargetID) (Target, error) {
	var copied Target
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		all, err := listTargets(ctx, q)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(all, func(t Target) bool { return t.ID == id })
		if i < 0 {
			return ErrNoTarget
		}
		copied = Target{
			ID: TargetID{UUID: storage.NewUUID()}, Tiers: all[i].Tiers, Cutoff: all[i].Cutoff, Filters: all[i].Filters, CreatedAt: storage.Now(),
		}
		for n := 1; copied.Name == "" || slices.ContainsFunc(all, func(t Target) bool { return t.Name == copied.Name }); n++ {
			copied.Name = all[i].Name + " copy"
			if n > 1 {
				copied.Name += fmt.Sprintf(" %d", n)
			}
		}
		if err := insertTarget(ctx, q, copied); err != nil {
			return err
		}
		copied.Contains = containedBy(copied, append(all, copied))
		return nil
	})
	if err != nil {
		return Target{}, err
	}
	t.cfg.Changed.Publish(ctx, Changed{})
	return copied, nil
}

// DeleteTarget deletes Target id, or refuses with *InUseError naming each
// Profile that holds it. A Target no Profile holds fills no cell, and its
// Grabs' history goes with it.
func (t *Targets) DeleteTarget(ctx context.Context, _ identity.Admin, id TargetID) error {
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		holders, err := q.ListHolders(ctx)
		if err != nil {
			return err
		}
		var by []string
		for _, h := range holders {
			if h.TargetID == id {
				by = append(by, h.Name)
			}
		}
		if len(by) > 0 {
			return &InUseError{By: by}
		}
		switch n, err := q.DeleteTarget(ctx, id); {
		case err != nil:
			return err
		case n == 0:
			return ErrNoTarget
		}
		return nil
	})
	if err != nil {
		return err
	}
	t.cfg.Changed.Publish(ctx, Changed{})
	return nil
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
	all, err := listTargets(ctx, q)
	if err != nil {
		return nil, err
	}
	out := make([]Target, len(rows))
	for i, row := range rows {
		out[i] = all[slices.IndexFunc(all, func(t Target) bool { return t.ID == row.ID })]
	}
	return out, nil
}

// listTargets reads every Target with its rules, holders and containment,
// by name.
func listTargets(ctx context.Context, q *db.Queries) ([]Target, error) {
	rows, err := q.ListTargets(ctx)
	if err != nil {
		return nil, err
	}
	qualities, err := q.ListQualities(ctx)
	if err != nil {
		return nil, err
	}
	filters, err := q.ListFilters(ctx)
	if err != nil {
		return nil, err
	}
	values, err := q.ListFilterValues(ctx)
	if err != nil {
		return nil, err
	}
	holders, err := q.ListHolders(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Target, len(rows))
	at := map[TargetID]*Target{}
	for i, row := range rows {
		out[i] = Target{ID: row.ID, Name: row.Name, Cutoff: int(row.CutoffTier), CreatedAt: row.CreatedAt}
		at[row.ID] = &out[i]
	}
	for _, row := range qualities {
		t := at[row.TargetID]
		for int(row.Tier) >= len(t.Tiers) {
			t.Tiers = append(t.Tiers, nil)
		}
		t.Tiers[row.Tier] = append(t.Tiers[row.Tier], Accepted{Quality: row.Quality, Size: Size{Min: row.MinMibPerMinute, Max: row.MaxMibPerMinute}})
	}
	named := map[TargetID]map[Attribute][]string{}
	for _, v := range values {
		if named[v.TargetID] == nil {
			named[v.TargetID] = map[Attribute][]string{}
		}
		named[v.TargetID][v.Attribute] = append(named[v.TargetID][v.Attribute], v.Value)
	}
	for _, f := range filters {
		if err := at[f.TargetID].Filters.Set(f.Attribute, f.Mode, named[f.TargetID][f.Attribute]); err != nil {
			return nil, fmt.Errorf("target %s: %w", f.TargetID, err)
		}
	}
	for _, h := range holders {
		t := at[h.TargetID]
		t.Profiles = append(t.Profiles, ProfileRef{ID: h.ID, Name: h.Name})
	}
	for i := range out {
		out[i].Contains = containedBy(out[i], out)
	}
	return out, nil
}

// containedBy is the id of every Target in all but t that t contains.
func containedBy(t Target, all []Target) []TargetID {
	var out []TargetID
	for _, o := range all {
		if o.ID != t.ID && Contains(t, o) {
			out = append(out, o.ID)
		}
	}
	return out
}
