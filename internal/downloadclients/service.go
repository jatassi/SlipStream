package downloadclients

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// ControlService serves DownloadingControlService: a Node's fetch of a
// Torrent of its intent it holds no data for, and of an external client's
// secrets.
type ControlService struct {
	Library *storage.DB[file]
	Sealer  *secret.Sealer
	Nodes   interface {
		Caller(ctx context.Context) (role.NodeID, bool)
	}
	Types interface {
		Describe(t works.TypeID) (works.Description, bool)
	}
}

// Torrent answers only the Torrent's own Node, since its metainfo carries
// the tracker's passkey, and a removed Torrent is NotFound. It carries the
// origin the Controller recorded, so a Node that lost its local row learns
// it again.
func (s ControlService) Torrent(ctx context.Context, req *dlv1.TorrentRequest) (*dlv1.TorrentResponse, error) {
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
	if row.Attached != nil {
		res.Origin = originOnWire[*row.Attached]
	}
	return res, nil
}

var originOnWire = map[bool]dlv1.Origin{false: dlv1.Origin_ORIGIN_ADDED, true: dlv1.Origin_ORIGIN_ATTACHED}

// ClientSecrets answers an external client's secrets and the settings
// version they belong to only to the client's own Node, and NotFound for a
// deleted client or another Node's.
func (s ControlService) ClientSecrets(ctx context.Context, req *dlv1.ClientSecretsRequest) (*dlv1.ClientSecretsResponse, error) {
	u, err := storage.UUIDFromBytes(req.GetId())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "client id: %v", err)
	}
	id := ClientID{UUID: u}
	caller, known := s.Nodes.Caller(ctx)
	row, err := db.New(s.Library).GetClientSecrets(ctx, id)
	switch {
	case errors.Is(err, storage.ErrNoRows), err == nil && (!known || row.NodeID != caller):
		return nil, rpc.Errorf(rpc.CodeNotFound, "the calling Node has no Download Client %s", id)
	case err != nil:
		return nil, err
	}
	secrets, err := unseal(s.Sealer, id, row.SecretsSealed)
	if err != nil {
		return nil, err
	}
	return &dlv1.ClientSecretsResponse{
		Secrets: &dlv1.Secrets{Password: secrets.Password, ApiKey: secrets.APIKey}, SettingsVersion: row.SettingsVersion,
	}, nil
}
