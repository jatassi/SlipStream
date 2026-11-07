package downloadclients_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	tc "github.com/jatassi/slipstream/internal/torrent/clients"
)

func transmission(t *testing.T, node role.NodeID, name, address string) dc.Imported {
	t.Helper()
	a, err := tc.ParseAddress(address, tc.Transmission)
	if err != nil {
		t.Fatal(err)
	}
	l, err := tc.ParseLabel("tv-sonarr", tc.Transmission)
	if err != nil {
		t.Fatal(err)
	}
	return dc.Imported{
		Node: node, Kind: tc.Transmission, Name: name, Address: a, Username: "jackson",
		Secrets: tc.Secrets{Password: "hunter2"}, Label: l, ImportDelay: 5 * time.Minute,
	}
}

func importClient(t *testing.T, lib library, sealer *secret.Sealer, c dc.Imported) (dc.ClientID, error) {
	t.Helper()
	var id dc.ClientID
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		id, err = dc.Import(ctx, tx, sealer, c, storage.Now())
		return err
	})
	return id, err
}

func mustImport(t *testing.T, lib library, sealer *secret.Sealer, c dc.Imported) dc.ClientID {
	t.Helper()
	id, err := importClient(t, lib, sealer, c)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func externalOf(t *testing.T, lib library, id dc.ClientID) (dc.External, error) {
	t.Helper()
	var e dc.External
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		e, err = dc.ExternalOf(ctx, r, id)
		return err
	})
	return e, err
}

func TestImportSavesAClientWithNoTest(t *testing.T) {
	lib, sealer := open(t), secrettest.Sealer(t)
	node := addNode(t, lib, "Media server")
	id := mustImport(t, lib, sealer, transmission(t, node, "Transmission", "http://192.168.1.20:9091"))

	e, err := externalOf(t, lib, id)
	if err != nil {
		t.Fatal(err)
	}
	if e.Node != node || e.Kind != tc.Transmission || e.Name != "Transmission" || e.Address.String() != "http://192.168.1.20:9091/transmission/rpc" ||
		e.Username != "jackson" || e.Label.String() != "tv-sonarr" || e.Folder != "" || len(e.Folders.Pairs()) != 0 || e.Daemon != nil {
		t.Errorf("the imported client reads %+v, want Transmission on Media server at http://192.168.1.20:9091/transmission/rpc, "+
			"jackson, label tv-sonarr, the client's default folder, no Client folders and no daemon", e)
	}
	if !e.On || e.Version != 1 || e.ImportDelay != 5*time.Minute || e.CheckEvery != dc.CheckEvery.Default {
		t.Errorf("the imported client is on=%t at version %d with Import delay %s and Check every %s, want on at version 1, 5m and the default %s",
			e.On, e.Version, e.ImportDelay, e.CheckEvery, dc.CheckEvery.Default)
	}
	var polled int
	var sealed []byte
	err = lib.QueryRowContext(t.Context(), "SELECT poll_state IS NOT NULL, secrets_sealed FROM downloadclients_client WHERE id = ?", id).Scan(&polled, &sealed)
	if err != nil {
		t.Fatal(err)
	}
	if polled != 0 {
		t.Error("the imported client has a poll, want none until its Node's first")
	}
	var s secret.Sealed
	if err := s.Scan(sealed); err != nil {
		t.Fatal(err)
	}
	plain, err := sealer.Open(s, secret.NewColumn("downloadclients_client.secrets_sealed").Row(id.String()))
	if err != nil {
		t.Fatal(err)
	}
	var opened struct{ Password string }
	if err := json.Unmarshal(plain, &opened); err != nil || opened.Password != "hunter2" {
		t.Errorf("the sealed secrets open to %s (%v), want the password hunter2", plain, err)
	}
}

func TestImportRefusesATakenNameOrAddress(t *testing.T) {
	lib, sealer := open(t), secrettest.Sealer(t)
	node := addNode(t, lib, "Media server")
	first := mustImport(t, lib, sealer, transmission(t, node, "Transmission", "http://192.168.1.20:9091"))
	for _, c := range []struct {
		name, address string
		err           error
	}{
		{"Transmission", "http://192.168.1.21:9091", dc.ErrNameTaken},
		{"Transmission 2", "HTTP://192.168.1.20:9091/transmission/rpc/", dc.ErrAddressTaken},
	} {
		if _, err := importClient(t, lib, sealer, transmission(t, node, c.name, c.address)); !errors.Is(err, c.err) {
			t.Errorf("importing %s at %s = %v, want %v", c.name, c.address, err, c.err)
		}
	}

	exec(t, lib, "UPDATE downloadclients_client SET deleted_at = ?, secrets_sealed = NULL WHERE id = ?", storage.Now(), first)
	if _, err := externalOf(t, lib, first); !errors.Is(err, dc.ErrNoClient) {
		t.Errorf("reading a deleted client = %v, want ErrNoClient", err)
	}
	mustImport(t, lib, sealer, transmission(t, node, "Transmission", "http://192.168.1.20:9091"))
}
