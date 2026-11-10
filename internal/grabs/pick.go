package grabs

import (
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Liftable is the soft refusals a Manual pick may grab past. covers_more
// and not_an_upgrade wait for S33 (Import pipeline). cut_not_wanted is
// lifted only by the Edition override.
var Liftable = []targets.ReasonCode{targets.WrongTitle, targets.SizeBelow, targets.SizeAbove, targets.ScoreBelowMinimum, targets.RepackOtherGroup}

// ManualPick is one ticked row, or one Release's Grab, as the sheet sends
// it.
type ManualPick struct {
	Release ReleaseRef
	// Lifts is the soft refusals the row said it gets past, by code and
	// Target.
	Lifts []LiftKey
	// Edition is the Edition override, zero for none.
	Edition targets.Edition
	// Client is the client chosen, nil for where Tracker routing sends it.
	Client  *downloadclients.Choice
	Unblock bool
	// Cancel is the Grab downloading a cell the pick fills, which the same
	// Write cancels first.
	Cancel *GrabID
}

type LiftKey struct {
	Code   targets.ReasonCode
	Target targets.TargetID
}

// PickResult is a pick's Grab, or why it was refused.
type PickResult struct {
	Grab GrabID
	Err  *PickError
}

type PickCode string

const (
	// PickConflict is two picks that fill one cell.
	PickConflict PickCode = "pick_conflict"
	// CellHeld is a pick for a cell a Grab downloads.
	CellHeld PickCode = "cell_held"
	// PickChanged is a pick whose cells or refusals changed since the plan.
	PickChanged PickCode = "pick_changed"
	// NotTaken is a pick its Indexer won't take now.
	NotTaken PickCode = "not_taken"
	// NotGrabbable is a hard refusal, covers_more or not_an_upgrade.
	NotGrabbable PickCode = "not_grabbable"
	// PickBusy is a pick whose Grab to cancel is placing its file.
	PickBusy PickCode = "grab_busy"
	// ClientRefused is a chosen client SlipStream can't use.
	ClientRefused PickCode = "download_client_field_refused"
	// NotDownloading is a built-in client on a Node that doesn't hold
	// Downloading.
	NotDownloading PickCode = "node_not_downloading"
)

// PickCodes is every PickCode.
var PickCodes = []PickCode{PickConflict, CellHeld, PickChanged, NotTaken, NotGrabbable, PickBusy, ClientRefused, NotDownloading}

// PickError is a refused pick in words. Held is the Grab downloading the
// cell, for CellHeld.
type PickError struct {
	Code  PickCode
	Words string
	Held  GrabID
}

// errRegrab rolls back a grab Write in which a pick was refused after a
// change of its own, so the Write runs again with that pick refused.
var errRegrab = errors.New("a pick was refused after it changed the library")

// GrabFromManual grabs picks from Manual search id of work in one Write.
// Each is judged again in that Write as the plan judges it, and a refused
// pick is refused alone and changes nothing. Two picks with one info hash
// make one Grab, and a pick whose cell a Grab from this search holds with
// its Release answers that Grab, so a request sent twice grabs once. It
// fails as a whole with ErrManualSearchReplaced, ErrReconciling, ErrNoWork
// or a library error.
func (gr *Grabs) GrabFromManual(ctx context.Context, by identity.Admin, work works.WorkID, id ManualID, picks []ManualPick) ([]PickResult, error) {
	refused := map[int]*PickError{}
	for {
		g := grabbing{gr: gr, by: actorOf(by), picks: picks, refused: refused}
		err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			g.tx = tx
			return g.run(ctx, work, id)
		})
		if errors.Is(err, errRegrab) {
			continue
		}
		if err != nil {
			return nil, err
		}
		g.publish(ctx)
		return g.results, nil
	}
}

// grabbing is one attempt at a grab Write. refused outlives it: each
// attempt that ends in errRegrab adds a pick to it.
type grabbing struct {
	gr      *Grabs
	tx      *storage.Tx[file]
	by      Actor
	picks   []ManualPick
	refused map[int]*PickError
	s       sheet
	results []PickResult
	// canceled is each Grab the Write canceled, by id.
	canceled  map[GrabID]canceledGrab
	grabbed   []Grab
	unblocked bool
}

