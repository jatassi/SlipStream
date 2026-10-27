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
	"github.com/jatassi/slipstream/internal/works"
)

// ErrNotFound is a Torrent the Controller doesn't hold: no Torrent has
// the id, or its Node parted from it.
var ErrNotFound = errors.New("no Torrent has this id")

// Row is one Torrent as the Torrents view lists it: its record and what
// is derived from it.
type Row struct {
	ID          ID
	Node        role.NodeID
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
	// QueuePosition is its place, from 1, among its Node's downloads
	// waiting for a slot, 0 while it waits for none.
	QueuePosition int
	// Pending is the wanted state its Node has yet to act on, "" when
	// none.
	Pending Wanted
	// Done is the bytes done as its Node last reported them.
	Done   int64
	Totals Totals
	// Tracker is nil for a Torrent whose metainfo names no tracker.
	Tracker *TrackerCell
	Error   string
	Fault   *Fault
	// LastKnown is when its Node last reported it, set only while the
	// Node is out of step.
	LastKnown time.Time

	report *Report
}

// TrackerCell is the tracker a row names: a working one when any works,
// else the one in the worst state.
type TrackerCell struct {
	// Host is "" when the Torrent's tracker list is unknown.
	Host    string
	Status  TrackerStatus
	Message string
	RetryAt time.Time
}

// Filter narrows List. Each zero field lets every Torrent through.
type Filter struct {
	IDs  []ID
	Tab  Tab
	Node role.NodeID
	Type works.TypeID
	// Search matches the Torrent's name, its Grabs' release titles and
	// their Works' labels, ignoring case.
	Search string
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
var Tabs = []Tab{TabDownloading, TabSeeding, TabProblems, TabAll}

// List is every Torrent live or Removing, in added order, as f narrows
// it.
func (c *Clients) List(ctx context.Context, f Filter) (Listing, error) {
	var out Listing
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := c.rows(ctx, r)
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
		f.Type != "" && row.ContentType != f.Type:
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

// rows is every Torrent live or Removing, in added order.
func (c *Clients) rows(ctx context.Context, r storage.Reader[file]) ([]Row, error) {
	held, err := db.New(r).ListHeld(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]ID, len(held))
	for i, h := range held {
		ids[i] = h.ID
	}
	uses, err := c.cfg.Uses.Of(ctx, r, ids)
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
		if row.State == StateQueued && row.Hold == HoldSlot {
			inLine[row.Node]++
			row.QueuePosition = inLine[row.Node]
		}
		rows[i] = row
	}
	return rows, nil
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
		ID: h.ID, Node: h.NodeID, Name: h.Name, Size: h.Size, ContentType: h.ContentType, AddedAt: h.AddedAt.Time(),
		Uses: uses, State: state, Severity: SeverityOf(facts, state), Tabs: TabsOf(facts, state),
		Done: rep.done(h.Size), Tracker: cellOf(hosts, rep), report: rep,
		Totals: Totals{
			Uploaded: h.Uploaded, Downloaded: h.Downloaded,
			Active: time.Duration(h.ActiveSeconds) * time.Second, Seeding: time.Duration(h.SeedingSeconds) * time.Second,
			At: h.TotalsAt.Time(),
		},
	}
	if h.Fault != nil {
		row.Fault = &Fault{Kind: FaultKind(*h.Fault), At: h.FaultAt.Time()}
	}
	if !facts.Removing {
		row.Pending = pendingOf(facts.Wanted, rep)
	}
	if rep == nil {
		if !c.cfg.Roster.InStep(h.NodeID) {
			row.LastKnown = row.AddedAt
		}
		return row, nil
	}
	row.Hold, row.Until = rep.Hold, rep.Until
	if rep.Engine == EngineFailed {
		row.Error = rep.Error
	}
	if !c.cfg.Roster.InStep(h.NodeID) {
		row.LastKnown = rep.At
	}
	return row, nil
}

// pendingOf is wanted when the Node's report doesn't yet show it: a pause
// the Node hasn't held it for, or a resume of one it still holds.
func pendingOf(wanted Wanted, rep *Report) Wanted {
	held := rep != nil && rep.Hold == HoldWanted
	if wanted == Paused && !held || wanted == Started && held {
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
	InfoHash bencode.InfoHash
	// SaveFolder is "" until its Node reports one.
	SaveFolder string
	PieceSize  int64
	Private    bool
	// Tracker is the Tracker its Grab's Release reached, nil for none.
	Tracker  *indexers.TrackerID
	Trackers []TrackerDetail
	Files    []FileDetail
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
}

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
		rows, err := c.rows(ctx, r)
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
	d := Detail{Row: row, InfoHash: m.InfoHash, PieceSize: m.Info.PieceLength, Private: details.Private == 1, Tracker: details.TrackerID}
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
	}
}

// atMillis is a wire time, zero for 0.
func atMillis(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}
