// Package rootstest gives tests a library root.
package rootstest

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type discard struct{}

func (discard) Publish(context.Context, roots.Changed) {}

// Root makes path a root of t on self, the way an Admin does, so it homes
// every rootless Work of t when it is t's first.
func Root(tb testing.TB, lib *storage.DB[storage.Library], self role.NodeID, t works.TypeID, path string) roots.Root {
	tb.Helper()
	svc, err := roots.Open(tb.Context(), roots.Config{Library: lib, Types: []works.TypeID{t}, Self: self, Contents: files.RootContents{}, Changed: discard{}})
	if err != nil {
		tb.Fatal(err)
	}
	v, err := svc.Create(tb.Context(), identity.Admin{}, roots.NewRoot{ContentType: t, Path: path})
	if err != nil {
		tb.Fatal(err)
	}
	return v.Root
}
