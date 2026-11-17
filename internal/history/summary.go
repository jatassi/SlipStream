package history

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Sentence is History's opening sentence.
type Sentence struct {
	Words string
	// NeedsYou is the failed Grabs it leads with, 0 when it counts the
	// week instead.
	NeedsYou int
}

// Summary is History's opening sentence: the failed Grabs that need the
// Admin when needsYou counts any, else what each kind with a Tally added
// since since. Nil: History is empty and nothing needs the Admin.
func (s *Service) Summary(ctx context.Context, _ identity.Admin, since time.Time, needsYou int) (*Sentence, error) {
	if needsYou > 0 {
		return &Sentence{Words: needsYouWords(needsYou), NeedsYou: needsYou}, nil
	}
	var tallies []tally
	empty := false
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		cleared, err := q.GetCleared(ctx)
		if err != nil {
			return err
		}
		newest, err := q.NewestID(ctx)
		if errors.Is(err, storage.ErrNoRows) || err == nil && newest <= cleared {
			empty = true
			return nil
		}
		if err != nil {
			return err
		}
		floor, err := floorOf(ctx, r, since)
		if err != nil {
			return err
		}
		tallies, err = s.tallies(ctx, q, floor)
		return err
	})
	if err != nil || empty {
		return nil, err
	}
	return &Sentence{Words: weekWords(tallies)}, nil
}

// tally is how many entries one Tally noun's kinds added.
type tally struct {
	noun works.Noun
	n    int
}

// tallies counts each Tally noun's kinds from floor, in catalog order.
// Zero floor: nothing is that recent.
func (s *Service) tallies(ctx context.Context, q *db.Queries, floor EventID) ([]tally, error) {
	var out []tally
	for _, spec := range s.cfg.Catalog.Specs() {
		if spec.Tally == (works.Noun{}) {
			continue
		}
		i := slices.IndexFunc(out, func(t tally) bool { return t.noun == spec.Tally })
		if i < 0 {
			out, i = append(out, tally{noun: spec.Tally}), len(out)
		}
		if floor == 0 {
			continue
		}
		n, err := q.CountKindFrom(ctx, db.CountKindFromParams{Kind: spec.Kind, Floor: floor})
		if err != nil {
			return nil, err
		}
		out[i].n += int(n)
	}
	return out, nil
}

func needsYouWords(n int) string {
	if n == 1 {
		return "1 Grab failed and needs attention."
	}
	return entry.Grouped(n) + " Grabs failed and need attention."
}

// weekWords is "There were 14 Grabs and 31 Imports in the last 7 days.",
// the first count deciding was or were.
func weekWords(ts []tally) string {
	var counted, nouns []string
	verb := ""
	for _, t := range ts {
		nouns = append(nouns, t.noun.Many)
		if t.n == 0 {
			continue
		}
		if verb == "" {
			verb = "were"
			if t.n == 1 {
				verb = "was"
			}
		}
		counted = append(counted, entry.Count(t.n, t.noun))
	}
	if len(counted) == 0 {
		return "There were no " + orWords(nouns) + " in the last 7 days."
	}
	return "There " + verb + " " + works.JoinWords(counted) + " in the last 7 days."
}

// orWords is "a", "a or b", "a, b or c".
func orWords(ws []string) string {
	if len(ws) < 2 {
		return strings.Join(ws, "")
	}
	return strings.Join(ws[:len(ws)-1], ", ") + " or " + ws[len(ws)-1]
}
