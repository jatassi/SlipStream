package downloadclients

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
	"github.com/jatassi/slipstream/internal/works"
)

// Uses is the Grabs that used Torrents, as grabs answers it. Nothing here
// writes a Grab.
type Uses interface {
	// Of is the Grabs of each of ids, newest first, in r's snapshot.
	Of(ctx context.Context, r storage.Reader[file], ids []ID) (map[ID][]Use, error)
}

// Blocklist is the Blocklist as grabs keeps it.
type Blocklist interface {
	// MarkedFailed adds an entry for each of ids whose Torrent a Grab at
	// downloading holds, by by, in the caller's Write. It refuses with
	// ErrReconciling while a restored Controller is Reconciling. Adding
	// twice keeps the first entry.
	MarkedFailed(ctx context.Context, tx *storage.Tx[file], ids []ID, by identity.Admin) error
	// Changed tells the Admins the Blocklist changed. It is called once the
	// Write MarkedFailed ran in has committed.
	Changed(ctx context.Context)
}

// ErrReconciling is a restored Controller Reconciling, which blocklists
// nothing.
var ErrReconciling = errors.New("the Controller is Reconciling after a restore")

// Use is one Grab of a Torrent.
type Use struct {
	Grab    storage.UUID
	Work    works.WorkID
	Edition string
	Release string
	Stage   UseStage
	// Files are the Torrent's files the Grab's Import took, none before
	// it took any.
	Files   []UseFile
	Created time.Time
	// Floor is the seeding rule its Release stated, nil for none.
	Floor seeding.Rule
}

// UseFile is the Torrent's file at Index, and the Items it is for.
type UseFile struct {
	Index int
	Items []works.ItemID
}

type UseStage string

const (
	UseDownloading UseStage = "downloading"
	// UseImporting is a Grab probing, transferring or importing the
	// Torrent's files.
	UseImporting UseStage = "importing"
	// UseWaiting is a Grab whose Import has a file waiting for the Admin
	// and none sending or placing.
	UseWaiting  UseStage = "waiting"
	UseImported UseStage = "imported"
	UseFailed   UseStage = "failed"
)

type ActionKind string

const (
	Pause    ActionKind = "pause"
	Resume   ActionKind = "resume"
	Announce ActionKind = "announce"
	Recheck  ActionKind = "recheck"
	// Retry checks the files of a failed Torrent that had completed: it
	// seeds again once all its data is there, and stays paused otherwise.
	Retry ActionKind = "retry"
	// MarkFailed removes a download a Grab at downloading holds, with its
	// download copy, puts its Release on the Blocklist, and fails its
	// downloading Grabs as download_failed.
	MarkFailed ActionKind = "mark-failed"
	Remove     ActionKind = "remove"
)

// ActionKinds is every ActionKind.
var ActionKinds = []ActionKind{Pause, Resume, Announce, Recheck, Retry, MarkFailed, Remove}

type Action struct {
	Kind ActionKind
	IDs  []ID
	// DeleteFiles deletes the download copy of a removed Torrent.
	DeleteFiles bool
	// DryRun answers what remove, mark-failed or recheck would do, and
	// changes nothing.
	DryRun bool
	// By is the Admin acting, whom a mark-failed's Blocklist entry names.
	By identity.Admin
}

// ErrNoPreview is a dry run of an action that has nothing to preview.
var ErrNoPreview = errors.New("only remove, mark-failed and recheck can be previewed")

// Result is the action's outcome for one Torrent: done, or refused.
type Result struct {
	ID      ID
	Refusal *Refusal
}

type RefusalCode string