// chosen is a pick judged again: the Release as[group] of runs[run], the
// cells it fills, the first leading, and how it stands there: its kind
// and Edition at the first cell, and every lift across them.
type chosen struct {
	run, group int
	cells      []int
	stance     stance
	// picks is the index of each pick naming it, the first leading.
	picks []int
}

func (g *grabbing) run(ctx context.Context, work works.WorkID, id ManualID) error {
	if err := g.open(ctx, work, id); err != nil {
		return err
	}
	g.results, g.canceled = make([]PickResult, len(g.picks)), map[GrabID]canceledGrab{}
	all := g.chosen()
	for _, c := range g.conflicted(all) {
		g.refuse(c, &PickError{Code: PickConflict, Words: g.s.conflictWords(c)})
	}
	for _, c := range all {
		if g.results[c.picks[0]].Err != nil {
			continue
		}
		id, e, err := g.grab(ctx, *c)
		if err != nil {
			return err
		}
		if e != nil {
			g.refuse(*c, e)
			continue
		}
		for _, i := range c.picks {
			g.results[i].Grab = id
		}
	}
	return nil
}

// open works out work's Manual search id in the Write, as the sheet
// shows it. It refuses the whole grab while Reconciling, as ADR 0019
// has a restored Controller start nothing new.
func (g *grabbing) open(ctx context.Context, work works.WorkID, id ManualID) error {
	reconciling, err := g.gr.cfg.Reconciling(ctx, g.tx)
	if err != nil {
		return err
	}
	if reconciling {
		return ErrReconciling
	}
	in, err := g.gr.manualInput(ctx, g.tx, work)
	if errors.Is(err, ErrNoManualSearch) {
		if _, err := g.gr.cfg.Works.TreeIn(ctx, g.tx, work); err != nil {
			return err
		}
		return ErrManualSearchReplaced
	}
	if err != nil {
		return err
	}
	if in.manual.ID != id {
		return ErrManualSearchReplaced
	}
	g.s, err = worked(in, nil)
	return err
}

// chosen is each pick not refused judged again, the picks of one Release
// together.
func (g *grabbing) chosen() []*chosen {
	var all []*chosen
	for i, p := range g.picks {
		if e, ok := g.refused[i]; ok {
			g.results[i].Err = e
			continue
		}
		c, e := g.s.choose(p)
		if e != nil {
			g.results[i].Err = e
			continue
		}
		if k := slices.IndexFunc(all, func(o *chosen) bool { return o.run == c.run && o.group == c.group }); k >= 0 {
			all[k].picks = append(all[k].picks, i)
			continue
		}
		c.picks = []int{i}
		all = append(all, &c)
	}
	return all
}

// refuse answers every pick of c with e, in this attempt and the ones
// that follow.
func (g *grabbing) refuse(c chosen, e *PickError) {
	for _, i := range c.picks {
		g.refused[i], g.results[i].Err = e, e
	}
}

// choose is pick judged again in the sheet: the cells it fills now, led
// by the first cell of its Edition its Release stands at, auto or lifted
// or on the Blocklist, that the plan walks or an unfinished Grab holds.
func (s sheet) choose(p ManualPick) (chosen, *PickError) {
	r := slices.IndexFunc(s.runs, func(rs runSheet) bool { return rs.run.ID == p.Release.Run })
	if r < 0 {
		return chosen{}, changedPick
	}
	rs := s.runs[r]
	grp := groupOf(rs.as, p.Release.Position)
	if grp < 0 {
		return chosen{}, changedPick
	}
	lead, e := s.lead(rs, grp, p.Edition)
	if e != nil {
		return chosen{}, e
	}
	at := rs.st[grp][lead]
	c := chosen{run: r, group: grp, stance: stance{kind: at.kind, edition: at.edition}}
	for k, cell := range rs.item {
		st := rs.st[grp][k]
		if !rs.open(k) || cell.Edition != rs.item[lead].Edition || st.kind != standsAuto && (st.kind != at.kind || at.kind == standsAuto) {
			continue
		}
		c.cells = append(c.cells, k)
		for _, l := range st.lifts {
			if !slices.Contains(c.stance.lifts, l) {
				c.stance.lifts = append(c.stance.lifts, l)
			}
		}
	}
	return c, nil
}

