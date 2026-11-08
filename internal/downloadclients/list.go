package downloadclients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
	"github.com/jatassi/slipstream/internal/works"
)

// ErrNotFound is a Torrent the Controller doesn't hold: no Torrent has
// the id, or its Node parted from it.
var ErrNotFound = errors.New("no Torrent has this id")

// Row is one Torrent as the Torrents view lists it: its record and what
// is derived from it.
type Row struct {
	ID       ID
	Node     role.NodeID
	InfoHash bencode.InfoHash
	Private  bool
	// Client is nil for the Node's built-in client.
	Client *ClientRef
	// Attached is an external torrent that was in its client before
	// SlipStream attached to it.
	Attached bool
	// DeleteFiles is a remove that may delete its download copy: false
	// for one that parts at once, attached or in an off client.
	DeleteFiles bool
	Name        string
	Size        int64
	ContentType works.TypeID
	AddedAt     time.Time
	// Uses are its Grabs, newest first.
	Uses     []Use
	State    State
	Hold     Hold
	Until    time.Time
	Severity Severity
	Tabs     Tab
	// QueuePosition is its place, from 1, among its client's downloads
	// waiting for a slot, 0 while it waits for none.
	QueuePosition int
	// Pending is the wanted state its Node has yet to act on, "" when
	// none.
	Pending Wanted
	// Done is the bytes done as its Node last reported them.
	Done   int64
	Totals Totals
	// Announce is nil for a Torrent whose metainfo names no tracker.
	Announce *TrackerCell
	// Tracker is the Tracker its first Grab's Release reached, nil for
	// none.
	Tracker *TrackerRef
	// Seeding is its info hash's seeding, as the Seeder judged it, zero
	// for an external client's Torrent, which no goal or window covers.
	Seeding seeding.View
	// Plan is what its goal does now, as the Seeder judged it, nil for an
	// external client's Torrent.
	Plan seeding.Plan
	// Error is the engine's error text while failed, or an external
	// client's words for its local error.
	Error string
	Fault *Fault
	// LastKnown is when its Node last reported it, set only while the row
	// is stale for LastKnownCause.
	LastKnown      time.Time
	LastKnownCause StaleCause
	// ImportAt is when a finished external torrent imports, set while its
	// client's Import delay runs.
	ImportAt time.Time
	// StoppedAtLimit is an external torrent its client stopped at its
	// seeding limit.
	StoppedAtLimit bool

	report *Report
}

// TrackerRef is a Tracker as a row names it.
type TrackerRef struct {
	ID   indexers.TrackerID
	Name string
}

// TrackerCell is the announce URL a row names: a working one when any
// works, else the one in the worst state.
type TrackerCell struct {
	// Host is "" when the Torrent's tracker list is unknown.
	Host    string
	Status  TrackerStatus
	Message string
	RetryAt time.Time
}

// Filter narrows List and Rows. Each zero field lets every Torrent
// through. Rows reads only the Torrents Tracker, NoTracker and Hashes let
// through.
type Filter struct {
	IDs  []ID
	Tab  Tab
	Node role.NodeID
	Type works.TypeID
	// Client lets through one client's Torrents when set.
	Client *ClientFilter
	// Search matches the Torrent's name, its Grabs' release titles and
	// their Works' labels, ignoring case.
	Search    string
	Tracker   *indexers.TrackerID
	NoTracker bool
	// Hashes is nil for every info hash.
	Hashes []bencode.InfoHash
}

// ClientFilter is one external client's Torrents, or every built-in
// client's for a nil ID.
type ClientFilter struct{ ID *ClientID }

func (f ClientFilter) lets(row Row) bool {
	if f.ID == nil || row.Client == nil {
		return f.ID == nil && row.Client == nil
	}
	return row.Client.ID == *f.ID
}

// Listing is the rows a Filter lets through, and counts of every Torrent
// live or Removing, filtered or not.
type Listing struct {
	Rows   []Row
	Counts Counts
}

