package downloadclients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/works"
)

// MovingTo is where a Move takes a torrent: a Node and its client, nil
// for the built-in one.
type MovingTo struct {
	Node   role.NodeID
	Client *ClientID
}

// Move is a Torrent's Move as its row shows it.
type Move struct {
	Unit, Batch storage.UUID
	Phase       string
	// Waiting is empty while it waits for nothing.
	Waiting    string
	ToNode     role.NodeID
	ToNodeName string
	// Words is its sentence, such as "Copying to den".
	Words string
}

// Moving is what downloadclients asks of S48 (Moves).
type Moving interface {
	// Moving is, for each of ids that is the source of a Move before its
	// switch, where it is moving.
	Moving(ctx context.Context, r storage.Reader[file], ids []ID) (map[ID]MovingTo, error)
	// MovesOf is the Move of each of ids that moves or keeps a failed
	// Move's copy.
	MovesOf(ctx context.Context, r storage.Reader[file], ids []ID) (map[ID]Move, error)
	// UsesClient is how many Moves that haven't ended move a torrent into
	// or out of client.
	UsesClient(ctx context.Context, r storage.Reader[file], client ClientID) (int64, error)
}

// NoMoving moves nothing.
var NoMoving Moving = noMoving{}

type noMoving struct{}

func (noMoving) Moving(context.Context, storage.Reader[file], []ID) (map[ID]MovingTo, error) {
	return nil, nil
}

func (noMoving) MovesOf(context.Context, storage.Reader[file], []ID) (map[ID]Move, error) {
	return nil, nil
}

func (noMoving) UsesClient(context.Context, storage.Reader[file], ClientID) (int64, error) {
	return 0, nil
}

// Arriving is a Move's destination Torrent, a new row of From's metainfo
// that stays out of every reader but the Move's until its switch.
type Arriving struct {
	// Unit is the Move's unit.
	Unit [16]byte
	From ID
	Node role.NodeID
	// Client is nil for the built-in client.
	Client *ClientID
	// Folder is the Client folder an external client adds the torrent
	// in, "" for the built-in client.
	Folder string
}

// InsertArriving adds a's Torrent in tx, paused, with the source's origin
// and totals. The caller Touches a.Node after commit: an external one adds
// it from the files the Move placed.
func InsertArriving(ctx context.Context, tx *storage.Tx[file], a Arriving) (ID, error) {
	q := db.New(tx)
	order, err := q.NextOrder(ctx)
	if err != nil {
		return ID{}, err
	}
	id := ID{UUID: storage.NewUUID()}
	var folder *string
	if a.Folder != "" {
		folder = &a.Folder
	}
	err = q.InsertArriving(ctx, db.InsertArrivingParams{
		ID: id, NodeID: a.Node, ClientID: a.Client, SaveFolder: folder, AddedOrder: order, AddedAt: storage.Now(),
		ArrivingUnit: a.Unit[:], FromID: a.From,
	})
	return id, err
}

// WatchIn has id's Node report when its trackers last answered and when
// they are next due, or stop. A watch that stops forgets what it saw. The
// caller Touches id's Node after commit.
func (c *Clients) WatchIn(ctx context.Context, tx *storage.Tx[file], id ID, on bool) error {
	if on {
		return db.New(tx).Watch(ctx, id)
	}
	return db.New(tx).Unwatch(ctx, id)
}

// StartIn asks arriving Torrent dst to start at now, and to give up on its
// own at until, with src's totals after its pause. The ask is kept to the
// millisecond its Node echoes.
func (c *Clients) StartIn(ctx context.Context, tx *storage.Tx[file], dst, src ID, now, until time.Time) error {
	asked := storage.FromTime(now.Truncate(time.Millisecond))
	n, err := db.New(tx).AskStart(ctx, db.AskStartParams{FromID: src, AskedAt: asked, UntilAt: storage.FromTime(until), ID: dst})
	if err == nil && n == 0 {
		err = fmt.Errorf("torrent %s is no Move's destination", dst)
	}
	return err
}

