package subtitles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
)

// errStale is an attempt's Write that found its lease gone: a doom, a
// language drop or a restart took the want. The attempt ends with no
// effect.
var errStale = errors.New("the attempt's lease on its want is gone")

// intend records c as w's intent under attempt a, in one Write that holds
// only while a still holds w's lease, then hands c's bytes to the Keeper.
// The intent replaces the bytes of w's Sidecar when they sit at the name
// c's go to.
func (s *Subtitles) intend(ctx context.Context, w WantKey, a storage.UUID, c Checked) error {
	sc := c.sidecar
	sc.File = w.File
	body, err := json.Marshal(sc)
	if err != nil {
		return fmt.Errorf("an intent's Sidecar: %w", err)
	}
	err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		held, err := q.HoldsLease(ctx, db.HoldsLeaseParams{FileID: w.File, Language: w.Language.String(), Forced: w.Forced, AttemptID: a})
		if err != nil {
			return err
		}
		if held == 0 {
			return errStale
		}
		scs, err := sidecarsIn(ctx, q, []files.ID{w.File})
		if err != nil {
			return err
		}
		var replaces []byte
		if old := sidecarOfWant(scs[w.File], w.Want); old != nil && old.Stray == "" && old.TailText == sc.TailText {
			replaces = old.SHA256[:]
		}
		return q.InsertIntent(ctx, db.InsertIntentParams{
			FileID: w.File, Language: w.Language.String(), Forced: w.Forced, AttemptID: a, Tail: sc.TailText, Sha256: sc.SHA256[:],
			Sidecar: string(body), Replaces: replaces, StartedAt: storage.FromTime(s.cfg.Clock()),
		})
	})
	if err != nil {
		return err
	}
	s.keeper.bytes.hand(w.File, a, c.bytes)
	s.keeper.soon(w.File)
	return nil
}

// intentBytes is each intent's bytes, by file and attempt, while the
// process that checked them lives. An intent without them settles from a
// survey.
type intentBytes struct {
	mu sync.Mutex
	of map[files.ID]map[storage.UUID][]byte
}

func (b *intentBytes) hand(f files.ID, a storage.UUID, bytes []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.of == nil {
		b.of = map[files.ID]map[storage.UUID][]byte{}
	}
	if b.of[f] == nil {
		b.of[f] = map[storage.UUID][]byte{}
	}
	b.of[f][a] = bytes
}

// held is f's intents' bytes, keeping only those of intents, so bytes
// whose intent settled or went are let go.
func (b *intentBytes) held(f files.ID, intents []db.SubtitlesWrite) map[storage.UUID][]byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := map[storage.UUID][]byte{}
	for _, w := range intents {
		if bytes, ok := b.of[f][w.AttemptID]; ok {
			out[w.AttemptID] = bytes
		}
	}
	if len(out) == 0 {
		delete(b.of, f)
	} else {
		b.of[f] = out
	}
	return maps.Clone(out)
}
