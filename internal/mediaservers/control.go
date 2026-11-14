package mediaservers

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

// ControlService serves MediaServerControlService: a reaching Node's fetch
// of its servers' credentials.
type ControlService struct {
	Library *storage.DB[storage.Library]
	Sealer  *secret.Sealer
	Nodes   interface {
		Caller(ctx context.Context) (role.NodeID, bool)
		Self() role.NodeID
	}
}

var errNotReaching = errors.New("the calling Node doesn't reach this Media Server")

// MediaServerSecrets answers a server's credential and the settings
// version it belongs to only to the server's reaching Node, and NotFound to
// any other caller and for a deleted server. It records each Worker that
// fetched it, since naming another Node doesn't take the credential back.
func (s ControlService) MediaServerSecrets(ctx context.Context, req *msv1.MediaServerSecretsRequest) (*msv1.MediaServerSecretsResponse, error) {
	id, err := serverIDOf(req.GetServerId())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	caller, known := s.Nodes.Caller(ctx)
	var (
		cred    link.Credential
		version int64
	)
	err = s.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		row, err := q.GetSecrets(ctx, id)
		switch {
		case errors.Is(err, storage.ErrNoRows), err == nil && (!known || row.ReachingNodeID != caller):
			return errNotReaching
		case err != nil:
			return err
		}
		if cred, err = unsealCredential(s.Sealer, id, row.SecretsSealed); err != nil {
			return err
		}
		version = row.SettingsVersion
		if caller == s.Nodes.Self() {
			return nil
		}
		return q.RecordFetch(ctx, db.RecordFetchParams{ServerID: id, NodeID: caller, SettingsVersion: version, FetchedAt: storage.Now()})
	})
	switch {
	case errors.Is(err, errNotReaching):
		return nil, rpc.Errorf(rpc.CodeNotFound, "the calling Node reaches no Media Server %s", id)
	case err != nil:
		return nil, err
	}
	return &msv1.MediaServerSecretsResponse{Credential: &msv1.Credential{Token: cred.Token, ApiKey: cred.APIKey}, SettingsVersion: version}, nil
}