// RollbackIn has a Move's handoff seed from src again: dst is paused with
// its ask, its result and any fault cleared, and src is wanted started
// with its watch off.
func (c *Clients) RollbackIn(ctx context.Context, tx *storage.Tx[file], src, dst ID) error {
	q := db.New(tx)
	if err := q.RollBackArriving(ctx, dst); err != nil {
		return err
	}
	if err := q.ResumeSource(ctx, src); err != nil {
		return err
	}
	return q.Unwatch(ctx, src)
}

// ResumeIn has a Move's source seed again in tx after its stop went
// unseen, its watch kept for the next guard.
func (c *Clients) ResumeIn(ctx context.Context, tx *storage.Tx[file], src ID) error {
	return db.New(tx).ResumeSource(ctx, src)
}

// SwitchIn makes arriving Torrent dst the torrent's live one in tx: src
// is removed moved with its files first, so it holds the hash no more,
// then dst takes the larger of its totals and src's final ones.
func (c *Clients) SwitchIn(ctx context.Context, tx *storage.Tx[file], src, dst ID) error {
	q := db.New(tx)
	n, err := q.RemoveMoved(ctx, db.RemoveMovedParams{At: storage.Now(), ID: src})
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("torrent %s was removed before its Move's switch", src)
	}
	n, err = q.Arrive(ctx, db.ArriveParams{FromID: src, ID: dst})
	if err == nil && n == 0 {
		err = fmt.Errorf("torrent %s is no Move's destination", dst)
	}
	return err
}

// Arrival is what a Move's destination Node reported of its Torrent.
type Arrival string

const (
	// ArrivalComplete is a complete add whose check found every piece.
	ArrivalComplete    Arrival = "complete"
	ArrivalCheckFailed Arrival = "check_failed"
	// ArrivalStarted is a start a tracker answered within the reply wait.
	ArrivalStarted Arrival = "started"
	ArrivalRefused Arrival = "refused"
)

var arrivalFromWire = map[dlv1.ArrivalResult]Arrival{
	dlv1.ArrivalResult_ARRIVAL_RESULT_COMPLETE:     ArrivalComplete,
	dlv1.ArrivalResult_ARRIVAL_RESULT_CHECK_FAILED: ArrivalCheckFailed,
	dlv1.ArrivalResult_ARRIVAL_RESULT_STARTED:      ArrivalStarted,
	dlv1.ArrivalResult_ARRIVAL_RESULT_REFUSED:      ArrivalRefused,
}

// Pair is a Move's source and its arriving Torrent, nil before its Place.
type Pair struct {
	From ID
	To   *ID
}

// Handoff is what a Move reads of its pair. Every time is the
// Controller's.
type Handoff struct {
	// Wanted is the source's.
	Wanted Wanted
	// AnnouncedAt and AnnounceDueAt are the source's latest successful
	// announce and its next due one, while a Move watches them.
	AnnouncedAt, AnnounceDueAt *time.Time
	// PausedAt is when the source was seen paused.
	PausedAt *time.Time
	// Final is the source's totals.
	Final Totals
	// StartAskedAt is the destination's open ask.
	StartAskedAt *time.Time
	// Arrival is the destination's result for its ask, or for its
	// complete add; ArrivalWhy is the Node's reason for a refused or a
	// check failed.
	Arrival    *Arrival
	ArrivalWhy string
	// Complete is the source's payload complete.
	Complete bool
	// Parted is the source parted from its Node.
	Parted bool
	// Removed is the source removed by anyone but the Move's switch.
	Removed bool
	// Gone is the destination removed before the switch.
	Gone bool
}

