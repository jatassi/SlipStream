package subtitles

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
	"github.com/jatassi/slipstream/internal/subtitling"
)

// converge writes a live file's intents whose bytes this process holds,
// times again each Sidecar that waits for it, and brings home each stray
// one whose name is free. A step that can't run now leaves its rows for
// the next wake.
func (k *keeper) converge(ctx context.Context, in settled) error {
	for _, w := range in.intents {
		if b := in.held[w.AttemptID]; b != nil {
			if err := k.write(ctx, in, w, b); err != nil {
				return err
			}
		}
	}
	for _, sc := range in.sidecars {
		var err error
		switch {
		case sc.TimingDue:
			err = k.retime(ctx, in, sc)
		case sc.Stray != "":
			err = k.home(ctx, in, sc)
		}
		if errors.Is(err, errMapDue) {
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func sha256Of(b []byte) [32]byte {
	var sha [32]byte
	copy(sha[:], b)
	return sha
}

// write writes intent w's bytes b beside the file. Written finishes the
// intent. A name that holds other bytes drops it and surveys, so the
// occupant is read as found or as SlipStream's own; a video that changed
// drops it and makes its map due. Either way the want stays due.
func (k *keeper) write(ctx context.Context, in settled, w db.SubtitlesWrite, b []byte) error {
	sub := in.subject
	calls, err := k.calls(ctx, sub.Root)
	if err != nil {
		return err
	}
	sc, err := decodeSidecar(w.Sidecar)
	if err != nil {
		return err
	}
	sw := storing.SidecarWrite{Root: sub.Root.UUID, Video: sub.Path, Identity: identityOf(*in.row, *sub), Tail: w.Tail, Bytes: b, SHA256: sha256Of(w.Sha256), Ledger: sc.Source == FromProvider}
	if len(w.Replaces) == len(sw.SHA256) {
		sw.Replaces = new(sha256Of(w.Replaces))
	}
	at, err := calls.Write(ctx, sw)
	err = callError(err)
	_, occupied := errors.AsType[*storing.OccupiedError](err)
	stale := errors.Is(err, storing.ErrStale)
	if err != nil && !occupied && !stale {
		return err
	}
	return k.commit(ctx, *sub, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		switch {
		case occupied:
			if _, err := q.DeleteIntent(ctx, intentKey(w)); err != nil {
				return err
			}
			return q.MarkSurveyDue(ctx, db.MarkSurveyDueParams{FileID: sub.ID})
		case stale:
			if _, err := q.DeleteIntent(ctx, intentKey(w)); err != nil {
				return err
			}
			if err := q.MapStale(ctx, sub.ID); err != nil {
				return err
			}
			return k.s.markMapsDue(ctx, q, []files.ID{sub.ID})
		}
		return k.finishIntent(ctx, tx, in, w, at)
	})
}

func intentKey(w db.SubtitlesWrite) db.DeleteIntentParams {
	return db.DeleteIntentParams{FileID: w.FileID, Language: w.Language, Forced: w.Forced, AttemptID: w.AttemptID}
}

// finishIntent makes the intent whose bytes landed at at the want's
// Sidecar, guarded by its attempt: an intent a doom, a drop or a Mark bad
// took in between left a parked or removal row that covers its bytes. The
// Sidecar it replaces goes, and its bytes too when they sat elsewhere.
func (k *keeper) finishIntent(ctx context.Context, tx *storage.Tx[storage.Library], in settled, w db.SubtitlesWrite, at storing.Rel) error {
	q := db.New(tx)
	sub := in.subject
	now := storage.FromTime(k.s.cfg.Clock())
	gone, err := q.DeleteIntent(ctx, intentKey(w))
	if err != nil {
		return err
	}
	if gone == 0 {
		k.soon(sub.ID)
		return nil
	}
	sc, err := decodeSidecar(w.Sidecar)
	if err != nil {
		return err
	}
	sc.ID, sc.File, sc.TailText, sc.SHA256, sc.Stray, sc.Added = storage.NewUUID(), sub.ID, w.Tail, sha256Of(w.Sha256), "", storage.Time{}
	current, err := sidecarsIn(ctx, q, []files.ID{sub.ID})
	if err != nil {
		return err
	}
	row, kind := addedRow, mediaservers.SidecarAdded
	if old := sidecarOfWant(current[sub.ID], sc.want()); old != nil {
		row, kind = replacedRow, mediaservers.SidecarReplaced
		if err := dropSidecar(ctx, q, *old, now); err != nil {
			return err
		}
		if was := old.at(sub.Path.String()); was != at.String() {
			if err := k.removal(ctx, q, sub.ID, sub.Root, was, old.SHA256, "superseded"); err != nil {
				return err
			}
			k.soon(sub.ID)
		}
	}
	if err := putSidecar(ctx, q, sc, now); err != nil {
		return err
	}
	if o := sc.Offer; o != nil {
		err := q.PutTried(ctx, db.PutTriedParams{
			FileID: sub.ID, Language: w.Language, Forced: w.Forced, CandidateKey: o.Key, Provider: some(string(o.Provider)), Release: some(o.Release),
			Match: matchOf(sc), Outcome: "passed", TriedAt: now,
		})
		if err != nil {
			return err
		}
	}
	if err := k.s.cfg.MediaServers.SidecarsChanged(ctx, tx, []mediaservers.SidecarChange{{Kind: kind, Root: sub.Root, Path: at, Sidecar: sc.ID}}); err != nil {
		return err
	}
	return k.s.recordSidecars(ctx, tx, row, *sub, []Sidecar{sc}, "")
}

// announce tells Media Servers and History of the Sidecars a survey
// finished from intents whose write's answer was lost.
func (k *keeper) announce(ctx context.Context, tx *storage.Tx[storage.Library], sub files.Subject, scs []Sidecar) error {
	if len(scs) == 0 {
		return nil
	}
	changes := make([]mediaservers.SidecarChange, 0, len(scs))
	for _, sc := range scs {
		at, err := storing.ParseRel(sc.at(sub.Path.String()))
		if err != nil {
			return err
		}
		changes = append(changes, mediaservers.SidecarChange{Kind: mediaservers.SidecarAdded, Root: sub.Root, Path: at, Sidecar: sc.ID})
	}
	if err := k.s.cfg.MediaServers.SidecarsChanged(ctx, tx, changes); err != nil {
		return err
	}
	return k.s.recordSidecars(ctx, tx, addedRow, sub, scs, "")
}

func matchOf(sc Sidecar) *int64 {
	if sc.Fit == nil {
		return nil
	}
	return new(int64(sc.Fit.Match))
}

func (k *keeper) removal(ctx context.Context, q *db.Queries, f files.ID, root roots.ID, at string, sha [32]byte, why string) error {
	return q.InsertRemoval(ctx, db.InsertRemovalParams{ID: storage.NewUUID(), FileID: f, RootID: root, Path: at, Sha256: sha[:], Why: why, NextAt: storage.FromTime(k.s.cfg.Clock())})
}

// holder is SubtitlesService on a Node with the Subtitles Role, false
// when none has it.
func (s *Subtitles) holder() (subtitling.Remote, bool) {
	ids := s.cfg.Roster.Holding(role.Subtitles)
	if len(ids) == 0 {
		return subtitling.Remote{}, false
	}
	return s.cfg.Holders(ids[0]), true
}

// retime times sc again against the file's current map on a holder, and
// writes the retimed bytes home. A fetched Sidecar that fails goes and
// its want searches again; an upload or a found one stays, untimed,
// because SlipStream deletes the Admin's own file only through Mark bad.
// A holder's answer the check refuses leaves sc due for another try.
func (k *keeper) retime(ctx context.Context, in settled, sc Sidecar) error {
	holder, ok := k.s.holder()
	if !ok {
		return nil
	}
	sub := in.subject
	ref, m, err := k.s.speechMap(ctx, *in.row, *sub)
	if err != nil {
		return err
	}
	b, err := k.readBack(ctx, in, sc)
	if b == nil || err != nil {
		return err
	}
	var g timing.Gate
	if err := k.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		g, err = k.s.gate(ctx, r)
		return err
	}); err != nil {
		return err
	}
	ans, err := holder.Sync(ctx, subtitling.SyncAsk{Name: sc.TailText, Bytes: b, Language: sc.Tail.Language, Map: m, Gate: g, Constants: k.s.cfg.Timings.Sync})
	if err != nil {
		return err
	}
	if ans.Verdict != subtitling.Passed {
		return k.retimeFailed(ctx, in, sc)
	}
	template := sc
	template.Fit = nil
	c, err := check(asked{sidecar: template, bound: boundOf(sub.Duration, ref.Frames(), k.s.cfg.Timings.CueMargin)}, ref, ans, g, k.s.cfg.Timings.Sync, k.s.cfg.FetchCap)
	if errors.Is(err, timing.ErrForged) {
		k.s.cfg.Log.WarnContext(ctx, "A subtitle the Subtitles Node retimed failed the Controller's check", "file", sub.ID, "sidecar", sc.ID, "error", err)
		return nil
	}
	if err != nil {
		return err
	}
	if !storing.WritableTail(sc.TailText) {
		return k.untimed(ctx, in, sc)
	}
	return k.place(ctx, in, sc, c, mediaservers.SidecarReplaced)
}

