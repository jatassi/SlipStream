package downloadclients

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/works"
)

// Uses is the Grabs that used Torrents, as grabs answers it. Nothing here
// writes a Grab.
type Uses interface {
	// Of is the Grabs of each of ids, newest first, in r's snapshot.
	Of(ctx context.Context, r storage.Reader[file], ids []ID) (map[ID][]Use, error)
}

// Use is one Grab of a Torrent.
type Use struct {
	Grab    storage.UUID
	Work    works.WorkID
	Item    works.ItemID
	Edition string
	Release string
	Stage   UseStage
	// File is the index of the Torrent's file the Grab took, nil until it
	// probed one.
	File    *int
	Created time.Time
}

type UseStage string

const (
	UseDownloading UseStage = "downloading"
	// UseImporting is a Grab probing, transferring or importing the
	// Torrent's files.
	UseImporting UseStage = "importing"
	UseImported  UseStage = "imported"
	UseFailed    UseStage = "failed"
)

type ActionKind string

const (
	Pause    ActionKind = "pause"
	Resume   ActionKind = "resume"
	Announce ActionKind = "announce"
	Recheck  ActionKind = "recheck"
	// MarkFailed removes a stalled download with its download copy, and
	// fails its downloading Grabs as download_failed.
	MarkFailed ActionKind = "mark-failed"
	Remove     ActionKind = "remove"
)

// ActionKinds is every ActionKind.
var ActionKinds = []ActionKind{Pause, Resume, Announce, Recheck, MarkFailed, Remove}

type Action struct {
	Kind ActionKind
	IDs  []ID
	// DeleteFiles deletes the download copy of a removed Torrent.
	DeleteFiles bool
	// DryRun answers what remove, mark-failed or recheck would do, and
	// changes nothing.
	DryRun bool
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
	// RefusedNotStalled is mark-failed on a Torrent that isn't stalled.
	RefusedNotStalled RefusalCode = "not_stalled"
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
)

type Refusal struct {
	Code RefusalCode
	// Work is the Work being imported, for RefusedImporting.
	Work works.WorkID
	// AcceptsAt is when an announce would be accepted, for RefusedWindow.
	AcceptsAt time.Time
	// Client and Node name the external client and its Node, for
	// RefusedClientAway and RefusedClientOff.
	Client, Node string
}

// Preview is what a remove or a recheck would do.
type Preview struct {
	// Cancels are the Grabs at downloading the remove fails.
	Cancels []Use
	// StopSeeding are the Works whose complete Torrents stop seeding.
	StopSeeding []works.WorkID
	// Frees is the bytes deleting the download copies would free, nil
	// while a Node can't say.
	Frees *int64
	// Linked is some file also being a library file: a recheck reads it.
	Linked bool
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
	var nodes []role.NodeID
	var changed []ID
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		results, nodes, changed = nil, nil, nil
		rows, err := readActed(ctx, tx, a.IDs)
		if err != nil {
			return err
		}
		for _, id := range a.IDs {
			r := Result{ID: id, Refusal: rows.refusal(id)}
			results = append(results, r)
			if r.Refusal != nil || Wanted(rows[id].Wanted) == wanted {
				continue
			}
			if err := db.New(tx).SetWanted(ctx, db.SetWantedParams{Wanted: string(wanted), ID: id}); err != nil {
				return err
			}
			nodes, changed = append(nodes, rows[id].NodeID), append(changed, id)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	c.committed(nodes, changed)
	return results, nil, nil
}

// remove removes each Torrent no Grab is importing from, in one Write that
// reads the Grabs, so an import that started first refuses it and one
// that starts after finds it removed. A dry run reads the same and writes
// nothing.
func (c *Clients) remove(ctx context.Context, a Action) ([]Result, *Preview, error) {
	reason, deleteFiles := ByAdmin, a.DeleteFiles
	if a.Kind == MarkFailed {
		reason, deleteFiles = ByMarkFailed, true
	}
	var plan removal
	if a.DryRun {
		err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
			var err error
			plan, err = c.planRemoval(ctx, r, a)
			return err
		})
		if err != nil {
			return nil, nil, err
		}
		nodes, ids := plan.deleting()
		plan.preview.Frees, plan.preview.Linked = c.frees(ctx, nodes, ids)
		return plan.results, plan.preview, nil
	}
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		if plan, err = c.planRemoval(ctx, tx, a); err != nil {
			return err
		}
		now := storage.Now()
		for _, id := range plan.ids {
			if err := removeIn(ctx, db.New(tx), plan.rows[id], reason, deleteFiles, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	c.committed(plan.nodes, plan.ids)
	return plan.results, nil, nil
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

func (c *Clients) planRemoval(ctx context.Context, r storage.Reader[file], a Action) (removal, error) {
	plan := removal{preview: &Preview{}}
	rows, err := readActed(ctx, r, a.IDs)
	if err != nil {
		return removal{}, err
	}
	plan.rows = rows
	uses, err := c.cfg.Uses.Of(ctx, r, a.IDs)
	if err != nil {
		return removal{}, err
	}
	for _, id := range a.IDs {
		res := Result{ID: id, Refusal: rows.refusal(id)}
		if res.Refusal == nil {
			if res.Refusal, err = removable(rows[id], uses[id], a.Kind); err != nil {
				return removal{}, err
			}
		}
		plan.results = append(plan.results, res)
		if res.Refusal == nil {
			plan.ids, plan.nodes = append(plan.ids, id), append(plan.nodes, rows[id].NodeID)
			plan.preview.add(rows[id], uses[id])
		}
	}
	return plan, nil
}

// removable is why a Torrent may not be removed by kind: a Grab importing
// from it, or for mark-failed, a download that isn't stalled.
func removable(row db.ListActedRow, uses []Use, kind ActionKind) (*Refusal, error) {
	if i := slices.IndexFunc(uses, func(u Use) bool { return u.Stage == UseImporting }); i >= 0 {
		return &Refusal{Code: RefusedImporting, Work: uses[i].Work}, nil
	}
	if kind != MarkFailed {
		return nil, nil
	}
	rep, err := reportOf(row.Report)
	if err != nil {
		return nil, fmt.Errorf("torrent %s: %w", row.ID, err)
	}
	if StateOf(Facts{Wanted: Wanted(row.Wanted), Report: rep}) != StateStalled {
		return &Refusal{Code: RefusedNotStalled}, nil
	}
	return nil, nil
}

// add counts a removed Torrent's Grabs into p: those downloading it are
// canceled, and once complete, every Work it seeds for stops seeding.
func (p *Preview) add(row db.ListActedRow, uses []Use) {
	for _, u := range uses {
		if u.Stage == UseDownloading {
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
