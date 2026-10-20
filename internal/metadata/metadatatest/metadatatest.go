// Package metadatatest serves metadata in process, for tests that open a
// Content Type's halves without a Node.
package metadatatest

import (
	"errors"
	"math/rand/v2"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// Serve is the Controller's metadata with one Node, itself, holding the
// Metadata Role: every call reaches sources through a Holder on the
// in-process path a Controller calls itself on.
func Serve(sources map[works.TypeID]metadata.Source) *metadata.Metadata {
	return metadata.New(alone(sources, &http.Client{Transport: noImages{}}, metadata.Timings{}))
}

// Minter signs the artwork tokens Artwork opens.
var Minter = artwork.NewMinter([]byte("metadatatest"))

// Artwork is Serve over lib, which holds the metadata schema, with an
// artwork store of its own, reaching image hosts through client under
// timings. It runs its background work until the test ends.
func Artwork(t testing.TB, lib *storage.DB[storage.Library], sources map[works.TypeID]metadata.Source, client *http.Client, timings metadata.Timings) *metadata.Metadata {
	t.Helper()
	cfg := alone(sources, client, timings)
	cfg.Library, cfg.Minter, cfg.Local, cfg.ArtworkDir = lib, Minter, storagetest.Local(t, artwork.Schema()), t.TempDir()
	m, err := metadata.Open(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.Run(t.Context())
	}()
	t.Cleanup(func() { <-done })
	return m
}

// alone is the Controller's metadata config with one Node, itself,
// holding the Metadata Role.
func alone(sources map[works.TypeID]metadata.Source, client *http.Client, timings metadata.Timings) metadata.Config {
	self := role.NodeID{UUID: storage.NewUUID()}
	d, cluster := rpc.Alone(self, func(role.Name) bool { return true })
	lines := metadata.NewLines(client, "SlipStream/test", timings, rand.New(generation{})) //nolint:gosec // a fixed draw the test replays
	metadatav1connect.MetadataService.Mount(d, metadata.NewHolder(metadata.HolderConfig{Types: sources, Lines: lines}))
	return metadata.Config{
		Roster: roster{self: self},
		Service: func(node role.NodeID) metadatav1connect.MetadataServiceHandler {
			return metadatav1connect.MetadataService.On(cluster, node)
		},
		Timings: timings,
		Rand:    rand.New(generation{}), //nolint:gosec // a fixed draw the test replays
	}
}

type roster struct{ self role.NodeID }

func (r roster) Self() role.NodeID { return r.self }

func (r roster) Pick(role.Name, role.NodeID) (role.NodeID, bool) { return r.self, true }

func (roster) PickOther(role.Name, role.NodeID) (role.NodeID, bool) { return role.NodeID{}, false }

func (roster) Why(role.Name) string { return "" }

// noImages answers no request: a test served here fetches no image.
type noImages struct{}

func (noImages) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, errors.New("metadatatest reaches no image host: " + r.URL.Host)
}

// generation draws the same cache generation every run.
type generation struct{}

func (generation) Uint64() uint64 { return 1 }