// lead is the first cell rs.as[grp] stands at as edition, of those open:
// not_grabbable when it stands refused at each, pick_changed when it
// stands at none.
func (s sheet) lead(rs runSheet, grp int, edition targets.Edition) (int, *PickError) {
	opened, stands := false, false
	for k := range rs.item {
		if !rs.open(k) {
			continue
		}
		opened = true
		st := rs.st[grp][k]
		if st.kind == standsRefused {
			continue
		}
		if st.edition == edition {
			return k, nil
		}
		stands = true
	}
	if opened && !stands {
		return -1, &PickError{Code: NotGrabbable, Words: s.refusal(rs, grp)}
	}
	return -1, changedPick
}

// open reports whether a pick may fill cell k: the plan walks it or an
// unfinished Grab holds it.
func (rs runSheet) open(k int) bool {
	_, held := rs.held[k]
	return rs.fill[k] || held
}

var changedPick = &PickError{Code: PickChanged, Words: "What this Release gets you changed since the plan. Check the plan and grab again."}

// conflicted is each of all that fills a cell another of all fills.
func (g *grabbing) conflicted(all []*chosen) []chosen {
	by := map[files.CellKey][]int{}
	for i, c := range all {
		for _, k := range c.cells {
			key := keyOf(g.s.runs[c.run].item[k])
			by[key] = append(by[key], i)
		}
	}
	var out []chosen
	for _, c := range all {
		if slices.ContainsFunc(c.cells, func(k int) bool { return len(by[keyOf(g.s.runs[c.run].item[k])]) > 1 }) {
			out = append(out, *c)
		}
	}
	return out
}

// conflictWords is "Two rows fill 1080p for E05. Untick one of them."
// for the first cell of c.
func (s sheet) conflictWords(c chosen) string {
	cell := s.runs[c.run].item[c.cells[0]]
	return "Two rows fill " + cellName(cell) + " for " + s.itemsWords([]works.ItemID{cell.Item.ID}, false) + ". Untick one of them."
}

// grab grabs c in the Write: the Grab it made, the Grab from this search
// that already holds its cells with its Release, or why it is refused.
// Every refusal before its first change returns a PickError. One after
// it fails the Write with errRegrab, refused for the attempt that
// follows.
func (g *grabbing) grab(ctx context.Context, c chosen) (GrabID, *PickError, error) {
	p := g.picks[c.picks[0]]
	for _, l := range c.stance.lifts {
		if !slices.Contains(p.Lifts, LiftKey{Code: l.Code, Target: l.Target}) {
			return GrabID{}, changedPick, nil
		}
	}
	if c.stance.kind == standsBlocked && !p.Unblock {
		return GrabID{}, &PickError{Code: PickChanged, Words: "This Release is on the Blocklist now. Remove it from the Blocklist to grab it."}, nil
	}
	cancel, same, e, err := g.held(ctx, c, p)
	if !same.IsZero() || e != nil || err != nil {
		return same, e, err
	}
	cl, e, err := g.clear(ctx, c, p)
	if e != nil || err != nil {
		return GrabID{}, e, err
	}
	cl.cancel = cancel
	id, err := g.make(ctx, c, cl)
	return id, nil, err
}

// held is the unfinished Grabs on c's cells that p cancels, or the Grab
// from this search that holds them with c's Release, or the refusal of
// a held cell p doesn't cancel.
func (g *grabbing) held(ctx context.Context, c chosen, p ManualPick) ([]Grab, GrabID, *PickError, error) {
	rs := g.s.runs[c.run]
	var out []Grab
	for _, k := range c.cells {
		h, ok := rs.held[k]
		if !ok || slices.ContainsFunc(out, func(o Grab) bool { return o.ID == h.ID }) {
			continue
		}
		held, err := grabIn(ctx, g.tx, h.ID)
		if err != nil {
			return nil, GrabID{}, nil, err
		}
		if held.Manual == g.s.in.manual.ID && sameRelease(held.Release, rs.js[c.group]) {
			return nil, held.ID, nil, nil
		}
		if p.Cancel == nil || *p.Cancel != held.ID {
			words := "A Grab is already downloading " + cellName(rs.item[k]) + " for " + g.s.itemsWords([]works.ItemID{held.ItemID}, false) + ". Cancel it to grab this one."
			return nil, GrabID{}, &PickError{Code: CellHeld, Words: words, Held: held.ID}, nil
		}
		out = append(out, held)
	}
	return out, GrabID{}, nil, nil
}

