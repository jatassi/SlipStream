package conversions

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/conversions/internal/plan"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Band is a part of the Conversions tab.
type Band string

const (
	BandAttention  Band = "attention"
	BandConverting Band = "converting"
	BandNext       Band = "next"
)

// Cursor pages Up next by lane and position.
type Cursor string

type ListQuery struct {
	Band   Band
	Node   *role.NodeID
	Type   works.TypeID
	Q      string
	Work   *works.WorkID
	Cursor Cursor
}

type List struct {
	Counts     Counts
	Attention  []ListRow
	Converting []ListRow
	Next       []LaneRows
	Cursor     Cursor
}

// Counts are the tab's sentence numbers. ClearsAt is the latest finish.
type Counts struct {
	Converting, Paused, Waiting, Failed int
	ClearsAt                            *time.Time
}

type LaneRows struct {
	Lane Lane
	Rows []Entry
}

// Entry is one row of Up next: a Conversion or a folded group.
type Entry struct {
	Row   *ListRow
	Group *GroupRow
}

// ListRow is a Conversion with its forecast and live progress. Servers
// are the Media Servers a transcode wait names.
type ListRow struct {
	Row           Row
	Items         []ItemLabel
	Source        RowSource
	NodeName      string
	Progress      *LiveRow
	Wait          *Wait
	WaitNode      string
	Servers       []string
	Start, Finish *time.Time
	// FailedOn names the Nodes its counted attempts failed on.
	FailedOn []string
	Attempts []Attempt
}

type ItemLabel struct {
	ID    works.ItemID
	Label string
}

// RowSource is the Target a row's source file fills, nil once no Profile
// row makes the row's Target from one.
type RowSource struct {
	Target *targets.TargetID
	Name   string
}

// GroupID names a backfill's group or a series season's.
type GroupID string

type GroupKind string

const (
	GroupBackfill GroupKind = "backfill"
	GroupSeason   GroupKind = "season"
)

type GroupRow struct {
	ID            GroupID
	Kind          GroupKind
	Label         string
	Count         int
	Nodes         []string
	Seconds       int64
	Bytes         int64
	Start, Finish *time.Time
}

type Group struct {
	Rows   []ListRow
	Cursor Cursor
}

// PageSize is the Up next entries, or a group's rows, one page holds.
const PageSize = 100

var lanes = []Lane{LaneMovedUp, LaneNew, LaneUpgrade, LaneBackfill}

// tab is the queue forecast to the horizon, with the failed rows the
// forecast leaves out.
type tab struct {
	c        *Conversions
	s        *snap
	f        plan.Forecast
	failed   []Row
	progress map[ID]RunProgress
	levels   map[works.ItemID]works.LevelID
}

func (c *Conversions) tab(ctx context.Context, r storage.Reader[storage.Library]) (*tab, error) {
	ros := c.roster()
	s, err := c.snapshot(ctx, r, ros)
	if err != nil {
		return nil, err
	}
	_, f := plan.Run(s.Snapshot, s.Now.Add(c.cfg.Timings.Horizon))
	raws, err := db.New(r).ListFailed(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the failed Conversions: %w", err)
	}
	t := &tab{c: c, s: s, f: f, progress: map[ID]RunProgress{}, levels: map[works.ItemID]works.LevelID{}}
	for _, raw := range raws {
		row, err := rowOf(raw)
		if err != nil {
			return nil, err
		}
		t.failed = append(t.failed, row)
	}
	for _, n := range s.Nodes {
		lengths := s.lengthsOn(n.ID)
		if len(lengths) == 0 {
			continue
		}
		fs, _ := ros.facts(n.ID)
		for id, p := range factsOf(fs, n.AtOnce, lengths).Progress {
			t.progress[id] = p
		}
	}
	return t, nil
}

// rows is every unfinished row: the failed newest first, then the queue
// in lane and position order.
func (t *tab) rows() []Row {
	out := slices.Clone(t.failed)
	for _, j := range t.s.Jobs {
		out = append(out, t.s.rows[ID{UUID: j.ID}])
	}
	return out
}