type Counts struct {
	States map[State]int
	Tabs   map[Tab]int
	Nodes  map[role.NodeID]int
}

// Tabs is each of the Torrents view's tabs, in its order.
var Tabs = []Tab{TabDownloading, TabSeeding, TabProblems, TabGoalMet, TabAll}

// List is every Torrent live or Removing, in added order, as f narrows
// it.
func (c *Clients) List(ctx context.Context, f Filter) (Listing, error) {
	var out Listing
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := c.Rows(ctx, r, Filter{})
		if err != nil {
			return err
		}
		labels := map[works.WorkID]string{}
		if f.Search != "" {
			if labels, err = workLabels(ctx, r); err != nil {
				return err
			}
		}
		out = Listing{Rows: []Row{}, Counts: countsOf(rows)}
		for _, row := range rows {
			if f.lets(row, labels) {
				out.Rows = append(out.Rows, row)
			}
		}
		return nil
	})
	return out, err
}

func (f Filter) lets(row Row, labels map[works.WorkID]string) bool {
	switch {
	case f.IDs != nil && !slices.Contains(f.IDs, row.ID),
		f.Tab != 0 && row.Tabs&f.Tab == 0,
		!f.Node.IsZero() && row.Node != f.Node,
		f.Type != "" && row.ContentType != f.Type,
		f.Tracker != nil && (row.Tracker == nil || row.Tracker.ID != *f.Tracker),
		f.NoTracker && row.Tracker != nil,
		f.Hashes != nil && !slices.Contains(f.Hashes, row.InfoHash),
		f.Client != nil && !f.Client.lets(row):
		return false
	}
	if f.Search == "" {
		return true
	}
	q := strings.ToLower(f.Search)
	found := func(s string) bool { return strings.Contains(strings.ToLower(s), q) }
	return found(row.Name) || slices.ContainsFunc(row.Uses, func(u Use) bool { return found(u.Release) || found(labels[u.Work]) })
}

func workLabels(ctx context.Context, r storage.Reader[file]) (map[works.WorkID]string, error) {
	summaries, err := works.Summaries(ctx, r)
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]string, len(summaries))
	for _, s := range summaries {
		out[s.ID] = s.Label
	}
	return out, nil
}

func countsOf(rows []Row) Counts {
	c := Counts{States: map[State]int{}, Tabs: map[Tab]int{}, Nodes: map[role.NodeID]int{}}
	for _, row := range rows {
		c.States[row.State]++
		c.Nodes[row.Node]++
		for _, t := range Tabs {
			if row.Tabs&t != 0 {
				c.Tabs[t]++
			}
		}
	}
	return c
}

// Rows is every Torrent live or Removing that f's Tracker, NoTracker and
// Hashes let through, in added order, its built-in client's ones judged
// by the Seeder in r's snapshot. QueuePosition counts only the rows
// read. Every reader of a Torrent's seeding reads it here, so the
// Torrents view, a Tracker's page and the goals agree.
func (c *Clients) Rows(ctx context.Context, r storage.Reader[file], f Filter) ([]Row, error) {
	held, err := db.New(r).ListHeld(ctx, f.listHeld())
	if err != nil {
		return nil, err
	}
	return c.rowsOf(ctx, r, held)
}

func (f Filter) listHeld() db.ListHeldParams {
	p := db.ListHeldParams{ByTracker: flag(f.Tracker != nil), TrackerID: f.Tracker, NoTracker: flag(f.NoTracker), ByHash: flag(f.Hashes != nil)}
	for _, h := range f.Hashes {
		p.InfoHashes = append(p.InfoHashes, h[:])
	}
	return p
}

