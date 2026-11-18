package subtitles

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
)

// claimAll settles the rows parked from a doomed file, one replacer at a
// time, once the replacer's bytes sit at their name.
func (k *keeper) claimAll(ctx context.Context, in settled) error {
	var order []files.ID
	by := map[files.ID][]db.SubtitlesParked{}
	for _, p := range in.parked {
		if by[p.ToFile] == nil {
			order = append(order, p.ToFile)
		}
		by[p.ToFile] = append(by[p.ToFile], p)
	}
	for _, to := range order {
		if err := k.claimFor(ctx, in, to, by[to]); err != nil {
			return err
		}
	}
	return nil
}

// replacer is what a claim reads of the file parked rows wait for.
type replacer struct {
	subject  files.Subject
	row      db.SubtitlesFile
	sidecars []Sidecar
}

func (k *keeper) replacer(ctx context.Context, to files.ID) (*replacer, error) {
	var out *replacer
	err := k.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		ids := []files.ID{to}
		subjects, err := k.s.cfg.Files.SubjectsOf(ctx, r, ids)
		if err != nil || len(subjects) == 0 || !subjects[0].AtName {
			return err
		}
		q := db.New(r)
		rows, err := q.ListFilesIn(ctx, ids)
		if err != nil || len(rows) == 0 {
			return err
		}
		scs, err := sidecarsIn(ctx, q, ids)
		if err != nil {
			return err
		}
		out = &replacer{subject: subjects[0], row: rows[0], sidecars: scs[to]}
		return nil
	})
	return out, err
}

// claimFor lists to's video and each parked place, and applies claim's
// verdict: relocations first, each in its own Write after its bytes land,
// then the rest in one.
func (k *keeper) claimFor(ctx context.Context, in settled, to files.ID, parked []db.SubtitlesParked) error {
	r, err := k.replacer(ctx, to)
	if err != nil || r == nil {
		return err
	}
	calls, err := k.calls(ctx, r.subject.Root)
	if err != nil {
		return err
	}
	l := storing.SidecarListing{Video: r.subject.Path}
	for _, p := range parked {
		at, err := storing.ParseRel(p.Path)
		if err != nil {
			return err
		}
		l.Also = append(l.Also, at)
	}
	listed, err := calls.List(ctx, r.subject.Root.UUID, in.videos, []storing.SidecarListing{l})
	if err != nil {
		return callError(err)
	}
	plan, err := claim(to, r.subject.Path.String(), parked, listed[0].Companions, r.sidecars)
	if err != nil {
		return err
	}
	for _, m := range plan.relocate {
		if err := k.relocate(ctx, *r, calls, m); err != nil {
			return err
		}
	}
	return k.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return k.finishClaim(ctx, tx, *r, plan)
	})
}

// finishClaim applies a claim's verdicts on rows still parked. A want
// another Sidecar took in between turns a fetched one into a removal and
// leaves an upload's or a found one's bytes where they sit.
func (k *keeper) finishClaim(ctx context.Context, tx *storage.Tx[storage.Library], r replacer, plan claimPlan) error {
	q := db.New(tx)
	sub := r.subject
	current, err := sidecarsIn(ctx, q, []files.ID{sub.ID})
	if err != nil {
		return err
	}
	have := current[sub.ID]
	var put []Sidecar
	for _, t := range plan.take {
		switch {
		case sidecarOfWant(have, t.sidecar.want()) == nil:
		case own(t.sidecar):
			plan.drop = append(plan.drop, t.parked)
			continue
		default:
			plan.remove = append(plan.remove, t.parked)
			continue
		}
		if n, err := q.DeleteParkedOnce(ctx, t.parked.ID); err != nil || n == 0 {
			if err != nil {
				return err
			}
			continue
		}
		have, put = append(have, t.sidecar), append(put, t.sidecar)
		if err := putSidecar(ctx, q, t.sidecar, storage.FromTime(k.s.cfg.Clock())); err != nil {
			return err
		}
	}
	if err := k.unparkAll(ctx, q, plan.remove, plan.drop); err != nil {
		return err
	}
	return k.claimed(ctx, tx, sub, put)
}

// unparkAll deletes parked rows still there: the bytes of remove go by
// hash, those of drop stay.
func (k *keeper) unparkAll(ctx context.Context, q *db.Queries, remove, drop []db.SubtitlesParked) error {
	for _, p := range remove {
		n, err := q.DeleteParkedOnce(ctx, p.ID)
		if err != nil {
			return err
		}
		if n == 1 {
			if err := k.removal(ctx, q, p.FromFile, p.RootID, p.Path, sha256Of(p.Sha256), "superseded"); err != nil {
				return err
			}
		}
	}
	for _, p := range drop {
		if _, err := q.DeleteParkedOnce(ctx, p.ID); err != nil {
			return err
		}
	}
	return nil
}

