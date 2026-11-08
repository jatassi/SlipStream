package targets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/cases"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
)

var (
	ErrFormatNameTaken = errors.New("another Custom Format has that name")
	ErrNoFormat        = errors.New("no Custom Format has that id")
)

// StoredFormat is a Custom Format as the library keeps it.
type StoredFormat struct {
	formats.Format
	// Revision counts its saves, from 1.
	Revision int
	// Timeouts counts the Releases one of its patterns ran past the limit
	// on since its last save.
	Timeouts int
	// ScoredBy is every Target that scores it, by name.
	ScoredBy  []TargetRef
	CreatedAt storage.Time
}

type TargetRef struct {
	ID   TargetID
	Name string
}

// NewFormatID mints a Custom Format's id.
func NewFormatID() formats.ID { return formats.ID(storage.NewUUID().Bytes()) }

// ParseFormatID reads a Custom Format's id from its UUID text.
func ParseFormatID(s string) (formats.ID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return formats.ID{}, err
	}
	return formats.ID(u.Bytes()), nil
}

func byFormatID(a, b formats.ID) int { return bytes.Compare(a[:], b[:]) }

func formatID(id db.FormatID) formats.ID { return formats.ID(id.Bytes()) }

func rowID(id formats.ID) db.FormatID {
	u, _ := storage.UUIDFromBytes(id[:])
	return db.FormatID{UUID: u}
}

func rowIDs(ids []formats.ID) []db.FormatID {
	out := make([]db.FormatID, len(ids))
	for i, id := range ids {
		out[i] = rowID(id)
	}
	return out
}

// condition is a formats.Condition in upstream's export form, as the
// conditions column holds it.
type condition struct {
	Name           string          `json:"name"`
	Implementation formats.Kind    `json:"implementation"`
	Negate         bool            `json:"negate"`
	Required       bool            `json:"required"`
	Fields         json.RawMessage `json:"fields,omitempty"`
}

func conditionsJSON(cs []formats.Condition) string {
	out := make([]condition, len(cs))
	for i, c := range cs {
		out[i] = condition{Name: c.Name, Implementation: c.Kind, Negate: c.Negate, Required: c.Required, Fields: c.Fields}
	}
	var b strings.Builder
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(out); err != nil {
		panic(err)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func storedOf(row db.TargetsFormat) (StoredFormat, error) {
	var cs []condition
	if err := json.Unmarshal([]byte(row.Conditions), &cs); err != nil {
		return StoredFormat{}, fmt.Errorf("custom format %s: %w", row.ID, err)
	}
	f := formats.Format{
		ID: formatID(row.ID), Name: row.Name, IncludeWhenRenaming: row.IncludeWhenRenaming, Dialect: row.Dialect,
		Conditions: make([]formats.Condition, len(cs)), Source: row.Source,
	}
	for i, c := range cs {
		f.Conditions[i] = formats.Condition{Name: c.Name, Kind: c.Implementation, Negate: c.Negate, Required: c.Required, Fields: c.Fields}
	}
	return StoredFormat{Format: f, Revision: int(row.Revision), Timeouts: int(row.Timeouts), ScoredBy: []TargetRef{}, CreatedAt: row.CreatedAt}, nil
}

// saveFormat checks f and writes it: a new format, or with replace over
// the format of f's id, ErrNoFormat when there is none.
func saveFormat(ctx context.Context, q *db.Queries, f formats.Format, replace bool) error {
	// The limit bounds a match, and a compile never runs one.
	if err := formats.Check(f, formats.Timings{}); err != nil {
		return err
	}
	id, folded := rowID(f.ID), cases.Fold().String(f.Name)
	switch taken, err := q.FormatNameTaken(ctx, db.FormatNameTakenParams{NameFolded: folded, ID: id}); {
	case err != nil:
		return err
	case taken:
		return fmt.Errorf("%w: \"%s\"", ErrFormatNameTaken, f.Name)
	}
	conditions := conditionsJSON(f.Conditions)
	if !replace {
		return q.InsertFormat(ctx, db.InsertFormatParams{
			ID: id, Name: f.Name, NameFolded: folded, IncludeWhenRenaming: f.IncludeWhenRenaming, Dialect: f.Dialect,
			Conditions: conditions, Source: f.Source, CreatedAt: storage.Now(),
		})
	}
	switch n, err := q.ReplaceFormat(ctx, db.ReplaceFormatParams{
		ID: id, Name: f.Name, NameFolded: folded, IncludeWhenRenaming: f.IncludeWhenRenaming, Dialect: f.Dialect,
		Conditions: conditions, Source: f.Source,
	}); {
	case err != nil:
		return err
	case n == 0:
		return ErrNoFormat
	}
	return nil
}

// formatIn reads format id, ErrNoFormat when none has it.
func formatIn(ctx context.Context, q *db.Queries, id formats.ID) (StoredFormat, error) {
	row, err := q.GetFormat(ctx, rowID(id))
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return StoredFormat{}, ErrNoFormat
	case err != nil:
		return StoredFormat{}, err
	}
	return storedOf(row)
}

// listFormats reads every format by name, with the Targets scoring each.
func listFormats(ctx context.Context, q *db.Queries) ([]StoredFormat, error) {
	rows, err := q.ListFormats(ctx)
	if err != nil {
		return nil, err
	}
	scoring, err := q.ListScoringTargets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]StoredFormat, len(rows))
	at := map[db.FormatID]*StoredFormat{}
	for i, row := range rows {
		if out[i], err = storedOf(row); err != nil {
			return nil, err
		}
		at[row.ID] = &out[i]
	}
	for _, s := range scoring {
		f := at[s.FormatID]
		f.ScoredBy = append(f.ScoredBy, TargetRef{ID: s.ID, Name: s.Name})
	}
	return out, nil
}

