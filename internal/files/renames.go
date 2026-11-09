package files

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// Draft is unsaved naming settings for one Content Type.
type Draft struct {
	Type  works.TypeID
	Patch setting.Patch
}

// Confirm is one Work of a plan an Admin confirmed: its digest as shown
// and the files left out.
type Confirm struct {
	Work   works.WorkID
	Digest Digest
	Leave  []ID
}

// Confirmed is what a confirm queued, and the Works it left because
// their rows differ from the confirmed ones now.
type Confirmed struct {
	Queued  int
	Changed []works.WorkID
}

// confirmChunk is the most Works one Write of a confirm renames, so the
// library's single writer is held for a chunk, never for a whole library.
const confirmChunk = 200

// Plan is the plan of each Work of s with a row or a file the disk queue
// still moves, under the stored settings, or draft over its type's. A
// draft the save check refuses is its *naming.RefusedError. Each root's
// Node is asked which files sit at the new paths, except a lost root's.
func (f *Files) Plan(ctx context.Context, _ identity.Admin, s Scope, draft *Draft) ([]WorkPlan, error) {
	ps, err := f.settled(ctx, s, draft, false)
	return slices.DeleteFunc(ps, func(p WorkPlan) bool { return len(p.Rows) == 0 && p.Moving == 0 }), err
}

// PlanOf is w's plan as Plan makes it, each row with its companions.
func (f *Files) PlanOf(ctx context.Context, _ identity.Admin, w works.WorkID, draft *Draft) (WorkPlan, error) {
	ps, err := f.settled(ctx, WorkScope(w), draft, true)
	if err != nil {
		return WorkPlan{}, err
	}
	return ps[0], nil
}

// Unmatched counts, per Content Type, the live files whose path from the
// templates differs from their stored one. It asks no Node.
func (f *Files) Unmatched(ctx context.Context, _ identity.Admin) (map[works.TypeID]int, error) {
	out := map[works.TypeID]int{}
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		trees, among, err := LibraryScope().trees(ctx, r)
		if err != nil {
			return err
		}
		pl, err := f.plans(ctx, r, trees, among, nil)
		for _, p := range pl {
			out[p.Type] += len(p.Rows)
		}
		return err
	})
	return out, err
}

// Rename renames the rows of each Work of cs whose plan under the stored
// settings still has the confirmed digest, but for the files it leaves
// out, every row of a group with one left out or blocked, and blocked
// rows. Other Works are left and named. It asks each root's Node which
// files sit at the new paths first, then recomputes and renames up to 200
// Works per Write, each judged by its digest alone, with the new paths of
// cs's other Works in the way as the plan had them.
func (f *Files) Rename(ctx context.Context, _ identity.Admin, cs []Confirm) (Confirmed, error) {
	asked := make(map[works.WorkID]Confirm, len(cs))
	var ids []works.WorkID
	for _, c := range cs {
		if _, ok := asked[c.Work]; !ok {
			asked[c.Work] = c
			ids = append(ids, c.Work)
		}
	}
	var first []WorkPlan
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		pl, err := f.plansOf(ctx, r, ids)
		first = workPlans(pl)
		return err
	})
	if err != nil {
		return Confirmed{}, err
	}
	found, _, err := f.survey(ctx, first, false)
	if err != nil {
		return Confirmed{}, err
	}
	var out Confirmed
	var renamed []works.WorkID
	seen := map[works.WorkID]bool{}
	for chunk := range slices.Chunk(ids, confirmChunk) {
		err := f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			pl, err := f.plansOf(ctx, tx, chunk)
			if err != nil {
				return err
			}
			q := db.New(tx)
			tracked, err := trackedIn(ctx, q)
			if err != nil {
				return err
			}
			inChunk := map[works.WorkID]bool{}
			for _, id := range chunk {
				inChunk[id] = true
			}
			for _, p := range first {
				if !inChunk[p.Work] {
					aside(tracked, p)
				}
			}
			ps := workPlans(pl)
			settle(ps, tracked, found)
			for i, p := range ps {
				seen[p.Work] = true
				if p.Digest != asked[p.Work].Digest {
					out.Changed = append(out.Changed, p.Work)
					continue
				}
				n, err := renameRows(ctx, q, pl[i], ps[i].Rows, asked[p.Work].Leave)
				if err != nil {
					return err
				}
				if n > 0 {
					out.Queued += n
					renamed = append(renamed, p.Work)
				}
			}
			return nil
		})
		if err != nil {
			return Confirmed{}, err
		}
	}
	for _, id := range ids {
		if !seen[id] {
			out.Changed = append(out.Changed, id)
		}
	}
	if len(renamed) > 0 {
		f.Poke()
		f.cfg.Moved.Publish(ctx, MovedOf(renamed))
	}
	return out, nil
}

