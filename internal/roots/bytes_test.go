package roots_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// payload is a Node's one finished download, whatever its torrent.
type payload string

func (p payload) Path(bencode.InfoHash, int) (string, error) { return string(p), nil }

func closed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestWatchClosesWhenARootsReportMoves(t *testing.T) {
	r := rig(t, false)
	r.run(t)
	watch := r.svc.Watch()
	r.arrive(t, r.report(storagev1.RootState_ROOT_STATE_UNMOUNTED, r.root.Place))
	if !closed(watch) {
		t.Fatal("the watch is open after the root's first report")
	}
	watch = r.svc.Watch()
	r.storage.send(r.node, r.report(storagev1.RootState_ROOT_STATE_UNMOUNTED, r.root.Place))
	r.arrived.Note(r.node, string(role.Storage))
	r.arrive(t, r.report(storagev1.RootState_ROOT_STATE_READY, r.root.Place))
	if !closed(watch) {
		t.Fatal("the watch is open after the unmounted root reported ready")
	}
	if closed(r.svc.Watch()) {
		t.Fatal("a fresh watch is closed before any report moved")
	}
}

func TestARootOnAnOfflineOrWrittenOffNodeIsUnreachable(t *testing.T) {
	reachable := func(t *testing.T, r *storageRig) error {
		t.Helper()
		var out error
		err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
			out = r.svc.Reachable(ctx, rd, r.root)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	r := rig(t, false)
	r.storage.send(r.node, r.report(storagev1.RootState_ROOT_STATE_UNMOUNTED, r.root.Place))
	if err := reachable(t, r); err != nil {
		t.Fatalf("an unmounted root on a Node in step is %v, want reachable", err)
	}

	away := rig(t, true)
	var unreachable *roots.UnreachableError
	if err := reachable(t, away); !errors.As(err, &unreachable) || unreachable.State != roots.Offline {
		t.Fatalf("a root on an offline Node is %v, want unreachable offline", err)
	}

	err := r.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'Media server', 0)", r.node); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO nodes_forfeit (node_id, role, declared_at) VALUES (?, ?, 0)", r.node, role.Storage)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	err = reachable(t, r)
	if !errors.As(err, &unreachable) || unreachable.State != roots.WrittenOff || err.Error() != "Media server's Storage was written off, so the library root there takes no new files." {
		t.Fatalf("a root on a written-off Node is %v, want unreachable written off, naming the Node", err)
	}
}

func TestBytesGoThroughTheRootsNode(t *testing.T) {
	lib := storagetest.Library(t, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), grabs.Schema(), files.Schema())
	node := role.NodeID{UUID: storage.NewUUID()}
	folder := t.TempDir()
	root := rootstest.Root(t, lib, node, alpha, folder)
	download := filepath.Join(t.TempDir(), "orchard.mkv")
	if err := os.WriteFile(download, []byte("orchard"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := rootstest.Held(t, lib, node, payload(download), alpha)
	ctx := t.Context()
	rel := func(p string) storing.Rel {
		t.Helper()
		r, err := storing.ParseRel(p)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	if err := os.MkdirAll(filepath.Join(folder, storing.StagingDir), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, storing.StagingDir, "g.part"), []byte("film"), 0o600); err != nil {
		t.Fatal(err)
	}
	held := rel(storing.HoldingDir + "/g.mkv")
	if size, err := svc.Move(ctx, root.ID, rel(storing.StagingDir+"/g.part"), held); err != nil || size != 4 {
		t.Fatalf("Move from staging = %d, %v; want the 4 bytes", size, err)
	}
	names, err := svc.ListHeld(ctx, root.ID, storing.Holding)
	if err != nil || !slices.Equal(names, []string{"g.mkv"}) {
		t.Fatalf("ListHeld(holding) = %v, %v; want the moved file", names, err)
	}
	if err := svc.Remove(ctx, root.ID, held); err != nil {
		t.Fatalf("Remove = %v", err)
	}
	if err := svc.Remove(ctx, root.ID, held); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Remove of a file already gone = %v, want fs.ErrNotExist", err)
	}
	placed := rel("Orchard/Orchard.mkv")
	if size, err := svc.Place(ctx, root.ID, placed, storing.Payload{}); err != nil || size != 7 {
		t.Fatalf("Place = %d, %v; want the download's 7 bytes", size, err)
	}
	if b, err := os.ReadFile(filepath.Join(folder, "Orchard", "Orchard.mkv")); err != nil || string(b) != "orchard" {
		t.Fatalf("the placed file holds %q, %v; want the download", b, err)
	}
	if _, err := svc.Move(ctx, roots.ID{UUID: storage.NewUUID()}, held, held); !errors.Is(err, roots.ErrNotFound) {
		t.Fatalf("Move in no root = %v, want ErrNotFound", err)
	}
}

func TestAWriteToARootItsNodeDoesntHoldIsNotReady(t *testing.T) {
	lib := storagetest.Library(t, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), grabs.Schema(), files.Schema())
	svc, err := roots.Open(t.Context(), rootstest.Config(t, lib, alpha))
	if err != nil {
		t.Fatal(err)
	}
	v, err := svc.Create(t.Context(), identity.Admin{}, roots.NewRoot{ContentType: alpha, Node: role.NodeID{UUID: storage.NewUUID()}, Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	held, _ := storing.ParseRel(storing.HoldingDir + "/g.mkv")
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var notReady *storing.NotReady
	if _, err := svc.Move(ctx, v.ID, held, held); !errors.As(err, &notReady) || !notReady.Unknown {
		t.Fatalf("Move into a root its Node doesn't hold yet = %v, want NotReady unknown", err)
	}
}