// readBack is sc's bytes where they sit, nil when they aren't there: the
// file is then surveyed, which finds where they went.
func (k *keeper) readBack(ctx context.Context, in settled, sc Sidecar) ([]byte, error) {
	return k.bytesOf(ctx, *in.subject, sc.at(in.subject.Path.String()), sc.SHA256)
}

// home brings a stray Sidecar's bytes home once its name is free.
func (k *keeper) home(ctx context.Context, in settled, sc Sidecar) error {
	if !storing.WritableTail(sc.TailText) {
		return nil
	}
	b, err := k.readBack(ctx, in, sc)
	if b == nil || err != nil {
		return err
	}
	c, err := found(sc, b)
	if err != nil {
		return err
	}
	return k.place(ctx, in, sc, c, mediaservers.SidecarAdded)
}

// place writes c home for sc, whose bytes sit at sc.at, then makes the row
// c's, guarded by sc's bytes. A name taken by other bytes waits. Bytes
// left at a stray place go by hash. A retime's rewrite is recorded as a
// Subtitle replaced.
func (k *keeper) place(ctx context.Context, in settled, sc Sidecar, c Checked, kind mediaservers.SidecarKind) error {
	sub := in.subject
	calls, err := k.calls(ctx, sub.Root)
	if err != nil {
		return err
	}
	sw := storing.SidecarWrite{Root: sub.Root.UUID, Video: sub.Path, Identity: identityOf(*in.row, *sub), Tail: sc.TailText, Bytes: c.bytes, SHA256: c.sidecar.SHA256, Ledger: sc.Source == FromProvider}
	if sc.Stray == "" {
		sw.Replaces = &sc.SHA256
	}
	at, err := calls.Write(ctx, sw)
	err = callError(err)
	if _, occupied := errors.AsType[*storing.OccupiedError](err); occupied || errors.Is(err, storing.ErrStale) {
		return nil
	}
	if err != nil {
		return err
	}
	return k.commit(ctx, *sub, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return k.placed(ctx, tx, *sub, sc, c, at, kind)
	})
}