// cleared is a pick that passed every check that changes nothing: what it
// grabs, the client it goes to and the Grabs it cancels.
type cleared struct {
	pick   Pick
	sent   *downloadclients.Target
	cancel []Grab
}

// clear checks p's client and whether its Indexers would take c now, as
// the plan reads their standings.
func (g *grabbing) clear(ctx context.Context, c chosen, p ManualPick) (cleared, *PickError, error) {
	rs := g.s.runs[c.run]
	j := rs.js[c.group]
	out := cleared{pick: Pick{Release: j.Release, Edition: rs.item[c.cells[0]].Edition, at: j.at, copies: j.copies}}
	for _, k := range c.cells {
		out.pick.Targets = append(out.pick.Targets, rs.item[k].Target.ID)
	}
	if p.Client != nil {
		t, err := downloadclients.Usable(ctx, g.tx, *p.Client, g.gr.downloading)
		if e := clientRefusal(err); e != nil {
			return cleared{}, e, nil
		}
		if err != nil {
			return cleared{}, nil, err
		}
		out.sent = &t
	}
	_, _, ok, err := choose([]Pick{out.pick}, nil, g.s.in.standings.Take)
	if err == nil && !ok {
		return cleared{}, notTaken(g.s.in.standings.Take(j.Release)), nil
	}
	return out, nil, err
}

// make removes c's Release from the Blocklist when it is on it, cancels
// the Grabs cl cancels, takes it on its Indexer and inserts its Grab.
func (g *grabbing) make(ctx context.Context, c chosen, cl cleared) (GrabID, error) {
	rs := g.s.runs[c.run]
	p := g.picks[c.picks[0]]
	n := newGrab{work: g.s.in.tree.ID, item: rs.run.ItemID, source: FromManual, manual: g.s.in.manual.ID, by: &g.by, sent: p.Client}
	for _, w := range rs.worded(c.stance.lifts, c.group) {
		n.overrides.Lifted = append(n.overrides.Lifted, Lifted{Code: w.Code, Words: w.Words})
	}
	if c.stance.kind == standsBlocked {
		a := rs.as[c.group]
		e, _ := g.s.in.blocks.of(a)
		n.overrides.Unblocked = &Unblocked{Cause: e.Cause, Since: e.Added}
		var keys [][]byte
		for _, h := range a.hashes() {
			keys = append(keys, h[:])
		}
		if err := db.New(g.tx).DeleteBlocked(ctx, keys); err != nil {
			return GrabID{}, err
		}
		g.unblocked = true
	}
	for _, held := range cl.cancel {
		if _, done := g.canceled[held.ID]; done {
			continue
		}
		canceled, err := g.gr.cancelIn(ctx, g.tx, held, g.by, false)
		if busy, ok := errors.AsType[*GrabBusyError](err); ok {
			return GrabID{}, g.regrab(c, &PickError{Code: PickBusy, Words: busyWords(busy)})
		}
		if err != nil {
			return GrabID{}, err
		}
		g.canceled[held.ID] = canceled
	}
	var taken bool
	var err error
	n.pick, n.ticket, taken, err = choose([]Pick{cl.pick}, nil, func(r indexers.Release) (indexers.Ticket, error) {
		return g.gr.cfg.Indexers.Take(ctx, g.tx, r)
	})
	if err != nil {
		return GrabID{}, err
	}
	if !taken {
		return GrabID{}, g.regrab(c, notTaken(g.s.in.standings.Take(rs.js[c.group].Release)))
	}
	if n.overrides, err = g.overridden(ctx, n, c, cl.sent); err != nil {
		return GrabID{}, err
	}
	made, err := g.gr.insertGrab(ctx, g.tx, n, storage.Now())
	if err != nil {
		return GrabID{}, err
	}
	g.grabbed = append(g.grabbed, made)
	return made.ID, nil
}

