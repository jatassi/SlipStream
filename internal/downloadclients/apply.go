package downloadclients

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
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
}

// Clients applies each Node's reports and tells their readers once they
// commit, and keeps each Node's client settings.
type Clients struct {
	cfg Config

	mu sync.Mutex
	// pending is what each Node's applied reports changed, until Applied.
	pending map[role.NodeID]*pending
	moved   chan struct{}
}

type pending struct {
	ids    []ID
	parted bool
}

func New(cfg Config) *Clients {
	return &Clients{cfg: cfg, pending: map[role.NodeID]*pending{}, moved: make(chan struct{})}
}

// Touch tells the Downloading Control nodes' intents may have changed.
func (c *Clients) Touch(nodes ...role.NodeID) { c.cfg.Touch(nodes...) }

// Moved is closed once reports that may move a Torrent's outcome commit.
func (c *Clients) Moved() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.moved
}

// Apply records one of node's reports in tx: the Torrents' latest
// reports, a totals batch, or a removal's confirmation. An id that is not
// node's Torrent changes nothing. Applied tells the readers once tx
// commits.
func (c *Clients) Apply(ctx context.Context, tx *storage.Tx[file], node role.NodeID, rep *dlv1.Report) error {
	var (
		ids    []ID
		parted bool
		err    error
	)
	q := db.New(tx)
	switch k := rep.GetKind().(type) {
	case *dlv1.Report_Torrents:
		ids, err = applyReports(ctx, q, node, k.Torrents.GetReports())
	case *dlv1.Report_Totals:
		ids, err = applyTotals(ctx, tx, node, k.Totals.GetRows())
	case *dlv1.Report_Parted:
		var id ID
		if id, err = idOf(k.Parted.GetId()); err == nil {
			var n int64
			n, err = q.SetParted(ctx, db.SetPartedParams{PartedAt: storage.Now(), Freed: new(k.Parted.GetFreed()), ID: id, NodeID: node})
			ids, parted = []ID{id}, n > 0
		}
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
	p.ids = append(p.ids, ids...)
	p.parted = p.parted || parted
	return nil
}

// Applied is role.ControlsConfig.Applied: once node's Downloading reports
// commit, it publishes the Torrents they changed, wakes what reads their
// outcomes, and touches node's intent when a discard parted.
func (c *Clients) Applied(node role.NodeID, scope string) {
	if scope != string(role.Downloading) {
		return
	}
	c.mu.Lock()
	p := c.pending[node]
	delete(c.pending, node)
	close(c.moved)
	c.moved = make(chan struct{})
	c.mu.Unlock()
	if p == nil {
		return
	}
	if p.parted {
		c.cfg.Touch(node)
	}
	if len(p.ids) == 0 {
		return
	}
	slices.SortFunc(p.ids, func(a, b ID) int { return a.Compare(b.UUID) })
	p.ids = slices.Compact(p.ids)
	ch := Changed{IDs: make([]string, len(p.ids))}
	for i, id := range p.ids {
		ch.IDs[i] = id.String()
	}
	c.cfg.Changed.Publish(context.Background(), ch)
}

func applyReports(ctx context.Context, q *db.Queries, node role.NodeID, reports []*dlv1.TorrentReport) ([]ID, error) {
	ids := make([]ID, 0, len(reports))
	for _, r := range reports {
		id, err := idOf(r.GetId())
		if err != nil {
			return nil, err
		}
		held, err := applyReport(ctx, q, node, id, r)
		if err != nil {
			return nil, err
		}
		if held {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// applyReport records r, false when id is not node's Torrent.
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
		ActiveSeconds: t.GetActiveSeconds(), SeedingSeconds: t.GetSeedingSeconds(),
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
		At:       time.UnixMilli(r.GetAtUnixMs()).UTC(),
		Engine:   engineFromWire[r.GetEngine()],
		Hold:     holdFromWire[r.GetHold()],
		Complete: r.GetComplete(),
		Stalled:  r.GetStalled(),
		Error:    r.GetError(),
		Done:     r.GetDone(),
	}
	if u := r.GetUntilUnixMs(); u != 0 {
		out.Until = time.UnixMilli(u).UTC()
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
		dlv1.Hold_HOLD_UNSPECIFIED: HoldNone,
		dlv1.Hold_HOLD_WANTED:      HoldWanted,
		dlv1.Hold_HOLD_WINDOW:      HoldWindow,
		dlv1.Hold_HOLD_DISK_FULL:   HoldDiskFull,
		dlv1.Hold_HOLD_SLOT:        HoldSlot,
		dlv1.Hold_HOLD_CHECK_SLOT:  HoldCheckSlot,
		dlv1.Hold_HOLD_DISK_GUARD:  HoldDiskGuard,
		dlv1.Hold_HOLD_DRAINING:    HoldDraining,
		dlv1.Hold_HOLD_CONTROLLER:  HoldController,
	}
	trackerFromWire = map[dlv1.TrackerStatus]TrackerStatus{
		dlv1.TrackerStatus_TRACKER_STATUS_WORKING:       TrackerWorking,
		dlv1.TrackerStatus_TRACKER_STATUS_UPDATING:      TrackerUpdating,
		dlv1.TrackerStatus_TRACKER_STATUS_ERROR:         TrackerError,
		dlv1.TrackerStatus_TRACKER_STATUS_UNREGISTERED:  TrackerUnregistered,
		dlv1.TrackerStatus_TRACKER_STATUS_NOT_CONTACTED: TrackerNotContacted,
	}
)