const (
	RefusedNotFound RefusalCode = "not_found"
	// RefusedRemoved is a Torrent already removed.
	RefusedRemoved RefusalCode = "removed"
	// RefusedImporting is a Torrent a Grab is importing from.
	RefusedImporting RefusalCode = "importing"
	// RefusedNotHeld is mark-failed on a Torrent no Grab at downloading
	// holds.
	RefusedNotHeld RefusalCode = "not_held"
	// RefusedNotThisRelease is mark-failed on a Torrent whose Grab's Import
	// waits for the Admin, who answers it with Not this Release on its
	// Pending import.
	RefusedNotThisRelease RefusalCode = "use_not_this_release"
	// RefusedReconciling is mark-failed while a restored Controller is
	// Reconciling, which removes no torrent.
	RefusedReconciling RefusalCode = "reconciling"
	// RefusedOffline is announce or recheck on a Torrent whose Node is out
	// of step.
	RefusedOffline RefusalCode = "offline"
	// RefusedClientAway is announce or recheck on a Torrent whose external
	// client isn't responding to its Node.
	RefusedClientAway RefusalCode = "client_away"
	// RefusedClientOff is announce or recheck on a Torrent whose external
	// client is off, so its Node doesn't check it.
	RefusedClientOff RefusalCode = "client_off"
	// RefusedWindow is an announce some tracker would reject until
	// AcceptsAt.
	RefusedWindow RefusalCode = "window"
	// RefusedNotRunning is an announce of a Torrent that isn't running.
	RefusedNotRunning RefusalCode = "not_running"
	// RefusedFailed is announce or recheck of a Torrent the engine failed.
	RefusedFailed RefusalCode = "failed"
	// RefusedNotRetryable is Retry of a Torrent that isn't failed, that
	// failed before it completed, or that an external client holds.
	RefusedNotRetryable RefusalCode = "not_retryable"
)

// RefusalCodes is every RefusalCode.
var RefusalCodes = []RefusalCode{
	RefusedNotFound, RefusedRemoved, RefusedImporting, RefusedNotHeld, RefusedNotThisRelease, RefusedReconciling, RefusedOffline,
	RefusedClientAway, RefusedClientOff, RefusedWindow, RefusedNotRunning, RefusedFailed, RefusedNotRetryable,
}

type Refusal struct {
	Code RefusalCode
	// Work is the Work being imported, for RefusedImporting and
	// RefusedNotThisRelease.
	Work works.WorkID
	// AcceptsAt is when an announce would be accepted, for RefusedWindow.
	AcceptsAt time.Time
	// Client and Node name the external client and its Node, for
	// RefusedClientAway and RefusedClientOff.
	Client, Node string
}

// Preview is what a remove or a recheck would do.
type Preview struct {
	// Cancels are the Grabs a remove ends: each at downloading, and each
	// whose Import waits for the Admin.
	Cancels []Use
	// StopSeeding are the Works whose complete Torrents stop seeding.
	StopSeeding []works.WorkID
	// Frees is the bytes deleting the download copies would free, nil
	// while a Node can't say.
	Frees *int64
	// Linked is some file also being a library file: a recheck reads it.
	Linked bool
	// HitAndRun are the Torrents whose removal counts as a hit and run, in
	// the action's order: each is the last live copy of an info hash that
	// still owes its Tracker.
	HitAndRun []HitAndRun
}

type HitAndRun struct {
	ID ID
	// Tracker is nil while the Tracker it owes isn't known.
	Tracker *TrackerRef
	// Owes is nil while its Tracker's rule isn't known.
	Owes *seeding.Owes
}

// Act runs a on each of its Torrents and answers for each, in a's order,
// with a Preview for a dry run. Pause, resume, remove and mark-failed
// write the wanted state, which reaches each Node in its intent, even
// while it is offline. Announce and recheck ask each Torrent's Node.
func (c *Clients) Act(ctx context.Context, a Action) ([]Result, *Preview, error) {
	switch {
	case a.DryRun && a.Kind == Recheck:
		return c.previewRecheck(ctx, a)
	case a.DryRun && a.Kind != Remove && a.Kind != MarkFailed:
		return nil, nil, ErrNoPreview
	}
	switch a.Kind {
	case Pause, Resume:
		return c.want(ctx, a)
	case Remove, MarkFailed:
		return c.remove(ctx, a)
	case Announce, Recheck:
		return c.ask(ctx, a)
	case Retry:
		return c.retry(ctx, a)
	}
	return nil, nil, fmt.Errorf("no action %q", a.Kind)
}

