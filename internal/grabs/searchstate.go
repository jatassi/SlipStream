package grabs

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Phase is a search group's queue state.
type Phase uint8

const (
	Idle Phase = iota
	InQueue
	Searching
)

var phaseNames = [...]string{Idle: "idle", InQueue: "queued", Searching: "searching"}

func (p Phase) String() string { return phaseNames[p] }

// GroupState is one search group of a Work as the hero, a level's header,
// Wanted and the cells read it.
type GroupState struct {
	Key   indexers.GroupKey
	Label string
	Phase Phase
	// Lane and Trigger are the queued entry's, else the open run's, ""
	// while Idle.
	Lane    string
	Trigger searches.Trigger
	// Again is whether it is queued again behind its own open run.
	Again bool
	// Behind is what holds it, nil when nothing does.
	Behind *Behind
	// Last is its latest finished run, nil before the first.
	Last *LastRun
	// Next is its due time while Idle and Backlog search is on, else zero.
	Next time.Time
}

// Behind is what holds a group: the Works ahead of it in its lane while
// queued, and the Indexer whose search limit holds its run's lines, or
// the start of its run.
type Behind struct {
	Ahead     int
	AheadLane string
	// AheadFrom is the trigger every Work ahead was queued by, "" when
	// they differ.
	AheadFrom searches.Trigger
	Indexer   indexers.ID
	Name      string
	// Until is when the limit frees a search, zero while Starved.
	Until time.Time
	// Starved is RSS sync's reserve filling the Indexer's whole limit.
	Starved bool
}

func (b Behind) holds() bool { return b.Ahead > 0 || !b.Indexer.IsZero() }

// LastRun is how a group's latest finished run ended.
type LastRun struct {
	At      time.Time
	Outcome Outcome
	Found   int
	Grabs   int
}

// Wait is one cell waiting for a better Release: when it grabs and the
// Releases it holds.
type Wait struct {
	Cell       files.CellKey
	Until      time.Time
	Candidates int
}

// Stop is a cell failed Grabs stopped SlipStream searching again at once,
// with each counted failure up to the stop, oldest first.
type Stop struct {
	Cell     files.CellKey
	Failures []CountedFailure
	// Next is its group's due time while Backlog search is on.
	Next time.Time
}

// CountedFailure is one failed Grab that counted toward a stop. Grab is
// zero once the Grab is pruned.
type CountedFailure struct {
	Code Code
	At   time.Time
	Grab GrabID
}

// WorkSearch is what SlipStream is doing to fill one Work's wanted cells.
type WorkSearch struct {
	Work   works.WorkID
	Groups []GroupState
	// Waits are by when they grab, earliest first.
	Waits []Wait
	Stops []Stop
}

// Lead is the hero's line: the first group searching, else the first
// queued, else the Work's earliest wait, else the group of its latest
// result. Of counts the groups queued or searching, for "Searching
// season 3 of 10".
type Lead struct {
	Group *GroupState
	Of    int
	Wait  *Wait
}

func (s WorkSearch) Lead() Lead {
	var l Lead
	for _, g := range s.Groups {
		if g.Phase != Idle {
			l.Of++
		}
	}
	for _, p := range []Phase{Searching, InQueue} {
		if i := slices.IndexFunc(s.Groups, func(g GroupState) bool { return g.Phase == p }); i >= 0 {
			l.Group = &s.Groups[i]
			return l
		}
	}
	if len(s.Waits) > 0 {
		l.Wait = &s.Waits[0]
		return l
	}
	l.Group = s.latest()
	return l
}

// Phase is the Lead's: what Wanted's row reads for the Work.
func (s WorkSearch) Phase() Phase {
	if g := s.Lead().Group; g != nil {
		return g.Phase
	}
	return Idle
}

// Last is the Work's latest result, nil before its first.
func (s WorkSearch) Last() *LastRun {
	if g := s.latest(); g != nil {
		return g.Last
	}
	return nil
}

// Next is the earliest due time of its groups, zero when none has one.
func (s WorkSearch) Next() time.Time {
	var next time.Time
	for _, g := range s.Groups {
		if !g.Next.IsZero() && (next.IsZero() || g.Next.Before(next)) {
			next = g.Next
		}
	}
	return next
}

