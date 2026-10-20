package downloadclients

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// TorrentService serves DownloadingControlService: a Node's fetch of a
// Torrent of its intent it holds no data for.
type TorrentService struct {
	Library *storage.DB[file]
	Nodes   interface {
		Caller(ctx context.Context) (role.NodeID, bool)
	}
	Types interface {
		Describe(t works.TypeID) (works.Description, bool)
	}
}

// Torrent answers only the Torrent's own Node, since its metainfo carries
// the tracker's passkey, and a removed Torrent is NotFound.
func (s TorrentService) Torrent(ctx context.Context, req *dlv1.TorrentRequest) (*dlv1.TorrentResponse, error) {
	id, err := idOf(req.GetId())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	caller, known := s.Nodes.Caller(ctx)
	row, err := db.New(s.Library).GetFetched(ctx, id)
	switch {
	case errors.Is(err, storage.ErrNoRows), err == nil && (!known || row.NodeID != caller || !row.RemovedAt.IsZero()):
		return nil, rpc.Errorf(rpc.CodeNotFound, "the calling Node holds no Torrent %s", id)
	case err != nil:
		return nil, err
	}
	desc, _ := s.Types.Describe(row.ContentType)
	res := &dlv1.TorrentResponse{
		Metainfo: row.Metainfo,
		Baseline: &dlv1.TotalsRow{
			Id: id.Bytes(), Uploaded: row.Uploaded, Downloaded: row.Downloaded,
			ActiveSeconds: row.ActiveSeconds, SeedingSeconds: row.SeedingSeconds,
		},
		ContentType: desc.Name,
	}
	if row.SaveFolder != nil {
		res.SaveFolder = *row.SaveFolder
	}
	return res, nil
}
