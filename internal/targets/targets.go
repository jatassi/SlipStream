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
	ErrNoProfile        = errors.New("no Profile has that id")
	ErrProfileWrongType = errors.New("the Profile is for another Content Type")
	ErrTargetTwice      = errors.New("a Profile holds a Target twice")
	ErrProfileIsDefault = errors.New("the default Profile can't be deleted")
	ErrMovesToItself    = errors.New("a Profile's Works can't move to the Profile being deleted")
	ErrNoReplacement    = errors.New("no Profile has the id a deleted Profile's Works would move to")
)

// InUseError is a Target that Profiles hold, By their names, or a cut
// that lists or files hold, By the Profiles and a count of the rest.
type InUseError struct{ By []string }

func (e *InUseError) Error() string {
	if len(e.By) == 1 {
		return e.By[0] + " uses it"
	}
	return strings.Join(e.By[:len(e.By)-1], ", ") + " and " + e.By[len(e.By)-1] + " use it"
}

type Config struct {
	Library *storage.DB[file]
	// Works describes each Content Type, whose name its default Profile
	// takes.
	Works Describer
	// Types is every Content Type this binary registers, by its facet.
	Types   map[works.TypeID]Facet
	Changed Publisher
}

// Describer is works.Works as Open reads it.
type Describer interface {
	Describe(t works.TypeID) (works.Description, bool)
}

type Targets struct{ cfg Config }

// Publish sends targets.changed, for a Change files applied and committed.
func (t *Targets) Publish(ctx context.Context) { t.cfg.Changed.Publish(ctx, Changed{}) }

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
	return insertRules(ctx, q, t)
}

// insertRules writes t's qualities, sizes and filters.
func insertRules(ctx context.Context, q *db.Queries, t Target) error {
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
