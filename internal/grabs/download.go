package grabs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// DownloadIntent is node's Downloading intent: the torrent of every Grab on
// node past fetching, whatever its stage, since every torrent seeds until
// seeding goals exist.
func DownloadIntent(ctx context.Context, r storage.Reader[file], node role.NodeID) (*dlv1.Intent, error) {
	rows, err := db.New(r).ListTorrentsOfNode(ctx, node)
	if err != nil {
		return nil, err
	}
	intent := &dlv1.Intent{}
	for _, row := range rows {
		intent.Torrents = append(intent.Torrents, &dlv1.Torrent{InfoHash: row.InfoHash, Metainfo: row.Metainfo, GrabId: row.ID.Bytes()})
	}
	return intent, nil
}

// Kept is every Grab on node whose torrent its Downloading intent lists,
// with the release's title, in the intent's order.
func Kept(ctx context.Context, r storage.Reader[file], node role.NodeID) ([]nodes.Grabbed, error) {
	rows, err := db.New(r).ListKeptOfNode(ctx, node)
	if err != nil {
		return nil, err
	}
	out := make([]nodes.Grabbed, len(rows))
	for i, row := range rows {
		out[i] = nodes.Grabbed{Grab: row.ID.UUID, InfoHash: bencode.InfoHash(row.InfoHash), Title: row.ReleaseTitle}
	}
	return out, nil
}

// ApplyDownload records how a Grab's download ended. It never moves the
// stage: the reconciler does within one progress tick. A result for a
// Grab past downloading, already recorded, or on another Node changes
// nothing, so a resend after a restart is harmless.
func ApplyDownload(ctx context.Context, tx *storage.Tx[file], node role.NodeID, res *dlv1.Result) error {
	id, err := storage.UUIDFromBytes(res.GetGrabId())
	if err != nil {
		return err
	}
	params := db.RecordDownloadParams{ID: GrabID{UUID: id}, NodeID: node}
	switch o := res.GetOutcome().(type) {
	case *dlv1.Result_Completed:
		fs := make([]engine.File, len(o.Completed.GetFiles()))
		for i, f := range o.Completed.GetFiles() {
			fs[i] = engine.File{Index: int(f.GetIndex()), Path: f.GetPath(), Size: f.GetSize()}
		}
		b, err := json.Marshal(fs)
		if err != nil {
			return fmt.Errorf("encode the download's files: %w", err)
		}
		params.DownloadFiles = new(string(b))
	case *dlv1.Result_Failed:
		params.DownloadError = new(o.Failed.GetError())
	}
	return db.New(tx).RecordDownload(ctx, params)
}