func (s WorkSearch) latest() *GroupState {
	var out *GroupState
	for i, g := range s.Groups {
		if g.Last != nil && (out == nil || g.Last.At.After(out.Last.At)) {
			out = &s.Groups[i]
		}
	}
	return out
}

// states is the queue rows of a batch of Works, each kind read once: the
// one model under Searching, Wanted and the cells' search facts.
type states struct {
	due     []db.ListEveryDueRow
	runs    []db.ListRunItemsRow
	waits   []db.ListWaitsRow
	backlog bool
}

func readStates(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (states, error) {
	q := db.New(r)
	var s states
	var err error
	s.due, err = works.Read(ws, func() ([]db.ListEveryDueRow, error) { return q.ListEveryDue(ctx) }, func(ids []works.WorkID) ([]db.ListEveryDueRow, error) {
		rows, err := q.ListDueOfWorks(ctx, ids)
		return convert(rows, func(r db.ListDueOfWorksRow) db.ListEveryDueRow { return db.ListEveryDueRow(r) }), err
	})
	if err != nil {
		return states{}, err
	}
	s.runs, err = works.Read(ws, func() ([]db.ListRunItemsRow, error) { return q.ListRunItems(ctx) }, func(ids []works.WorkID) ([]db.ListRunItemsRow, error) {
		rows, err := q.ListRunItemsOfWorks(ctx, ids)
		return convert(rows, func(r db.ListRunItemsOfWorksRow) db.ListRunItemsRow { return db.ListRunItemsRow(r) }), err
	})
	if err != nil {
		return states{}, err
	}
	s.waits, err = works.Read(ws, func() ([]db.ListWaitsRow, error) { return q.ListWaits(ctx) }, func(ids []works.WorkID) ([]db.ListWaitsRow, error) {
		rows, err := q.ListWaitsOfWorks(ctx, ids)
		return convert(rows, func(r db.ListWaitsOfWorksRow) db.ListWaitsRow { return db.ListWaitsRow(r) }), err
	})
	if err != nil {
		return states{}, err
	}
	settings, err := q.GetSettings(ctx)
	if err != nil {
		return states{}, err
	}
	s.backlog = BacklogOn.Value(settings.BacklogOn)
	return s, nil
}

// byWork is each Work's search, its groups in key order and unlabeled.
func (s states) byWork() map[works.WorkID]*WorkSearch {
	out := map[works.WorkID]*WorkSearch{}
	of := func(w works.WorkID) *WorkSearch {
		if out[w] == nil {
			out[w] = &WorkSearch{Work: w}
		}
		return out[w]
	}
	group := func(w works.WorkID, key indexers.GroupKey) *GroupState {
		ws := of(w)
		i := slices.IndexFunc(ws.Groups, func(g GroupState) bool { return g.Key == key })
		if i < 0 {
			ws.Groups = append(ws.Groups, GroupState{Key: key})
			i = len(ws.Groups) - 1
		}
		return &ws.Groups[i]
	}
	for _, d := range s.due {
		g := group(d.WorkID, d.GroupKey)
		switch {
		case !d.SearchID.IsZero():
			g.Phase, g.Again = Searching, d.Lane != nil
		case d.Lane != nil:
			g.Phase = InQueue
		case s.backlog:
			g.Next = d.DueAt.Time()
		}
		if d.Lane != nil {
			g.Lane, g.Trigger = laneOf(d.Lane).String(), searches.Trigger(*d.Trigger)
		}
	}
	for _, run := range s.runs {
		group(run.WorkID, *run.GroupKey).Last = lastOf(run)
	}
	for _, w := range s.waits {
		ws := of(w.WorkID)
		ws.Waits = append(ws.Waits, Wait{
			Cell: files.CellKey{Item: w.ItemID, Edition: w.Edition, Target: w.TargetID}, Until: w.UntilAt.Time(), Candidates: int(w.Candidates),
		})
	}
	for _, ws := range out {
		slices.SortFunc(ws.Groups, func(a, b GroupState) int { return cmp.Compare(a.Key, b.Key) })
	}
	return out
}

func lastOf(run db.ListRunItemsRow) *LastRun {
	return &LastRun{At: run.FinishedAt.Time(), Outcome: *run.Outcome, Found: int(run.Candidates), Grabs: int(run.Grabs)}
}

// items is each Item's search fact: its latest run, and its group's due
// time while that group is idle and Backlog search is on.
func (s states) items() map[works.ItemID]files.SearchFact {
	due := map[works.WorkID]map[indexers.GroupKey]time.Time{}
	for _, d := range s.due {
		if s.backlog && d.SearchID.IsZero() && d.Lane == nil {
			if due[d.WorkID] == nil {
				due[d.WorkID] = map[indexers.GroupKey]time.Time{}
			}
			due[d.WorkID][d.GroupKey] = d.DueAt.Time()
		}
	}
	out := map[works.ItemID]files.SearchFact{}
	for _, run := range s.runs {
		out[run.ItemID] = files.SearchFact{
			NotFound:   *run.Outcome == NothingFound || *run.Outcome == NothingAcceptable,
			SearchedAt: run.FinishedAt.Time(), NextAt: due[run.WorkID][*run.GroupKey],
		}
	}
	return out
}

// Searching is what SlipStream is doing for work: each search group of
// its counted Items in tree order with its state, its waits and its
// stopped cells.
func (gr *Grabs) Searching(ctx context.Context, work works.WorkID) (WorkSearch, error) {
	var out WorkSearch
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		bare, err := works.BareIn(ctx, r, work)
		if err != nil {
			return err
		}
		s, err := readStates(ctx, r, works.Only([]works.WorkID{work}))
		if err != nil {
			return err
		}
		read := WorkSearch{Work: work}
		if ws := s.byWork()[work]; ws != nil {
			read = *ws
		}
		var counted []works.ItemID
		for _, it := range bare.Items {
			if it.Counts() {
				counted = append(counted, it.ID)
			}
		}
		groups, err := gr.groupsOf(ctx, r, bare.Summary, counted)
		if err != nil {
			return err
		}
		out = WorkSearch{Work: work, Waits: read.Waits}
		of := map[works.ItemID]indexers.GroupKey{}
		for _, g := range groups {
			for _, it := range g.Items {
				of[it] = g.Key
			}
			i := slices.IndexFunc(read.Groups, func(s GroupState) bool { return s.Key == g.Key })
			state := GroupState{Key: g.Key}
			if i >= 0 {
				state = read.Groups[i]
				read.Groups = slices.Delete(read.Groups, i, i+1)
			}
			state.Label = g.Label
			out.Groups = append(out.Groups, state)
		}
		out.Groups = append(out.Groups, read.Groups...)
		if err := gr.holding(ctx, r, bare.ContentType, &out); err != nil {
			return err
		}
		out.Stops, err = stopsOf(ctx, r, work, out.Groups, of)
		return err
	})
	if err != nil {
		return WorkSearch{}, err
	}
	return out, nil
}

