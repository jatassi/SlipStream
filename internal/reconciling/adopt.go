package reconciling

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reconciling/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// Decision is one held torrent's verdict.
type Decision string

const (
	// Adopt: unknown here, so it is added, adopted, once its metainfo
	// checks.
	Adopt Decision = "adopt"
	// Kept: recorded on the Node that holds it.
	Kept Decision = "kept"
	// Rerecord: recorded on a Node that reported without it, or was
	// written off, so it is recorded again on the Node that holds it.
	Rerecord Decision = "move"
	// Wait: recorded on a Node that hasn't reported. It is kept and
	// decided again.
	Wait Decision = "wait"
	// Conflict: the Node it is recorded on holds it too. Both keep it, and
	// it goes on the review list.
	Conflict Decision = "conflict"
	// Stranded: unknown here and nothing can give its metainfo, so it is
	// kept as it is and never adopted.
	Stranded Decision = "stranded"
)

// Said is what the Node a torrent is recorded on has said of it.
type Said int

const (
	// NotYet: that Node hasn't reported its Holdings.
	NotYet Said = iota
	// Holds: it reported the torrent, or keeps nothing back, so its record
	// stands.
	Holds
	// Lacks: it reported its Holdings without the torrent.
	Lacks
	// WrittenOff: an Admin wrote off what it held.
	WrittenOff
)

// Recorded is the Torrent the library records for a held torrent's hash.
type Recorded struct {
	Torrent downloadclients.ID
	Node    nodes.ID
	Said    Said
}

// decide is the whole adoption table: what happens to a torrent node
// holds, given what the library records for its hash and what that
// record's Node has said. content is whether node has its resume data.
func decide(here nodes.ID, content bool, rec *Recorded) Decision {
	switch {
	case rec == nil && !content:
		return Stranded
	case rec == nil:
		return Adopt
	case rec.Node == here && rec.Said != WrittenOff:
		return Kept
	case rec.Said == Lacks || rec.Said == WrittenOff:
		return Rerecord
	case rec.Said == NotYet:
		return Wait
	default:
		return Conflict
	}
}

// fetched is what a Node answered for an Adopt candidate's metainfo.
type fetched struct {
	metainfo []byte
	// stranded: the Node has no metainfo for it, or one that hashes to
	// another torrent.
	stranded bool
}

// fetchers is how many metainfo asks one page has in flight: enough to
// hide a link's round trip, few enough not to crowd the Node's other
// calls.
const fetchers = 8

// pass decides every Node's open held torrents once, one page per Write,
// and finishes each Node whose round is decided. changed is any Write
// that changed a decision.
func (s *Service) pass(ctx context.Context) (changed, retry bool, err error) {
	var ready []db.ListReadyRow
	err = s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		ready, err = db.New(r).ListReady(ctx)
		return err
	})
	if err != nil {
		return false, false, fmt.Errorf("list the Nodes to adopt: %w", err)
	}
	for _, n := range ready {
		c, again, err := s.adoptNode(ctx, n.NodeID)
		changed, retry = changed || c, retry || again
		if err != nil {
			return changed, retry, err
		}
		if again || n.Adopted {
			continue
		}
		c, err = s.finish(ctx, n.NodeID)
		changed = changed || c
		if err != nil {
			return changed, retry, err
		}
	}
	return changed, retry, nil
}

// adoptNode decides node's open held torrents, a page at a time.
func (s *Service) adoptNode(ctx context.Context, node nodes.ID) (changed, retry bool, err error) {
	after := []byte{}
	for {
		var rows []db.ReconcilingHeld
		var want map[bencode.InfoHash]downloadclients.Holding
		err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
			var err error
			if rows, err = db.New(r).ListOpenHeld(ctx, db.ListOpenHeldParams{NodeID: node, After: after, Page: int64(s.cfg.Page)}); err != nil {
				return err
			}
			want, err = s.candidates(ctx, r, node, rows)
			return err
		})
		if err != nil {
			return changed, retry, fmt.Errorf("read a Node's Holdings: %w", err)
		}
		if len(rows) == 0 {
			return changed, retry, nil
		}
		after = rows[len(rows)-1].InfoHash
		got, again := s.fetch(ctx, node, want)
		retry = retry || again
		var touched []role.NodeID
		err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			var c bool
			c, touched, err = s.applyPage(ctx, tx, node, rows, got)
			changed = changed || c
			return err
		})
		if err != nil {
			return changed, retry, fmt.Errorf("adopt a page of a Node's Holdings: %w", err)
		}
		s.touch(touched)
	}
}

