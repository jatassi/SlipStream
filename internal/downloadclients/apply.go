package downloadclients

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// Changed names the Torrents whose record moved.
type Changed struct {
	IDs []string `json:"ids" nullable:"false" doc:"The Torrents whose record changed"`
}

var ChangedEvent = live.AdminEvent[Changed]("torrents.changed")

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

type Config struct {
	Library *storage.DB[file]
	// Touch tells the Downloading Control a Node's intent may have
	// changed.
	Touch   func(nodes ...role.NodeID)
	Changed Publisher[Changed]
	// CardsChanged tells the Admins a Node's client card or settings
	// moved.
	CardsChanged Publisher[ClientsChanged]
	// Facts is a Node's latest client facts, false while it has none.
	Facts func(role.NodeID) (*dlv1.Facts, bool)
	// Remote is a Node's client as the Controller calls it.
	Remote func(role.NodeID) downloadingv1connect.DownloadingServiceHandler
	// Sealer seals and opens external clients' secrets.
	Sealer  *secret.Sealer
	Roster  Roster
	Uses    Uses
	Seeding Seeder
	Live    LiveConfig
	// Hub wakes Run when the storage levels the disk guard follows change.
	Hub Hub
	Log *slog.Logger
}

type Hub interface {
	Watch(ctx context.Context, events ...live.Event) <-chan struct{}
}

// Roster is the Nodes and the Roles they hold.
type Roster interface {
	// InStep reports whether node's Sync session is attached.
	InStep(node role.NodeID) bool
	// Holds reports whether node is assigned r, in step or not.
	Holds(r role.Name, node role.NodeID) bool
	// WrittenOff reports whether the Admin wrote off what node holds for
	// r.
	WrittenOff(r role.Name, node role.NodeID) bool
	// Changed is closed at the roster's next change.
	Changed() <-chan struct{}
	// Serving is every Node r is assigned to or drains, in id order.
	Serving(r role.Name) []role.NodeID
}

// Clients applies each Node's reports and tells their readers once they
// commit, and keeps each Node's client settings.
type Clients struct {
	cfg Config

	mu sync.Mutex
	// pending is what each Node's applied reports changed, until Applied.
	pending map[role.NodeID]*pending
	moved   chan struct{}

	watching watching
	disk     disk
}

type pending struct {
	ids []ID
	// touch is a change to the Node's intent: a discard parted, a Torrent
	// its client removed, or a wanted state its client changed.
	touch bool
	// polled is the clients whose poll state changed.
	polled []ClientID
}

func New(cfg Config) *Clients {
	return &Clients{
		cfg: cfg, pending: map[role.NodeID]*pending{}, moved: make(chan struct{}),
		watching: watching{nodes: map[role.NodeID]*watched{}, moved: map[ID]Sample{}},
		disk:     disk{guarded: map[role.NodeID]bool{}, recorded: make(chan struct{})},
	}
}

// Touch tells the Downloading Control nodes' intents may have changed.
func (c *Clients) Touch(nodes ...role.NodeID) { c.cfg.Touch(nodes...) }

// Moved is closed once a change that may move a Torrent's outcome
// commits: a Node's reports, an Admin's action or a write-off.
func (c *Clients) Moved() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.moved
}

// Apply records one of node's reports in tx: the Torrents' latest
// reports, a totals batch, a removal's confirmation, an external client's
// poll, a torrent its client stopped listing, or an external client's raw
// counters. An id that is not node's changes nothing, and each applies
// alike when the outbox replays it. Applied tells the readers once tx
// commits.
func (c *Clients) Apply(ctx context.Context, tx *storage.Tx[file], node role.NodeID, rep *dlv1.Report) error {
	var (
		got pending
		err error
	)
	q := db.New(tx)
	switch k := rep.GetKind().(type) {
	case *dlv1.Report_Torrents:
		got.ids, got.touch, err = applyReports(ctx, q, node, k.Torrents.GetReports())
	case *dlv1.Report_Totals:
		got.ids, err = applyTotals(ctx, tx, node, k.Totals.GetRows())
	case *dlv1.Report_Parted:
		var id ID
		if id, err = idOf(k.Parted.GetId()); err == nil {
			var n int64
			n, err = q.SetParted(ctx, db.SetPartedParams{PartedAt: storage.Now(), Freed: new(k.Parted.GetFreed()), ID: id, NodeID: node})
			got.ids, got.touch = []ID{id}, n > 0
		}
	case *dlv1.Report_Client:
		got.polled, got.ids, err = applyPoll(ctx, q, node, k.Client)
	case *dlv1.Report_Vanished:
		var id ID
		if id, err = idOf(k.Vanished.GetId()); err == nil {
			var n int64
			n, err = q.Vanish(ctx, db.VanishParams{At: storage.Now(), ID: id, NodeID: node})
			got.ids, got.touch = []ID{id}, n > 0
		}
	case *dlv1.Report_ClientTotals:
		got.ids, err = applyClientTotals(ctx, q, node, k.ClientTotals.GetRows())
	default:
		return fmt.Errorf("a report of kind %T", k)
	}
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.pending[node]
	if p == nil {
		p = &pending{}
		c.pending[node] = p
	}
	p.ids = append(p.ids, got.ids...)
	p.touch = p.touch || got.touch
	p.polled = append(p.polled, got.polled...)
	return nil
}