// acted is the Torrents an action names, as it reads them.
type acted map[ID]db.ListActedRow

func readActed(ctx context.Context, r storage.Reader[file], ids []ID) (acted, error) {
	rows, err := db.New(r).ListActed(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := acted{}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

// refusal is why no action may touch id: not found, or removed.
func (a acted) refusal(id ID) *Refusal {
	row, ok := a[id]
	switch {
	case !ok:
		return &Refusal{Code: RefusedNotFound}
	case !row.RemovedAt.IsZero():
		return &Refusal{Code: RefusedRemoved}
	}
	return nil
}

func (c *Clients) want(ctx context.Context, a Action) ([]Result, *Preview, error) {
	wanted := Started
	if a.Kind == Pause {
		wanted = Paused
	}
	var results []Result
	var touched Touched
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		results, touched, err = c.WantIn(ctx, tx, a.IDs, wanted)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	c.Touched(touched)
	return results, nil, nil
}

// Touched is what a write inside a caller's Write changed, for Touched
// once that Write commits. Only RemoveIn and WantIn make one.
type Touched struct {
	nodes []role.NodeID
	ids   []ID
}

// Touched tells each changed Torrent's Node and the Admins, once the
// Write that made ts committed.
func (c *Clients) Touched(ts ...Touched) {
	var all Touched
	for _, t := range ts {
		all.nodes, all.ids = append(all.nodes, t.nodes...), append(all.ids, t.ids...)
	}
	c.committed(all.nodes, all.ids)
}

// WantIn sets each of ids to wanted in tx, and refuses one not found or
// removed.
func (c *Clients) WantIn(ctx context.Context, tx *storage.Tx[file], ids []ID, wanted Wanted) ([]Result, Touched, error) {
	rows, err := readActed(ctx, tx, ids)
	if err != nil {
		return nil, Touched{}, err
	}
	var (
		results []Result
		touched Touched
	)
	for _, id := range ids {
		r := Result{ID: id, Refusal: rows.refusal(id)}
		results = append(results, r)
		if r.Refusal != nil || Wanted(rows[id].Wanted) == wanted {
			continue
		}
		if err := db.New(tx).SetWanted(ctx, db.SetWantedParams{Wanted: string(wanted), ID: id}); err != nil {
			return nil, Touched{}, err
		}
		touched.nodes, touched.ids = append(touched.nodes, rows[id].NodeID), append(touched.ids, id)
	}
	return results, touched, nil
}

func (c *Clients) previewRemoval(ctx context.Context, torrents []ID, reason RemoveReason) ([]Result, *Preview, error) {
	var plan removal
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if plan, err = c.planRemoval(ctx, r, torrents, reason); err != nil {
			return err
		}
		plan.preview.HitAndRun, err = c.hitsAndRuns(ctx, r, plan)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	nodes, ids := plan.deleting()
	plan.preview.Frees, plan.preview.Linked = c.frees(ctx, nodes, ids)
	return plan.results, plan.preview, nil
}

// removedBy names the Admin who removed the Torrents, for the Grab they fed;
// an API key has no name to give.
func removedBy(ctx context.Context, tx *storage.Tx[file], by identity.Admin, torrents []ID) error {
	account, ok := by.Account()
	if !ok {
		return nil
	}
	return db.New(tx).SetRemovedBy(ctx, db.SetRemovedByParams{Name: account.Profile().DisplayName, Ids: torrents})
}

// remove removes each Torrent no Grab is importing from, as RemoveIn
// does, and for mark-failed blocklists their Releases in the same Write.
// While Reconciling a mark-failed refuses each Torrent it would remove and
// removes nothing. A dry run reads the same and writes nothing.
func (c *Clients) remove(ctx context.Context, a Action) ([]Result, *Preview, error) {
	reason, deleteFiles := ByAdmin, a.DeleteFiles
	if a.Kind == MarkFailed {
		reason, deleteFiles = ByMarkFailed, true
	}
	if a.DryRun {
		return c.previewRemoval(ctx, a.IDs, reason)
	}
	var results []Result
	var touched Touched
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		results, touched, err = c.RemoveIn(ctx, tx, a.IDs, reason, deleteFiles)
		if err != nil || len(touched.ids) == 0 {
			return err
		}
		if err := removedBy(ctx, tx, a.By, touched.ids); err != nil {
			return err
		}
		if a.Kind != MarkFailed {
			return nil
		}
		return c.cfg.Blocklist.MarkedFailed(ctx, tx, touched.ids, a.By)
	})
	if errors.Is(err, ErrReconciling) {
		for i := range results {
			if results[i].Refusal == nil {
				results[i].Refusal = &Refusal{Code: RefusedReconciling}
			}
		}
		return results, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	c.Touched(touched)
	if a.Kind == MarkFailed && len(touched.ids) > 0 {
		c.cfg.Blocklist.Changed(ctx)
	}
	return results, nil, nil
}

// RemoveIn removes each of ids for reason in tx, with its download copy
// when deleteFiles says so. It reads the Grabs in tx, so it refuses a
// Torrent a Grab is importing from, and an import that starts after it
// finds the Torrent removed. For ByMarkFailed it also refuses a Torrent no
// Grab at downloading holds.
func (c *Clients) RemoveIn(ctx context.Context, tx *storage.Tx[file], ids []ID, reason RemoveReason, deleteFiles bool) ([]Result, Touched, error) {
	plan, err := c.planRemoval(ctx, tx, ids, reason)
	if err != nil {
		return nil, Touched{}, err
	}
	now := storage.Now()
	for _, id := range plan.ids {
		if err := removeIn(ctx, db.New(tx), plan.rows[id], reason, deleteFiles, now); err != nil {
			return nil, Touched{}, err
		}
	}
	return plan.results, Touched{nodes: plan.nodes, ids: plan.ids}, nil
}

// removalWay is how a Torrent's removal is carried out.
type removalWay uint8

const (
	// removedByNode puts it in its Node's discards: the Node removes it
	// from its client, with its files when asked, and parts it once done.
	// For an external client that isn't responding, that waits until the
	// client responds again.
	removedByNode removalWay = iota + 1
	// partedAtOnce parts it in the same Write and leaves its client and
	// its files as they are.
	partedAtOnce
)

// removalOf is how removing a Torrent of client, which SlipStream
// attached to when attached says so, goes while the client is on or off.
// The built-in client and a torrent SlipStream added to an on client go
// through the Node, responding or not. An attached torrent only stops
// being tracked, and an off client isn't checked, so both part at once.
func removalOf(client *ClientID, attached *bool, on bool) removalWay {
	switch {
	case client == nil:
		return removedByNode
	case attached != nil && *attached, !on:
		return partedAtOnce
	}
	return removedByNode
}

// removeIn removes row's Torrent for reason in q's Write, as removalOf
// says.
func removeIn(ctx context.Context, q *db.Queries, row db.ListActedRow, reason RemoveReason, deleteFiles bool, now storage.Time) error {
	if removalOf(row.ClientID, row.Attached, row.ClientOn) == partedAtOnce {
		return q.RemoveParted(ctx, db.RemovePartedParams{At: now, RemovedReason: new(string(reason)), ID: row.ID})
	}
	return q.Remove(ctx, db.RemoveParams{RemovedAt: now, RemovedReason: new(string(reason)), DeleteFiles: boolInt(deleteFiles), ID: row.ID})
}

// removal is what a remove does: its answer for each Torrent, the
// Torrents it removes with their Nodes, and what that cancels.
type removal struct {
	results []Result
	ids     []ID
	nodes   []role.NodeID
	hashes  []bencode.InfoHash
	rows    acted
	preview *Preview
}

// deleting is the Torrents of p whose Node removes them, with their
// Nodes: the ones whose files a remove may delete.
func (p removal) deleting() (nodes []role.NodeID, ids []ID) {
	for i, id := range p.ids {
		row := p.rows[id]
		if removalOf(row.ClientID, row.Attached, row.ClientOn) == removedByNode {
			nodes, ids = append(nodes, p.nodes[i]), append(ids, id)
		}
	}
	return nodes, ids
}

func (c *Clients) planRemoval(ctx context.Context, r storage.Reader[file], ids []ID, reason RemoveReason) (removal, error) {
	plan := removal{preview: &Preview{}}
	rows, err := readActed(ctx, r, ids)
	if err != nil {
		return removal{}, err
	}
	plan.rows = rows
	uses, err := c.cfg.Uses.Of(ctx, r, ids)
	if err != nil {
		return removal{}, err
	}
	for _, id := range ids {
		res := Result{ID: id, Refusal: rows.refusal(id)}
		if res.Refusal == nil {
			res.Refusal = removable(uses[id], reason)
		}
		plan.results = append(plan.results, res)
		if res.Refusal == nil {
			plan.ids, plan.nodes, plan.hashes = append(plan.ids, id), append(plan.nodes, rows[id].NodeID), append(plan.hashes, bencode.InfoHash(rows[id].InfoHash))
			plan.preview.add(rows[id], uses[id])
		}
	}
	return plan, nil
}

// removable is why a Torrent its Grabs uses may not be removed for
// reason: a Grab importing from it, or for ByMarkFailed, a Grab whose
// Import waits on it or no Grab at downloading holding it. An import reads
// the files a removal deletes, so it refuses mark-failed too, though its
// Grab is unfinished. A waiting Import reads none, so a remove ends it.
func removable(uses []Use, reason RemoveReason) *Refusal {
	if i := slices.IndexFunc(uses, func(u Use) bool { return u.Stage == UseImporting }); i >= 0 {
		return &Refusal{Code: RefusedImporting, Work: uses[i].Work}
	}
	if reason != ByMarkFailed {
		return nil
	}
	if i := slices.IndexFunc(uses, func(u Use) bool { return u.Stage == UseWaiting }); i >= 0 {
		return &Refusal{Code: RefusedNotThisRelease, Work: uses[i].Work}
	}
	if !slices.ContainsFunc(uses, func(u Use) bool { return u.Stage == UseDownloading }) {
		return &Refusal{Code: RefusedNotHeld}
	}
	return nil
}

// hitsAndRuns is each Torrent plan removes that leaves its info hash with
// no live built-in copy while the hash owes its Tracker, once per hash. An
// external client's copy is no part of the hash's seeding.
func (c *Clients) hitsAndRuns(ctx context.Context, r storage.Reader[file], plan removal) ([]HitAndRun, error) {
	rows, err := c.Rows(ctx, r, Filter{Hashes: plan.hashes})
	if err != nil {
		return nil, err
	}
	byID := make(map[ID]Row, len(rows))
	stays := map[bencode.InfoHash]bool{}
	for _, row := range rows {
		byID[row.ID] = row
		if row.Client == nil && row.State != StateRemoving && !slices.Contains(plan.ids, row.ID) {
			stays[row.InfoHash] = true
		}
	}
	var out []HitAndRun
	for _, id := range plan.ids {
		row := byID[id]
		w := row.Seeding.Window
		if stays[row.InfoHash] || w == nil || w.State != seeding.Owing {
			continue
		}
		stays[row.InfoHash] = true
		out = append(out, HitAndRun{ID: id, Tracker: row.Tracker, Owes: row.Seeding.Owes})
	}
	return out, nil
}

// add counts a removed Torrent's Grabs into p: those downloading it are
// canceled, and once complete, every Work it seeds for stops seeding.
func (p *Preview) add(row db.ListActedRow, uses []Use) {
	for _, u := range uses {
		if u.Stage == UseDownloading || u.Stage == UseWaiting {
			p.Cancels = append(p.Cancels, u)
		}
		if row.Completion != nil && !slices.Contains(p.StopSeeding, u.Work) {
			p.StopSeeding = append(p.StopSeeding, u.Work)
		}
	}
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// frees is what deleting the download copies of ids, on nodes, frees, and
// whether any file is also a library file. Each Node answers within a live
// sample, or the bytes are unknown.
func (c *Clients) frees(ctx context.Context, nodes []role.NodeID, ids []ID) (*int64, bool) {
	var total int64
	known, linked := true, false
	for node, of := range byNode(nodes, ids) {
		freed, err := c.freesOn(ctx, node, of)
		if err != nil {
			c.cfg.Log.Debug("count what deleting download copies frees", "node", node.String(), "err", err.Error())
			known = false
			continue
		}
		for _, f := range freed {
			total += f.GetBytes()
			linked = linked || f.GetLinked()
		}
	}
	if !known {
		return nil, linked
	}
	return &total, linked
}

func (c *Clients) freesOn(ctx context.Context, node role.NodeID, ids []ID) ([]*dlv1.Freed, error) {
	if !c.cfg.Roster.InStep(node) {
		return nil, errors.New("the Node is out of step")
	}
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Live.Sample)
	defer cancel()
	res, err := c.cfg.Remote(node).Frees(ctx, &dlv1.FreesRequest{Ids: wireIDs(ids)})
	return res.GetFreed(), err
}

func (c *Clients) previewRecheck(ctx context.Context, a Action) ([]Result, *Preview, error) {
	results, live, nodes, err := c.live(ctx, a.IDs)
	if err != nil {
		return nil, nil, err
	}
	_, linked := c.frees(ctx, nodes, live)
	return results, &Preview{Linked: linked}, nil
}

// live answers for each of ids the refusal of one not found or removed,
// and lists the others with their Nodes.
func (c *Clients) live(ctx context.Context, ids []ID) (results []Result, live []ID, nodes []role.NodeID, err error) {
	err = c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := readActed(ctx, r, ids)
		if err != nil {
			return err
		}
		for _, id := range ids {
			r := Result{ID: id, Refusal: rows.refusal(id)}
			results = append(results, r)
			if r.Refusal == nil {
				live, nodes = append(live, id), append(nodes, rows[id].NodeID)
			}
		}
		return nil
	})
	return results, live, nodes, err
}