// placed makes sc's row c's once c's bytes sit at at.
func (k *keeper) placed(ctx context.Context, tx *storage.Tx[storage.Library], sub files.Subject, sc Sidecar, c Checked, at storing.Rel, kind mediaservers.SidecarKind) error {
	q := db.New(tx)
	held, err := q.GetSidecarHeld(ctx, db.GetSidecarHeldParams{ID: sc.ID, Sha256: sc.SHA256[:]})
	if err != nil {
		return err
	}
	k.soon(sub.ID)
	if held == 0 {
		if c.sidecar.SHA256 == sc.SHA256 && sc.Stray == "" {
			return nil
		}
		return k.removal(ctx, q, sub.ID, sub.Root, at.String(), c.sidecar.SHA256, "superseded")
	}
	if sc.Stray != "" {
		if err := k.removal(ctx, q, sub.ID, sub.Root, sc.Stray, sc.SHA256, "superseded"); err != nil {
			return err
		}
	}
	next := c.sidecar
	next.Stray, next.TimingDue = "", false
	if err := putSidecar(ctx, q, next, storage.FromTime(k.s.cfg.Clock())); err != nil {
		return err
	}
	if err := k.s.cfg.MediaServers.SidecarsChanged(ctx, tx, []mediaservers.SidecarChange{{Kind: kind, Root: sub.Root, Path: at, Sidecar: sc.ID}}); err != nil {
		return err
	}
	if kind != mediaservers.SidecarReplaced {
		return nil
	}
	return k.s.recordSidecars(ctx, tx, replacedRow, sub, []Sidecar{next}, "")
}

// retimeFailed settles a Sidecar whose timing failed against the file's
// map: a fetched one goes, by hash, and its want is due now; the Admin's
// own stays, untimed.
func (k *keeper) retimeFailed(ctx context.Context, in settled, sc Sidecar) error {
	if sc.Source == FromUpload || sc.Source == Found {
		return k.untimed(ctx, in, sc)
	}
	sub := in.subject
	return k.commit(ctx, *sub, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		held, err := q.GetSidecarHeld(ctx, db.GetSidecarHeldParams{ID: sc.ID, Sha256: sc.SHA256[:]})
		if err != nil || held == 0 {
			return err
		}
		now := storage.FromTime(k.s.cfg.Clock())
		at := sc.at(sub.Path.String())
		if err := k.removal(ctx, q, sub.ID, sub.Root, at, sc.SHA256, "retime_failed"); err != nil {
			return err
		}
		if err := dropSidecar(ctx, q, sc, now); err != nil {
			return err
		}
		if err := q.DueNow(ctx, db.DueNowParams{Now: now, FileID: sub.ID, Language: sc.Tail.Language.String(), Forced: sc.Tail.Forced}); err != nil {
			return err
		}
		k.soon(sub.ID)
		rel, err := storing.ParseRel(at)
		if err != nil {
			return err
		}
		if err := k.s.cfg.MediaServers.SidecarsChanged(ctx, tx, []mediaservers.SidecarChange{{Kind: mediaservers.SidecarRemoved, Root: sub.Root, Path: rel, Sidecar: sc.ID}}); err != nil {
			return err
		}
		return k.s.recordSidecars(ctx, tx, removedRow, *sub, []Sidecar{sc}, "retime_failed")
	})
}

