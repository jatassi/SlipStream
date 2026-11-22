package moves

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// TorrentTo is one seeding torrent and the Download Client it moves to.
type TorrentTo struct {
	Torrent downloadclients.ID
	To      Side
}

var ErrNoTorrent = errors.New("no torrent has this id")

// CheckTorrent is a torrent unit's own state.
const CheckTorrent Check = "torrent"

// torrentRead is what the planner reads for ask's torrents.
type torrentRead struct {
	sources map[downloadclients.ID]downloadclients.MoveSource
	moving  map[downloadclients.ID]bool
	// held is, per destination Node, each hash's live Torrents there.
	held  map[role.NodeID]map[bencode.InfoHash][]*downloadclients.ClientID
	nodes map[role.NodeID]string
	// clients is each destination client's Node, and its name.
	clients map[downloadclients.ClientID]downloadclients.External
}

// planTorrents plans and judges a unit per torrent of ts.
func (m *Moves) planTorrents(ctx context.Context, ts []TorrentTo) ([]PlanUnit, error) {
	if len(ts) == 0 {
		return nil, nil
	}
	var rd torrentRead
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		rd, err = readTorrents(ctx, r, ts)
		return err
	})
	if err != nil {
		return nil, err
	}
	out := make([]PlanUnit, 0, len(ts))
	for _, t := range ts {
		src, ok := rd.sources[t.Torrent]
		if !ok {
			return nil, ErrNoTorrent
		}
		out = append(out, planTorrent(t, src, rd, m.now()))
	}
	return out, nil
}

func readTorrents(ctx context.Context, r storage.Reader[file], ts []TorrentTo) (torrentRead, error) {
	rd := torrentRead{held: map[role.NodeID]map[bencode.InfoHash][]*downloadclients.ClientID{}, clients: map[downloadclients.ClientID]downloadclients.External{}}
	ids := make([]downloadclients.ID, len(ts))
	for i, t := range ts {
		ids[i] = t.Torrent
	}
	var err error
	if rd.sources, err = downloadclients.MoveSources(ctx, r, ids); err != nil {
		return rd, err
	}
	if rd.moving, err = torrentsMoving(ctx, r, ids); err != nil {
		return rd, err
	}
	if rd.nodes, err = nodeNamesIn(ctx, r); err != nil {
		return rd, err
	}
	hashes := map[role.NodeID][]bencode.InfoHash{}
	for _, t := range ts {
		hashes[t.To.Node] = append(hashes[t.To.Node], rd.sources[t.Torrent].InfoHash)
		if c := t.To.Client; c != nil {
			e, err := downloadclients.ExternalOf(ctx, r, *c)
			if err != nil && !errors.Is(err, downloadclients.ErrNoClient) {
				return rd, err
			}
			if err == nil {
				rd.clients[*c] = e
			}
		}
	}
	for node, hs := range hashes {
		if rd.held[node], err = downloadclients.HeldOn(ctx, r, node, hs); err != nil {
			return rd, err
		}
	}
	return rd, nil
}

// torrentsMoving is which of ids an open unit moves or keeps a copy of,
// read in r.
func torrentsMoving(ctx context.Context, r storage.Reader[file], ids []downloadclients.ID) (map[downloadclients.ID]bool, error) {
	rows, err := db.New(r).ListOpenUnitsOfTorrents(ctx, uuidsOf(ids))
	out := make(map[downloadclients.ID]bool, len(rows))
	for _, row := range rows {
		out[downloadclients.ID{UUID: row.TorrentID}] = true
	}
	return out, err
}

// planTorrent is t's unit, its payload files as planned at now.
func planTorrent(t TorrentTo, src downloadclients.MoveSource, rd torrentRead, now time.Time) PlanUnit {
	from := Side{Node: src.Node}
	if src.Client != nil {
		from.Client = &src.Client.ID
	}
	u := PlanUnit{Torrent: &TorrentSubject{Torrent: t.Torrent, From: from, To: t.To}, Label: src.Name, Lane: torrentLane(from, t.To)}
	var bad bool
	for i, f := range src.Files {
		rel, err := storing.ParseRel(f.Path)
		bad = bad || err != nil
		u.Files = append(u.Files, File{Position: i, Kind: KindPayload, Sits: rel, Index: f.Index, Size: f.Size, Seen: storing.Seen{Modified: now}})
	}
	if from.Node != t.To.Node {
		u.Bytes = src.Size
	}
	u.Verdict = torrentVerdict(t, src, rd, bad)
	return u
}

