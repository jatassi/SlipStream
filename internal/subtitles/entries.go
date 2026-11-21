package subtitles

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
)

// Priority is who asked a want to search ahead of the schedule.
type Priority uint8

const (
	ByAdmin Priority = iota + 1
	ByReport
)

var priorities = map[Priority]struct {
	name string
	rank int64
}{ByAdmin: {"admin", 1}, ByReport: {"report", 0}}

// ErrNoSubtitle is a Mark bad of a want no Sidecar meets.
var ErrNoSubtitle = errors.New("no subtitle of that language and kind sits beside the file")

// SearchNow queues w on each of fs ahead of the schedule, in tx, at p's
// rank, and resets its gap to the interval. A file whose audio isn't
// scanned yet scans ahead of the others and queues once it is. The drain
// wakes after tx commits.
func (s *Subtitles) SearchNow(ctx context.Context, tx *storage.Tx[storage.Library], fs []files.ID, w Want, p Priority) error {
	every, err := s.cfg.Tasks.EveryIn(ctx, tx, TaskID)
	if err != nil {
		return err
	}
	q, now, pr := db.New(tx), storage.FromTime(s.cfg.Clock()), priorities[p]
	for _, f := range fs {
		n, err := q.SearchNow(ctx, db.SearchNowParams{
			Priority: &pr.name, Now: now, GapSeconds: int64(Every.Value(every) / time.Second), Rank: &pr.rank,
			FileID: f, Language: w.Language.String(), Forced: w.Forced,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("file %s: %w", f, ErrNotWanted)
		}
		if err := q.RankMap(ctx, db.RankMapParams{Rank: pr.rank, FileID: f}); err != nil {
			return err
		}
	}
	s.maps.soon()
	s.drain.soon()
	return nil
}

// MarkBad removes, in tx, each file's Sidecar of w by its bytes, records
// it as marked_bad in Tried before so no search takes it again, and
// searches for another at the Admin's rank. A found or uploaded
// Sidecar's bytes go too.
func (s *Subtitles) MarkBad(ctx context.Context, tx *storage.Tx[storage.Library], fs []files.ID, w Want, by entry.Actor) error {
	q, now := db.New(tx), s.cfg.Clock()
	subjects, err := s.cfg.Files.SubjectsOf(ctx, tx, fs)
	if err != nil {
		return err
	}
	scs, err := sidecarsIn(ctx, q, fs)
	if err != nil {
		return err
	}
	for _, f := range fs {
		sub, ok := subjectOf(subjects, f)
		if !ok {
			return fmt.Errorf("file %s: %w", f, files.ErrNoFile)
		}
		sc := sidecarOfWant(scs[f], w)
		if sc == nil {
			return fmt.Errorf("file %s: %w", f, ErrNoSubtitle)
		}
		if err := s.markBad(ctx, tx, q, sub, *sc, now, by); err != nil {
			return err
		}
	}
	return s.SearchNow(ctx, tx, fs, w, ByAdmin)
}

func (s *Subtitles) markBad(ctx context.Context, tx *storage.Tx[storage.Library], q *db.Queries, sub files.Subject, sc Sidecar, now time.Time, by entry.Actor) error {
	at := sc.at(sub.Path.String())
	if err := s.keeper.removal(ctx, q, sub.ID, sub.Root, at, sc.SHA256, "marked_bad"); err != nil {
		return err
	}
	if err := dropSidecar(ctx, q, sc, storage.FromTime(now)); err != nil {
		return err
	}
	if err := putTry(ctx, q, WantKey{File: sub.ID, Want: sc.want()}, markedBad(sc), now); err != nil {
		return err
	}
	rel, err := storing.ParseRel(at)
	if err != nil {
		return err
	}
	if err := s.cfg.MediaServers.SidecarsChanged(ctx, tx, []mediaservers.SidecarChange{{Kind: mediaservers.SidecarRemoved, Root: sub.Root, Path: rel, Sidecar: sc.ID}}); err != nil {
		return err
	}
	work, err := entry.WorkIn(ctx, tx, s.cfg.Works, sub.Work)
	if err != nil {
		return err
	}
	if err := markedBadRow.Record(ctx, tx, s.cfg.History, MarkedBadFacts{Work: work, File: sub.Path.String(), Subtitle: s.factOf(sc), By: by}); err != nil {
		return err
	}
	s.keeper.soon(sub.ID)
	return nil
}

// markedBad is sc as Tried before keeps it marked bad: by its offer, the
// sibling it was copied from, or its bytes.
func markedBad(sc Sidecar) try {
	t := try{key: hex.EncodeToString(sc.SHA256[:]), outcome: TriedMarkedBad}
	switch {
	case sc.Offer != nil:
		t.key, t.provider, t.release = sc.Offer.Key, sc.Offer.Provider, sc.Offer.Release
	case sc.Source == FromSibling:
		t.key = siblingKey(sc.SiblingOf)
	}
	if sc.Fit != nil {
		t.match = new(int64(sc.Fit.Match))
	}
	return t
}

func subjectOf(subjects []files.Subject, f files.ID) (files.Subject, bool) {
	for _, sub := range subjects {
		if sub.ID == f {
			return sub, true
		}
	}
	return files.Subject{}, false
}

// NoGapCap is Config.GapCap until S47 (Subtitle reports) caps a want's
// gap.
func NoGapCap(context.Context, storage.Reader[storage.Library], WantKey) (time.Duration, bool) {
	return 0, false
}