// renameRows renames rows of p's Work, as settled, but for the files of
// leave, every row of a group one of them or a blocked row is in, and the
// blocked rows, and answers how many it renamed. A gone file whose path a
// row takes is forgotten: its bytes could no longer return there.
func renameRows(ctx context.Context, q *db.Queries, p worldPlan, rows []Row, leave []ID) (int, error) {
	left := map[ID]bool{}
	for _, id := range leave {
		left[id] = true
	}
	stays := map[int]bool{}
	for _, r := range rows {
		if r.Group != 0 && (left[r.File] || r.Blocked != (Blocked{})) {
			stays[r.Group] = true
		}
	}
	nm := make(map[ID]name, len(p.w.files))
	for _, g := range p.w.files {
		nm[g.ID] = name{Label: g.Label, Extra: g.Extra, Path: g.Path}
	}
	n := 0
	for _, r := range rows {
		if left[r.File] || r.Blocked != (Blocked{}) || stays[r.Group] {
			continue
		}
		if err := q.ForgetGoneAt(ctx, db.ForgetGoneAtParams{RootID: r.root, Path: r.To}); err != nil {
			return 0, err
		}
		nm[r.File] = p.names[r.File]
		n++
	}
	if n == 0 {
		return 0, nil
	}
	return n, apply(ctx, q, placed{before: p.w, after: p.w, fills: p.w.fills, names: nm})
}

// settled is the plan of each Work of s, read in one Read, its rows
// blocked by what each root's Node finds at their new paths and by each
// other, with each row's companions when asked.
func (f *Files) settled(ctx context.Context, s Scope, draft *Draft, companions bool) ([]WorkPlan, error) {
	if err := f.known(s, draft); err != nil {
		return nil, err
	}
	var ps []WorkPlan
	var tracked map[place][]works.WorkID
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		trees, among, err := s.trees(ctx, r)
		if err != nil {
			return err
		}
		pl, err := f.plans(ctx, r, trees, among, draft)
		if err != nil {
			return err
		}
		ps = workPlans(pl)
		tracked, err = trackedIn(ctx, db.New(r))
		return err
	})
	if err != nil {
		return nil, err
	}
	found, beside, err := f.survey(ctx, ps, companions)
	if err != nil {
		return nil, err
	}
	settle(ps, tracked, found)
	for i := range ps {
		for j := range ps[i].Rows {
			ps[i].Rows[j].Companions = beside[ps[i].Rows[j].File]
		}
	}
	return ps, nil
}

// known is nil when s and draft name only Content Types the library has.
func (f *Files) known(s Scope, draft *Draft) error {
	var types []works.TypeID
	if s.kind == oneType {
		types = append(types, s.typ)
	}
	if draft != nil {
		types = append(types, draft.Type)
	}
	for _, t := range types {
		if _, ok := f.cfg.Names[t]; !ok {
			return works.ErrNoType
		}
	}
	return nil
}

// plansOf is the plan of each of ids a Work has, under the stored
// settings, in r.
func (f *Files) plansOf(ctx context.Context, r storage.Reader[file], ids []works.WorkID) ([]worldPlan, error) {
	trees, err := works.OutlinesIn(ctx, r, works.Only(ids))
	if err != nil {
		return nil, err
	}
	bares := make([]works.Bare, len(trees))
	have := make([]works.WorkID, len(trees))
	for i, t := range trees {
		bares[i], have[i] = works.BareOf(t), t.ID
	}
	return f.plans(ctx, r, bares, works.Only(have), nil)
}