func flag(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func (c *Clients) rowsOf(ctx context.Context, r storage.Reader[file], held []db.ListHeldRow) ([]Row, error) {
	ids := make([]ID, len(held))
	for i, h := range held {
		ids[i] = h.ID
	}
	uses, err := c.cfg.Uses.Of(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	externals, err := externalsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	rows := make([]Row, len(held))
	inLine := map[role.NodeID]int{}
	for i, h := range held {
		row, err := c.rowOf(h, uses[h.ID])
		if err != nil {
			return nil, fmt.Errorf("torrent %s: %w", h.ID, err)
		}
		if row.LastKnownCause = staleOf(c.cfg.Roster.InStep(h.NodeID), row.Client, externals); row.LastKnownCause != "" {
			row.LastKnown = row.AddedAt
			if row.report != nil {
				row.LastKnown = row.report.At
			}
		}
		switch {
		case row.State != StateQueued || row.Hold != HoldSlot:
		case row.Client != nil:
			row.QueuePosition = row.report.QueuePosition
		default:
			inLine[row.Node]++
			row.QueuePosition = inLine[row.Node]
		}
		rows[i] = row
	}
	return rows, c.judge(ctx, r, rows)
}

// judge fills each built-in row's Tracker, seeding and plan as the Seeder
// judges them. A row whose goal is met gains TabGoalMet, and one at risk
// of a hit and run gains TabProblems.
func (c *Clients) judge(ctx context.Context, r storage.Reader[file], rows []Row) error {
	var builtIn []Row
	for _, row := range rows {
		if row.Client == nil {
			builtIn = append(builtIn, row)
		}
	}
	judged, err := c.cfg.Seeding.Judge(ctx, r, builtIn)
	if err != nil {
		return err
	}
	for i := range rows {
		j, ok := judged[rows[i].ID]
		if !ok {
			continue
		}
		rows[i].Tracker, rows[i].Seeding, rows[i].Plan = j.Tracker, j.View, j.Plan
		if j.View.GoalMet {
			rows[i].Tabs |= TabGoalMet
		}
		if j.View.Risk == seeding.AtRisk {
			rows[i].Tabs |= TabProblems
		}
	}
	return nil
}

func (c *Clients) rowOf(h db.ListHeldRow, uses []Use) (Row, error) {
	rep, err := reportOf(h.Report)
	if err != nil {
		return Row{}, err
	}
	var hosts []string
	if err := json.Unmarshal([]byte(h.Trackers), &hosts); err != nil {
		return Row{}, fmt.Errorf("trackers: %w", err)
	}
	facts := Facts{Removing: !h.RemovedAt.IsZero(), Wanted: Wanted(h.Wanted), Report: rep}
	state := StateOf(facts)
	row := Row{
		ID: h.ID, Node: h.NodeID, InfoHash: bencode.InfoHash(h.InfoHash), Private: h.Private == 1, Name: h.Name, Size: h.Size, ContentType: h.ContentType, AddedAt: h.AddedAt.Time(),
		Uses: uses, State: state, Severity: SeverityOf(facts, state), Tabs: TabsOf(facts, state),
		Client: refOf(h.ClientID, h.ClientKind, h.ClientName), Attached: h.Attached != nil && *h.Attached,
		DeleteFiles: removalOf(h.ClientID, h.Attached, h.ClientOn) == removedByNode,
		Done:        rep.done(h.Size), Announce: cellOf(hosts, rep), report: rep,
		Totals: Totals{
			Uploaded: h.Uploaded, Downloaded: h.Downloaded,
			Active: time.Duration(h.ActiveSeconds) * time.Second, Seeding: time.Duration(h.SeedingSeconds) * time.Second,
			Announced: time.Duration(h.AnnouncedSeconds) * time.Second, At: h.TotalsAt.Time(),
		},
	}
	if h.Fault != nil {
		row.Fault = &Fault{Kind: FaultKind(*h.Fault), At: h.FaultAt.Time()}
	}
	if !facts.Removing {
		row.Pending = pendingOf(facts.Wanted, rep)
	}
	if rep == nil {
		return row, nil
	}
	row.Hold, row.Until, row.ImportAt, row.StoppedAtLimit = rep.Hold, rep.Until, rep.ImportAt, rep.StoppedAtLimit
	if rep.Engine == EngineFailed || rep.Hold == HoldClientError {
		row.Error = rep.Error
	}
	return row, nil
}

// refOf is a Torrent's client as a row names it, nil for the built-in
// client.
func refOf(id *ClientID, kind clients.Kind, name string) *ClientRef {
	if id == nil {
		return nil
	}
	return &ClientRef{ID: *id, Kind: kind, Name: name}
}

// pendingOf is wanted when the Node's report doesn't yet show it: a pause
// the client doesn't hold it for, or a resume of one it still holds. The
// built-in client holds it with HoldWanted, an external one with a paused
// run. A stop at a seeding limit is no pause, so it leaves no resume
// pending.
func pendingOf(wanted Wanted, rep *Report) Wanted {
	held := rep != nil && (rep.Hold == HoldWanted || rep.Run == Paused)
	limit := rep != nil && rep.StoppedAtLimit
	if wanted == Paused && !held || wanted == Started && held && !limit {
		return wanted
	}
	return ""
}

// trackerRank orders the tracker states a row names, the first shown.
var trackerRank = []TrackerStatus{TrackerWorking, TrackerUnregistered, TrackerError, TrackerUpdating, TrackerNotContacted}

func cellOf(hosts []string, rep *Report) *TrackerCell {
	var lines []TrackerLine
	if rep != nil {
		lines = rep.Trackers
	}
	if len(lines) == 0 {
		if len(hosts) == 0 {
			return nil
		}
		return &TrackerCell{Host: hosts[0], Status: TrackerNotContacted}
	}
	l := slices.MinFunc(lines, func(a, b TrackerLine) int {
		return slices.Index(trackerRank, a.Status) - slices.Index(trackerRank, b.Status)
	})
	cell := &TrackerCell{Status: l.Status, Message: l.Message, RetryAt: l.RetryAt}
	if l.Index >= 0 && l.Index < len(hosts) {
		cell.Host = hosts[l.Index]
	}
	return cell
}

// Detail is one Torrent as the Inspector shows it: its row, its metainfo's
// facts, and its trackers and files, live from its Node while it answers.
type Detail struct {
	Row
	// SaveFolder is "" until its Node reports one.
	SaveFolder string
	PieceSize  int64
	Trackers   []TrackerDetail
	Files      []FileDetail
	// Live: its Node answered, so Trackers, Files and AnnounceAt are its
	// word now rather than its last report.
	Live bool
	// AnnounceAt is the earliest every tracker would accept an announce,
	// zero unless Live.
	AnnounceAt time.Time
}

type TrackerDetail struct {
	// URL is redacted.
	URL     string
	Tier    int
	Status  TrackerStatus
	Message string
	// RetryAt is set while Status is error or unregistered and the
	// detail is not Live.
	RetryAt      time.Time
	AnsweredAt   time.Time
	NextAnnounce time.Time
	MinInterval  time.Duration
	AcceptsAt    time.Time
	// Swarm is the counts its last announce response carried, nil while
	// none did or the detail is not Live.
	Swarm *Swarm
}

// Swarm is a tracker's count of a Torrent's seeders and leechers.
type Swarm struct{ Seeders, Leechers int }

type FileDetail struct {
	Index int
	Path  string
	Size  int64
	// Done is nil while unknown: the detail is not Live and the Torrent
	// is not complete.
	Done *int64
	// Item is the Item the file filled once a Grab imported it, zero
	// before.
	Item works.ItemID
}

// Get is Torrent id, live or Removing, with its Node's word on its
// trackers and files when the Node is in step and answers within a live
// sample.
func (c *Clients) Get(ctx context.Context, id ID) (Detail, error) {
	var d Detail
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		details, err := db.New(r).GetDetails(ctx, id)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		rows, err := c.Rows(ctx, r, Filter{})
		if err != nil {
			return err
		}
		i := slices.IndexFunc(rows, func(row Row) bool { return row.ID == id })
		if i < 0 {
			return ErrNotFound
		}
		d, err = detailOf(rows[i], details)
		return err
	})
	if err != nil {
		return Detail{}, err
	}
	c.inspect(ctx, &d)
	return d, nil
}

