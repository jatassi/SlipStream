package downloadclients_test

import (
	"context"
	"slices"
	"testing"
	"time"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
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
	c := dc.New(dc.Config{Library: lib})
	o, err := c.NodeSettings(t.Context(), node)
	if err != nil {
		t.Fatal(err)
	}
	want := dc.NodeSettings{ActiveDownloads: 3, ActiveChecks: 1, Connections: 8000, UTP: true, StalledAfter: 10 * time.Minute}
	if got := o.Settings(); got != want {
		t.Errorf("settings = %+v, want %+v", got, want)
	}
	th, err := c.Thresholds(t.Context())
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

func TestNodeSettingsAreEachNodesOwnAndReachItsIntent(t *testing.T) {
	lib := open(t)
	seedbox, home := addNode(t, lib, "seedbox"), addNode(t, lib, "home")
	r, cards := &readers{}, &told{}
	c := dc.New(dc.Config{Library: lib, Touch: r.touch, Changed: r, CardsChanged: cards})
	limit := int64(512 * 1024)
	off := false
	set := func(ch dc.NodeChange) dc.NodeSettings {
		t.Helper()
		o, err := c.SetNodeSettings(t.Context(), seedbox, ch)
		if err != nil {
			t.Fatal(err)
		}
		return o.Settings()
	}
	read := func(node role.NodeID) dc.NodeSettings {
		t.Helper()
		o, err := c.NodeSettings(t.Context(), node)
		if err != nil {
			t.Fatal(err)
		}
		return o.Settings()
	}

	if got := set(dc.NodeChange{UploadLimit: parsed(dc.UploadLimit.Parse(&limit)), UTP: new(dc.UTP.Parse(&off))}); got != read(seedbox) || got.UploadLimit != limit || got.UTP || got.ActiveDownloads != 3 {
		t.Errorf("seedbox = %+v, want a 512 KB/s upload limit, uTP off and the rest default", got)
	}
	if got := read(home); got.UploadLimit != 0 || !got.UTP {
		t.Errorf("home = %+v, want defaults", got)
	}
	if !slices.Equal(r.touched, []role.NodeID{seedbox}) {
		t.Errorf("touched %v, want seedbox's intent alone", r.touched)
	}
	if want := [][]string{{seedbox.String()}}; !slices.EqualFunc(cards.nodes, want, slices.Equal) {
		t.Errorf("download-clients.changed named %v, want seedbox once", cards.nodes)
	}
	var in *dlv1.Intent
	err := lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		in, err = dc.Intent(ctx, rd, seedbox)
		return err
	})
	if err != nil || in.GetSettings().GetUploadLimit() != limit || in.GetSettings().GetUtp() {
		t.Errorf("seedbox's intent carries %v, %v; want its new settings", in.GetSettings(), err)
	}

	if got := set(dc.NodeChange{UploadLimit: parsed(dc.UploadLimit.Parse(nil))}); got.UploadLimit != 0 || got.UTP {
		t.Errorf("after a reset seedbox = %+v, want the upload limit default and uTP still off", got)
	}
}

// told is the download-clients.changed events published.
type told struct{ nodes [][]string }

func (t *told) Publish(_ context.Context, c dc.ClientsChanged) { t.nodes = append(t.nodes, c.NodeIDs) }

func TestThresholdsKeepWhatIsLeftAndReachEveryDownloadingNode(t *testing.T) {
	lib := open(t)
	seedbox, home := addNode(t, lib, "seedbox"), addNode(t, lib, "home")
	r, cards := &readers{}, &told{}
	c := dc.New(dc.Config{Library: lib, Touch: r.touch, Changed: r, CardsChanged: cards, Roster: &roster{serving: []role.NodeID{seedbox, home}}})
	gap := int64(3600)
	o, err := c.SetThresholds(t.Context(), dc.ThresholdChange{UnregisteredGap: parsed(dc.UnregisteredGap.Parse(&gap))})
	if err != nil {
		t.Fatal(err)
	}
	if got := o.Thresholds(); got != (dc.Thresholds{FaultStall: 12 * time.Hour, UnregisteredGap: time.Hour}) {
		t.Errorf("thresholds = %+v, want 12h and 1h", got)
	}
	if read, err := c.Thresholds(t.Context()); err != nil || read.Thresholds() != o.Thresholds() {
		t.Errorf("read back %+v, %v; want %+v", read, err, o)
	}
	if !slices.Equal(r.touched, []role.NodeID{seedbox, home}) {
		t.Errorf("touched %v, want both downloading Nodes' intents", r.touched)
	}
	if want := [][]string{{seedbox.String(), home.String()}}; !slices.EqualFunc(cards.nodes, want, slices.Equal) {
		t.Errorf("download-clients.changed named %v, want both Nodes once", cards.nodes)
	}
	var in *dlv1.Intent
	err = lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		in, err = dc.Intent(ctx, rd, home)
		return err
	})
	if err != nil || in.GetThresholds().GetUnregisteredGapSeconds() != 3600 || in.GetThresholds().GetFaultStallSeconds() != 12*3600 {
		t.Errorf("home's intent carries %v, %v; want the new gap and the default stall", in.GetThresholds(), err)
	}
}
