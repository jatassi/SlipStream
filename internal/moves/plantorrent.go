package moves

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
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
	// owing is each built-in source whose Tracker needs more seeding of
	// it: a Move into an external client stops SlipStream counting it.
	owing map[downloadclients.ID]owing
}

// owing is a torrent's Tracker and what its Hit and run window still
// needs.
type owing struct {
	tracker string
	owes    seeding.Owes
}

// planTorrents plans and judges a unit per torrent of ts. A drain's
// unfinished torrents move once they finish.
func (m *Moves) planTorrents(ctx context.Context, ts []TorrentTo, drain bool) ([]PlanUnit, error) {
	if len(ts) == 0 {
		return nil, nil
	}
	var out []PlanUnit
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		out, err = m.planTorrentsIn(ctx, r, ts, drain)
		return err
	})
	return out, err
}

// planTorrentsIn is planTorrents in r.
func (m *Moves) planTorrentsIn(ctx context.Context, r storage.Reader[file], ts []TorrentTo, drain bool) ([]PlanUnit, error) {
	rd, err := readTorrents(ctx, r, ts)
	if err != nil {
		return nil, err
	}
	if rd.owing, err = m.owingOf(ctx, r, ts, rd); err != nil {
		return nil, err
	}
	out := make([]PlanUnit, 0, len(ts))
	for _, t := range ts {
		src, ok := rd.sources[t.Torrent]
		if !ok {
			return nil, ErrNoTorrent
		}
		out = append(out, planTorrent(t, src, rd, m.now(), drain))
	}
	return out, torrentRooms(ctx, r, out)
}

