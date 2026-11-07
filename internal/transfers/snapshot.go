package transfers

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/nodes"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/internal/ledger"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// snapshot is everything judge reads, taken at one instant. Each field
// names the wake that tells the loop it changed.
type snapshot struct {
	// now: the loop's timer at the last judgment's next.
	now time.Time
	// transfers, limits, pairs, dests, levels, sources and peers are read
	// in one library Read, so the same library gives the same space
	// verdict. transfers is in (Created, ID) order. Wakes: Touch after a
	// purpose's commit, Applied, SetNodeSettings, the loop's own writes,
	// RootsMoved and roots.changed, volumes.changed, health.changed, Moved
	// and nodes.changed.
	transfers []Transfer
	limits    map[role.NodeID]int64
	pairs     map[pairKey]ledger.Pair
	dests     map[roots.ID]roots.Destination
	levels    health.Levels
	sources   map[sourceAt]source.Readiness
	peers     map[role.NodeID]nodes.Peer
	// ends: the Board's Changed.
	ends map[role.NodeID]End
	// facts and fullAt: the Facts hook, Noted.
	facts  map[role.NodeID]*tv1.Facts
	fullAt map[ID]time.Time

	routeRetry time.Duration
}

type pairKey struct{ source, dest role.NodeID }

type sourceAt struct {
	node role.NodeID
	ref  source.Ref
}

// take reads a snapshot: the library from rd, the rest from memory and
// board.
func (t *Transfers) take(ctx context.Context, rd storage.Reader[storage.Library], board Board) (snapshot, error) {
	s := snapshot{now: time.Now(), routeRetry: t.cfg.Timings.RouteRetry}
	if err := t.records(ctx, rd, &s); err != nil {
		return snapshot{}, err
	}
	if err := t.readiness(ctx, rd, &s); err != nil {
		return snapshot{}, err
	}
	q := ledger.New(rd)
	var err error
	if s.limits, err = limitsIn(ctx, q); err != nil {
		return snapshot{}, err
	}
	pairs, err := q.ListPairs(ctx)
	if err != nil {
		return snapshot{}, err
	}
	s.pairs = make(map[pairKey]ledger.Pair, len(pairs))
	for _, p := range pairs {
		s.pairs[pairKey{p.SourceNode, p.DestinationNode}] = p
	}
	if s.levels, err = health.LevelsIn(ctx, rd); err != nil {
		return snapshot{}, err
	}
	if s.peers, err = nodes.PeersIn(ctx, rd); err != nil {
		return snapshot{}, err
	}
	t.remember(&s, board)
	return s, nil
}

// records reads every Transfer and its destination into s.
func (t *Transfers) records(ctx context.Context, rd storage.Reader[storage.Library], s *snapshot) error {
	q := ledger.New(rd)
	rows, err := q.ListTransfers(ctx)
	if err != nil {
		return err
	}
	all, err := q.ListFiles(ctx)
	if err != nil {
		return err
	}
	files := map[ID][]ledger.File{}
	for _, f := range all {
		files[f.TransferID] = append(files[f.TransferID], f)
	}
	var ids []roots.ID
	for _, row := range rows {
		if !slices.Contains(ids, *row.DestinationRoot) {
			ids = append(ids, *row.DestinationRoot)
		}
	}
	if s.dests, err = t.cfg.Destinations(ctx, rd, ids); err != nil {
		return err
	}
	for _, row := range rows {
		tr, err := transferOf(row, files[row.ID], s.dests[*row.DestinationRoot].Place)
		if err != nil {
			return err
		}
		s.transfers = append(s.transfers, tr)
	}
	return nil
}

// readiness reads the readiness of every unplaced file's source into s.
func (t *Transfers) readiness(ctx context.Context, rd storage.Reader[storage.Library], s *snapshot) error {
	refs := map[role.NodeID][]source.Ref{}
	for _, tr := range s.transfers {
		for _, f := range tr.Files {
			if !f.Placed && !slices.Contains(refs[tr.Source], f.Source) {
				refs[tr.Source] = append(refs[tr.Source], f.Source)
			}
		}
	}
	s.sources = map[sourceAt]source.Readiness{}
	for node, rs := range refs {
		ready, err := t.cfg.Readiness(ctx, rd, node, rs)
		if err != nil {
			return err
		}
		for ref, r := range ready {
			s.sources[sourceAt{node, ref}] = r
		}
	}
	return nil
}

// remember reads into s what memory holds of each end of every Transfer:
// its standing from board, its facts and its full disks.
func (t *Transfers) remember(s *snapshot, board Board) {
	s.ends = map[role.NodeID]End{}
	s.facts = map[role.NodeID]*tv1.Facts{}
	for _, tr := range s.transfers {
		for _, node := range []role.NodeID{tr.Source, s.dests[rootOf(tr.Destination)].Node} {
			if e, ok := board.End(node); ok {
				s.ends[node] = e
			}
			if f, ok := t.cfg.Facts(node); ok {
				s.facts[node] = f
			}
		}
	}
	t.mu.Lock()
	s.fullAt = maps.Clone(t.fullAt)
	t.mu.Unlock()
}

func limitsIn(ctx context.Context, q *ledger.Queries) (map[role.NodeID]int64, error) {
	rows, err := q.ListNodeSettings(ctx)
	if err != nil {
		return nil, err
	}
	limits := make(map[role.NodeID]int64, len(rows))
	for _, row := range rows {
		limits[row.NodeID] = SendLimit.Value(row.SendLimit)
	}
	return limits, nil
}

// judgeIn judges a snapshot of rd and board.
func (t *Transfers) judgeIn(ctx context.Context, rd storage.Reader[storage.Library], board Board) (judgment, error) {
	s, err := t.take(ctx, rd, board)
	if err != nil {
		return judgment{}, err
	}
	return judge(s), nil
}

// judgeNow judges a snapshot of the library now.
func (t *Transfers) judgeNow(ctx context.Context, board Board) (judgment, error) {
	var j judgment
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		j, err = t.judgeIn(ctx, rd, board)
		return err
	})
	return j, err
}
