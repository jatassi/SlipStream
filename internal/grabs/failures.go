package grabs

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// next is what a failed Grab's cells do next.
type next uint8

const (
	// searchAgain searches the cells again at once, in the now lane.
	searchAgain next = iota + 1
	// backlogPace leaves the cells to Backlog search at their due time.
	// Searching them again at once would grab the same Release again,
	// since nothing refuses it, and loop on a link that stays down.
	backlogPace
	// keepDue changes nothing: the cells keep their due time.
	keepDue
	// waitForAdmin changes nothing, and the cells read Failed until the
	// Admin acts.
	waitForAdmin
)

// failureRule is what a failure of one Code does: what its cells do next,
// whether it counts toward the stop, and whether it refuses its Release,
// tried, for the Work.
type failureRule struct {
	next    next
	counts  bool
	refuses bool
}

// afterFailure is the failure table, one row per Code.
var afterFailure = map[Code]failureRule{
	ReleaseFault:       {searchAgain, true, true},
	MarkedFailed:       {searchAgain, true, true},
	NoVideoFile:        {searchAgain, true, true},
	Unreadable:         {searchAgain, true, true},
	DownloadFailed:     {searchAgain, true, true},
	Blocklisted:        {searchAgain, true, true},
	TorrentInvalid:     {searchAgain, true, true},
	InfohashMismatch:   {searchAgain, true, true},
	TorrentV2Only:      {searchAgain, true, true},
	MagnetInvalid:      {searchAgain, true, true},
	NoMetadata:         {searchAgain, true, true},
	NoPeerSource:       {searchAgain, true, true},
	TorrentUnreachable: {backlogPace, false, false},
	IndexerUnavailable: {backlogPace, false, false},
	IndexerRemoved:     {backlogPace, false, false},
	IndexerStopped:     {backlogPace, false, false},
	HomeOffline:        {backlogPace, false, false},
	ClientError:        {backlogPace, false, false},
	Removed:            {keepDue, false, true},
	ClientRemoved:      {keepDue, false, true},
	ClientDeleted:      {keepDue, false, true},
	GoalMet:            {keepDue, false, true},
	NodeWrittenOff:     {keepDue, false, true},
	Canceled:           {keepDue, false, false},
	Declined:           {keepDue, false, false},
	ProbeFailed:        {waitForAdmin, false, false},
	FillsNothing:       {waitForAdmin, false, false},
	NameTaken:          {waitForAdmin, false, false},
	NameTooLong:        {waitForAdmin, false, false},
	TransferFailed:     {waitForAdmin, false, false},
	ImportFailed:       {waitForAdmin, false, false},
	RootChanged:        {waitForAdmin, false, false},
}

// ruleOf is g's failure rule. A Grab the Admin picked in Manual search
// never searches again by itself.
func ruleOf(g Grab, code Code) failureRule {
	r := afterFailure[code]
	if g.Source == FromManual {
		r.next = keepDue
	}
	return r
}

// searchingAgain is whether failed Grab g's cell is searched again rather
// than read Failed: it isn't stopped, and g's rule searches it again.
func searchingAgain(g Grab, stopped bool) bool {
	n := ruleOf(g, g.Failed.Code).next
	return !stopped && (n == searchAgain || n == backlogPace)
}

// failed is what g failing with code does to the cells it held unsettled,
// in the Write that fails it: a failure row each, the stop, and for a
// rule that searches again a nudge of the cells not stopped.
func (gr *Grabs) failed(ctx context.Context, tx *storage.Tx[file], g Grab, code Code) error {
	rule := ruleOf(g, code)
	q := db.New(tx)
	targets, err := q.ListUnsettledTargets(ctx, g.ID)
	if err != nil {
		return err
	}
	cells := make([]files.CellKey, len(targets))
	for i, t := range targets {
		cells[i] = files.CellKey{Item: t.ItemID, Edition: g.Edition, Target: t.TargetID}
	}
	stopped, err := recordFailures(ctx, q, g, code, cells, rule)
	if err != nil {
		return err
	}
	if rule.next != searchAgain {
		return nil
	}
	cells = slices.DeleteFunc(cells, func(k files.CellKey) bool { return stopped[k] })
	if len(cells) == 0 {
		return nil
	}
	return gr.nudge(ctx, tx, g.WorkID, searches.FailedGrab, pace.Now, itemsOfKeys(cells))
}