// listRow is r with its forecast and progress, without what a page reads.
func (t *tab) listRow(r Row) ListRow {
	out := ListRow{Row: r}
	if r.Node != nil {
		out.NodeName = t.s.names[*r.Node]
	}
	if p, ok := t.progress[r.ID]; ok {
		live := liveRow(r.ID, p, t.s.Jobs[t.s.at[r.ID]].Length)
		out.Progress = &live
	}
	p, ok := t.f[r.ID.UUID]
	if !ok {
		return out
	}
	out.Start, out.Finish = p.Start, p.Finish
	if r.Stage == StageRunning || r.Stage == StagePaused {
		out.Start = nil
	}
	if p.Wait.Code != "" {
		w := p.Wait
		out.Wait = &w
		if w.Node != nil {
			out.WaitNode = t.s.names[*w.Node]
		}
	}
	return out
}

// on is the Node a row is placed or forecast on.
func (t *tab) on(r Row) *role.NodeID {
	if r.Node != nil {
		return r.Node
	}
	return t.f[r.ID.UUID].Node
}

func (t *tab) matches(r Row, q ListQuery) bool {
	switch {
	case q.Work != nil && r.Work != *q.Work:
	case q.Type != "" && r.Type != q.Type:
	case q.Q != "" && !strings.Contains(strings.ToLower(r.Snapshot.Work), strings.ToLower(q.Q)):
	case q.Node != nil && (t.on(r) == nil || *t.on(r) != *q.Node):
	default:
		return true
	}
	return false
}

func bandOf(r Row) Band {
	switch r.Stage {
	case StageFailed:
		return BandAttention
	case StageRunning, StagePaused:
		return BandConverting
	case StageQueued, StageCopying, StageReady, StageChecking, StageReturning, StageLanding, StageDone, StageCanceled:
	}
	return BandNext
}

// List is the Conversions tab, forecast to the horizon at read time.
func (c *Conversions) List(ctx context.Context, q ListQuery) (List, error) {
	var out List
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		t, err := c.tab(ctx, r)
		if err != nil {
			return err
		}
		out, err = t.list(ctx, r, q)
		return err
	})
	return out, err
}

func (t *tab) list(ctx context.Context, r storage.Reader[storage.Library], q ListQuery) (List, error) {
	out, next := t.banded(q)
	entries, err := t.fold(ctx, r, next)
	if err != nil {
		return List{}, err
	}
	page, more := pageOf(entries, q.Cursor)
	if more && len(page) > 0 {
		out.Cursor = cursorOf(page[len(page)-1].anchor)
	}
	for _, e := range page {
		if len(out.Next) == 0 || out.Next[len(out.Next)-1].Lane != e.anchor.Lane {
			out.Next = append(out.Next, LaneRows{Lane: e.anchor.Lane})
		}
		lane := &out.Next[len(out.Next)-1]
		lane.Rows = append(lane.Rows, e.Entry)
	}
	err = t.fillList(ctx, r, &out)
	return out, err
}

// banded counts the rows q matches and puts each in its band, the Up next
// ones unfolded.
func (t *tab) banded(q ListQuery) (List, []ListRow) {
	out := List{Attention: []ListRow{}, Converting: []ListRow{}, Next: []LaneRows{}}
	var next []ListRow
	for _, row := range t.rows() {
		if !t.matches(row, q) {
			continue
		}
		lr := t.listRow(row)
		out.Counts.add(lr)
		band := bandOf(row)
		switch {
		case q.Band != "" && q.Band != band:
		case band == BandNext:
			next = append(next, lr)
		case q.Cursor != "":
		case band == BandAttention:
			out.Attention = append(out.Attention, lr)
		default:
			out.Converting = append(out.Converting, lr)
		}
	}
	return out, next
}