// ask has each Torrent's Node announce or recheck it. A Torrent whose
// external client is off, or away, is refused without a call,
// and so is one whose Node is out of step or can't be reached.
func (c *Clients) ask(ctx context.Context, a Action) ([]Result, *Preview, error) {
	var rows acted
	var names map[role.NodeID]string
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if rows, err = readActed(ctx, r, a.IDs); err != nil {
			return err
		}
		names, err = namesIn(ctx, r)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	results := make([]Result, len(a.IDs))
	var live []ID
	var nodes []role.NodeID
	for i, id := range a.IDs {
		results[i] = Result{ID: id, Refusal: rows.refusal(id)}
		if results[i].Refusal == nil {
			results[i].Refusal = rows.askRefusal(id, c.reachOf(rows[id]), names)
		}
		if results[i].Refusal == nil {
			live, nodes = append(live, id), append(nodes, rows[id].NodeID)
		}
	}
	act := dlv1.Act_ACT_ANNOUNCE
	if a.Kind == Recheck {
		act = dlv1.Act_ACT_RECHECK
	}
	answers := map[ID]*Refusal{}
	for node, of := range byNode(nodes, live) {
		maps.Copy(answers, c.askOn(ctx, node, act, of))
	}
	for i, r := range results {
		if r.Refusal != nil {
			continue
		}
		results[i].Refusal = answers[r.ID]
		if f := results[i].Refusal; f != nil && f.Code == RefusedClientAway {
			f.Client, f.Node = rows[r.ID].ClientName, names[rows[r.ID].NodeID]
		}
	}
	return results, nil, nil
}

