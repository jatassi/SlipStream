package subtitles

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/works"
)

// siblingKey is the Tried before key of a sibling's Sidecar.
func siblingKey(id storage.UUID) string { return "sibling:" + id.String() }

// trySiblings times the want's Sidecars on the other files of its Item
// and Edition against this file's map, up to Timings.Siblings of them,
// newest first, skipping any tried before. The first that passes is the
// want's intent, as a sibling's copy. It spends no allowance.
func (a *attempt) trySiblings(ctx context.Context) (bool, error) {
	siblings, err := a.siblings(ctx)
	if err != nil {
		return false, err
	}
	var e ending
	for _, sib := range siblings {
		key := siblingKey(sib.ID)
		if a.skips("", key) {
			continue
		}
		b, err := a.s.keeper.bytesOf(ctx, sib.of, sib.at, sib.SHA256)
		if err != nil || b == nil {
			if err != nil {
				a.s.cfg.Log.WarnContext(ctx, "A sibling's subtitle wasn't read", "file", a.key.File, "sibling", sib.File, "error", err)
			}
			continue
		}
		template := Sidecar{Tail: Tail{Language: a.key.Language, Forced: a.key.Forced, HI: sib.Tail.HI}, Source: FromSibling, SiblingOf: sib.ID}
		j, err := a.sync(ctx, key, sib.TailText, b, nil, template)
		if err != nil {
			return false, err
		}
		if j.checked != nil {
			return true, a.pass(ctx, *j.checked, e)
		}
		if j.try != nil {
			e.failed(*j.try)
		}
	}
	return false, a.note(ctx, e.tries)
}

type sibling struct {
	Sidecar
	at string
	of files.Subject
}

// siblings is the want's Sidecars on the other files with this file's
// Items and cut, at most Timings.Siblings.
func (a *attempt) siblings(ctx context.Context) ([]sibling, error) {
	var out []sibling
	err := a.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := db.New(r).ListSiblings(ctx, db.ListSiblingsParams{WorkID: a.sub.Work, FileID: a.key.File, Language: a.key.Language.String(), Forced: a.key.Forced})
		if err != nil || len(rows) == 0 {
			return err
		}
		ids := make([]files.ID, len(rows))
		for i, row := range rows {
			ids[i] = row.FileID
		}
		subjects, err := a.s.cfg.Files.SubjectsOf(ctx, r, ids)
		if err != nil {
			return err
		}
		for _, row := range rows {
			i := slices.IndexFunc(subjects, func(s files.Subject) bool { return s.ID == row.FileID })
			if i < 0 || !subjects[i].AtName || subjects[i].Cut != a.sub.Cut || !slices.Equal(subjects[i].Items, a.sub.Items) {
				continue
			}
			sc, err := sidecarOf(row)
			if err != nil {
				return err
			}
			out = append(out, sibling{Sidecar: sc, at: sc.at(subjects[i].Path.String()), of: subjects[i]})
			if len(out) == a.s.cfg.Timings.Siblings {
				return nil
			}
		}
		return nil
	})
	return out, err
}

// packKey names the members of one season pack held for one want's
// language and kind.
type packKey struct {
	work     works.WorkID
	season   int
	language language.Code
	forced   bool
}

// heldMember is a pack's other episode, held for Timings.PackHold after
// the pack's download.
type heldMember struct {
	offer  subtitling.Offer
	member subtitling.Member
}

// packs is the season packs' members the Controller holds.
type packs struct {
	mu sync.Mutex
	of map[packKey]heldPack
}

type heldPack struct {
	members []heldMember
	until   time.Time
}

// hold keeps the other members of a pack a's want downloaded, for the
// season's other wants of the same language and kind.
func (s *Subtitles) hold(a *attempt, o subtitling.Offer, members []subtitling.Member) {
	if len(members) == 0 || a.subject.Kind != subtitling.Episode {
		return
	}
	k := packKey{work: a.sub.Work, season: a.subject.Season, language: a.key.Language, forced: a.key.Forced}
	s.packs.mu.Lock()
	defer s.packs.mu.Unlock()
	if s.packs.of == nil {
		s.packs.of = map[packKey]heldPack{}
	}
	p := heldPack{until: s.cfg.Clock().Add(s.cfg.Timings.PackHold)}
	for _, m := range members {
		p.members = append(p.members, heldMember{offer: o, member: m})
	}
	s.packs.of[k] = p
}

// heldFor is the pack members held for a's want that name every episode of
// its file.
func (s *Subtitles) heldFor(a *attempt) []heldMember {
	k := packKey{work: a.sub.Work, season: a.subject.Season, language: a.key.Language, forced: a.key.Forced}
	s.packs.mu.Lock()
	defer s.packs.mu.Unlock()
	p, ok := s.packs.of[k]
	if !ok || !s.cfg.Clock().Before(p.until) {
		delete(s.packs.of, k)
		return nil
	}
	var out []heldMember
	for _, m := range p.members {
		if m.member.Season == a.subject.Season && len(m.member.Episodes) > 0 && !slices.ContainsFunc(a.subject.Episodes, func(e int) bool {
			return !slices.Contains(m.member.Episodes, e)
		}) {
			out = append(out, m)
		}
	}
	return out
}

// tryHeld times each held pack member of the want's episodes, skipping
// any tried before. The first that passes is the want's intent as the
// pack's subtitle. It spends no allowance.
func (a *attempt) tryHeld(ctx context.Context) (bool, error) {
	if a.subject.Kind != subtitling.Episode {
		return false, nil
	}
	var e ending
	for _, m := range a.s.heldFor(a) {
		o := m.offer
		o.Key = m.offer.Key + "/" + m.member.Name
		o.Episodes, o.Pack = m.member.Episodes, false
		if a.skips(o.Provider, o.Key) {
			continue
		}
		template := Sidecar{Tail: Tail{Language: a.key.Language, Forced: a.key.Forced}, Source: FromProvider}
		j, err := a.sync(ctx, o.Key, m.member.Name, m.member.Bytes, &o, template)
		if err != nil {
			return false, err
		}
		if j.checked != nil {
			return true, a.pass(ctx, *j.checked, e)
		}
		if j.try != nil {
			e.failed(*j.try)
		}
	}
	return false, a.note(ctx, e.tries)
}

// note keeps tries that failed before the search goes on, under the
// lease.
func (a *attempt) note(ctx context.Context, tries []try) error {
	if len(tries) == 0 {
		return nil
	}
	return a.write(ctx, func(ctx context.Context, _ *storage.Tx[storage.Library], q *db.Queries) error {
		now := a.s.cfg.Clock()
		for _, t := range tries {
			if err := putTry(ctx, q, a.key, t, now); err != nil {
				return err
			}
		}
		return nil
	})
}