// groupsOf is the search groups of items, items of w, none when its
// Content Type can't search.
func (gr *Grabs) groupsOf(ctx context.Context, r storage.Reader[file], w works.Summary, items []works.ItemID) ([]indexers.Group, error) {
	terms, ok := gr.cfg.Terms[w.ContentType]
	if !ok || len(items) == 0 {
		return nil, nil
	}
	return terms.Groups(ctx, r, w.ID, items)
}

// holding fills each of s's groups' open run lane and trigger, and what
// holds it: the Works ahead in its lane, and the Indexer whose search
// limit holds its run's planned lines, or would hold its run's start.
func (gr *Grabs) holding(ctx context.Context, r storage.Reader[file], typ works.TypeID, s *WorkSearch) error {
	q := db.New(r)
	rooms, err := gr.cfg.Indexers.Room(ctx, r)
	if err != nil {
		return err
	}
	runs, err := q.ListOpenRunsOfWork(ctx, s.Work)
	if err != nil {
		return err
	}
	planned, err := q.ListPlannedOfWork(ctx, s.Work)
	if err != nil {
		return err
	}
	var lines []indexers.Line
	lanes := map[pace.Lane][]db.ListLaneRow{}
	for i := range s.Groups {
		g := &s.Groups[i]
		var b Behind
		switch g.Phase {
		case Searching:
			run := runs[slices.IndexFunc(runs, func(run db.ListOpenRunsOfWorkRow) bool { return *run.GroupKey == g.Key })]
			if !g.Again {
				g.Lane, g.Trigger = laneOf(run.Lane).String(), searches.Trigger(*run.Trigger)
			}
			for _, p := range planned {
				if p.SearchID == run.ID {
					b.limited(rooms, p.IndexerID, laneOf(run.Lane))
				}
			}
		case InQueue:
			lane := laneNamed(g.Lane)
			if lanes[lane] == nil {
				if lanes[lane], err = q.ListLane(ctx, new(int64(lane))); err != nil {
					return err
				}
			}
			b.ahead(lanes[lane], s.Work, g.Key, lane)
			if lines == nil {
				if lines, err = gr.cfg.Indexers.Lines(ctx, r, typ, indexers.AutoSearch); err != nil {
					return err
				}
			}
			for _, l := range lines {
				if l.Skipped == nil {
					b.limited(rooms, l.Indexer, lane)
				}
			}
		case Idle:
		}
		if b.holds() {
			g.Behind = &b
		}
	}
	return nil
}