// candidates is each row of rows decide adopts, read in r.
func (s *Service) candidates(ctx context.Context, r storage.Reader[file], node nodes.ID, rows []db.ReconcilingHeld) (map[bencode.InfoHash]downloadclients.Holding, error) {
	recorded, err := recordedOf(ctx, r, node, rows)
	if err != nil {
		return nil, err
	}
	out := map[bencode.InfoHash]downloadclients.Holding{}
	for _, row := range rows {
		h := bencode.InfoHash(row.InfoHash)
		if decide(node, row.Name != nil, recorded[h]) != Adopt {
			continue
		}
		held, err := holdingOf(row)
		if err != nil {
			return nil, err
		}
		out[h] = held
	}
	return out, nil
}

// fetch asks node for each candidate's metainfo, outside any Write, and
// checks it hashes to the torrent. retry is a candidate the Node couldn't
// answer for.
func (s *Service) fetch(ctx context.Context, node nodes.ID, want map[bencode.InfoHash]downloadclients.Holding) (map[bencode.InfoHash]*fetched, bool) {
	var (
		mu    sync.Mutex
		out   = map[bencode.InfoHash]*fetched{}
		retry bool
		wg    sync.WaitGroup
		slots = make(chan struct{}, fetchers)
	)
	for h, held := range want {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			raw, err := s.cfg.Metainfo(ctx, node, h, held.Client)
			f, ok := judge(h, held, raw, err)
			mu.Lock()
			defer mu.Unlock()
			if !ok {
				retry = true
				s.cfg.Log.Debug("a Node couldn't give a torrent's metainfo", "node", node, "info_hash", h, "err", err)
				return
			}
			out[h] = &f
		})
	}
	wg.Wait()
	return out, retry
}

// judge is what a Node's metainfo answer for held means. ok is false
// when it should be asked again.
func judge(h bencode.InfoHash, held downloadclients.Holding, raw []byte, err error) (f fetched, ok bool) {
	switch {
	case err == nil:
		m, err := bencode.DecodeMetainfo(raw)
		if err != nil || m.InfoHash != h {
			return fetched{stranded: true}, true
		}
		return fetched{metainfo: raw}, true
	case rpc.CodeOf(err) == rpc.CodeUnimplemented && held.Client != nil:
		return fetched{}, true
	case rpc.CodeOf(err) == rpc.CodeNotFound:
		return fetched{stranded: true}, true
	}
	return fetched{}, false
}

// applyPage decides rows again in tx and does what each decision says.
func (s *Service) applyPage(ctx context.Context, tx *storage.Tx[file], node nodes.ID, rows []db.ReconcilingHeld, got map[bencode.InfoHash]*fetched) (changed bool, touched []role.NodeID, err error) {
	q := db.New(tx)
	recorded, err := recordedOf(ctx, tx, node, rows)
	if err != nil {
		return false, nil, err
	}
	now := storage.Now()
	for _, row := range rows {
		rec := recorded[bencode.InfoHash(row.InfoHash)]
		d, ok, err := apply(ctx, tx, node, row, rec, got[bencode.InfoHash(row.InfoHash)], now)
		if err != nil {
			return changed, touched, err
		}
		if !ok {
			continue
		}
		if d == Rerecord {
			touched = append(touched, rec.Node)
		}
		var settled storage.Time
		if d != Wait {
			settled = now
		}
		decision := string(d)
		if err := q.Decide(ctx, db.DecideParams{Decision: &decision, SettledAt: settled, NodeID: node, InfoHash: row.InfoHash}); err != nil {
			return changed, touched, err
		}
		changed = true
	}
	if changed {
		touched = append(touched, node)
	}
	return changed, touched, nil
}

