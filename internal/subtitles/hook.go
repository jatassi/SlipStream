package subtitles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/targets"
)

var _ files.Subtitles = (*Subtitles)(nil)

// Changed hears files' path facts in the Write that makes them true. A
// file that landed at its name, moved, was put back or changed in place
// gets its facts and wants again and a survey; a changed one's map is
// built again and its Sidecars timed again; parked rows waiting for a
// landing wake.
func (s *Subtitles) Changed(ctx context.Context, tx *storage.Tx[storage.Library], cs []files.PathChange) error {
	var ids []files.ID
	for _, c := range cs {
		if c.Kind != files.PathRemoved && !slices.Contains(ids, c.File) {
			ids = append(ids, c.File)
		}
	}
	if err := s.reconcileIn(ctx, tx, ids); err != nil {
		return err
	}
	q := db.New(tx)
	var wake []files.ID
	for _, c := range cs {
		switch c.Kind {
		case files.PathRemoved:
			continue
		case files.PathReplaced:
			if err := s.replaced(ctx, q, c.File); err != nil {
				return err
			}
		case files.PathAdded:
			parked, err := q.ListParkedTo(ctx, c.File)
			if err != nil {
				return err
			}
			for _, p := range parked {
				wake = append(wake, p.FromFile)
			}
		case files.PathMoved, files.PathUnmoved:
		}
		if err := q.MarkSurveyDue(ctx, db.MarkSurveyDueParams{FileID: c.File, SurveyFrom: some(c.From.String())}); err != nil {
			return err
		}
		wake = append(wake, c.File)
	}
	s.keeper.soon(wake...)
	return nil
}

// Leaving hears that d leaves, in the doom Write while its row is still
// there. With by set, each of d's Sidecars and intents parks for by; with
// by zero, each gets removal rows at every place its bytes can sit. Either
// way, rows parked for d follow it.
func (s *Subtitles) Leaving(ctx context.Context, tx *storage.Tx[storage.Library], d files.File, by files.ID) error {
	q := db.New(tx)
	now := storage.FromTime(s.cfg.Clock())
	scs, err := sidecarsIn(ctx, q, []files.ID{d.ID})
	if err != nil {
		return err
	}
	intents, err := q.ListIntentsIn(ctx, []files.ID{d.ID})
	if err != nil {
		return err
	}
	sits, err := q.GetDoomedSits(ctx, d.ID)
	if err != nil {
		return err
	}
	places := []string{files.HoldingOf(d.ID.UUID, path.Ext(d.Path))}
	if sits != nil {
		places = append(places, *sits)
	}
	l := leaving{q: q, d: d, by: by, now: now}
	for _, sc := range scs[d.ID] {
		if err := l.leave(ctx, sc, sc.at(d.Path), places, whatSidecar); err != nil {
			return err
		}
	}
	for _, w := range intents {
		sc, err := decodeSidecar(w.Sidecar)
		if err != nil {
			return err
		}
		sc.File, sc.TailText = d.ID, w.Tail
		copy(sc.SHA256[:], w.Sha256)
		if err := l.leave(ctx, sc, besideAt(d.Path, w.Tail), places, whatIntent); err != nil {
			return err
		}
	}
	if err := l.follow(ctx); err != nil {
		return err
	}
	s.keeper.soon(d.ID)
	return nil
}

const (
	whatSidecar = "sidecar"
	whatIntent  = "intent"
)

type leaving struct {
	q   *db.Queries
	d   files.File
	by  files.ID
	now storage.Time
}

// leave parks sc, whose bytes sit or will sit at at, or writes its
// removal rows. The other places are where d's bytes may still be: its
// holding stem and where an open move took them.
func (l leaving) leave(ctx context.Context, sc Sidecar, at string, places []string, what string) error {
	others := make([]string, len(places))
	for i, p := range places {
		others[i] = besideAt(p, sc.TailText)
	}
	if l.by.IsZero() {
		return l.remove(ctx, l.d.ID, append([]string{at}, others...), sc.SHA256, "deleted")
	}
	body, err := json.Marshal(sc)
	if err != nil {
		return fmt.Errorf("park a Sidecar: %w", err)
	}
	err = l.q.InsertParked(ctx, db.InsertParkedParams{
		ID: storage.NewUUID(), FromFile: l.d.ID, ToFile: l.by, What: what, Sidecar: string(body), RootID: l.d.RootID, Path: at, Sha256: sc.SHA256[:], ParkedAt: l.now,
	})
	if err != nil {
		return err
	}
	return l.remove(ctx, l.d.ID, others, sc.SHA256, "superseded")
}

