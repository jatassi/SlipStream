package mediaservers_test

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/link"
	mslink "github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1/mediaserverv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

func schemas() []storage.Schema[storage.Library] {
	return []storage.Schema[storage.Library]{
		secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema(), works.Schema(), roots.Schema(), mediaservers.Schema(),
	}
}

// addServer saves a Jellyfin server named name that reaching reaches at
// address, with an API key of key- and its name.
func addServer(t *testing.T, lib *storage.DB[storage.Library], sealer *secret.Sealer, name string, reaching role.NodeID, address string, fallback *string) mslink.ServerID {
	t.Helper()
	id := mslink.ServerID{UUID: storage.NewUUID()}
	sealed, err := mediaservers.SealCredential(sealer, id, mslink.Credential{APIKey: "key-" + name})
	if err != nil {
		t.Fatal(err)
	}
	must(t, lib, `INSERT INTO mediaservers_server (id, kind, name, address, fallback_address, identity, version,
		monitor_delay_ms, reaching_node_id, device_id, secrets_sealed, created_at, updated_at)
		VALUES (?, 'jellyfin', ?, ?, ?, ?, '10.10.7', 60000, ?, ?, ?, 0, 0)`,
		id, name, address, fallback, "id-"+name, reaching, "device-"+name, sealed)
	return id
}

func addNode(t *testing.T, lib *storage.DB[storage.Library], name string) role.NodeID {
	t.Helper()
	id := role.NodeID{UUID: storage.NewUUID()}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, id, name)
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

type callers struct {
	byKey map[link.Key]role.NodeID
	self  role.NodeID
}

func (c callers) Caller(ctx context.Context) (role.NodeID, bool) {
	k, _ := rpc.CallerOf(ctx)
	n, ok := c.byKey[k]
	return n, ok
}

func (c callers) Self() role.NodeID { return c.self }

type noRoster struct{}

func (noRoster) Key(role.NodeID) (link.Key, bool)   { return link.Key{}, false }
func (noRoster) Serves(role.Name, role.NodeID) bool { return false }

// TestMediaServerSecretsAnswerOnlyTheReachingNode fetches the credentials
// of Den, which seedbox reaches, and Loft, which home reaches, as each
// Node and as a key the roster doesn't know: each server answers only its
// reaching Node, and only seedbox's fetch is recorded, at the settings
// version it got.
func TestMediaServerSecretsAnswerOnlyTheReachingNode(t *testing.T) {
	lib, sealer := storagetest.Library(t, schemas()...), secrettest.Sealer(t)
	home, seedbox, crate := addNode(t, lib, "home"), addNode(t, lib, "seedbox"), addNode(t, lib, "crate")
	den := addServer(t, lib, sealer, "Den", seedbox, "http://127.0.0.1:8096", nil)
	loft := addServer(t, lib, sealer, "Loft", home, "http://192.168.1.30:8096", nil)
	keys := map[string]link.Key{"home": {1}, "seedbox": {2}, "crate": {3}, "stranger": {4}}
	roster := callers{byKey: map[link.Key]role.NodeID{keys["home"]: home, keys["seedbox"]: seedbox, keys["crate"]: crate}, self: home}
	fetch := func(as string, id mslink.ServerID) (*msv1.MediaServerSecretsResponse, error) {
		d := rpc.NewDispatch(nil, keys[as], keys[as], func(role.Name) bool { return true })
		mediaserverv1connect.MediaServerControlService.Mount(rpc.NewCluster(d, noRoster{}), mediaservers.ControlService{Library: lib, Sealer: sealer, Nodes: roster})
		return mediaserverv1connect.MediaServerControlService.Call(d).MediaServerSecrets(t.Context(), &msv1.MediaServerSecretsRequest{ServerId: id.Bytes()})
	}

	must(t, lib, "UPDATE mediaservers_server SET settings_version = 3 WHERE id = ?", den)
	for _, c := range []struct {
		as   string
		id   mslink.ServerID
		want *msv1.MediaServerSecretsResponse
	}{
		{"seedbox", den, &msv1.MediaServerSecretsResponse{Credential: &msv1.Credential{ApiKey: "key-Den"}, SettingsVersion: 3}},
		{"home", loft, &msv1.MediaServerSecretsResponse{Credential: &msv1.Credential{ApiKey: "key-Loft"}, SettingsVersion: 1}},
		{"home", den, nil},
		{"crate", den, nil},
		{"stranger", den, nil},
		{"seedbox", loft, nil},
		{"seedbox", mslink.ServerID{UUID: storage.NewUUID()}, nil},
	} {
		res, err := fetch(c.as, c.id)
		if c.want == nil {
			if rpc.CodeOf(err) != rpc.CodeNotFound {
				t.Errorf("%s fetched server %s: %v, %v; want NotFound", c.as, c.id, res, err)
			}
			continue
		}
		if err != nil || !proto.Equal(res, c.want) {
			t.Errorf("%s fetched server %s: %v, %v; want %v", c.as, c.id, res, err, c.want)
		}
	}

	var fetched string
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, `SELECT group_concat(s.name || ' by ' || n.name || ' at ' || f.settings_version, ', ')
			FROM mediaservers_credential_fetch f JOIN mediaservers_server s ON s.id = f.server_id
			JOIN nodes_node n ON n.id = f.node_id`).Scan(&fetched)
	})
	if err != nil {
		t.Fatal(err)
	}
	if fetched != "Den by seedbox at 3" {
		t.Errorf("the recorded fetches are %q, want only Den by seedbox at 3", fetched)
	}
}