// claimed tells Media Servers of the claimed Sidecars already home, makes
// the replacer's map due for the rest to be timed, and wakes it.
func (k *keeper) claimed(ctx context.Context, tx *storage.Tx[storage.Library], sub files.Subject, put []Sidecar) error {
	var changes []mediaservers.SidecarChange
	for _, sc := range put {
		if sc.Stray != "" {
			continue
		}
		at, err := storing.ParseRel(sc.at(sub.Path.String()))
		if err != nil {
			return err
		}
		changes = append(changes, mediaservers.SidecarChange{Kind: mediaservers.SidecarAdded, Root: sub.Root, Path: at, Sidecar: sc.ID})
	}
	if len(changes) > 0 {
		if err := k.s.cfg.MediaServers.SidecarsChanged(ctx, tx, changes); err != nil {
			return err
		}
	}
	k.soon(sub.ID)
	return k.s.markMapsDue(ctx, db.New(tx), []files.ID{sub.ID})
}

// relocate writes an upload's or a found Sidecar's bytes beside the
// replacer under the doomed file's short id, then removes them from their
// old place. Bytes no longer there wait for the next claim, which drops
// them; a name other bytes hold leaves them where they sit; a replacer
// that moved waits.
func (k *keeper) relocate(ctx context.Context, r replacer, calls storing.SidecarCalls, m relocation) error {
	from, err := storing.ParseRel(m.parked.Path)
	if err != nil {
		return err
	}
	b, err := calls.Read(ctx, r.subject.Root.UUID, from, m.sidecar.SHA256)
	err = callError(err)
	if _, changed := errors.AsType[*storing.ChangedError](err); changed || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	c, err := found(m.sidecar, b)
	if err != nil {
		return err
	}
	_, err = calls.Write(ctx, storing.SidecarWrite{
		Root: r.subject.Root.UUID, Video: r.subject.Path, Identity: identityOf(r.row, r.subject), Tail: m.tail, Bytes: c.bytes, SHA256: c.sidecar.SHA256,
	})
	err = callError(err)
	_, occupied := errors.AsType[*storing.OccupiedError](err)
	if errors.Is(err, storing.ErrStale) {
		return nil
	}
	if err != nil && !occupied {
		return err
	}
	return k.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		if n, err := q.DeleteParkedOnce(ctx, m.parked.ID); err != nil || n == 0 || occupied {
			return err
		}
		k.soon(r.subject.ID, m.parked.FromFile)
		if err := q.MarkSurveyDue(ctx, db.MarkSurveyDueParams{FileID: r.subject.ID}); err != nil {
			return err
		}
		return k.removal(ctx, q, m.parked.FromFile, m.parked.RootID, m.parked.Path, m.sidecar.SHA256, "superseded")
	})
}

// claimPlan is claim's verdict on each parked row.
type claimPlan struct {
	// take become the replacer's Sidecars.
	take []taken
	// remove are fetched bytes no Sidecar keeps.
	remove []db.SubtitlesParked
	// relocate are an upload's or a found one's bytes for which the
	// replacer has a Sidecar.
	relocate []relocation
	// drop are rows whose bytes aren't listed where they were parked, or
	// that the replacer already holds, or an upload's or found one's that
	// can't be written beside it and stay where they sit.
	drop []db.SubtitlesParked
}

type taken struct {
	parked  db.SubtitlesParked
	sidecar Sidecar
}

type relocation struct {
	parked  db.SubtitlesParked
	sidecar Sidecar
	tail    string
}

// own is a Sidecar the Admin brought: an upload or one found beside the
// file. SlipStream removes it only through Mark bad.
func own(sc Sidecar) bool { return sc.Source == FromUpload || sc.Source == Found }