// recordFailures writes g's failure row for each of cells, once per Grab
// and cell, and stops a cell once its counted rows inside the stop window
// reach the stop count. It answers which cells are stopped. A rule that
// neither counts nor refuses writes nothing, since no reader needs its
// row.
func recordFailures(ctx context.Context, q *db.Queries, g Grab, code Code, cells []files.CellKey, r failureRule) (map[files.CellKey]bool, error) {
	if !r.counts && !r.refuses {
		return nil, nil
	}
	s, err := q.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	now := storage.Now()
	p := db.InsertFailureParams{
		GrabID: g.ID, WorkID: g.WorkID, Edition: g.Edition, Code: code, IndexerID: g.Ticket.Indexer, Guid: orNil(g.GUID),
		FailedAt: now, Counted: r.counts,
	}
	if h := g.infoHash(); h != nil {
		p.InfoHash = h[:]
	}
	p.Refuses = r.refuses && (p.InfoHash != nil || !p.IndexerID.IsZero() && p.Guid != nil)
	stop := stopRule{count: StopCount.Value(s.StopCount), since: storage.FromTime(now.Time().Add(-StopWindow.Value(s.StopWindowSeconds)))}
	stopped := map[files.CellKey]bool{}
	for _, k := range cells {
		p.ItemID, p.TargetID = k.Item, k.Target
		if stopped[k], err = recordFailure(ctx, q, p, stop); err != nil {
			return nil, err
		}
	}
	return stopped, nil
}

// stopRule is when a cell stops: once it has count counted rows since.
type stopRule struct {
	count int64
	since storage.Time
}

// recordFailure writes p's row, stops its cell when the row is new and
// brings its counted rows to the stop, and answers whether the cell is
// stopped.
func recordFailure(ctx context.Context, q *db.Queries, p db.InsertFailureParams, stop stopRule) (bool, error) {
	n, err := q.InsertFailure(ctx, p)
	if err != nil {
		return false, err
	}
	if n == 1 && p.Counted {
		count, err := q.CountFailuresSince(ctx, db.CountFailuresSinceParams{ItemID: p.ItemID, Edition: p.Edition, TargetID: p.TargetID, Since: stop.since})
		if err != nil {
			return false, err
		}
		if count >= stop.count {
			at := db.StopAtParams{StoppedAt: p.FailedAt, GrabID: p.GrabID, ItemID: p.ItemID, Edition: p.Edition, TargetID: p.TargetID}
			if err := q.StopAt(ctx, at); err != nil {
				return false, err
			}
		}
	}
	return q.CellStopped(ctx, db.CellStoppedParams{ItemID: p.ItemID, Edition: p.Edition, TargetID: p.TargetID})
}

func itemsOfKeys(cells []files.CellKey) []works.ItemID {
	var out []works.ItemID
	for _, k := range cells {
		if !slices.Contains(out, k.Item) {
			out = append(out, k.Item)
		}
	}
	return out
}

// uncount clears the failure count and the stop of cells, keeping their
// refusals: the Admin's Search.
func uncount(ctx context.Context, tx *storage.Tx[file], cells []files.Cell) error {
	q := db.New(tx)
	for _, c := range cells {
		if err := q.Uncount(ctx, db.UncountParams{ItemID: c.Item.ID, Edition: c.Edition, TargetID: c.Target.ID}); err != nil {
			return err
		}
	}
	return nil
}

// clearFailures deletes every failure row of cells, which filled.
func clearFailures(ctx context.Context, tx *storage.Tx[file], cells []files.CellKey) error {
	q := db.New(tx)
	for _, k := range cells {
		if err := q.DeleteFailuresOf(ctx, db.DeleteFailuresOfParams{ItemID: k.Item, Edition: k.Edition, TargetID: k.Target}); err != nil {
			return err
		}
	}
	return nil
}

// clearPlaced clears the failures of the cells g's Import placed.
func clearPlaced(ctx context.Context, tx *storage.Tx[file], g Grab) error {
	placed, err := db.New(tx).ListPlacedTargets(ctx, g.ID)
	if err != nil {
		return err
	}
	cells := make([]files.CellKey, len(placed))
	for i, t := range placed {
		cells[i] = files.CellKey{Item: t.ItemID, Edition: g.Edition, Target: t.TargetID}
	}
	return clearFailures(ctx, tx, cells)
}
