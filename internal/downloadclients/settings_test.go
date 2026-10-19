package downloadclients_test

import (
	"context"
	"testing"
	"time"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type library = *storage.DB[storage.Library]

func open(t *testing.T) library {
	t.Helper()
	return storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), dc.Schema())
}

func addNode(t *testing.T, lib library, name string) role.NodeID {
	t.Helper()
	id := role.NodeID{UUID: storage.NewUUID()}
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), ?, 0)", id, name)
		return err
	})
	return id
}

func write(t *testing.T, lib library, fn func(context.Context, *storage.Tx[storage.Library]) error) {
	t.Helper()
	if err := lib.Write(t.Context(), fn); err != nil {
		t.Fatal(err)
	}
}

func TestEveryClientSettingIsDeclaredSoundly(t *testing.T) {
	for _, d := range []interface{ Check() error }{
		dc.DownloadLimit, dc.UploadLimit, dc.ActiveDownloads, dc.ActiveChecks, dc.Connections,
		dc.StalledAfter, dc.FaultStall, dc.UnregisteredGap,
	} {
		if err := d.Check(); err != nil {
			t.Error(err)
		}
	}
}

func TestNodeSettingsReadDefaultsWhenAbsent(t *testing.T) {
	lib := open(t)
	node := addNode(t, lib, "seedbox")
	o, err := dc.NodeOverridesOf(t.Context(), lib, node)
	if err != nil {
		t.Fatal(err)
	}
	want := dc.NodeSettings{ActiveDownloads: 3, ActiveChecks: 1, Connections: 8000, UTP: true, StalledAfter: 10 * time.Minute}
	if got := o.Settings(); got != want {
		t.Errorf("settings = %+v, want %+v", got, want)
	}
	th, err := dc.ThresholdOverridesOf(t.Context(), lib)
	if err != nil {
		t.Fatal(err)
	}
	if got := th.Thresholds(); got != (dc.Thresholds{FaultStall: 12 * time.Hour, UnregisteredGap: 30 * time.Minute}) {
		t.Errorf("thresholds = %+v, want 12h and 30m", got)
	}
}

func parsed[T any](s setting.Stored[T], err error) *setting.Stored[T] {
	if err != nil {
		panic(err)
	}
	return &s
}

func TestNodeSettingsAreEachNodesOwn(t *testing.T) {
	lib := open(t)
	seedbox, home := addNode(t, lib, "seedbox"), addNode(t, lib, "home")
	limit := int64(512 * 1024)
	off := false
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return dc.SetNodeSettings(ctx, tx, seedbox, dc.NodeChange{
			UploadLimit: parsed(dc.UploadLimit.Parse(&limit)),
			UTP:         new(dc.UTP.Parse(&off)),
		})
	})
	read := func(node role.NodeID) dc.NodeSettings {
		o, err := dc.NodeOverridesOf(t.Context(), lib, node)
		if err != nil {
			t.Fatal(err)
		}
		return o.Settings()
	}
	if got := read(seedbox); got.UploadLimit != limit || got.UTP || got.ActiveDownloads != 3 {
		t.Errorf("seedbox = %+v, want a 512 KB/s upload limit, uTP off and the rest default", got)
	}
	if got := read(home); got.UploadLimit != 0 || !got.UTP {
		t.Errorf("home = %+v, want defaults", got)
	}

	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return dc.SetNodeSettings(ctx, tx, seedbox, dc.NodeChange{UploadLimit: parsed(dc.UploadLimit.Parse(nil))})
	})
	if got := read(seedbox); got.UploadLimit != 0 || got.UTP {
		t.Errorf("after a reset seedbox = %+v, want the upload limit default and uTP still off", got)
	}
}

func TestThresholdsKeepWhatIsLeft(t *testing.T) {
	lib := open(t)
	gap := int64(3600)
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return dc.SetThresholds(ctx, tx, dc.ThresholdChange{UnregisteredGap: parsed(dc.UnregisteredGap.Parse(&gap))})
	})
	o, err := dc.ThresholdOverridesOf(t.Context(), lib)
	if err != nil {
		t.Fatal(err)
	}
	if got := o.Thresholds(); got != (dc.Thresholds{FaultStall: 12 * time.Hour, UnregisteredGap: time.Hour}) {
		t.Errorf("thresholds = %+v, want 12h and 1h", got)
	}
}