// Applied is role.ControlsConfig.Applied: once node's Downloading reports
// commit, it publishes the Torrents and client cards they changed, wakes
// what reads their outcomes, and touches node's intent when they changed
// it.
func (c *Clients) Applied(node role.NodeID, scope string) {
	if scope != string(role.Downloading) {
		return
	}
	c.mu.Lock()
	p := c.pending[node]
	delete(c.pending, node)
	c.mu.Unlock()
	if p == nil {
		c.wake()
		return
	}
	var touched []role.NodeID
	if p.touch {
		touched = []role.NodeID{node}
	}
	if len(p.polled) > 0 {
		c.clientsChanged(p.polled, node)
	}
	c.committed(touched, p.ids)
}

// committed tells the readers once a change to ids commits: the intents
// of nodes, the Admins, and whatever reads the Torrents' outcomes.
func (c *Clients) committed(nodes []role.NodeID, ids []ID) {
	if len(nodes) > 0 {
		c.cfg.Touch(nodes...)
	}
	if len(ids) > 0 {
		ids = slices.SortedFunc(slices.Values(ids), func(a, b ID) int { return a.Compare(b.UUID) })
		ids = slices.Compact(ids)
		ch := Changed{IDs: make([]string, len(ids))}
		for i, id := range ids {
			ch.IDs[i] = id.String()
		}
		c.cfg.Changed.Publish(context.Background(), ch)
	}
	c.wake()
}

func (c *Clients) wake() {
	c.mu.Lock()
	defer c.mu.Unlock()
	close(c.moved)
	c.moved = make(chan struct{})
}

// applyReports records each report, and whether one changed a Torrent's
// wanted state or settled its Retry.
func applyReports(ctx context.Context, q *db.Queries, node role.NodeID, reports []*dlv1.TorrentReport) (ids []ID, touched bool, err error) {
	ids = make([]ID, 0, len(reports))
	for _, r := range reports {
		id, err := idOf(r.GetId())
		if err != nil {
			return nil, false, err
		}
		held, err := applyReport(ctx, q, node, id, r)
		if err != nil {
			return nil, false, err
		}
		if !held {
			continue
		}
		ids = append(ids, id)
		changed, err := applyExternal(ctx, q, node, id, r)
		if err != nil {
			return nil, false, err
		}
		settled, err := settleRetry(ctx, q, node, id, r)
		if err != nil {
			return nil, false, err
		}
		touched = touched || changed || settled
	}
	return ids, touched, nil
}

// applyExternal records what only an external client's report says: the
// origin where none is recorded, and a pause or resume made in the client
// as a compare and set on the wanted state, true when it landed. A stop at
// the client's seeding limit is no pause, so it never changes wanted.
func applyExternal(ctx context.Context, q *db.Queries, node role.NodeID, id ID, r *dlv1.TorrentReport) (bool, error) {
	if o := r.GetOrigin(); o != dlv1.Origin_ORIGIN_UNSPECIFIED {
		if err := q.SetOrigin(ctx, db.SetOriginParams{Attached: new(o == dlv1.Origin_ORIGIN_ATTACHED), ID: id, NodeID: node}); err != nil {
			return false, err
		}
	}
	wc := r.GetWantedChange()
	if wc == nil || r.GetStoppedAtLimit() {
		return false, nil
	}
	n, err := q.ChangeWanted(ctx, db.ChangeWantedParams{
		FromWanted: string(wantedFromWire[wc.GetFrom()]), ToWanted: string(wantedFromWire[wc.GetTo()]), ID: id, NodeID: node,
	})
	return n > 0, err
}

