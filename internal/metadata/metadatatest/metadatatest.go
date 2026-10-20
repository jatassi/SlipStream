// Package metadatatest serves metadata in process, for tests that open a
// Content Type's halves without a Node.
package metadatatest

import (
	"math/rand/v2"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Serve is the Controller's metadata with one Node, itself, holding the
// Metadata Role: every call reaches sources through a Holder on the
// in-process path a Controller calls itself on.
func Serve(sources map[works.TypeID]metadata.Source) *metadata.Metadata {
	a := alone{self: role.NodeID{UUID: storage.NewUUID()}}
	d, cluster := rpc.Alone(a.self, func(role.Name) bool { return true })
	metadatav1connect.MetadataService.Mount(d, metadata.NewHolder(metadata.HolderConfig{Types: sources}))
	return metadata.New(metadata.Config{
		Roster: a,
		Service: func(node role.NodeID) metadatav1connect.MetadataServiceHandler {
			return metadatav1connect.MetadataService.On(cluster, node)
		},
		Rand: rand.New(generation{}), //nolint:gosec // a fixed draw the test replays
	})
}

type alone struct{ self role.NodeID }

func (a alone) Self() role.NodeID { return a.self }

func (a alone) Pick(role.Name, role.NodeID) (role.NodeID, bool) { return a.self, true }

func (alone) PickOther(role.Name, role.NodeID) (role.NodeID, bool) { return role.NodeID{}, false }

func (alone) Why(role.Name) string { return "" }

// generation draws the same cache generation every run.
type generation struct{}

func (generation) Uint64() uint64 { return 1 }