// claim is the pure decision on rows parked for the replacer to, whose
// video sits at video with Sidecars rows. A parked row counts only where
// its bytes are listed. For each want the replacer lacks, the
// highest-ranked fetched one becomes its Sidecar, stray where its bytes
// sit and due to be timed, else the first upload or found one, untimed
// and final. Other fetched ones are removed, and other uploads and found
// ones are written beside the replacer as "<stem>.<8 hex of the doomed
// file>.<tail>", except a tail SlipStream can't write, which stays.
func claim(to files.ID, video string, parked []db.SubtitlesParked, listed []storing.SidecarCompanion, rows []Sidecar) (claimPlan, error) {
	wants, of, drop, err := claimable(video, parked, listed, rows)
	if err != nil {
		return claimPlan{}, err
	}
	plan := claimPlan{drop: drop}
	for _, w := range wants {
		ts := of[w]
		slices.SortStableFunc(ts, func(a, b taken) int { return better(b.sidecar, a.sidecar) })
		plan.decide(to, video, ts, sidecarOfWant(rows, w) == nil)
	}
	return plan, nil
}

// claimable is the parked rows whose bytes are listed where they were
// parked and that no row of the replacer holds, by want in parked order,
// and the rest to drop.
func claimable(video string, parked []db.SubtitlesParked, listed []storing.SidecarCompanion, rows []Sidecar) ([]Want, map[Want][]taken, []db.SubtitlesParked, error) {
	held := func(p db.SubtitlesParked) bool {
		return slices.ContainsFunc(listed, func(c storing.SidecarCompanion) bool {
			return c.Path.String() == p.Path && bytes.Equal(c.SHA256[:], p.Sha256)
		})
	}
	kept := func(p db.SubtitlesParked) bool {
		return slices.ContainsFunc(rows, func(sc Sidecar) bool { return sc.at(video) == p.Path && bytes.Equal(sc.SHA256[:], p.Sha256) })
	}
	var wants []Want
	var drop []db.SubtitlesParked
	of := map[Want][]taken{}
	for _, p := range parked {
		sc, err := decodeSidecar(p.Sidecar)
		if err != nil {
			return nil, nil, nil, err
		}
		sc.SHA256 = sha256Of(p.Sha256)
		if !held(p) || kept(p) {
			drop = append(drop, p)
			continue
		}
		w := sc.want()
		if of[w] == nil {
			wants = append(wants, w)
		}
		of[w] = append(of[w], taken{parked: p, sidecar: sc})
	}
	return wants, of, drop, nil
}

// decide is the verdict on one want's parked rows ts, best first, while
// free says the replacer has no Sidecar of it.
func (plan *claimPlan) decide(to files.ID, video string, ts []taken, free bool) {
	for _, t := range ts {
		switch {
		case own(t.sidecar):
		case free:
			plan.take = append(plan.take, t.as(to, video, true))
			free = false
		default:
			plan.remove = append(plan.remove, t.parked)
		}
	}
	for _, t := range ts {
		if !own(t.sidecar) {
			continue
		}
		tail := shortID(t.parked.FromFile) + "." + t.sidecar.TailText
		switch {
		case free:
			plan.take = append(plan.take, t.as(to, video, false))
			free = false
		case storing.WritableTail(tail):
			plan.relocate = append(plan.relocate, relocation{parked: t.parked, sidecar: t.sidecar, tail: tail})
		default:
			plan.drop = append(plan.drop, t.parked)
		}
	}
}

// as is t's Sidecar as the replacer's: a fetched one due to be timed, an
// upload or a found one untimed and final.
func (t taken) as(to files.ID, video string, fetched bool) taken {
	sc := t.sidecar
	sc.ID, sc.File, sc.Added = storage.NewUUID(), to, storage.Time{}
	sc.Stray = strayAt(t.parked.Path, besideAt(video, sc.TailText))
	if fetched {
		sc.TimingDue = true
	} else {
		sc.Fit, sc.Final, sc.TimingDue = nil, true, false
	}
	t.sidecar = sc
	return t
}

// better orders two Sidecars of one want: final first, then the higher
// timing match.
func better(a, b Sidecar) int {
	if a.Final != b.Final {
		if a.Final {
			return 1
		}
		return -1
	}
	return cmp.Compare(matchOr(a), matchOr(b))
}

func matchOr(sc Sidecar) int {
	if sc.Fit == nil {
		return 0
	}
	return sc.Fit.Match
}

// shortID is the first 8 hex digits of f, the mark a moved companion
// carries.
func shortID(f files.ID) string { return strings.ReplaceAll(f.String(), "-", "")[:8] }

// strays adds to also the place of each stray Sidecar, so a survey lists
// bytes a claim left beside the file they came from.
func strays(also []storing.Rel, scs []Sidecar) ([]storing.Rel, error) {
	for _, sc := range scs {
		if sc.Stray == "" {
			continue
		}
		at, err := storing.ParseRel(sc.Stray)
		if err != nil {
			return nil, err
		}
		also = append(also, at)
	}
	return also, nil
}