func (l leaving) remove(ctx context.Context, f files.ID, at []string, sha [32]byte, why string) error {
	for _, p := range at {
		err := l.q.InsertRemoval(ctx, db.InsertRemovalParams{ID: storage.NewUUID(), FileID: f, RootID: l.d.RootID, Path: p, Sha256: sha[:], Why: why, NextAt: l.now})
		if err != nil {
			return err
		}
	}
	return nil
}

// follow moves the rows parked for d: to by, or into removal rows of the
// files they came from.
func (l leaving) follow(ctx context.Context) error {
	if !l.by.IsZero() {
		return l.q.Repark(ctx, db.ReparkParams{By: l.by, D: l.d.ID})
	}
	parked, err := l.q.ListParkedTo(ctx, l.d.ID)
	if err != nil {
		return err
	}
	for _, p := range parked {
		if err := unpark(ctx, l.q, p, l.now); err != nil {
			return err
		}
	}
	return nil
}

// unpark turns a parked row whose replacer is gone into a removal of its
// bytes, keyed by the file they came from.
func unpark(ctx context.Context, q *db.Queries, p db.SubtitlesParked, now storage.Time) error {
	err := q.InsertRemoval(ctx, db.InsertRemovalParams{ID: storage.NewUUID(), FileID: p.FromFile, RootID: p.RootID, Path: p.Path, Sha256: p.Sha256, Why: "deleted", NextAt: now})
	if err != nil {
		return err
	}
	return q.DeleteParked(ctx, p.ID)
}

func decodeSidecar(body string) (Sidecar, error) {
	var sc Sidecar
	if err := json.Unmarshal([]byte(body), &sc); err != nil {
		return Sidecar{}, fmt.Errorf("a parked Sidecar: %w", err)
	}
	t, ok := ParseTail(sc.TailText)
	if !ok {
		return Sidecar{}, errors.New("a parked Sidecar's tail names no language: " + sc.TailText)
	}
	sc.Tail = t
	return sc, nil
}

// Languages answers what a change of p's Wanted Languages from was to now
// does, in the change Write, and unless preview makes its drops so: the
// fetched Sidecars and intents of a dropped want get removal rows, and its
// want rows go. A gained want reaches its rows through the bulk pass,
// which this wakes.
func (s *Subtitles) Languages(ctx context.Context, tx *storage.Tx[storage.Library], p targets.ProfileID, was, now []targets.WantedLanguage, preview bool) (files.SubtitlesImpact, error) {
	q := db.New(tx)
	var impact files.SubtitlesImpact
	searches := 0
	for _, g := range gainedOf(was, now) {
		n, scan, err := gainedCount(ctx, q, p, g, now)
		if err != nil {
			return files.SubtitlesImpact{}, err
		}
		searches += n
		impact.Gained = append(impact.Gained, files.GainedLanguage{Language: g.Language, Forced: g.Forced, Files: n, NeedScan: scan})
	}
	for _, d := range gainedOf(now, was) {
		n, err := q.ImpactDropped(ctx, db.ImpactDroppedParams{Language: d.Language.String(), Forced: d.Forced, ProfileID: p})
		if err != nil {
			return files.SubtitlesImpact{}, err
		}
		impact.Dropped = append(impact.Dropped, files.DroppedLanguage{Language: d.Language, Forced: d.Forced, Deleted: int(n)})
		if !preview {
			if err := s.drop(ctx, q, p, Want{d.Language, d.Forced}); err != nil {
				return files.SubtitlesImpact{}, err
			}
		}
	}
	days, err := s.days(ctx, q, searches)
	if err != nil {
		return files.SubtitlesImpact{}, err
	}
	impact.Days = days
	if !preview {
		s.soonReconcile()
	}
	return impact, nil
}

// gainedOf is the wants of now that was lacks.
func gainedOf(was, now []targets.WantedLanguage) []targets.WantedLanguage {
	var out []targets.WantedLanguage
	for _, n := range now {
		if !slices.ContainsFunc(was, func(w targets.WantedLanguage) bool { return w.Language == n.Language && w.Forced == n.Forced }) {
			out = append(out, n)
		}
	}
	return out
}

