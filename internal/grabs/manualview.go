package grabs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// ManualView is a Work's Manual search as the sheet shows it, every
// sentence written.
type ManualView struct {
	ID ManualID
	// Scope is what it was started for: its levels and Items, or its one
	// cell, all empty for the whole Work.
	Scope ShownScope
	// Title is the scope in words: "Heat (1995)", "Season 2 of Severance".
	Title string
	// Query is the typed query, "" when each run sent its Item's terms.
	Query     string
	Started   time.Time
	StartedBy Actor
	// Searching is true while a run has a line with no outcome.
	Searching bool
	// Queryable is true when the scope is one Item, so a typed query can
	// search it.
	Queryable bool
	// Line is "Searching 6 Indexers. 4 have responded." or "SlipStream
	// searched 6 Indexers. 2 didn't respond."
	Line     string
	Indexers []IndexerLine
	Plan     ManualPlan
	Folded   Folded
	// Releases is how many Releases All Releases lists.
	Releases int
	// Pins is the pins still in effect, for the sheet to keep. A pin that
	// names no Release, or that no longer fills its cell, is dropped.
	Pins []Pin
}

// ShownScope is a Manual search's scope as it was started.
type ShownScope struct {
	Levels []works.LevelID
	Items  []works.ItemID
	Cell   *files.CellKey
}

// IndexerLine is one Indexer across the search's runs.
type IndexerLine struct {
	// Indexer is zero once the Indexer is deleted.
	Indexer indexers.ID
	Name    string
	// Sentence is S26 (Indexers and Trackers)'s outcome sentence, or
	// "TorrentLeech searched 6 of 10 episodes. Its limit resets at 14:05."
	Sentence string
	// Done is true once it has an outcome for every run that asked it.
	Done bool
	// Automatic is true when automatic search uses it.
	Automatic bool
	// Failed is a fault at the site, which the sheet shows red with Retry
	// and Open Indexer.
	Failed bool
}

// ManualPlan is the fewest Grabs that fill the scope's missing, failed and
// upgradable cells, worked out around the pins.
type ManualPlan struct {
	// Headline is "2 Grabs get you everything.", "1 Grab gets you 2 of the
	// 3 files you're missing. SlipStream hasn't found a Release for 4K
	// yet." or "Every Target here already has its file."
	Headline string
	// Rows is a Grab or a cell nothing fills, in Want order.
	Rows []Row
	// Held is each cell an unfinished Grab downloads, left out of the plan.
	Held []HeldCell
}

// Row is a plan row in words. Pick is nil for a cell nothing fills.
type Row struct {
	// Cells is the cells it fills, the first leading it. A pin chosen from
	// its choices replaces the pin of its first cell.
	Cells   []files.CellKey
	Pick    *PlanPick
	Pinned  bool
	Outcome string // "It gets you 1080p and 720p for E05."
	// Missing is a cell nothing fills: "Missing: 720p for E05.", "Could be
	// better: 1080p SDR." or "Failed: 1080p for E10."
	Missing string
	// Cause is Missing's plain cause: "No Indexer listed a 720p Release."
	Cause string
	// Failure is the cell's last Grab's failure, for a failed cell.
	Failure string
	Choices []Choice
}

// PlanPick is what a pick of the row sends back verbatim, with the
// Release it names.
type PlanPick struct {
	Release ReleaseDetails
	// Lifts is the soft refusals it gets past, each a sentence.
	Lifts   []Worded
	Edition *EditionOverride
	// Routed is the client Tracker routing sends it to in words,
	// "qBittorrent on Den", "" while no Node may take it.
	Routed string
}

// EditionOverride is the Edition a Release is grabbed as when the cell
// leading its cells doesn't take its cut, nil when it does.
type EditionOverride struct {
	Edition targets.Edition
	// Words is "Its name gives the Extended Edition, which the Work
	// doesn't want. It's grabbed as the Standard Edition."
	Words string
}

