// Package rootstest gives tests library roots.
package rootstest

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/files"
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
		Changed:  discard{},
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