// reachOf is row's external client's reach now, "" for the built-in
// client.
func (c *Clients) reachOf(row db.ListActedRow) ClientReach {
	if row.ClientID == nil {
		return ""
	}
	var state *clients.PollState
	if row.PollState != nil {
		state = new(clients.PollState(*row.PollState))
	}
	return reachOf(row.ClientOn, c.cfg.Roster.InStep(row.NodeID), state)
}

// askRefusal is why id's Node can't be asked to announce or recheck it
// while its client is at reach: the client is off, or away.
func (a acted) askRefusal(id ID, reach ClientReach, names map[role.NodeID]string) *Refusal {
	row := a[id]
	switch {
	case reach == ReachOff:
		return &Refusal{Code: RefusedClientOff, Client: row.ClientName, Node: names[row.NodeID]}
	case reach.Away():
		return &Refusal{Code: RefusedClientAway, Client: row.ClientName, Node: names[row.NodeID]}
	}
	return nil
}

func (c *Clients) askOn(ctx context.Context, node role.NodeID, act dlv1.Act, ids []ID) map[ID]*Refusal {
	out := make(map[ID]*Refusal, len(ids))
	for _, id := range ids {
		out[id] = &Refusal{Code: RefusedOffline}
	}
	if !c.cfg.Roster.InStep(node) {
		return out
	}
	res, err := c.cfg.Remote(node).Act(ctx, &dlv1.ActRequest{Act: act, Ids: wireIDs(ids)})
	if err != nil {
		c.cfg.Log.Debug("ask a Node's torrent client to act", "node", node.String(), "err", err.Error())
		return out
	}
	for _, r := range res.GetResults() {
		id, err := idOf(r.GetId())
		if err != nil || out[id] == nil {
			continue
		}
		out[id] = refusalFromWire(r)
	}
	return out
}

