package subtitles

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
)

// Derived gives made, the file a Conversion made from source, the
// Sidecars source downloaded, in the landing Write after Landed. A want
// made's own tracks meet has no row, so nothing is copied for it. Each
// other want made has no Sidecar of becomes an intent to copy source's
// bytes unchanged, which the Keeper writes from beside source, and keeps
// source's next search, so only what neither has searches now.
func (s *Subtitles) Derived(ctx context.Context, tx *storage.Tx[storage.Library], source, made files.ID) error {
	q := db.New(tx)
	scs, err := sidecarsIn(ctx, q, []files.ID{source, made})
	if err != nil {
		return err
	}
	from, err := q.ListWantsIn(ctx, []files.ID{source})
	if err != nil {
		return err
	}
	to, err := q.ListWantsIn(ctx, []files.ID{made})
	if err != nil {
		return err
	}
	intents, err := q.ListIntentsIn(ctx, []files.ID{made})
	if err != nil {
		return err
	}
	now := storage.FromTime(s.cfg.Clock())
	for _, sc := range scs[source] {
		w := sc.want()
		if !downloaded(sc) || wantRow(to, w) == nil || sidecarOfWant(scs[made], w) != nil || slices.ContainsFunc(intents, intentOf(w)) {
			continue
		}
		c := sc
		c.Source, c.SiblingOf, c.Offer = FromSibling, sc.ID, nil
		body, err := json.Marshal(c)
		if err != nil {
			return fmt.Errorf("a copied Sidecar: %w", err)
		}
		err = q.InsertIntent(ctx, db.InsertIntentParams{
			FileID: made, Language: w.Language.String(), Forced: w.Forced, AttemptID: storage.NewUUID(), Tail: sc.TailText, Sha256: sc.SHA256[:],
			Sidecar: string(body), StartedAt: now,
		})
		if err != nil {
			return err
		}
		if was := wantRow(from, w); !sc.Final && was != nil && !was.DueAt.IsZero() {
			if err := q.DueNow(ctx, db.DueNowParams{Now: was.DueAt, FileID: made, Language: w.Language.String(), Forced: w.Forced}); err != nil {
				return err
			}
		}
	}
	s.keeper.soon(made)
	return nil
}

// downloaded is whether SlipStream fetched sc's bytes, and they sit home
// and timed where a copy can read them.
func downloaded(sc Sidecar) bool {
	return (sc.Source == FromProvider || sc.Source == FromSibling) && sc.Stray == "" && !sc.TimingDue
}

func intentOf(w Want) func(db.SubtitlesWrite) bool {
	return func(i db.SubtitlesWrite) bool { return i.Language == w.Language.String() && i.Forced == w.Forced }
}