// Choice is one other Release that could fill a row's first cell.
type Choice struct {
	Pin     Pin
	Release ReleaseDetails
	// Cells is the cells its row fills once it is pinned, the first
	// leading.
	Cells   []files.CellKey
	Lifts   []Worded
	Edition *EditionOverride
	// Note is "From TorrentLeech, which automatic search doesn't use."
	Note string
}

// HeldCell is a cell a Grab downloads, left out of the plan.
type HeldCell struct {
	Cell files.CellKey
	Grab GrabID
	// Words is "A Grab is downloading WEBDL-1080p · FLUX · 6.2 GB for
	// 1080p."
	Words string
	Stage Stage
}

// Folded is "SlipStream wouldn't grab 14 more Releases." and its groups.
type Folded struct {
	Line   string
	Groups []FoldGroup
}

// FoldGroup is one group of the Releases SlipStream wouldn't grab: "8 are
// Bluray-2160p, which no Target here takes." Fault is a real fault, which
// the sheet shows red.
type FoldGroup struct {
	Words string
	Fault bool
}

// ReleaseRef names a Manual run's Release: its run and its position in
// arrival order. A pick names its group's primary when the sheet drew
// it, and the grab Write takes whichever copy is primary then.
type ReleaseRef struct {
	Run      SearchID
	Position int
}

// Pin is an Admin's swap: Release fills Cell, whatever ranks above it,
// grabbed as Cell's Edition.
type Pin struct {
	Cell    files.CellKey
	Release ReleaseRef
}

// ReleaseDetails is one Release with its facts and its verdicts in words.
type ReleaseDetails struct {
	Ref   ReleaseRef
	Title string
	// Words is the Release in words, as release.Describe writes it.
	Words string
	// Indexers is the Indexers that listed it, by priority.
	Indexers []string
	Tracker  string
	Size     int64
	// Published is zero when the Indexer sent no date.
	Published time.Time
	Seeders   int
	Freeleech bool
	Formats   []formats.ID
	// Verdicts is one per Target of its Item: "It would fill 1080p.", "It
	// meets 720p. It isn't better than the file you have."
	Verdicts []string
	// LostAt is where it lost to its first cell's pick, nil when it
	// didn't.
	LostAt *Loss
	// Blocked is "It's been on the Blocklist since 2 Nov because
	// TorrentLeech reported it unregistered.", "" when it isn't on it.
	Blocked string
}

// Offer is one Release as All Releases lists it, one per group of copies,
// in rank order.
type Offer struct {
	Release ReleaseDetails
	// Rank is its place in the ranking from 1, 0 when no cell's Target
	// takes it. Score and Quality are its best Target's, for sorting.
	Rank    int
	Score   int
	Quality int
	Grab    GrabOffer
}

// GrabOffer is what grabbing an Offer by itself does, or why it can't.
type GrabOffer struct {
	// Grabbable is true when it fills some cell and its Indexer would take
	// it now. A Release on the Blocklist is grabbable only with Unblock.
	Grabbable bool
	Outcome   string
	// Cells is the cells it fills, the first leading, any a Grab
	// downloads among them.
	Cells   []files.CellKey
	Lifts   []Worded
	Edition *EditionOverride
	// Cancels is each Grab downloading a cell it would also fill.
	Cancels []HeldCell
	// Waits is "TorrentLeech is at its grab limit until 14:05."
	Waits string
	// Refusal is "It's a pack of more than one episode, and SlipStream
	// can't import packs yet."
	Refusal string
}