// Handoffs is the Handoff of each of pairs, by its source.
func Handoffs(ctx context.Context, r storage.Reader[file], pairs []Pair) (map[ID]Handoff, error) {
	ids := make([]ID, 0, 2*len(pairs))
	for _, p := range pairs {
		ids = append(ids, p.From)
		if p.To != nil {
			ids = append(ids, *p.To)
		}
	}
	rows, err := db.New(r).ListHandoffs(ctx, ids)
	if err != nil {
		return nil, err
	}
	of := make(map[ID]db.ListHandoffsRow, len(rows))
	for _, row := range rows {
		of[row.ID] = row
	}
	out := make(map[ID]Handoff, len(pairs))
	for _, p := range pairs {
		src, ok := of[p.From]
		if !ok {
			out[p.From] = Handoff{Removed: true, Parted: true}
			continue
		}
		h := Handoff{
			Wanted: Wanted(src.Wanted), AnnouncedAt: timeOrNil(src.AnnouncedAt), AnnounceDueAt: timeOrNil(src.AnnounceDueAt),
			PausedAt: timeOrNil(src.PausedAt), Complete: src.Completed, Parted: !src.PartedAt.IsZero(),
			Removed: !src.RemovedAt.IsZero() && (src.RemovedReason == nil || RemoveReason(*src.RemovedReason) != Moved),
			Final: Totals{
				Uploaded: src.Uploaded, Downloaded: src.Downloaded, Active: time.Duration(src.ActiveSeconds) * time.Second,
				Seeding: time.Duration(src.SeedingSeconds) * time.Second, Announced: time.Duration(src.AnnouncedSeconds) * time.Second,
				At: src.TotalsAt.Time(),
			},
		}
		if p.To != nil {
			dst, ok := of[*p.To]
			h.Gone = !ok || !dst.RemovedAt.IsZero()
			h.StartAskedAt = timeOrNil(dst.StartAskedAt)
			if dst.Arrival != nil {
				h.Arrival = new(Arrival(*dst.Arrival))
			}
			if dst.ArrivalReason != nil {
				h.ArrivalWhy = *dst.ArrivalReason
			}
		}
		out[p.From] = h
	}
	return out, nil
}

func timeOrNil(t storage.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return new(t.Time())
}

// MoveSource is what a Move's preview reads of a torrent it would move.
type MoveSource struct {
	Node        role.NodeID
	Client      *ClientRef
	InfoHash    bencode.InfoHash
	Name        string
	Size        int64
	ContentType *works.TypeID
	// Metainfo is whether its metainfo is stored, which a Move's
	// destination is added from.
	Metainfo bool
	Wanted   Wanted
	// Private is a private torrent, which a Hit and run window may hold.
	Private bool
	// Withheld is on the review list after a restore.
	Withheld bool
	Removed  bool
	// Arriving is a Move's destination itself.
	Arriving bool
	// Complete is its payload complete.
	Complete bool
	// Files is its payload's files, from its completion.
	Files []FileOf
	// Version is its external client's version as last polled, zero
	// for none or the built-in client.
	Version clients.Version
}

// FileOf is one file of a torrent's payload, its path '/' separated as
// the metainfo names it.
type FileOf struct {
	Index int
	Path  string
	Size  int64
}

var ErrNotATorrent = errors.New("no such Torrent")

// MoveSources is each of ids as a Move's preview reads it, ErrNotATorrent
// for one not found.
func MoveSources(ctx context.Context, r storage.Reader[file], ids []ID) (map[ID]MoveSource, error) {
	rows, err := db.New(r).ListMoveSources(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[ID]MoveSource, len(rows))
	for _, row := range rows {
		s := MoveSource{
			Node: row.NodeID, Client: refOf(row.ClientID, row.ClientKind, row.ClientName), InfoHash: bencode.InfoHash(row.InfoHash),
			Name: row.Name, Size: row.Size, ContentType: row.ContentType, Metainfo: row.HasMetainfo, Wanted: Wanted(row.Wanted),
			Private: row.Private, Withheld: !row.WithheldAt.IsZero(), Removed: !row.RemovedAt.IsZero(), Arriving: row.Arriving,
		}
		if row.ClientVersion != nil {
			if s.Version, err = clients.ParseVersion(*row.ClientVersion); err != nil {
				return nil, fmt.Errorf("torrent %s's client version: %w", row.ID, err)
			}
		}
		if row.Completion != nil {
			var c Completion
			if err := json.Unmarshal([]byte(*row.Completion), &c); err != nil {
				return nil, fmt.Errorf("torrent %s's completion: %w", row.ID, err)
			}
			s.Complete = true
			for _, f := range c.Files {
				s.Files = append(s.Files, FileOf{Index: f.Index, Path: f.Path, Size: f.Size})
			}
		}
		out[row.ID] = s
	}
	for _, id := range ids {
		if _, ok := out[id]; !ok {
			return nil, fmt.Errorf("torrent %s: %w", id, ErrNotATorrent)
		}
	}
	return out, nil
}