// ListFormats lists every Custom Format by name.
func (t *Targets) ListFormats(ctx context.Context, _ identity.Admin) ([]StoredFormat, error) {
	var out []StoredFormat
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		out, err = listFormats(ctx, db.New(r))
		return err
	})
	return out, err
}

// GetFormat reads Custom Format id, ErrNoFormat when none has it.
func (t *Targets) GetFormat(ctx context.Context, by identity.Admin, id formats.ID) (StoredFormat, error) {
	all, err := t.ListFormats(ctx, by)
	if err != nil {
		return StoredFormat{}, err
	}
	i := slices.IndexFunc(all, func(f StoredFormat) bool { return f.ID == id })
	if i < 0 {
		return StoredFormat{}, ErrNoFormat
	}
	return all[i], nil
}

// ScoredIn is every format some Target of goals scores.
func ScoredIn(goals Goals) formats.Set {
	var out formats.Set
	for _, t := range goals.Targets {
		out = slices.AppendSeq(out, maps.Keys(t.Scores.Formats))
	}
	slices.SortFunc(out, byFormatID)
	return slices.Compact(out)
}

// MatchLimit is how long one condition's pattern may run against one
// name before it counts as not matched.
func (t *Targets) MatchLimit() time.Duration { return t.cfg.Formats.Match }

// catalog is the last compiled catalog and the formats' revisions it was
// compiled from.
type catalog struct {
	mu        sync.Mutex
	revisions []db.ListFormatRevisionsRow
	compiled  *formats.Catalog
}

// FormatsIn is every Custom Format in r, compiled with the Match limit.
// A catalog is compiled once per set of format revisions and shared until
// a save changes one.
func (t *Targets) FormatsIn(ctx context.Context, r storage.Reader[file]) (*formats.Catalog, error) {
	q := db.New(r)
	revisions, err := q.ListFormatRevisions(ctx)
	if err != nil {
		return nil, err
	}
	c := &t.catalog
	c.mu.Lock()
	compiled, hit := c.compiled, c.compiled != nil && slices.Equal(c.revisions, revisions)
	c.mu.Unlock()
	if hit {
		return compiled, nil
	}
	rows, err := q.ListFormats(ctx)
	if err != nil {
		return nil, err
	}
	fs := make([]formats.Format, len(rows))
	for i, row := range rows {
		s, err := storedOf(row)
		if err != nil {
			return nil, err
		}
		fs[i] = s.Format
	}
	compiled = formats.Compile(fs, t.cfg.Formats)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.revisions, c.compiled = revisions, compiled
	return compiled, nil
}

// AddTimeouts adds to each format's count the Releases whose match ran
// one of its patterns past the limit: one per Result naming it.
func AddTimeouts(ctx context.Context, tx *storage.Tx[file], rs []formats.Result) error {
	releases := map[formats.ID]int64{}
	for _, r := range rs {
		seen := map[formats.ID]bool{}
		for _, to := range r.Timeouts {
			if !seen[to.Format] {
				seen[to.Format] = true
				releases[to.Format]++
			}
		}
	}
	q := db.New(tx)
	for _, id := range slices.SortedFunc(maps.Keys(releases), byFormatID) {
		if err := q.AddFormatTimeouts(ctx, db.AddFormatTimeoutsParams{ID: rowID(id), Releases: releases[id]}); err != nil {
			return err
		}
	}
	return nil
}