// manualIn is everything the view reads, in one snapshot.
type manualIn struct {
	manual db.GrabsManual
	scope  ShownScope
	// cell is the one cell a cell scope plans for, nil for any other scope.
	cell  *files.CellKey
	tree  works.Tree
	nouns works.Description
	// describe is a Release titled title, of size bytes, in words.
	describe func(title string, size int64) string
	runs     []Search
	// heard is each run's stored Releases, in arrival order.
	heard   map[SearchID][]heardRelease
	cells   []files.Cell
	grabs   []db.ListCellGrabsOfWorkRow
	judging judging
	blocks  blocks
	// standings is how each Indexer stands now, which the plan chooses by
	// as automatic search takes.
	standings indexers.Standings
	// routed is where Tracker routing sends a Grab of each Indexer, in
	// words.
	routed map[indexers.ID]string
	line   func(*indexers.Outcome) string
	// clock is a time of day in sentences, "14:05".
	clock func(time.Time) string
	now   time.Time
}

// heardRelease is one Release a Manual run stored as it was heard: the
// Release, its title's reading, the formats it matched and its Indexer's
// name then.
type heardRelease struct {
	listed  listed
	read    release.Read
	matched formats.Set
	indexer string
}

// manualInput reads, in r, the view of work's Manual search.
func (gr *Grabs) manualInput(ctx context.Context, r storage.Reader[file], work works.WorkID) (manualIn, error) {
	q := db.New(r)
	m, err := q.GetManualOfWork(ctx, work)
	if errors.Is(err, storage.ErrNoRows) {
		return manualIn{}, ErrNoManualSearch
	}
	if err != nil {
		return manualIn{}, err
	}
	in := manualIn{manual: m, now: storage.Now().Time()}
	if in.tree, err = gr.cfg.Works.TreeIn(ctx, r, work); err != nil {
		return manualIn{}, err
	}
	typ := in.tree.ContentType
	in.nouns, _ = gr.cfg.Works.Describe(typ)
	names := gr.cfg.Names[typ]
	in.describe = func(title string, size int64) string { return releaseWordsOf(title, size, names) }
	in.line = func(o *indexers.Outcome) string { return gr.cfg.Indexers.Line(typ, o) }
	in.clock = gr.cfg.Indexers.Clock
	var scope storedScope
	if err := json.Unmarshal([]byte(m.Scope), &scope); err != nil {
		return manualIn{}, fmt.Errorf("decode Manual search %s's scope: %w", m.ID, err)
	}
	if in.scope, err = scope.shown(); err != nil {
		return manualIn{}, err
	}
	in.cell = in.scope.Cell
	if in.runs, err = manualRuns(ctx, q, m.ID); err != nil {
		return manualIn{}, err
	}
	counts, err := q.CountKeptOfManual(ctx, m.ID)
	if err != nil {
		return manualIn{}, err
	}
	kept := make(map[SearchID]int, len(counts))
	for _, c := range counts {
		kept[c.SearchID] = int(c.Kept)
	}
	in.heard, err = gr.heardIn(m.ID, kept, func(run SearchID) ([]db.GrabsRelease, error) { return q.ListKeptReleasesOfSearch(ctx, run) })
	if err != nil {
		return manualIn{}, err
	}
	var all []assessed
	for _, hs := range in.heard {
		for _, h := range hs {
			all = append(all, assessed{listed: h.listed})
		}
	}
	if in.blocks, err = blocksOf(ctx, r, all); err != nil {
		return manualIn{}, err
	}
	if in.cells, err = gr.cfg.Files.Cells(ctx, r, work); err != nil {
		return manualIn{}, err
	}
	if in.grabs, err = q.ListCellGrabsOfWork(ctx, work); err != nil {
		return manualIn{}, err
	}
	expect, err := gr.cfg.Terms[typ].Expect(ctx, r, work)
	if err != nil {
		return manualIn{}, err
	}
	if in.judging, err = gr.judgingOf(ctx, r, typ, Search{WorkID: work}, expect); err != nil {
		return manualIn{}, err
	}
	if in.standings, err = gr.cfg.Indexers.Standings(ctx, r); err != nil {
		return manualIn{}, err
	}
	in.routed, err = gr.routedWords(ctx, r, work, in.standings)
	return in, err
}

