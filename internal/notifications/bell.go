package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Bell is the bell's own events and warnings switch. It changes only what
// later releases show.
type Bell struct {
	// Events are the kinds it shows, in catalog order.
	Events   []entry.Kind
	Warnings bool
}

// bellRow is the stored bell: the Admin's choices over each kind's
// BellByDefault, so a later kind needs no migration.
type bellRow struct {
	choices  map[entry.Kind]bool
	warnings bool
}

func (b bellRow) has(k entry.Kind, c *catalog) bool {
	if on, ok := b.choices[k]; ok {
		return on
	}
	return c.entry(k).Declared().BellByDefault
}

func (s *Service) bellIn(ctx context.Context, q *db.Queries) (bellRow, error) {
	row, err := q.GetBell(ctx)
	if err != nil {
		return bellRow{}, err
	}
	b := bellRow{warnings: row.Warnings}
	if err := json.Unmarshal([]byte(row.Choices), &b.choices); err != nil {
		return bellRow{}, fmt.Errorf("decode the bell's events: %w", err)
	}
	return b, nil
}

func (s *Service) bellOf(b bellRow) Bell {
	out := Bell{Events: []entry.Kind{}, Warnings: b.warnings}
	for _, spec := range s.Events() {
		if !spec.Fixed && b.has(spec.Kind, s.catalog) {
			out.Events = append(out.Events, spec.Kind)
		}
	}
	return out
}

func (s *Service) Bell(ctx context.Context) (Bell, error) {
	var b bellRow
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		b, err = s.bellIn(ctx, db.New(r))
		return err
	})
	return s.bellOf(b), err
}

// SetBell stores the kinds the bell shows and its warnings switch. A nil
// events or warnings keeps what is stored. A kind outside the catalog, or
// Test, is ErrUnknownEvent.
func (s *Service) SetBell(ctx context.Context, events []entry.Kind, warnings *bool) (Bell, error) {
	if err := s.checkEvents(events); err != nil {
		return Bell{}, err
	}
	var b bellRow
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		var err error
		if b, err = s.bellIn(ctx, q); err != nil {
			return err
		}
		if events != nil {
			b.choices = map[entry.Kind]bool{}
			for _, spec := range s.Events() {
				if !spec.Fixed {
					b.choices[spec.Kind] = slices.Contains(events, spec.Kind)
				}
			}
		}
		if warnings != nil {
			b.warnings = *warnings
		}
		return q.SetBell(ctx, db.SetBellParams{Choices: *jsonText(b.choices), Warnings: b.warnings})
	})
	return s.bellOf(b), err
}

// UnknownEventError is a kind no catalog row has, or one that can't be
// chosen.
type UnknownEventError struct{ Kind entry.Kind }

func (e *UnknownEventError) Error() string {
	return fmt.Sprintf("notifications: %s isn't an event that can be chosen", e.Kind)
}

func (s *Service) checkEvents(events []entry.Kind) error {
	specs := s.Events()
	for _, k := range events {
		if !slices.ContainsFunc(specs, func(spec entry.Spec) bool { return spec.Kind == k && !spec.Fixed }) {
			return &UnknownEventError{Kind: k}
		}
	}
	return nil
}

// entry is k's row. Every stored kind has one: Recorder refuses a kind its
// owner didn't register.
func (c *catalog) entry(k entry.Kind) entry.Entry {
	for _, e := range c.all() {
		if e.Declared().Kind == k {
			return e
		}
	}
	panic(fmt.Sprintf("notifications: %s isn't in the catalog", k))
}
