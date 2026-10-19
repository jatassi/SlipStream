package downloadclients

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// Intent is node's Downloading intent: each of its live Torrents, by info
// hash, with its id where the wire names a Grab.
func Intent(ctx context.Context, r storage.Reader[file], node role.NodeID) (*dlv1.Intent, error) {
	rows, err := db.New(r).ListIntent(ctx, node)
	if err != nil {
		return nil, err
	}
	intent := &dlv1.Intent{}
	for _, row := range rows {
		intent.Torrents = append(intent.Torrents, &dlv1.Torrent{InfoHash: row.InfoHash, Metainfo: row.Metainfo, GrabId: row.ID.Bytes()})
	}
	return intent, nil
}

// Apply records how node's client ended a Torrent's download: the first
// completion's files, or the engine's failure, each as the latest report.
// A result for an id that is not node's Torrent changes nothing, and so
// does a second completion.
func Apply(ctx context.Context, tx *storage.Tx[file], node role.NodeID, res *dlv1.Result) error {
	uuid, err := storage.UUIDFromBytes(res.GetGrabId())
	if err != nil {
		return err
	}
	id, now, q := ID{UUID: uuid}, storage.Now(), db.New(tx)
	rep := Report{At: now.Time()}
	switch o := res.GetOutcome().(type) {
	case *dlv1.Result_Completed:
		c := Completion{Files: make([]engine.File, len(o.Completed.GetFiles()))}
		for i, f := range o.Completed.GetFiles() {
			c.Files[i] = engine.File{Index: int(f.GetIndex()), Path: f.GetPath(), Size: f.GetSize()}
		}
		completion, err := json.Marshal(c)
		if err != nil {
			return fmt.Errorf("encode the completion: %w", err)
		}
		if err := q.SetCompletion(ctx, db.SetCompletionParams{Completion: new(string(completion)), CompletedAt: now, ID: id, NodeID: node}); err != nil {
			return err
		}
		rep.Engine, rep.Complete = EngineSeeding, true
	case *dlv1.Result_Failed:
		rep.Engine, rep.Error = EngineFailed, o.Failed.GetError()
	}
	report, err := json.Marshal(rep)
	if err != nil {
		return fmt.Errorf("encode the report: %w", err)
	}
	return q.SetReport(ctx, db.SetReportParams{Report: new(string(report)), ReportedAt: now, ID: id, NodeID: node})
}

// Held is a live Torrent as a Node's client keeps it.
type Held struct {
	ID       ID
	InfoHash bencode.InfoHash
}

// Kept is node's live Torrents in its intent's order.
func Kept(ctx context.Context, r storage.Reader[file], node role.NodeID) ([]Held, error) {
	rows, err := db.New(r).ListIntent(ctx, node)
	if err != nil {
		return nil, err
	}
	out := make([]Held, len(rows))
	for i, row := range rows {
		out[i] = Held{ID: row.ID, InfoHash: bencode.InfoHash(row.InfoHash)}
	}
	return out, nil
}

// DownloadingOwner counts a Node's live Torrents. They seed until seeding
// goals exist, so a Node never finishes them on its own.
var DownloadingOwner = nodes.Owner{Role: role.Downloading, Held: held}

func held(ctx context.Context, r storage.Reader[file], node role.NodeID) (nodes.Held, error) {
	row, err := db.New(r).CountHeld(ctx, node)
	return nodes.Held{Units: row.Torrents, Noun: nodes.Noun{One: "torrent", Many: "torrents"}, Bytes: row.Bytes, Endless: true}, err
}

// Holding is every Node with a Torrent not yet parted from it.
func Holding(ctx context.Context, r storage.Reader[file]) ([]role.NodeID, error) {
	return db.New(r).ListHoldingNodes(ctx)
}

// WriteOff parts every Torrent node holds at once, files kept, as the
// Admin gave up what it holds for Downloading: each leaves its intent, and
// no removal waits on a Node that won't confirm it. It counts what it
// parted, so a second call parts nothing.
func WriteOff(ctx context.Context, tx *storage.Tx[file], node role.NodeID, now storage.Time) (int64, error) {
	return db.New(tx).WriteOff(ctx, db.WriteOffParams{At: now, NodeID: node})
}