func refusalFromWire(r *dlv1.ActResult) *Refusal {
	switch r.GetRefused() {
	case dlv1.Refusal_REFUSAL_UNSPECIFIED:
		return nil
	case dlv1.Refusal_REFUSAL_WINDOW:
		return &Refusal{Code: RefusedWindow, AcceptsAt: time.UnixMilli(r.GetAcceptsAtUnixMs()).UTC()}
	case dlv1.Refusal_REFUSAL_NOT_RUNNING:
		return &Refusal{Code: RefusedNotRunning}
	case dlv1.Refusal_REFUSAL_FAILED:
		return &Refusal{Code: RefusedFailed}
	case dlv1.Refusal_REFUSAL_CLIENT_AWAY:
		return &Refusal{Code: RefusedClientAway}
	case dlv1.Refusal_REFUSAL_NOT_HELD:
		return &Refusal{Code: RefusedNotFound}
	}
	return &Refusal{Code: RefusedNotFound}
}

// byNode groups ids by the Node at the same position in nodes.
func byNode(nodes []role.NodeID, ids []ID) map[role.NodeID][]ID {
	out := map[role.NodeID][]ID{}
	for i, id := range ids {
		out[nodes[i]] = append(out[nodes[i]], id)
	}
	return out
}

func wireIDs(ids []ID) [][]byte {
	out := make([][]byte, len(ids))
	for i, id := range ids {
		out[i] = id.Bytes()
	}
	return out
}