func (c *Counts) add(lr ListRow) {
	if lr.Finish != nil && (c.ClearsAt == nil || lr.Finish.After(*c.ClearsAt)) {
		c.ClearsAt = lr.Finish
	}
	switch lr.Row.Stage {
	case StageFailed:
		c.Failed++
	case StagePaused:
		c.Paused++
	case StageRunning:
		c.Converting++
	case StageQueued, StageCopying, StageReady, StageChecking, StageReturning, StageLanding, StageDone, StageCanceled:
		c.Waiting++
	}
}

// fillList fills the rows out shows with what their words name.
func (t *tab) fillList(ctx context.Context, r storage.Reader[storage.Library], out *List) error {
	shown := slices.Concat(out.Attention, out.Converting)
	for _, l := range out.Next {
		for _, e := range l.Rows {
			if e.Row != nil {
				shown = append(shown, *e.Row)
			}
		}
	}
	filled, err := t.fill(ctx, r, shown)
	if err != nil {
		return err
	}
	for _, rows := range [][]ListRow{out.Attention, out.Converting} {
		for i := range rows {
			rows[i] = filled[rows[i].Row.ID]
		}
	}
	for _, l := range out.Next {
		for _, e := range l.Rows {
			if e.Row != nil {
				*e.Row = filled[e.Row.Row.ID]
			}
		}
	}
	return nil
}

// folded is an Up next entry with the row whose lane and position it
// takes.
type folded struct {
	Entry
	anchor Row
}

// fold folds next's rows of one backfill, and the rows of one series
// season, into a group in the lane and at the place of its first row. A
// group of one is its row.
func (t *tab) fold(ctx context.Context, r storage.Reader[storage.Library], next []ListRow) ([]folded, error) {
	keys := make([]GroupID, len(next))
	if err := t.readLevels(ctx, r, next); err != nil {
		return nil, err
	}
	members := map[GroupID][]int{}
	for i, lr := range next {
		keys[i] = t.groupOf(lr.Row)
		if keys[i] != "" {
			members[keys[i]] = append(members[keys[i]], i)
		}
	}
	var out []folded
	for i, lr := range next {
		k := keys[i]
		if k == "" || len(members[k]) < 2 {
			out = append(out, folded{Entry: Entry{Row: &next[i]}, anchor: lr.Row})
			continue
		}
		if members[k][0] != i {
			continue
		}
		g, err := t.groupRow(ctx, r, k, next, members[k])
		if err != nil {
			return nil, err
		}
		out = append(out, folded{Entry: Entry{Group: &g}, anchor: lr.Row})
	}
	return out, nil
}

// readLevels reads the level of each Item of the Works with more than one
// row in rows.
func (t *tab) readLevels(ctx context.Context, r storage.Reader[storage.Library], rows []ListRow) error {
	per := map[works.WorkID]int{}
	for _, lr := range rows {
		per[lr.Row.Work]++
	}
	var ws []works.WorkID
	for w, n := range per {
		if n > 1 {
			ws = append(ws, w)
		}
	}
	if len(ws) == 0 {
		return nil
	}
	items, err := works.ItemsOf(ctx, r, works.Only(ws))
	if err != nil {
		return fmt.Errorf("read the Items: %w", err)
	}
	for _, its := range items {
		for _, it := range its {
			t.levels[it.ID] = it.LevelID
		}
	}
	return nil
}

const (
	backfillPrefix = "backfill:"
	seasonPrefix   = "season:"
)

// groupOf is the group r folds into: its backfill's, else its season's,
// "" for none.
func (t *tab) groupOf(r Row) GroupID {
	if r.Backfill != nil {
		return GroupID(backfillPrefix + string(r.Lane) + ":" + r.Backfill.String())
	}
	if len(r.Items) == 0 {
		return ""
	}
	level := t.levels[r.Items[0]]
	if level.IsZero() {
		return ""
	}
	return GroupID(seasonPrefix + string(r.Lane) + ":" + r.Work.String() + ":" + level.String())
}