func detailOf(row Row, details db.GetDetailsRow) (Detail, error) {
	m, err := bencode.DecodeMetainfo(details.Metainfo)
	if err != nil {
		return Detail{}, fmt.Errorf("torrent %s: %w", row.ID, err)
	}
	d := Detail{Row: row, PieceSize: m.Info.PieceLength}
	if details.SaveFolder != nil {
		d.SaveFolder = *details.SaveFolder
	}
	var lines map[int]TrackerLine
	complete := false
	if rep := row.report; rep != nil {
		lines, complete = map[int]TrackerLine{}, rep.Complete
		for _, l := range rep.Trackers {
			lines[l.Index] = l
		}
	}
	i := 0
	for tier, urls := range tiersOf(m) {
		for _, u := range urls {
			t := TrackerDetail{URL: Redact(u), Tier: tier, Status: TrackerNotContacted}
			if l, ok := lines[i]; ok {
				t.Status, t.Message, t.RetryAt = l.Status, l.Message, l.RetryAt
			}
			d.Trackers = append(d.Trackers, t)
			i++
		}
	}
	items := map[int]works.ItemID{}
	for _, u := range row.Uses {
		if u.Stage == UseImported && u.File != nil {
			items[*u.File] = u.Item
		}
	}
	for i, f := range m.Info.Files {
		fd := FileDetail{Index: i, Path: pathOf(m.Info, f), Size: f.Length, Item: items[i]}
		if complete {
			fd.Done = new(f.Length)
		}
		d.Files = append(d.Files, fd)
	}
	return d, nil
}