// trees reads each Work of s's bare tree in r, and the Among naming them.
func (s Scope) trees(ctx context.Context, r storage.Reader[file]) ([]works.Bare, works.Among[works.WorkID], error) {
	if s.kind == oneWork {
		t, err := works.BareIn(ctx, r, s.work)
		return []works.Bare{t}, works.Only([]works.WorkID{s.work}), err
	}
	all, err := works.OutlinesIn(ctx, r, works.Every[works.WorkID](nil))
	if err != nil {
		return nil, works.Among[works.WorkID]{}, err
	}
	var out []works.Bare
	var ids []works.WorkID
	for _, t := range all {
		if s.kind == library || t.ContentType == s.typ {
			out, ids = append(out, works.BareOf(t)), append(ids, t.ID)
		}
	}
	if s.kind == library {
		return out, works.Every(ids), nil
	}
	return out, works.Only(ids), nil
}

func workPlans(pl []worldPlan) []WorkPlan {
	out := make([]WorkPlan, len(pl))
	for i, p := range pl {
		out[i] = p.WorkPlan
	}
	return out
}

// trackedIn is each place a live file's bytes belong or sit, and its
// Work.
func trackedIn(ctx context.Context, q *db.Queries) (map[place][]works.WorkID, error) {
	rows, err := q.ListTracked(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[place][]works.WorkID, len(rows))
	for _, row := range rows {
		out[place{row.RootID, row.Path}] = append(out[place{row.RootID, row.Path}], row.WorkID)
		if row.Sits != nil {
			out[place{row.RootID, *row.Sits}] = append(out[place{row.RootID, *row.Sits}], row.WorkID)
		}
	}
	return out, nil
}

// aside adds the new path of each row of p not blocked before settling to
// tracked, as a place p's Work takes.
func aside(tracked map[place][]works.WorkID, p WorkPlan) {
	for _, r := range p.Rows {
		if r.Blocked == (Blocked{}) {
			tracked[place{r.root, r.To}] = append(tracked[place{r.root, r.To}], p.Work)
		}
	}
}

// survey asks the Node of each root ps's rows are in, but a lost root's,
// which files sit at the new path of each row not blocked yet, and with
// companions the files beside its old path that move with it, both by
// the row's file.
func (f *Files) survey(ctx context.Context, ps []WorkPlan, companions bool) (found, beside map[ID][]string, err error) {
	byRoot := map[roots.ID][]Row{}
	for _, p := range ps {
		for _, r := range p.Rows {
			if r.Blocked == (Blocked{}) {
				byRoot[r.root] = append(byRoot[r.root], r)
			}
		}
	}
	if len(byRoot) == 0 {
		return nil, nil, nil
	}
	var lost map[roots.ID]bool
	err = f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		lost, err = roots.LostIn(ctx, r)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	found, beside = map[ID][]string{}, map[ID][]string{}
	for _, root := range slices.SortedFunc(maps.Keys(byRoot), func(a, b roots.ID) int { return a.Compare(b.UUID) }) {
		if lost[root] {
			continue
		}
		rows := byRoot[root]
		planned := make([]storing.Planned, len(rows))
		for i, r := range rows {
			if planned[i], err = plannedOf(r); err != nil {
				return nil, nil, err
			}
		}
		answers, err := f.cfg.Disk.Survey(ctx, root, planned, companions)
		if err != nil {
			return nil, nil, fmt.Errorf("list the files at the new paths in library root %s: %w", root, err)
		}
		for i, a := range answers {
			for _, o := range a.Occupants {
				found[rows[i].File] = append(found[rows[i].File], o.Path.String())
			}
			for _, c := range a.Companions {
				beside[rows[i].File] = append(beside[rows[i].File], c.String())
			}
		}
	}
	return found, beside, nil
}

func plannedOf(r Row) (storing.Planned, error) {
	from, err := storing.ParseRel(r.From)
	if err != nil {
		return storing.Planned{}, fmt.Errorf("file %s: %w", r.File, err)
	}
	to, err := storing.ParseRel(r.To)
	if err != nil {
		return storing.Planned{}, fmt.Errorf("file %s: %w", r.File, err)
	}
	return storing.Planned{From: from, To: to}, nil
}