func (t *tab) groupRow(ctx context.Context, r storage.Reader[storage.Library], id GroupID, next []ListRow, at []int) (GroupRow, error) {
	first := next[at[0]].Row
	g := GroupRow{ID: id, Kind: GroupBackfill, Count: len(at), Nodes: []string{}, Label: "Backfill of " + first.Snapshot.Target}
	if first.Backfill == nil {
		g.Kind = GroupSeason
		label, err := seasonLabel(ctx, r, first, t.levels[first.Items[0]])
		if err != nil {
			return GroupRow{}, err
		}
		g.Label = label
	}
	for _, i := range at {
		lr := next[i]
		j := t.s.Jobs[t.s.at[lr.Row.ID]]
		g.Bytes += j.OutBytes
		if lr.Start != nil && lr.Finish != nil {
			g.Seconds += int64(lr.Finish.Sub(*lr.Start).Seconds())
		}
		if lr.Start != nil && (g.Start == nil || lr.Start.Before(*g.Start)) {
			g.Start = lr.Start
		}
		if lr.Finish != nil && (g.Finish == nil || lr.Finish.After(*g.Finish)) {
			g.Finish = lr.Finish
		}
		if n := t.on(lr.Row); n != nil && !slices.Contains(g.Nodes, t.s.names[*n]) {
			g.Nodes = append(g.Nodes, t.s.names[*n])
		}
	}
	return g, nil
}

// seasonLabel is a season group's label, such as The Office season 2.
func seasonLabel(ctx context.Context, r storage.Reader[storage.Library], row Row, level works.LevelID) (string, error) {
	levels, err := works.LevelsOf(ctx, r, row.Work)
	if err != nil {
		return "", fmt.Errorf("read the Work's levels: %w", err)
	}
	i := slices.IndexFunc(levels, func(l works.Level) bool { return l.ID == level })
	if i < 0 {
		return row.Snapshot.Work, nil
	}
	label := levels[i].Label
	first, size := utf8.DecodeRuneInString(label)
	return row.Snapshot.Work + " " + string(unicode.ToLower(first)) + label[size:], nil
}

func laneAt(l Lane) int { return slices.Index(lanes, l) }

// after reports whether r runs after the row cursor names.
func after(r Row, c Cursor) bool {
	lane, pos, _ := strings.Cut(string(c), "/")
	if d := cmp.Compare(laneAt(r.Lane), laneAt(Lane(lane))); d != 0 {
		return d > 0
	}
	return string(r.Position) > pos
}

func cursorOf(r Row) Cursor { return Cursor(string(r.Lane) + "/" + string(r.Position)) }

// pageOf is the PageSize entries after cursor, and whether more follow.
func pageOf(entries []folded, c Cursor) ([]folded, bool) {
	if c != "" {
		i := slices.IndexFunc(entries, func(e folded) bool { return after(e.anchor, c) })
		if i < 0 {
			return nil, false
		}
		entries = entries[i:]
	}
	if len(entries) > PageSize {
		return entries[:PageSize], true
	}
	return entries, false
}

// fill reads what each shown row's words name: its attempts, its Items'
// labels, the Target its source fills and the servers a transcode wait
// names.
func (t *tab) fill(ctx context.Context, r storage.Reader[storage.Library], shown []ListRow) (map[ID]ListRow, error) {
	out := make(map[ID]ListRow, len(shown))
	if len(shown) == 0 {
		return out, nil
	}
	var ws []works.WorkID
	for _, lr := range shown {
		if !slices.Contains(ws, lr.Row.Work) {
			ws = append(ws, lr.Row.Work)
		}
	}
	items, err := works.ItemsOf(ctx, r, works.Only(ws))
	if err != nil {
		return nil, fmt.Errorf("read the Items: %w", err)
	}
	sourced, err := t.c.cfg.Files.Sourced(ctx, r, works.Only(ws))
	if err != nil {
		return nil, fmt.Errorf("read what each Work's files are made from: %w", err)
	}
	servers := map[role.NodeID][]string{}
	q := db.New(r)
	for _, lr := range shown {
		if lr.Attempts, err = attemptsOf(ctx, q, lr.Row.ID); err != nil {
			return nil, err
		}
		lr.FailedOn = failedOn(lr.Row, lr.Attempts)
		lr.Items = labelsOf(lr.Row.Items, items[lr.Row.Work])
		lr.Source = t.sourceOf(lr.Row, sourced[lr.Row.Work])
		if w := lr.Wait; w != nil && w.Code == plan.Transcode && w.Node != nil {
			if _, ok := servers[*w.Node]; !ok {
				if servers[*w.Node], err = transcodingOn(ctx, t.c, r, *w.Node); err != nil {
					return nil, err
				}
			}
			lr.Servers = servers[*w.Node]
		}
		out[lr.Row.ID] = lr
	}
	return out, nil
}