// applyPoll stores p on its client when the client is node's and p ran
// under settings at least as new as the stored poll's, naming the client
// and its live Torrents, whose rows may turn stale or current, when it
// did.
func applyPoll(ctx context.Context, q *db.Queries, node role.NodeID, p *dlv1.ClientPoll) (polled []ClientID, held []ID, err error) {
	u, err := storage.UUIDFromBytes(p.GetClientId())
	if err != nil {
		return nil, nil, fmt.Errorf("client id: %w", err)
	}
	id := ClientID{UUID: u}
	state, ok := pollStateFromWire[p.GetState()]
	if !ok {
		return nil, nil, fmt.Errorf("client %s: a poll in state %s", id, p.GetState())
	}
	var problem *string
	if pr := p.GetProblem(); pr != nil {
		got, err := problemFromWire(pr)
		if err != nil {
			return nil, nil, fmt.Errorf("client %s: %w", id, err)
		}
		if problem, err = storedOf(got); err != nil {
			return nil, nil, err
		}
	}
	n, err := q.SetPoll(ctx, db.SetPollParams{
		PollState: new(string(state)), PollSince: storage.FromTime(time.UnixMilli(p.GetSinceUnixMs())), PollProblem: problem,
		PollSettingsVersion: new(p.GetSettingsVersion()), PollCheck: p.GetCheck(), ClientVersion: text(p.GetVersion()), ID: id, NodeID: node,
	})
	if err != nil || n == 0 {
		return nil, nil, err
	}
	held, err = q.ListLiveOfClient(ctx, &id)
	return []ClientID{id}, held, err
}

func problemFromWire(pr *dlv1.Problem) (clients.Problem, error) {
	cause, ok := causeFromWire[pr.GetCause()]
	if !ok {
		return clients.Problem{}, fmt.Errorf("a problem with cause %s", pr.GetCause())
	}
	p := clients.Problem{
		Cause: cause, Status: int(pr.GetStatus()), Folder: pr.GetFolder(), Paired: pr.GetPaired(), Host: pr.GetHost(), Missing: int(pr.GetMissing()),
	}
	if v := pr.GetVersion(); v != "" {
		var err error
		if p.Version, err = clients.ParseVersion(v); err != nil {
			return clients.Problem{}, err
		}
	}
	return p, nil
}

// storedOf is p as a poll's problem is stored.
func storedOf(p clients.Problem) (*string, error) {
	s := storedProblem{Cause: p.Cause, Status: p.Status, Folder: p.Folder, Paired: p.Paired, Host: p.Host, Missing: p.Missing}
	if p.Cause == clients.VersionTooOld {
		s.Version = p.Version.String()
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("encode the problem: %w", err)
	}
	return new(string(b)), nil
}

func applyReport(ctx context.Context, q *db.Queries, node role.NodeID, id ID, r *dlv1.TorrentReport) (bool, error) {
	stored, err := json.Marshal(reportFrom(r))
	if err != nil {
		return false, fmt.Errorf("encode the report: %w", err)
	}
	at := storage.FromTime(time.UnixMilli(r.GetAtUnixMs()))
	n, err := q.SetReport(ctx, db.SetReportParams{Report: new(string(stored)), ReportedAt: at, ID: id, NodeID: node})
	if err != nil || n == 0 {
		return false, err
	}
	if f := r.GetSaveFolder(); f != "" {
		if err := q.SetSaveFolder(ctx, db.SetSaveFolderParams{SaveFolder: &f, ID: id, NodeID: node}); err != nil {
			return false, err
		}
	}
	if c := r.GetCompletion(); c != nil {
		if err := setCompletion(ctx, q, node, id, c, at); err != nil {
			return false, err
		}
	}
	if f := r.GetFault(); f != nil {
		kind := string(faultFromWire[f.GetKind()])
		err := q.SetFault(ctx, db.SetFaultParams{Fault: &kind, FaultAt: storage.FromTime(time.UnixMilli(f.GetAtUnixMs())), ID: id, NodeID: node})
		if err != nil {
			return false, err
		}
	}
	if t := r.GetTotals(); t != nil {
		if err := q.MergeTotals(ctx, totalsParams(node, id, t, at)); err != nil {
			return false, err
		}
	}
	return true, nil
}

func setCompletion(ctx context.Context, q *db.Queries, node role.NodeID, id ID, c *dlv1.Completion, at storage.Time) error {
	files := make([]engine.File, len(c.GetFiles()))
	for i, f := range c.GetFiles() {
		files[i] = engine.File{Index: int(f.GetIndex()), Path: f.GetPath(), Size: f.GetSize()}
	}
	stored, err := json.Marshal(Completion{Files: files})
	if err != nil {
		return fmt.Errorf("encode the completion: %w", err)
	}
	return q.SetCompletion(ctx, db.SetCompletionParams{Completion: new(string(stored)), CompletedAt: at, ID: id, NodeID: node})
}

