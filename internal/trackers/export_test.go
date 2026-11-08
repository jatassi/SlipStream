package trackers

import (
	"context"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// ViewOf is h's seeding as every read shows it, live or removed.
func (t *Trackers) ViewOf(ctx context.Context, r storage.Reader[file], h bencode.InfoHash) (seeding.View, error) {
	j, err := t.judge(ctx, r, []bencode.InfoHash{h}, t.cfg.Roster)
	return j.hashes[h].verdict.View, err
}