// untimed keeps sc where it is with no timing, guarded by its bytes.
func (k *keeper) untimed(ctx context.Context, in settled, sc Sidecar) error {
	return k.commit(ctx, *in.subject, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		held, err := q.GetSidecarHeld(ctx, db.GetSidecarHeldParams{ID: sc.ID, Sha256: sc.SHA256[:]})
		if err != nil || held == 0 {
			return err
		}
		sc.Fit, sc.TimingDue = nil, false
		if sc.Stray != "" {
			k.soon(in.subject.ID)
		}
		return putSidecar(ctx, q, sc, storage.FromTime(k.s.cfg.Clock()))
	})
}

// bytesOf is the bytes of sha beside sub at at, for an attempt to time
// against another file: nil when they aren't there, and sub is then
// surveyed.
func (k *keeper) bytesOf(ctx context.Context, sub files.Subject, at string, sha [32]byte) ([]byte, error) {
	calls, err := k.calls(ctx, sub.Root)
	if err != nil {
		return nil, err
	}
	rel, err := storing.ParseRel(at)
	if err != nil {
		return nil, err
	}
	b, err := calls.Read(ctx, sub.Root.UUID, rel, sha)
	err = callError(err)
	if _, changed := errors.AsType[*storing.ChangedError](err); changed || errors.Is(err, fs.ErrNotExist) {
		return nil, k.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			k.soon(sub.ID)
			return db.New(tx).MarkSurveyDue(ctx, db.MarkSurveyDueParams{FileID: sub.ID})
		})
	}
	return b, err
}

// hold reads which of in's intents this process holds the bytes of, and
// which copy another file's Sidecar unchanged: an intent Derived made,
// whose bytes the Keeper reads from beside that file.
func (k *keeper) hold(ctx context.Context, f files.ID, in *settled) error {
	in.held = k.bytes.held(f, in.intents)
	in.copies = map[storage.UUID]sibling{}
	if in.subject == nil {
		return nil
	}
	for _, w := range in.intents {
		if in.held[w.AttemptID] != nil {
			continue
		}
		sc, err := decodeSidecar(w.Sidecar)
		if err != nil {
			return err
		}
		if sc.Source != FromSibling || sc.SiblingOf.IsZero() {
			continue
		}
		from, ok, err := k.copied(ctx, *in.subject, w, sc.SiblingOf)
		if err != nil {
			return err
		}
		if ok {
			in.copies[w.AttemptID] = from
		}
	}
	return nil
}

func (in *settled) copying(w db.SubtitlesWrite) bool {
	_, ok := in.copies[w.AttemptID]
	return ok
}

// fetch reads each copy's bytes once the file's map is built, when its
// identity is known, so converge writes them.
func (k *keeper) fetch(ctx context.Context, in settled) error {
	if in.row.MapState != "built" {
		return nil
	}
	for a, from := range in.copies {
		b, err := k.bytesOf(ctx, from.of, from.at, from.SHA256)
		if err != nil {
			return err
		}
		if b != nil {
			k.bytes.hand(in.subject.ID, a, b)
			in.held[a] = b
		}
	}
	return nil
}

// copied is the Sidecar of sub's Work with id that holds intent w's bytes,
// on a file at its name.
func (k *keeper) copied(ctx context.Context, sub files.Subject, w db.SubtitlesWrite, id storage.UUID) (sibling, bool, error) {
	var out sibling
	var ok bool
	err := k.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := db.New(r).ListSiblings(ctx, db.ListSiblingsParams{WorkID: sub.Work, FileID: sub.ID, Language: w.Language, Forced: w.Forced})
		if err != nil {
			return err
		}
		i := slices.IndexFunc(rows, func(row db.SubtitlesSidecar) bool { return row.ID == id && bytes.Equal(row.Sha256, w.Sha256) })
		if i < 0 {
			return nil
		}
		subjects, err := k.s.cfg.Files.SubjectsOf(ctx, r, []files.ID{rows[i].FileID})
		if err != nil || len(subjects) != 1 || !subjects[0].AtName {
			return err
		}
		sc, err := sidecarOf(rows[i])
		if err != nil {
			return err
		}
		out, ok = sibling{Sidecar: sc, at: sc.at(subjects[0].Path.String()), of: subjects[0]}, true
		return nil
	})
	return out, ok, err
}