// torrentLane is a torrent's lane: its Node pair, or its client pair on
// one Node.
func torrentLane(from, to Side) Lane {
	if from.Node != to.Node {
		return laneOf(End{Node: from.Node}, End{Node: to.Node}, "", "")
	}
	key := func(s Side) string {
		if s.Client == nil {
			return "built-in"
		}
		return s.Client.String()
	}
	return Lane("vol:" + from.Node.String() + ":" + key(from) + ">" + key(to))
}

// torrentVerdict judges one torrent's Move. Each reason but one is red:
// a torrent either moves whole or stays. A private torrent out of an
// external client into the built-in one is amber, its Hit and run window
// counted from now.
func torrentVerdict(t TorrentTo, src downloadclients.MoveSource, rd torrentRead, badPath bool) Verdict {
	red := func(c Check, code, words string) Verdict {
		return Verdict{Color: Red, Reasons: []Reason{{Check: c, Color: Red, Code: code, Words: words}}}
	}
	same := src.Node == t.To.Node && (src.Client == nil) == (t.To.Client == nil) && (t.To.Client == nil || src.Client.ID == *t.To.Client)
	switch {
	case src.Removed:
		return red(CheckTorrent, "torrent_removed", "It was removed from its Download Client.")
	case src.Arriving:
		return red(CheckTorrent, "arriving", "It's the copy another Move is placing.")
	case rd.moving[t.Torrent]:
		return red(CheckMoving, "already_moving", "It's already moving in another Move.")
	case !src.Metainfo:
		return red(CheckTorrent, "no_metainfo", "SlipStream doesn't have its torrent file, so no other client can add it.")
	case src.Withheld:
		return red(CheckTorrent, "on_review", "It's on the review list. Review it first.")
	case !src.Complete:
		return red(CheckTorrent, "incomplete", "It hasn't finished downloading.")
	case badPath:
		return red(CheckNames, "bad_path", "SlipStream can't place a file of it under its name.")
	case same:
		return red(CheckDestination, "same_client", "It's already in this Download Client.")
	case rd.nodes[t.To.Node] == "":
		return red(CheckDestination, "no_node", "The destination Node is gone. Pick another Download Client.")
	}
	if c := t.To.Client; c != nil {
		e, ok := rd.clients[*c]
		if !ok || e.Node != t.To.Node {
			return red(CheckDestination, "no_client", "The destination Download Client is gone. Pick another.")
		}
	}
	held := slices.DeleteFunc(slices.Clone(rd.held[t.To.Node][src.InfoHash]), func(c *downloadclients.ClientID) bool {
		return src.Node == t.To.Node && (c == nil) == (src.Client == nil) && (c == nil || *c == src.Client.ID)
	})
	if len(held) > 0 {
		return red(CheckDestination, "already_held", "A Download Client on "+rd.nodes[t.To.Node]+" already holds this torrent.")
	}
	if src.Private && src.Client != nil && t.To.Client == nil {
		return Verdict{Color: Amber, Reasons: []Reason{{Check: CheckHitAndRun, Color: Amber, Code: "window_starts", Words: "SlipStream starts counting this Hit and run window from now."}}}
	}
	return Verdict{}
}

// recheckTorrent is the reason a torrent unit planned as pu can't move
// now, read in tx.
func recheckTorrent(ctx context.Context, tx *storage.Tx[file], pu PlanUnit, now time.Time) (Reason, bool, error) {
	ts := []TorrentTo{{Torrent: pu.Torrent.Torrent, To: pu.Torrent.To}}
	rd, err := readTorrents(ctx, tx, ts)
	if err != nil {
		return Reason{}, false, err
	}
	src, ok := rd.sources[pu.Torrent.Torrent]
	if !ok {
		return Reason{Check: CheckTorrent, Color: Red, Code: "torrent_removed", Words: "It was removed from its Download Client."}, true, nil
	}
	v := planTorrent(ts[0], src, rd, now).Verdict
	return worst(v), v.Color == Red, nil
}