// shown is s as its ids.
func (s storedScope) shown() (ShownScope, error) {
	var out ShownScope
	for _, l := range s.Levels {
		id, err := storage.ParseUUID(l)
		if err != nil {
			return ShownScope{}, fmt.Errorf("decode a Manual search's level: %w", err)
		}
		out.Levels = append(out.Levels, works.LevelID{UUID: id})
	}
	for _, it := range s.Items {
		id, err := storage.ParseUUID(it)
		if err != nil {
			return ShownScope{}, fmt.Errorf("decode a Manual search's Item: %w", err)
		}
		out.Items = append(out.Items, works.ItemID{UUID: id})
	}
	if c := s.Cell; c != nil {
		var err error
		if out.Cell, err = c.key(); err != nil {
			return ShownScope{}, err
		}
	}
	return out, nil
}

func (c storedCell) key() (*files.CellKey, error) {
	item, err := storage.ParseUUID(c.Item)
	if err != nil {
		return nil, fmt.Errorf("decode a Manual search's cell: %w", err)
	}
	target, err := storage.ParseUUID(c.Target)
	if err != nil {
		return nil, fmt.Errorf("decode a Manual search's cell: %w", err)
	}
	return &files.CellKey{Item: works.ItemID{UUID: item}, Edition: targets.Edition(c.Edition), Target: targets.TargetID{UUID: target}}, nil
}