// apply does what row's decision says in tx, f being its fetched
// metainfo, nil when none was. ok is false when row stays as it is: an
// Adopt whose metainfo wasn't fetched, or a Wait already recorded.
func apply(ctx context.Context, tx *storage.Tx[file], node nodes.ID, row db.ReconcilingHeld, rec *Recorded, f *fetched, now storage.Time) (d Decision, ok bool, err error) {
	d = decide(node, row.Name != nil, rec)
	switch d {
	case Adopt:
		switch {
		case f == nil:
			return d, false, nil
		case f.stranded:
			return Stranded, true, nil
		}
		held, err := holdingOf(row)
		if err != nil {
			return d, false, err
		}
		_, err = downloadclients.Adopt(ctx, tx, downloadclients.Adopted{Node: node, Client: row.ClientID, Held: held, Metainfo: f.metainfo})
		return d, true, err
	case Rerecord:
		return d, true, rerecordOn(ctx, tx, rec.Torrent, node, row.ClientID)
	case Conflict:
		return d, true, db.New(tx).PutReview(ctx, db.PutReviewParams{TorrentID: rec.Torrent, NodeID: rec.Node, Reason: string(Conflict), Since: now})
	case Wait:
		return d, row.Decision == nil || *row.Decision != string(Wait), nil
	case Kept, Stranded:
	}
	return d, true, nil
}

// rerecordOn records Torrent id on client on node, off the review list if
// it was on it.
func rerecordOn(ctx context.Context, tx *storage.Tx[file], id downloadclients.ID, node nodes.ID, client *downloadclients.ClientID) error {
	if err := downloadclients.Rerecord(ctx, tx, id, node, client); err != nil {
		return err
	}
	if err := downloadclients.Release(ctx, tx, []downloadclients.ID{id}); err != nil {
		return err
	}
	return db.New(tx).DeleteReview(ctx, id)
}

// recordedOf is what the library records for each of rows' hashes, and
// what each record's Node has said, read in r.
func recordedOf(ctx context.Context, r storage.Reader[file], node nodes.ID, rows []db.ReconcilingHeld) (map[bencode.InfoHash]*Recorded, error) {
	hashes := make([]bencode.InfoHash, len(rows))
	for i, row := range rows {
		hashes[i] = bencode.InfoHash(row.InfoHash)
	}
	records, err := downloadclients.RecordedIn(ctx, r, node, hashes)
	if err != nil {
		return nil, err
	}
	q := db.New(r)
	out := make(map[bencode.InfoHash]*Recorded, len(records))
	for h, rec := range records {
		said, err := saidOf(ctx, q, rec, h)
		if err != nil {
			return nil, err
		}
		out[h] = &Recorded{Torrent: rec.ID, Node: rec.Node, Said: said}
	}
	return out, nil
}

// saidOf is what rec's Node has said of hash h: only its own Holdings
// round, once every page of it is in, says it lacks the torrent.
func saidOf(ctx context.Context, q *db.Queries, rec downloadclients.Recorded, h bencode.InfoHash) (Said, error) {
	if rec.WrittenOff {
		return WrittenOff, nil
	}
	row, err := q.GetNode(ctx, rec.Node)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return Holds, nil
	case err != nil:
		return NotYet, err
	case row.ReportedAt.IsZero() || !row.RefusedAt.IsZero():
		return NotYet, nil
	}
	holds, err := q.Holds(ctx, db.HoldsParams{NodeID: rec.Node, InfoHash: h[:]})
	if err != nil || holds {
		return Holds, err
	}
	return Lacks, nil
}

// finish is node's last step once every torrent it holds is decided: each
// torrent recorded on it that it didn't report moves to a Node that holds
// it and waits on it, or after a restore goes on the review list. Then
// one Write grants the Downloading Role and sets adopted_at.
func (s *Service) finish(ctx context.Context, node nodes.ID) (changed bool, err error) {
	var row db.ReconcilingNode
	var undecided int64
	err = s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		var err error
		if row, err = q.GetNode(ctx, node); err != nil {
			return err
		}
		undecided, err = q.CountUndecided(ctx, node)
		return err
	})
	if err != nil {
		return false, fmt.Errorf("read a Node's adoption: %w", err)
	}
	if undecided > 0 || !row.AdoptedAt.IsZero() {
		return false, nil
	}
	changed, err = s.settleNotHeld(ctx, node, Cause(row.Cause))
	if err != nil {
		return changed, err
	}
	var granted bool
	err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		cur, err := q.GetNode(ctx, node)
		if err != nil || !cur.AdoptedAt.IsZero() {
			return err
		}
		now := storage.Now()
		if granted, err = grantOnAdoption(ctx, tx, node, cur.Held); err != nil {
			return err
		}
		var at storage.Time
		if granted {
			at = now
		}
		return q.SetAdopted(ctx, db.SetAdoptedParams{At: now, GrantedAt: at, NodeID: node})
	})
	if err != nil {
		return changed, fmt.Errorf("finish a Node's adoption: %w", err)
	}
	s.cfg.Log.Info("adopted a Node's torrents", "node", node, "torrents", row.Held, "downloading_granted", granted)
	s.touch([]role.NodeID{node})
	return true, nil
}