func transcodingOn(ctx context.Context, c *Conversions, r storage.Reader[storage.Library], node role.NodeID) ([]string, error) {
	ts, err := c.cfg.TranscodingOn(ctx, r, node)
	if err != nil {
		return nil, fmt.Errorf("read the Media Servers on a Node: %w", err)
	}
	out := []string{}
	for _, t := range ts {
		if t.Known && t.Now {
			out = append(out, t.Name)
		}
	}
	return out, nil
}

// failedOn names the Nodes r's attempts since its last Retry failed on.
func failedOn(r Row, as []Attempt) []string {
	var counted []Attempt
	for _, a := range as {
		if a.N >= r.BudgetFrom && a.Outcome != "" && a.Outcome != AttemptPutBack {
			counted = append(counted, a)
		}
	}
	return names(counted)
}

func labelsOf(ids []works.ItemID, items []works.Item) []ItemLabel {
	out := []ItemLabel{}
	for _, it := range items {
		if slices.Contains(ids, it.ID) {
			out = append(out, ItemLabel{ID: it.ID, Label: it.Label})
		}
	}
	return out
}

// sourceOf is the Target the made-from cell of r makes its file from.
func (t *tab) sourceOf(r Row, cells []files.Sourced) RowSource {
	if r.Target == nil {
		return RowSource{}
	}
	i := slices.IndexFunc(cells, func(s files.Sourced) bool {
		return s.Target == *r.Target && s.Edition == r.Edition && len(r.Items) > 0 && slices.Contains(s.Items, r.Items[0])
	})
	if i < 0 {
		return RowSource{}
	}
	from := cells[i].From
	return RowSource{Target: &from, Name: t.s.goals[from].Name}
}

// Group is one folded group's rows, a page at a time.
func (c *Conversions) Group(ctx context.Context, id GroupID, cur Cursor) (Group, error) {
	var out Group
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		t, err := c.tab(ctx, r)
		if err != nil {
			return err
		}
		ids, err := t.c.groupMembers(ctx, r, id)
		if err != nil {
			return err
		}
		var rows []folded
		for _, row := range t.rows() {
			if slices.Contains(ids, row.ID) && bandOf(row) == BandNext {
				lr := t.listRow(row)
				rows = append(rows, folded{Entry: Entry{Row: &lr}, anchor: row})
			}
		}
		page, more := pageOf(rows, cur)
		out.Rows = make([]ListRow, len(page))
		for i, e := range page {
			out.Rows[i] = *e.Row
		}
		if more {
			out.Cursor = cursorOf(page[len(page)-1].anchor)
		}
		filled, err := t.fill(ctx, r, out.Rows)
		for i, lr := range out.Rows {
			out.Rows[i] = filled[lr.Row.ID]
		}
		return err
	})
	return out, err
}

// groupMembers is the unfinished Conversions of group id, ErrNoConversion
// for an id no group has.
func (c *Conversions) groupMembers(ctx context.Context, r storage.Reader[storage.Library], id GroupID) ([]ID, error) {
	q := db.New(r)
	if rest, ok := strings.CutPrefix(string(id), backfillPrefix); ok {
		lane, raw, _ := strings.Cut(rest, ":")
		u, err := storage.ParseUUID(raw)
		if err != nil {
			return nil, ErrNoConversion
		}
		ids, err := q.ListOpenOfBackfill(ctx, BackfillID{UUID: u})
		if err != nil {
			return nil, fmt.Errorf("read the backfill's Conversions: %w", err)
		}
		return c.inLane(ctx, q, ids, Lane(lane))
	}
	rest, ok := strings.CutPrefix(string(id), seasonPrefix)
	parts := strings.Split(rest, ":")
	if !ok || len(parts) != 3 {
		return nil, ErrNoConversion
	}
	w, errW := storage.ParseUUID(parts[1])
	l, errL := storage.ParseUUID(parts[2])
	if errW != nil || errL != nil {
		return nil, ErrNoConversion
	}
	return seasonMembers(ctx, r, works.WorkID{UUID: w}, l, Lane(parts[0]))
}