func laneNamed(name string) pace.Lane {
	i := slices.IndexFunc(pace.Lanes, func(l pace.Lane) bool { return l.String() == name })
	return pace.Lanes[i]
}

// limited names Indexer id when its search limit holds a line of lane,
// keeping the first Indexer named.
func (b *Behind) limited(rooms map[indexers.ID]indexers.Room, id indexers.ID, lane pace.Lane) {
	room, ok := rooms[id]
	if !ok || !b.Indexer.IsZero() {
		return
	}
	f := room.Background
	if lane.Press() {
		f = room.Press
	}
	if f.Free > 0 || (f.Until.IsZero() && !f.Starved) {
		return
	}
	b.Indexer, b.Name, b.Until, b.Starved = id, room.Name, f.Until, f.Starved
}

// ahead counts the Works ahead of work's group key in lane's rows, in
// lane order, and the trigger they share.
func (b *Behind) ahead(rows []db.ListLaneRow, work works.WorkID, key indexers.GroupKey, lane pace.Lane) {
	seen := map[works.WorkID]bool{}
	var from searches.Trigger
	mixed := false
	for _, row := range rows {
		if row.WorkID == work && row.GroupKey == key {
			break
		}
		if row.WorkID == work || seen[row.WorkID] {
			continue
		}
		seen[row.WorkID] = true
		t := searches.Trigger(*row.Trigger)
		if from != "" && t != from {
			mixed = true
		}
		from = t
	}
	b.Ahead = len(seen)
	if b.Ahead == 0 {
		return
	}
	b.AheadLane = lane.String()
	if !mixed {
		b.AheadFrom = from
	}
}

// stopsOf is work's stopped cells with the failures that stopped them,
// each with its group's due time from groups, of naming each Item's
// group.
func stopsOf(ctx context.Context, r storage.Reader[file], work works.WorkID, groups []GroupState, of map[works.ItemID]indexers.GroupKey) ([]Stop, error) {
	rows, err := db.New(r).ListStopsOfWork(ctx, work)
	if err != nil {
		return nil, err
	}
	var out []Stop
	for _, row := range rows {
		cell := files.CellKey{Item: row.ItemID, Edition: row.Edition, Target: row.TargetID}
		if len(out) == 0 || out[len(out)-1].Cell != cell {
			s := Stop{Cell: cell}
			if i := slices.IndexFunc(groups, func(g GroupState) bool { return g.Key == of[row.ItemID] }); i >= 0 {
				s.Next = groups[i].Next
			}
			out = append(out, s)
		}
		f := CountedFailure{Code: row.Code, At: row.FailedAt.Time()}
		if row.Kept {
			f.Grab = row.GrabID
		}
		last := &out[len(out)-1]
		last.Failures = append(last.Failures, f)
	}
	return out, nil
}

func convert[F, T any](rows []F, to func(F) T) []T {
	out := make([]T, len(rows))
	for i, r := range rows {
		out[i] = to(r)
	}
	return out
}