// gainedCount is how many files g searches for under p, and how many need
// their audio scanned first, by the same rules wantsOf reads. Whether a
// full want of g's language covers a forced-only g is decided here: an
// Always one covers every file, a NoAudio one the files without audio in
// the language.
func gainedCount(ctx context.Context, q *db.Queries, p targets.ProfileID, g targets.WantedLanguage, now []targets.WantedLanguage) (searches, needScan int, err error) {
	w := Want{g.Language, g.Forced}
	params := db.ImpactGainedParams{
		ProfileID: p, Key: w.key(), Want: w.String(), Base: "," + string(g.Language.Base()) + ",", Language: g.Language.String(), Forced: g.Forced,
		NoAudio: flag(g.Audio == targets.NoAudio),
	}
	if i := slices.IndexFunc(now, func(o targets.WantedLanguage) bool { return g.Forced && !o.Forced && o.Language == g.Language }); i >= 0 {
		if now[i].Audio == targets.Always || g.Audio == targets.NoAudio {
			return 0, 0, nil
		}
		params.Spoken = 1
	}
	row, err := q.ImpactGained(ctx, params)
	return int(row.Searches), int(row.NeedScan), err
}

// drop writes removal rows for w's fetched Sidecars and intents under p,
// and deletes them and w's want rows. Uploads and found Sidecars stay.
func (s *Subtitles) drop(ctx context.Context, q *db.Queries, p targets.ProfileID, w Want) error {
	now := storage.FromTime(s.cfg.Clock())
	scs, err := q.ListDropped(ctx, db.ListDroppedParams{Language: w.Language.String(), Forced: w.Forced, ProfileID: p})
	if err != nil {
		return err
	}
	var wake []files.ID
	for _, sc := range scs {
		at := besideAt(sc.Path, sc.Tail)
		if sc.Stray != nil {
			at = *sc.Stray
		}
		err := q.InsertRemoval(ctx, db.InsertRemovalParams{ID: storage.NewUUID(), FileID: sc.FileID, RootID: sc.RootID, Path: at, Sha256: sc.Sha256, Why: "dropped", NextAt: now})
		if err != nil {
			return err
		}
		var sha [32]byte
		copy(sha[:], sc.Sha256)
		if err := dropSidecar(ctx, q, Sidecar{ID: sc.ID, File: sc.FileID, Tail: Tail{Language: w.Language, Forced: w.Forced}, SHA256: sha}, now); err != nil {
			return err
		}
		wake = append(wake, sc.FileID)
	}
	intents, err := q.ListDroppedIntents(ctx, db.ListDroppedIntentsParams{Language: w.Language.String(), Forced: w.Forced, ProfileID: p})
	if err != nil {
		return err
	}
	for _, in := range intents {
		err := q.InsertRemoval(ctx, db.InsertRemovalParams{
			ID: storage.NewUUID(), FileID: in.FileID, RootID: in.RootID, Path: besideAt(in.Path, in.Tail), Sha256: in.Sha256, Why: "dropped", NextAt: now,
		})
		if err != nil {
			return err
		}
		_, err = q.DeleteIntent(ctx, db.DeleteIntentParams{FileID: in.FileID, Language: w.Language.String(), Forced: w.Forced, AttemptID: in.AttemptID})
		if err != nil {
			return err
		}
		wake = append(wake, in.FileID)
	}
	if err := q.DeleteWantsOfProfile(ctx, db.DeleteWantsOfProfileParams{Language: w.Language.String(), Forced: w.Forced, ProfileID: p}); err != nil {
		return err
	}
	s.keeper.soon(wake...)
	return nil
}

// days is how long searches take at the smallest daily download allowance
// a provider last reported, zero when none reported one.
func (s *Subtitles) days(ctx context.Context, q *db.Queries, searches int) (int, error) {
	if searches == 0 {
		return 0, nil
	}
	providers, err := q.ListProviders(ctx)
	if err != nil {
		return 0, err
	}
	least := 0
	for _, p := range providers {
		a, err := allowanceOf(p)
		if err != nil {
			return 0, err
		}
		if p.IsOn && a != nil && a.Downloads != nil && a.Downloads.Limit > 0 && (least == 0 || a.Downloads.Limit < least) {
			least = a.Downloads.Limit
		}
	}
	if least == 0 {
		return 0, nil
	}
	return (searches + least - 1) / least, nil
}

// Landed marks a file that lands fresh, in the landing Write, giving it
// its rows first: its map is built ahead of other files', and its first
// search tells History when nothing passed.
func (s *Subtitles) Landed(ctx context.Context, tx *storage.Tx[storage.Library], f files.File) error {
	if err := s.reconcileIn(ctx, tx, []files.ID{f.ID}); err != nil {
		return err
	}
	return db.New(tx).MarkFresh(ctx, f.ID)
}