// HeldOn is each of hashes node holds live, in any client, leaving out a
// Move's arriving Torrent, with the client that holds it.
func HeldOn(ctx context.Context, r storage.Reader[file], node role.NodeID, hashes []bencode.InfoHash) (map[bencode.InfoHash][]*ClientID, error) {
	raw := make([][]byte, len(hashes))
	for i, h := range hashes {
		raw[i] = h[:]
	}
	rows, err := db.New(r).ListHeldOnNode(ctx, db.ListHeldOnNodeParams{NodeID: node, Hashes: raw})
	if err != nil {
		return nil, err
	}
	out := map[bencode.InfoHash][]*ClientID{}
	for _, row := range rows {
		h := bencode.InfoHash(row.InfoHash)
		out[h] = append(out[h], row.ClientID)
	}
	return out, nil
}

// refusedMoving is, for each of ids, the refusal of a Move's source
// before its switch or of its arriving Torrent, read in r. An Admin may
// change neither until the Move ends. A source's refusal names its
// destination's Node only when it is another Node.
func (c *Clients) refusedMoving(ctx context.Context, r storage.Reader[file], ids []ID) (map[ID]*Refusal, error) {
	rows, err := readActed(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	moving, err := c.cfg.Moving.Moving(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	out := map[ID]*Refusal{}
	var names map[role.NodeID]string
	for _, id := range ids {
		row, ok := rows[id]
		to, src := moving[id]
		if !ok || !row.Arriving && !src {
			continue
		}
		if names == nil {
			if names, err = namesIn(ctx, r); err != nil {
				return nil, err
			}
		}
		if out[id], err = movingRefusal(ctx, r, row, to, names); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// movingRefusal is the refusal of row, an arriving Torrent or a source
// moving to.
func movingRefusal(ctx context.Context, r storage.Reader[file], row db.ListActedRow, to MovingTo, names map[role.NodeID]string) (*Refusal, error) {
	if row.Arriving {
		return &Refusal{Code: RefusedMoving, Node: names[row.NodeID], Client: row.ClientName}, nil
	}
	f := &Refusal{Code: RefusedMoving}
	if to.Node != row.NodeID {
		f.Node = names[to.Node]
	}
	if to.Client != nil {
		cl, err := db.New(r).GetClient(ctx, *to.Client)
		if err != nil {
			return nil, err
		}
		f.Client = cl.Name
	}
	return f, nil
}

// unrefused is ids less those refused names.
func unrefused(ids []ID, refused map[ID]*Refusal) []ID {
	if len(refused) == 0 {
		return ids
	}
	return slices.DeleteFunc(slices.Clone(ids), func(id ID) bool { return refused[id] != nil })
}

// withRefused is results for ids in order, each refused one answered by
// refused.
func withRefused(ids []ID, results []Result, refused map[ID]*Refusal) []Result {
	if len(refused) == 0 {
		return results
	}
	of := make(map[ID]Result, len(results))
	for _, r := range results {
		of[r.ID] = r
	}
	out := make([]Result, len(ids))
	for i, id := range ids {
		if f := refused[id]; f != nil {
			out[i] = Result{ID: id, Refusal: f}
		} else {
			out[i] = of[id]
		}
	}
	return out
}

// applyMove records what a report of node's Torrent id says for a Move,
// each Node time converted to the Controller's received: when its
// trackers last answered and are next due, its pause once each tracker
// answered its stopped, and its arrival. The built-in client holds a
// paused torrent with HOLD_WANTED, an external one reports its run paused.
func applyMove(ctx context.Context, q *db.Queries, node role.NodeID, id ID, r *dlv1.TorrentReport, received time.Time) error {
	ctl := func(ms int64) storage.Time {
		if ms == 0 {
			return storage.Time{}
		}
		return storage.FromTime(received.Add(-time.Duration(r.GetAtUnixMs()-ms) * time.Millisecond))
	}
	answered, due, stopped := announcesOf(r.GetTrackers(), ctl)
	if !answered.IsZero() || !due.IsZero() {
		if err := q.FoldAnnounces(ctx, db.FoldAnnouncesParams{AnsweredAt: answered, DueAt: due, ID: id, NodeID: node}); err != nil {
			return err
		}
	}
	if r.GetHold() == dlv1.Hold_HOLD_WANTED || r.GetRun() == dlv1.Wanted_WANTED_PAUSED {
		err := q.SetPausedAt(ctx, db.SetPausedAtParams{At: storage.FromTime(received), ID: id, NodeID: node, Stopped: boolInt(stopped), Blind: stoppedBlind})
		if err != nil {
			return err
		}
	}
	return setArrival(ctx, q, node, id, r.GetArrival(), received)
}

// announcesOf is when lines last answered and are next due, in the
// Controller's time by ctl, and whether each working one announced its
// stop.
func announcesOf(lines []*dlv1.TrackerLine, ctl func(int64) storage.Time) (answered, due storage.Time, stopped bool) {
	stopped = true
	for _, l := range lines {
		if a := ctl(l.GetAnsweredAtUnixMs()); !a.IsZero() && (answered.IsZero() || a.Time().After(answered.Time())) {
			answered = a
		}
		if d := ctl(l.GetNextAnnounceUnixMs()); !d.IsZero() && (due.IsZero() || d.Time().Before(due.Time())) {
			due = d
		}
		if l.GetStatus() == dlv1.TrackerStatus_TRACKER_STATUS_WORKING && !l.GetStopped() {
			stopped = false
		}
	}
	return answered, due, stopped
}

// setArrival records a, the arrival node reported for its Torrent id.
func setArrival(ctx context.Context, q *db.Queries, node role.NodeID, id ID, a *dlv1.Arrival, received time.Time) error {
	got, ok := arrivalFromWire[a.GetResult()]
	if !ok {
		return nil
	}
	var why *string
	if got == ArrivalRefused || got == ArrivalCheckFailed {
		why = new(a.GetReason())
	}
	p := db.SetArrivalParams{Arrival: new(string(got)), ArrivalReason: why, At: storage.FromTime(received), ID: id, NodeID: node}
	if got == ArrivalComplete || got == ArrivalCheckFailed {
		p.Added = 1
	} else {
		p.AskedAt = storage.FromTime(time.UnixMilli(a.GetAskedUnixMs()))
	}
	return q.SetArrival(ctx, p)
}

// stoppedBlind is the kinds that can't tell a stopped announce apart, so a
// Move's stop counts from their paused state alone.
var stoppedBlind = func() []clients.Kind {
	var out []clients.Kind
	for _, k := range clients.Kinds {
		if !k.Facts().StoppedRead {
			out = append(out, k)
		}
	}
	return out
}()

// TorrentsIn is every live Torrent of node's client, nil for its built-in
// one, in added order, leaving out a Move's arriving Torrents.
func TorrentsIn(ctx context.Context, r storage.Reader[file], node role.NodeID, client *ClientID) ([]ID, error) {
	return db.New(r).ListTorrentsInClient(ctx, db.ListTorrentsInClientParams{NodeID: node, ClientID: client})
}

// Describer describes a Content Type.
type Describer interface {
	Describe(t works.TypeID) (works.Description, bool)
}

// TypeFolder is the folder a torrent of Content Type ct sits in under a
// Download Client's download folder, the type's name, which a grab and a
// Move both send its Node. It is "" for an adopted torrent.
func TypeFolder(types Describer, ct *works.TypeID) string {
	if ct == nil {
		return ""
	}
	d, _ := types.Describe(*ct)
	return d.Name
}