// torrentRooms reds each unit of us that copies into a built-in client
// whose downloads volume has no room left once the units before it took
// theirs.
func torrentRooms(ctx context.Context, r storage.Reader[file], us []PlanUnit) error {
	room := map[role.NodeID]*int64{}
	for i, u := range us {
		to := u.Torrent.To
		if u.Bytes == 0 || to.Client != nil || u.Verdict.Color == Red {
			continue
		}
		left, ok := room[to.Node]
		if !ok {
			n, known, err := downloadclients.DownloadRoom(ctx, r, to.Node)
			if err != nil {
				return err
			}
			if known {
				left = &n
			}
			room[to.Node] = left
		}
		switch {
		case left == nil:
		case *left < u.Bytes:
			reason := Reason{Check: CheckSpace, Color: Red, Code: "space_error", Words: "The destination's downloads folder has no room for it. Free up space there, then preview again."}
			us[i].Verdict = Verdict{Color: Red, Reasons: []Reason{reason}}
		default:
			*left -= u.Bytes
		}
	}
	return nil
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
	if err := titled(ctx, r, rd.sources, nil); err != nil {
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

// owingOf is what each torrent of ts moving out of the built-in client
// into an external one still owes its Tracker, read in r.
func (m *Moves) owingOf(ctx context.Context, r storage.Reader[file], ts []TorrentTo, rd torrentRead) (map[downloadclients.ID]owing, error) {
	var hashes []bencode.InfoHash
	for _, t := range ts {
		if src, ok := rd.sources[t.Torrent]; ok && src.Client == nil && src.Private && t.To.Client != nil {
			hashes = append(hashes, src.InfoHash)
		}
	}
	if len(hashes) == 0 {
		return nil, nil
	}
	rows, err := m.cfg.Clients.Rows(ctx, r, downloadclients.Filter{Hashes: hashes})
	out := map[downloadclients.ID]owing{}
	for _, row := range rows {
		if row.Tracker != nil && row.Seeding.Owes != nil {
			out[row.ID] = owing{tracker: row.Tracker.Name, owes: *row.Seeding.Owes}
		}
	}
	return out, err
}

// owedWords is what o still needs, in the Hit and run window's words.
func owedWords(o owing) string {
	const day = 24 * time.Hour
	switch d := o.owes.Time; {
	case d >= day:
		n := int((d + day - 1) / day)
		return fmt.Sprintf("It needs %d more %s of seeding on %s.", n, plural(n, "day", "days"), o.tracker)
	case d > 0:
		n := int((d + time.Hour - 1) / time.Hour)
		return fmt.Sprintf("It needs %d more %s of seeding on %s.", n, plural(n, "hour", "hours"), o.tracker)
	}
	return "It needs more seeding on " + o.tracker + "."
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

// planTorrent is t's unit, its payload files as planned at now. A drain's
// unfinished torrent has none yet: it takes them when it takes its lane.
func planTorrent(t TorrentTo, src downloadclients.MoveSource, rd torrentRead, now time.Time, drain bool) PlanUnit {
	from := Side{Node: src.Node}
	if src.Client != nil {
		from.Client = &src.Client.ID
	}
	u := PlanUnit{Torrent: &TorrentSubject{Torrent: t.Torrent, From: from, To: t.To}, Label: src.Name, Lane: torrentLane(from, t.To)}
	var bad bool
	u.Files, bad = payloadFiles(src, now)
	if from.Node != t.To.Node {
		u.Bytes = src.Size
	}
	u.Verdict = torrentVerdict(t, src, rd, bad, drain)
	return u
}

// payloadFiles is src's completed payload as a unit's files at now, and
// whether a path of it can't be placed.
func payloadFiles(src downloadclients.MoveSource, now time.Time) ([]File, bool) {
	var out []File
	bad := false
	for i, f := range src.Files {
		rel, err := storing.ParseRel(f.Path)
		bad = bad || err != nil
		out = append(out, File{Position: i, Kind: KindPayload, Sits: rel, Index: f.Index, Size: f.Size, Seen: storing.Seen{Modified: now}})
	}
	return out, bad
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

// torrentVerdict judges one torrent's Move. A torrent either moves whole
// or stays, so only its notes are amber: a drain's torrent still
// downloading, and a Hit and run window that starts or stops counting.
func torrentVerdict(t TorrentTo, src downloadclients.MoveSource, rd torrentRead, badPath, drain bool) Verdict {
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
	case !src.Complete && !drain:
		return red(CheckTorrent, "incomplete", "It hasn't finished downloading.")
	case badPath:
		return red(CheckNames, "bad_path", `One of its file paths has a backslash, a drive letter or a ".." folder, so SlipStream can't copy it.`)
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
	var notes []Reason
	if !src.Complete {
		notes = append(notes, Reason{Check: CheckTorrent, Color: Amber, Code: "downloading",
			Words: rd.nodes[src.Node] + " is finishing its work. This torrent moves once it finishes downloading."})
	}
	if src.Private && src.Client != nil && t.To.Client == nil {
		notes = append(notes, Reason{Check: CheckHitAndRun, Color: Amber, Code: "window_starts", Words: "SlipStream starts counting this Hit and run window from now."})
	}
	if o, ok := rd.owing[t.Torrent]; ok {
		words := owedWords(o) + " " + rd.clients[*t.To.Client].Name + " on " + rd.nodes[t.To.Node] +
			" follows its own rules, and SlipStream stops counting this Hit and run window."
		notes = append(notes, Reason{Check: CheckHitAndRun, Color: Amber, Code: "window_stops", Words: words})
	}
	if len(notes) == 0 {
		return Verdict{}
	}
	return Verdict{Color: Amber, Reasons: notes}
}

// recheckTorrent is the reason a torrent unit planned as pu can't move
// now, read in tx.
func recheckTorrent(ctx context.Context, tx *storage.Tx[file], pu PlanUnit, now time.Time, drain bool) (Reason, bool, error) {
	ts := []TorrentTo{{Torrent: pu.Torrent.Torrent, To: pu.Torrent.To}}
	rd, err := readTorrents(ctx, tx, ts)
	if err != nil {
		return Reason{}, false, err
	}
	src, ok := rd.sources[pu.Torrent.Torrent]
	if !ok {
		return Reason{Check: CheckTorrent, Color: Red, Code: "torrent_removed", Words: "It was removed from its Download Client."}, true, nil
	}
	v := planTorrent(ts[0], src, rd, now, drain).Verdict
	return worst(v), v.Color == Red, nil
}