// manualRuns is Manual search id's runs, each with its lines.
func manualRuns(ctx context.Context, q *db.Queries, id ManualID) ([]Search, error) {
	rows, err := q.ListManualRuns(ctx, id)
	if err != nil {
		return nil, err
	}
	lines, err := q.ListLinesOfManual(ctx, id)
	if err != nil {
		return nil, err
	}
	of := map[SearchID][]SearchIndexer{}
	for _, l := range lines {
		of[l.SearchID] = append(of[l.SearchID], SearchIndexer{Indexer: l.IndexerID, Name: l.IndexerName, Outcome: outcomeOf(l)})
	}
	out := make([]Search, len(rows))
	for i, row := range rows {
		if out[i], err = searchOf(row, of[row.ID]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// decodedRuns is the decoded Releases of each run of the Manual search
// read last. A run's rows are only ever appended, so a run whose count of
// kept rows is unchanged is current, and a read decodes only the runs an
// Indexer answered since. Decoding is a third of a view's cost.
type decodedRuns struct {
	mu     sync.Mutex
	manual ManualID
	runs   map[SearchID][]heardRelease
}

// heardIn is each run of manual's stored Releases, in arrival order, kept
// being each run's count of kept rows and rowsOf a run's rows. It decodes
// only the runs whose count changed since the last read.
func (gr *Grabs) heardIn(manual ManualID, kept map[SearchID]int, rowsOf func(SearchID) ([]db.GrabsRelease, error)) (map[SearchID][]heardRelease, error) {
	d := &gr.decoded
	d.mu.Lock()
	held := d.runs
	if d.manual != manual {
		held = nil
	}
	d.mu.Unlock()
	out := make(map[SearchID][]heardRelease, len(kept))
	for run, n := range kept {
		if hs := held[run]; len(hs) == n {
			out[run] = hs
			continue
		}
		rows, err := rowsOf(run)
		if err != nil {
			return nil, err
		}
		hs := make([]heardRelease, len(rows))
		for i, row := range rows {
			if hs[i], err = gr.heardOf(row); err != nil {
				return nil, err
			}
		}
		out[run] = hs
	}
	d.mu.Lock()
	d.manual, d.runs = manual, out
	d.mu.Unlock()
	return out, nil
}

// heardOf is row, a Release a Manual run stored, decoded.
func (gr *Grabs) heardOf(row db.GrabsRelease) (heardRelease, error) {
	r, err := gr.kept(row)
	if err != nil {
		return heardRelease{}, err
	}
	var ids []string
	if err := json.Unmarshal([]byte(row.Formats), &ids); err != nil {
		return heardRelease{}, fmt.Errorf("search %s: decode the formats of Release %d: %w", row.SearchID, row.Position, err)
	}
	h := heardRelease{listed: listed{Release: r, at: int(row.Position)}, read: row.Read.Read, indexer: row.IndexerName}
	for _, s := range ids {
		id, err := targets.ParseFormatID(s)
		if err != nil {
			return heardRelease{}, fmt.Errorf("search %s: Release %d: %w", row.SearchID, row.Position, err)
		}
		h.matched = append(h.matched, id)
	}
	return h, nil
}

// routedWords is where Tracker routing sends a Grab of each Indexer of
// standings now, in words, read in r.
func (gr *Grabs) routedWords(ctx context.Context, r storage.Reader[file], work works.WorkID, standings indexers.Standings) (map[indexers.ID]string, error) {
	root, err := roots.OfWork(ctx, r, work)
	if err != nil {
		return nil, err
	}
	named, err := nodes.Names(ctx, r)
	if err != nil {
		return nil, err
	}
	names := map[role.NodeID]string{}
	for _, n := range named {
		names[n.ID] = n.Name
	}
	out := map[indexers.ID]string{}
	for id := range standings {
		t, err := gr.cfg.Router.Route(ctx, r, downloadclients.Ask{Root: root.Node, Indexer: id})
		if _, unrouted := errors.AsType[*nodes.Unrouted](err); unrouted {
			out[id] = ""
			continue
		}
		if err != nil {
			return nil, err
		}
		client := "the built-in client on " + names[t.Node]
		if t.Client != nil {
			e, err := downloadclients.ExternalOf(ctx, r, *t.Client)
			if err != nil {
				return nil, err
			}
			client = e.Name + " on " + names[t.Node]
		}
		out[id] = client
	}
	return out, nil
}

// sheet is a Manual search worked out: each run's plan, and what its
// words are written from.
type sheet struct {
	in   manualIn
	runs []runSheet
}

// runSheet is one run worked out.
type runSheet struct {
	run Search
	planIn
	// held is the unfinished Grab of each of item's cells it holds.
	held map[int]db.ListCellGrabsOfWorkRow
	// failure is the failure of the latest Grab of each failed cell.
	failure map[int]string
	// heard is each Release's Indexer's name, by position.
	heard map[int]string
	rows  []planned
	live  map[int]int
}

// walked is the states a Manual plan walks: Unreleased reads as missing,
// since a Manual search ignores release dates.
var walked = []files.State{files.Unreleased, files.Missing, files.Failed, files.Upgradable}

// worked is in worked out around pins, which drops each pin that names no
// Release of a run or no cell of its Item.
func worked(in manualIn, pins []Pin) (sheet, error) {
	s := sheet{in: in}
	gs := grabsByCell(in.grabs)
	for _, run := range in.runs {
		rs := runSheet{run: run, planIn: planIn{take: in.standings.Take}}
		place(&rs, in, gs)
		judgeRun(&rs, in)
		pin(&rs, pins)
		var err error
		if rs.rows, rs.live, err = plan(rs.planIn); err != nil {
			return sheet{}, err
		}
		s.runs = append(s.runs, rs)
	}
	return s, nil
}

// cellGrabs is the latest Grab of each cell, and its unfinished one.
type cellGrabs struct {
	latest, unfinished map[files.CellKey]db.ListCellGrabsOfWorkRow
}

func grabsByCell(grabs []db.ListCellGrabsOfWorkRow) cellGrabs {
	out := cellGrabs{latest: map[files.CellKey]db.ListCellGrabsOfWorkRow{}, unfinished: map[files.CellKey]db.ListCellGrabsOfWorkRow{}}
	for _, g := range grabs {
		key := files.CellKey{Item: g.ItemID, Edition: g.Edition, Target: g.TargetID}
		out.latest[key] = g
		if g.Stage != Imported && g.Stage != Failed {
			out.unfinished[key] = g
		}
	}
	return out
}

// place sets rs's cells: what each holds, which the plan walks and
// which a row may fill.
func place(rs *runSheet, in manualIn, gs cellGrabs) {
	rs.held, rs.failure = map[int]db.ListCellGrabsOfWorkRow{}, map[int]string{}
	for _, c := range in.cells {
		if c.Item.ID == rs.run.ItemID {
			rs.item = append(rs.item, c)
		}
	}
	rs.fill = make([]bool, len(rs.item))
	for k, c := range rs.item {
		key := keyOf(c)
		if g, ok := gs.unfinished[key]; ok {
			rs.held[k] = g
			continue
		}
		if g := gs.latest[key]; c.State == files.Failed && g.FailureMessage != nil {
			rs.failure[k] = *g.FailureMessage
		}
		if !slices.Contains(walked, c.State) {
			continue
		}
		rs.fill[k] = in.cell == nil || in.cell.Item == key.Item
		if in.cell == nil || *in.cell == key {
			rs.walk = append(rs.walk, k)
		}
	}
}

// judgeRun dedupes rs's stored Releases, gates them by the Blocklist
// and judges each at every cell of its Item.
func judgeRun(rs *runSheet, in manualIn) {
	j := in.judging
	j.searched = rs.run.ItemID
	heard := in.heard[rs.run.ID]
	rs.heard = make(map[int]string, len(heard))
	byAt := make(map[int]int, len(heard))
	ls := make([]listed, len(heard))
	for i, h := range heard {
		ls[i] = h.listed
		rs.heard[h.listed.at] = h.indexer
		byAt[h.listed.at] = i
	}
	kept := dedupe(ls)
	as := make([]assessed, len(kept))
	for i, l := range kept {
		h := &heard[byAt[l.at]]
		as[i] = assessStored(j, l, h.read, h.matched)
	}
	rs.as = in.blocks.gate(as)
	all := cells{open: rs.item}
	rs.js, rs.st = make([]judged, len(rs.as)), make([][]stance, len(rs.as))
	for g := range rs.as {
		rs.js[g] = judge(all, &rs.as[g])
		rs.st[g] = stances(rs.item, rs.js[g])
	}
}

// pin keeps each of pins that names a Release of rs's run and a cell of
// its Item.
func pin(rs *runSheet, pins []Pin) {
	rs.pins = map[int]int{}
	for _, p := range pins {
		if p.Release.Run != rs.run.ID {
			continue
		}
		k := slices.IndexFunc(rs.item, func(c files.Cell) bool { return keyOf(c) == p.Cell })
		if g := groupOf(rs.as, p.Release.Position); k >= 0 && g >= 0 {
			rs.pins[k] = g
		}
	}
}

func keyOf(c files.Cell) files.CellKey {
	return files.CellKey{Item: c.Item.ID, Edition: c.Edition, Target: c.Target.ID}
}

// ManualSearch is work's Manual search with its plan worked out around
// pins. It fails with ErrNoManualSearch and ErrNoWork.
func (gr *Grabs) ManualSearch(ctx context.Context, _ identity.Admin, work works.WorkID, pins []Pin) (ManualView, error) {
	var v ManualView
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		in, err := gr.manualInput(ctx, r, work)
		if err != nil {
			return err
		}
		v, err = manualView(in, pins)
		return err
	})
	return v, err
}

// ManualReleases is every Release of work's Manual search, one per group
// of copies, in rank order. It fails with ErrNoManualSearch and ErrNoWork.
func (gr *Grabs) ManualReleases(ctx context.Context, _ identity.Admin, work works.WorkID) ([]Offer, error) {
	var out []Offer
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		in, err := gr.manualInput(ctx, r, work)
		if err != nil {
			return err
		}
		s, err := worked(in, nil)
		if err != nil {
			return err
		}
		out = s.offers()
		return nil
	})
	return out, err
}
