package moves

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// linkTorrents puts before each Work unit of us that crosses Nodes the
// unit of a seeding torrent its files are hard links to, moving to the
// destination Node's "New Grabs go to" client. The payload crosses once:
// the Work unit places its linked files from it, via arriving, and sends
// only the rest.
func (m *Moves) linkTorrents(ctx context.Context, us []PlanUnit) ([]PlanUnit, error) {
	var ws []works.WorkID
	for _, u := range us {
		if u.Torrent == nil && u.Verdict.Color != Red && u.Subject.From.Node != u.Subject.To.Node {
			ws = append(ws, u.Subject.Work)
		}
	}
	if len(ws) == 0 {
		return us, nil
	}
	var of map[works.WorkID][]downloadclients.ID
	var srcs map[downloadclients.ID]downloadclients.MoveSource
	tos := map[role.NodeID]*downloadclients.ClientID{}
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if of, err = grabs.TorrentsOfWorks(ctx, r, ws); err != nil {
			return err
		}
		var ids []downloadclients.ID
		for _, ts := range of {
			for _, t := range ts {
				ids = appendOnce(ids, t)
			}
		}
		if srcs, err = downloadclients.MoveSources(ctx, r, ids); err != nil {
			return err
		}
		for _, u := range us {
			if _, ok := tos[u.Subject.To.Node]; !ok && u.Torrent == nil {
				if tos[u.Subject.To.Node], err = downloadclients.GrabsToOn(ctx, r, u.Subject.To.Node); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	asked := map[downloadclients.ID]bool{}
	for _, u := range us {
		if u.Torrent != nil {
			asked[u.Torrent.Torrent] = true
		}
	}
	var out []PlanUnit
	for _, u := range us {
		if u.Torrent == nil {
			if t, ok, err := m.linkOne(ctx, &u, of[u.Subject.Work], srcs, tos[u.Subject.To.Node], asked); err != nil {
				return nil, err
			} else if ok {
				out = append(out, t)
			}
		}
		out = append(out, u)
	}
	return out, nil
}

// linkOne links Work unit u to the first of ts that seeds complete on its
// source Node and whose payload holds one of u's files, answering the
// torrent's unit. A torrent already in the plan isn't taken again.
func (m *Moves) linkOne(ctx context.Context, u *PlanUnit, ts []downloadclients.ID, srcs map[downloadclients.ID]downloadclients.MoveSource,
	to *downloadclients.ClientID, asked map[downloadclients.ID]bool) (PlanUnit, bool, error) {
	for _, id := range ts {
		src := srcs[id]
		if asked[id] || src.Node != u.Subject.From.Node || src.Removed || src.Arriving || !src.Complete {
			continue
		}
		req := storing.ProbeMove{}
		for _, f := range src.Files {
			req.Payloads = append(req.Payloads, storing.Payload{InfoHash: src.InfoHash, FileIndex: f.Index})
		}
		probed, err := m.probePayloads(ctx, src.Node, req)
		if err != nil {
			return PlanUnit{}, false, err
		}
		linked := false
		for i, f := range u.Files {
			for p, got := range probed.Payloads {
				if f.Kind == KindLibrary && !f.Seen.ID.IsZero() && got.Seen.ID == f.Seen.ID && p.InfoHash == src.InfoHash {
					u.Files[i].Via, u.Files[i].Index = ViaArriving, p.FileIndex
					u.Bytes -= f.Size
					linked = true
				}
			}
		}
		if !linked {
			continue
		}
		var t []PlanUnit
		err = m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
			var err error
			t, err = m.planTorrentsIn(ctx, r, []TorrentTo{{Torrent: id, To: Side{Node: u.Subject.To.Node, Client: to}}}, false)
			return err
		})
		if err != nil {
			return PlanUnit{}, false, err
		}
		if t[0].Verdict.Color == Red {
			u.Files, u.Bytes = fromLibrary(u.Files), u.Bytes+arrivingBytes(u.Files)
			continue
		}
		asked[id], u.Linked = true, &id
		return t[0], true, nil
	}
	return PlanUnit{}, false, nil
}

// probePayloads reads req's payload files on node, under probe_deadline.
func (m *Moves) probePayloads(ctx context.Context, node role.NodeID, req storing.ProbeMove) (storing.Probed, error) {
	ctx, cancel := context.WithTimeout(ctx, m.cfg.Timings.ProbeDeadline)
	defer cancel()
	return m.cfg.Disk.ProbeMove(ctx, node, req)
}

func arrivingBytes(fs []File) int64 {
	var n int64
	for _, f := range fs {
		if f.Via == ViaArriving {
			n += f.Size
		}
	}
	return n
}

// fromLibrary is fs with every unplaced file via arriving from the
// library instead.
func fromLibrary(fs []File) []File {
	out := slices.Clone(fs)
	for i := range out {
		if out[i].Via == ViaArriving && out[i].Placed == nil {
			out[i].Via, out[i].Index = ViaLibrary, 0
		}
	}
	return out
}

// releasePair, in the Write that ends torrent unit t's Place, has the
// Work unit paired with it take its lane.
func releasePair(ctx context.Context, tx *storage.Tx[file], t UnitID, now time.Time) error {
	return db.New(tx).ReleasePair(ctx, db.ReleasePairParams{Now: storage.FromTime(now), TorrentUnit: t})
}

// unpair, in the Write that fails or cancels torrent unit t, has its
// paired Work unit send the files it hasn't placed from the payload from
// its library instead, so t never takes the payload from under it.
func unpair(ctx context.Context, tx *storage.Tx[file], t UnitID, now time.Time) error {
	q := db.New(tx)
	if err := q.UnpairFiles(ctx, t); err != nil {
		return err
	}
	return q.Unpair(ctx, db.UnpairParams{Now: storage.FromTime(now), TorrentUnit: t})
}

// arrivingSource is where file f of Work unit u lies at its destination
// Node: the payload file its paired torrent unit placed there, read in r.
func arrivingSource(ctx context.Context, r storage.Reader[file], u Unit, f File) (storing.Arriving, error) {
	t, err := load(ctx, r, *u.PairsWith)
	if err != nil {
		return storing.Arriving{}, err
	}
	src, err := sourceIn(ctx, r, t)
	if err != nil {
		return storing.Arriving{}, err
	}
	a := storing.Arriving{InfoHash: src.InfoHash}
	if src.ContentType != nil {
		a.ContentType = string(*src.ContentType)
	}
	if c := t.Torrent.To.Client; c != nil {
		a.Client = new([16]byte(c.Bytes()))
	}
	i := slices.IndexFunc(src.Files, func(p downloadclients.FileOf) bool { return p.Index == f.Index })
	a.Path, err = storing.ParseRel(src.Files[i].Path)
	return a, err
}