func applyTotals(ctx context.Context, tx *storage.Tx[file], node role.NodeID, rows []*dlv1.TotalsRow) ([]ID, error) {
	q, now := db.New(tx), storage.Now()
	ids := make([]ID, 0, len(rows))
	for _, row := range rows {
		id, err := idOf(row.GetId())
		if err != nil {
			return nil, err
		}
		if err := q.MergeTotals(ctx, totalsParams(node, id, row, now)); err != nil {
			return nil, err
		}
		if row.GetDone() > 0 {
			if err := q.SetDone(ctx, db.SetDoneParams{Done: row.GetDone(), ID: id, NodeID: node}); err != nil {
				return nil, err
			}
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func totalsParams(node role.NodeID, id ID, t *dlv1.TotalsRow, at storage.Time) db.MergeTotalsParams {
	return db.MergeTotalsParams{
		Uploaded: t.GetUploaded(), Downloaded: t.GetDownloaded(),
		ActiveSeconds: t.GetActiveSeconds(), SeedingSeconds: t.GetSeedingSeconds(), AnnouncedSeconds: t.GetAnnouncedSeconds(),
		TotalsAt: at, ID: id, NodeID: node,
	}
}

func idOf(b []byte) (ID, error) {
	u, err := storage.UUIDFromBytes(b)
	if err != nil {
		return ID{}, fmt.Errorf("torrent id: %w", err)
	}
	return ID{UUID: u}, nil
}

func reportFrom(r *dlv1.TorrentReport) Report {
	out := Report{
		At:             time.UnixMilli(r.GetAtUnixMs()).UTC(),
		Engine:         engineFromWire[r.GetEngine()],
		Hold:           holdFromWire[r.GetHold()],
		Complete:       r.GetComplete(),
		Stalled:        r.GetStalled(),
		Error:          r.GetError(),
		Done:           r.GetDone(),
		Run:            wantedFromWire[r.GetRun()],
		QueuePosition:  int(r.GetQueuePosition()),
		StoppedAtLimit: r.GetStoppedAtLimit(),
	}
	if u := r.GetUntilUnixMs(); u != 0 {
		out.Until = time.UnixMilli(u).UTC()
	}
	if u := r.GetImportAtUnixMs(); u != 0 {
		out.ImportAt = time.UnixMilli(u).UTC()
	}
	for _, t := range r.GetTrackers() {
		line := TrackerLine{Index: int(t.GetIndex()), Status: trackerFromWire[t.GetStatus()], Message: t.GetMessage()}
		if u := t.GetRetryAtUnixMs(); u != 0 {
			line.RetryAt = time.UnixMilli(u).UTC()
		}
		out.Trackers = append(out.Trackers, line)
	}
	return out
}

var (
	engineFromWire = map[dlv1.EngineState]EngineState{
		dlv1.EngineState_ENGINE_STATE_CHECKING:    EngineChecking,
		dlv1.EngineState_ENGINE_STATE_DOWNLOADING: EngineDownloading,
		dlv1.EngineState_ENGINE_STATE_SEEDING:     EngineSeeding,
		dlv1.EngineState_ENGINE_STATE_FAILED:      EngineFailed,
	}
	holdFromWire = map[dlv1.Hold]Hold{
		dlv1.Hold_HOLD_UNSPECIFIED:  HoldNone,
		dlv1.Hold_HOLD_WANTED:       HoldWanted,
		dlv1.Hold_HOLD_WINDOW:       HoldWindow,
		dlv1.Hold_HOLD_DISK_FULL:    HoldDiskFull,
		dlv1.Hold_HOLD_SLOT:         HoldSlot,
		dlv1.Hold_HOLD_CHECK_SLOT:   HoldCheckSlot,
		dlv1.Hold_HOLD_DISK_GUARD:   HoldDiskGuard,
		dlv1.Hold_HOLD_DRAINING:     HoldDraining,
		dlv1.Hold_HOLD_CONTROLLER:   HoldController,
		dlv1.Hold_HOLD_CLIENT_ERROR: HoldClientError,
	}
	faultFromWire = map[dlv1.FaultKind]FaultKind{
		dlv1.FaultKind_FAULT_KIND_ENGINE_ERROR: FaultEngineError,
		dlv1.FaultKind_FAULT_KIND_UNREGISTERED: FaultUnregistered,
		dlv1.FaultKind_FAULT_KIND_NO_PAYLOAD:   FaultNoPayload,
	}
	trackerFromWire = map[dlv1.TrackerStatus]TrackerStatus{
		dlv1.TrackerStatus_TRACKER_STATUS_WORKING:       TrackerWorking,
		dlv1.TrackerStatus_TRACKER_STATUS_UPDATING:      TrackerUpdating,
		dlv1.TrackerStatus_TRACKER_STATUS_ERROR:         TrackerError,
		dlv1.TrackerStatus_TRACKER_STATUS_UNREGISTERED:  TrackerUnregistered,
		dlv1.TrackerStatus_TRACKER_STATUS_NOT_CONTACTED: TrackerNotContacted,
	}
)