// settleNotHeld runs notHeld over every Torrent recorded on node, a page
// per Write.
func (s *Service) settleNotHeld(ctx context.Context, node nodes.ID, cause Cause) (changed bool, err error) {
	var after int64
	for {
		var page []downloadclients.OnNode
		err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
			var err error
			page, err = downloadclients.LiveOnNode(ctx, r, node, after, s.cfg.Page)
			return err
		})
		if err != nil || len(page) == 0 {
			return changed, err
		}
		after = page[len(page)-1].Order
		var touched []role.NodeID
		err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			var c bool
			c, touched, err = notHeld(ctx, tx, node, cause, page)
			changed = changed || c
			return err
		})
		if err != nil {
			return changed, fmt.Errorf("settle a Node's torrents it didn't report: %w", err)
		}
		s.touch(touched)
	}
}

// notHeld settles each Torrent of page, recorded on node, that node didn't
// report: a Node holding it that waits on node takes its record, and
// otherwise after a restore it goes on the review list, out of every
// intent. After a pairing it stays, so node's intent asks for it again.
func notHeld(ctx context.Context, tx *storage.Tx[file], node nodes.ID, cause Cause, page []downloadclients.OnNode) (changed bool, touched []role.NodeID, err error) {
	q := db.New(tx)
	var withhold []downloadclients.ID
	now := storage.Now()
	for _, t := range page {
		holds, err := q.Holds(ctx, db.HoldsParams{NodeID: node, InfoHash: t.InfoHash[:]})
		if err != nil {
			return changed, touched, err
		}
		if holds {
			continue
		}
		waiting, err := q.ListWaiting(ctx, t.InfoHash[:])
		if err != nil {
			return changed, touched, err
		}
		if len(waiting) > 0 {
			w := waiting[0]
			if err := downloadclients.Rerecord(ctx, tx, t.ID, w.NodeID, w.ClientID); err != nil {
				return changed, touched, err
			}
			rerecord := string(Rerecord)
			if err := q.Decide(ctx, db.DecideParams{Decision: &rerecord, SettledAt: now, NodeID: w.NodeID, InfoHash: t.InfoHash[:]}); err != nil {
				return changed, touched, err
			}
			changed, touched = true, append(touched, w.NodeID, node)
			continue
		}
		if cause != Restore {
			continue
		}
		if err := q.PutReview(ctx, db.PutReviewParams{TorrentID: t.ID, NodeID: node, Reason: "not_held", Since: now}); err != nil {
			return changed, touched, err
		}
		withhold = append(withhold, t.ID)
	}
	if len(withhold) > 0 {
		changed, touched = true, append(touched, node)
	}
	return changed, touched, downloadclients.Withhold(ctx, tx, withhold)
}

// grantOnAdoption is the one place adoption gives a Role: a Node whose
// first Holdings listed torrents gets Downloading, if it lacks it, in the
// Write that adopts them, so it goes from Keeping to holding in one
// commit. granted is whether it gave it.
func grantOnAdoption(ctx context.Context, tx *storage.Tx[file], node nodes.ID, held int64) (granted bool, err error) {
	if held == 0 {
		return false, nil
	}
	return nodes.AssignIn(ctx, tx, node, role.Downloading)
}

func (s *Service) touch(ids []role.NodeID) {
	if len(ids) > 0 {
		s.cfg.Touch(ids...)
	}
}

func seconds(n int64) time.Duration { return time.Duration(n) * time.Second }