// overridden is n's overrides with its Edition's name and its client's
// words.
func (g *grabbing) overridden(ctx context.Context, n newGrab, c chosen, sent *downloadclients.Target) (Overrides, error) {
	o := n.overrides
	if e := c.stance.edition; e != "" {
		known, err := g.gr.cfg.Targets.CutsIn(ctx, g.tx)
		if err != nil {
			return Overrides{}, err
		}
		o.Edition = string(e)
		if cut, ok := known.Get(release.CutKey(e)); ok {
			o.Edition = cut.Name
		}
	}
	if sent == nil {
		return o, nil
	}
	words, err := clientWords(ctx, g.tx, *sent)
	if err != nil {
		return Overrides{}, err
	}
	o.Client = &ClientOverride{Sent: words}
	routed, err := routeOf(ctx, g.tx, g.gr.cfg.Router, Grab{WorkID: n.work, Ticket: n.ticket})
	if _, unrouted := errors.AsType[*nodes.Unrouted](err); unrouted {
		return o, nil
	}
	if err != nil {
		return Overrides{}, err
	}
	o.Client.Routed, err = clientWords(ctx, g.tx, routed)
	return o, err
}

// regrab refuses c for the attempt that follows and fails this one.
func (g *grabbing) regrab(c chosen, e *PickError) error {
	for _, i := range c.picks {
		g.refused[i] = e
	}
	return errRegrab
}

// publish tells what the Write changed once it commits.
func (g *grabbing) publish(ctx context.Context) {
	for _, c := range g.canceled {
		c.publish(ctx, g.gr)
	}
	for _, made := range g.grabbed {
		g.gr.cfg.Indexers.Taken(ctx, made.Ticket)
		g.gr.cfg.Changed.Publish(ctx, Changed{GrabID: made.ID.String(), WorkID: made.WorkID.String()})
	}
	if g.unblocked {
		g.gr.cfg.BlocklistChanged.Publish(ctx, BlocklistChanged{})
	}
	g.gr.wake()
}

// sameRelease reports whether r is j's Release or a copy of it: the same
// info hash, or the same title and size when either lacks one.
func sameRelease(r Release, j judged) bool {
	same := func(o indexers.Release) bool {
		if r.InfoHash != nil && o.InfoHash != nil {
			return *r.InfoHash == *o.InfoHash
		}
		return r.Title == o.Title && r.Size == o.Size
	}
	return same(j.Release) || slices.ContainsFunc(j.copies, func(c copied) bool { return same(c.Release) })
}

// clientRefusal is err, a chosen client Usable refused, as a pick's
// refusal, nil for any other error.
func clientRefusal(err error) *PickError {
	if off, ok := errors.AsType[*downloadclients.ClientOffError](err); ok {
		return &PickError{Code: ClientRefused, Words: off.Name + " is off. Turn it on or choose another client."}
	}
	switch {
	case errors.Is(err, downloadclients.ErrNoClient):
		return &PickError{Code: ClientRefused, Words: "That Download Client was deleted. Choose another client."}
	case errors.Is(err, downloadclients.ErrNotDownloading):
		return &PickError{Code: NotDownloading, Words: "That Node doesn't hold Downloading. Choose another client."}
	}
	return nil
}

// notTaken is the refusal of a pick whose Indexers won't take it now,
// from err, the primary's refusal: "TorrentLeech is at its grab limit
// until 14:05."
func notTaken(_ indexers.Ticket, err error) *PickError {
	words := "No Indexer that listed this Release takes it now."
	if e, ok := errors.AsType[*indexers.NotTakenError](err); ok {
		words = e.Sentence
	}
	return &PickError{Code: NotTaken, Words: words}
}

func busyWords(e *GrabBusyError) string {
	if e.Words != "" {
		return e.Words
	}
	return "The Grab it replaces is placing its file in the library. It can't be canceled until that finishes."
}
