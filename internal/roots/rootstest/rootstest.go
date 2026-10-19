// Package rootstest gives tests library roots.
package rootstest

import (
	"context"
	"errors"
	"iter"
	"log/slog"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1/storagev1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

type discard struct{}

func (discard) Publish(context.Context, roots.Changed) {}

type discardVolumes struct{}

func (discardVolumes) Publish(context.Context, roots.VolumesChanged) {}

// Roster is a roster where every Node holds Storage and is in step.
type Roster struct{}

func (Roster) Holds(role.Name, role.NodeID) bool { return true }
func (Roster) InStep(role.NodeID) bool           { return true }

// Unreported is Storage's control half before any Node reported.
type Unreported struct{}

func (Unreported) Touch(...role.NodeID)                       {}
func (Unreported) Facts(role.NodeID) (*storagev1.Facts, bool) { return nil, false }

// Remote reaches, on every Node, one Store on this machine whose roots may
// not overlap data and downloads.
func Remote(tb testing.TB, data, downloads string) storing.Remote {
	tb.Helper()
	disk, err := storing.NewOSDisk(data, downloads)
	if err != nil {
		tb.Fatal(err)
	}
	store := storing.NewStore(storing.Config{Disk: disk, Timings: storing.Timings{Stat: 10 * time.Second}})
	return func(role.NodeID) storagev1connect.StorageServiceHandler { return store }
}

// Config is the roots over lib for types: every Node holds Storage, none
// reported yet, and each reaches one Store on this machine whose data and
// downloads folders are fresh temp folders.
func Config(tb testing.TB, lib *storage.DB[storage.Library], types ...works.TypeID) roots.Config {
	tb.Helper()
	return roots.Config{
		Library:  lib,
		Types:    types,
		Roster:   Roster{},
		Storage:  Unreported{},
		Remote:   Remote(tb, tb.TempDir(), tb.TempDir()),
		Arrivals: roots.NewArrivals(),
		Contents: files.RootContents{},
		Inflight: grabs.RootGrabs{},
		Changed:  discard{},
		Volumes:  discardVolumes{},
		Log:      slog.New(slog.DiscardHandler),
	}
}

// Root makes path a root of t on node, the way an Admin does, so it homes
// every rootless Work of t when it is t's first.
func Root(tb testing.TB, lib *storage.DB[storage.Library], node role.NodeID, t works.TypeID, path string) roots.Root {
	tb.Helper()
	svc, err := roots.Open(tb.Context(), Config(tb, lib, t))
	if err != nil {
		tb.Fatal(err)
	}
	v, err := svc.Create(tb.Context(), identity.Admin{}, roots.NewRoot{ContentType: t, Node: node, Path: path})
	if err != nil {
		tb.Fatal(err)
	}
	return v.Root
}

// Held is the roots over lib for types whose every Node reaches one Store
// on this machine, which finds downloads in payloads, holds node's roots
// as lib records them now and has adopted each. Make the roots first.
func Held(tb testing.TB, lib *storage.DB[storage.Library], node role.NodeID, payloads storing.Payloads, types ...works.TypeID) *roots.Roots {
	tb.Helper()
	return held(tb, lib, node, payloads, false, types)
}

// Draining is Held while node drains Storage.
func Draining(tb testing.TB, lib *storage.DB[storage.Library], node role.NodeID, payloads storing.Payloads, types ...works.TypeID) *roots.Roots {
	tb.Helper()
	return held(tb, lib, node, payloads, true, types)
}

func held(tb testing.TB, lib *storage.DB[storage.Library], node role.NodeID, payloads storing.Payloads, draining bool, types []works.TypeID) *roots.Roots {
	tb.Helper()
	disk, err := storing.NewOSDisk(tb.TempDir(), tb.TempDir())
	if err != nil {
		tb.Fatal(err)
	}
	store := storing.NewStore(storing.Config{Disk: disk, Payloads: payloads, Timings: storing.Timings{Stat: 10 * time.Second}})
	var intent *storagev1.Intent
	err = lib.Read(tb.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		intent, err = roots.StorageIntent(files.RootContents{})(ctx, r, node)
		return err
	})
	if err != nil {
		tb.Fatal(err)
	}
	h := &holder{intent: intent, draining: draining, facts: make(chan *storagev1.Facts, 1)}
	ctx, cancel := context.WithCancel(context.WithoutCancel(tb.Context()))
	done := make(chan error, 1)
	go func() { done <- store.Run(ctx, h) }()
	tb.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			tb.Errorf("the Store's run = %v", err)
		}
	})
	for _, rf := range (<-h.facts).GetRoots() {
		if rf.GetState() != storagev1.RootState_ROOT_STATE_READY {
			tb.Fatalf("the Store finds a root %s: %s", rf.GetState(), rf.GetDetail())
		}
	}
	cfg := Config(tb, lib, types...)
	cfg.Remote = func(role.NodeID) storagev1connect.StorageServiceHandler { return store }
	svc, err := roots.Open(tb.Context(), cfg)
	if err != nil {
		tb.Fatal(err)
	}
	return svc
}

// holder hands a Store one intent, draining or not, and keeps its latest
// facts.
type holder struct {
	intent   *storagev1.Intent
	draining bool
	facts    chan *storagev1.Facts
}

func (h *holder) Intent(ctx context.Context) iter.Seq[role.Snapshot[*storagev1.Intent]] {
	return func(yield func(role.Snapshot[*storagev1.Intent]) bool) {
		if yield(role.Snapshot[*storagev1.Intent]{Intent: h.intent, Standing: role.Live, Draining: h.draining}) {
			<-ctx.Done()
		}
	}
}

func (h *holder) SetFacts(f *storagev1.Facts) {
	select {
	case <-h.facts:
	default:
	}
	h.facts <- f
}

func (h *holder) Report(context.Context, *storage.Tx[storage.Local], *storagev1.Result) error {
	return errors.New("storage reports by facts")
}