// tiersOf is m's trackers by tier, in the order of Metainfo.Trackers.
func tiersOf(m bencode.Metainfo) [][]string {
	if len(m.AnnounceList) > 0 {
		return m.AnnounceList
	}
	if m.Announce == "" {
		return nil
	}
	return [][]string{{m.Announce}}
}

// pathOf is a file's path under the Torrent's save folder.
func pathOf(info bencode.Info, f bencode.File) string {
	if f.Path == nil {
		return info.Name
	}
	return info.Name + "/" + strings.Join(f.Path, "/")
}

// inspect takes d's Node's word on its trackers and files, when the Node
// is in step and answers within a live sample.
func (c *Clients) inspect(ctx context.Context, d *Detail) {
	if !c.cfg.Roster.InStep(d.Node) {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Live.Sample)
	defer cancel()
	res, err := c.cfg.Remote(d.Node).Inspect(ctx, &dlv1.InspectRequest{Id: d.ID.Bytes()})
	if err != nil {
		c.cfg.Log.Debug("inspect a torrent on its Node", "node", d.Node.String(), "torrent", d.ID.String(), "err", err.Error())
		return
	}
	d.Live, d.AnnounceAt = true, atMillis(res.GetAnnounceAtUnixMs())
	for _, f := range res.GetFiles() {
		if i := int(f.GetIndex()); i >= 0 && i < len(d.Files) {
			d.Files[i].Done = new(f.GetDone())
		}
	}
	for _, t := range res.GetTrackers() {
		i := int(t.GetIndex())
		if i < 0 || i >= len(d.Trackers) {
			continue
		}
		tr := &d.Trackers[i]
		tr.Status, tr.Message, tr.RetryAt = trackerFromWire[t.GetStatus()], t.GetMessage(), time.Time{}
		tr.AnsweredAt, tr.NextAnnounce, tr.AcceptsAt = atMillis(t.GetAnsweredAtUnixMs()), atMillis(t.GetNextAnnounceUnixMs()), atMillis(t.GetAcceptsAtUnixMs())
		tr.MinInterval = time.Duration(t.GetMinIntervalSeconds()) * time.Second
		if t.Seeders != nil && t.Leechers != nil {
			tr.Swarm = &Swarm{Seeders: int(t.GetSeeders()), Leechers: int(t.GetLeechers())}
		}
	}
}

// atMillis is a wire time, zero for 0.
func atMillis(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}