// seasonMembers is the unfinished Conversions of work in lane, outside a
// backfill, whose first Item is in level.
func seasonMembers(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID, level storage.UUID, lane Lane) ([]ID, error) {
	raws, err := db.New(r).ListOpenOfWork(ctx, work)
	if err != nil {
		return nil, fmt.Errorf("read the Work's Conversions: %w", err)
	}
	items, err := works.ItemsOf(ctx, r, works.Only([]works.WorkID{work}))
	if err != nil {
		return nil, fmt.Errorf("read the Items: %w", err)
	}
	var out []ID
	for _, raw := range raws {
		row, err := rowOf(raw)
		if err != nil {
			return nil, err
		}
		if row.Backfill != nil || row.Lane != lane || len(row.Items) == 0 {
			continue
		}
		i := slices.IndexFunc(items[work], func(it works.Item) bool { return it.ID == row.Items[0] })
		if i >= 0 && items[work][i].LevelID.UUID == level {
			out = append(out, row.ID)
		}
	}
	if len(out) == 0 {
		return nil, ErrNoConversion
	}
	return out, nil
}

func (c *Conversions) inLane(ctx context.Context, q *db.Queries, ids []ID, lane Lane) ([]ID, error) {
	var out []ID
	for _, id := range ids {
		r, err := readRow(ctx, q, id)
		if err != nil {
			return nil, err
		}
		if r.Lane == lane {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil, ErrNoConversion
	}
	return out, nil
}

// row is Conversion id as the tab shows it, with its forecast.
func (c *Conversions) row(ctx context.Context, id ID) (ListRow, error) {
	var out ListRow
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		t, err := c.tab(ctx, r)
		if err != nil {
			return err
		}
		row, ok := t.s.rows[id]
		if !ok {
			i := slices.IndexFunc(t.failed, func(f Row) bool { return f.ID == id })
			if i < 0 {
				if row, err = readRow(ctx, db.New(r), id); err != nil {
					return err
				}
			} else {
				row = t.failed[i]
			}
		}
		filled, err := t.fill(ctx, r, []ListRow{t.listRow(row)})
		out = filled[id]
		return err
	})
	return out, err
}

// Shell is what the app shell's counts read: the Conversions running,
// whether one failed and waits for an Admin, and whether no Node can
// convert one.
type Shell struct {
	Running        int
	Failed, NoNode bool
}

func (c *Conversions) Shell(ctx context.Context) (Shell, error) {
	n, err := db.New(c.cfg.Library).CountForShell(ctx)
	if err != nil {
		return Shell{}, fmt.Errorf("count the Conversions: %w", err)
	}
	return Shell{Running: int(n.Started), Failed: n.Failed > 0, NoNode: n.Causes > 0}, nil
}

// Older is how many made files t's Conversion settings before version
// made.
func (c *Conversions) Older(ctx context.Context, t targets.TargetID, version int) (int, error) {
	var n int
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		made, err := files.MadeBefore(ctx, r, t, version)
		n = len(made)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("read the files older settings made: %w", err)
	}
	return n, nil
}

// Titles is the title of each of ws.
func (c *Conversions) Titles(ctx context.Context, ws []works.WorkID) (map[works.WorkID]string, error) {
	out := make(map[works.WorkID]string, len(ws))
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		summaries, err := works.SummariesOf(ctx, r, ws)
		for id, s := range summaries {
			out[id] = s.Label
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read the Works: %w", err)
	}
	return out, nil
}
