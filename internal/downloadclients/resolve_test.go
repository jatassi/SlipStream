package downloadclients_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/bencode"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/magnet"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

type resolver struct {
	downloadingv1connect.UnimplementedDownloadingServiceHandler
	res   *dlv1.ResolveResponse
	asked *dlv1.ResolveRequest
}

func (r *resolver) Resolve(_ context.Context, req *dlv1.ResolveRequest) (*dlv1.ResolveResponse, error) {
	r.asked = req
	return r.res, nil
}

func TestResolveAsksTheNodeAndReadsEachAnswer(t *testing.T) {
	since := time.UnixMilli(1_700_000_000_000).UTC()
	link := magnet.Link{InfoHash: bencode.InfoHash{0xab}, Name: "Orchard", Trackers: []string{"https://tracker.test/announce/secret"}}
	endsAt := since.Add(30 * time.Minute)
	for _, row := range []struct {
		name string
		res  *dlv1.ResolveResponse
		want dc.Resolution
		is   error
	}{
		{name: "the info dictionary", res: &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_Info{Info: []byte("d4:name7:Orcharde")}}, want: dc.Resolved{Info: []byte("d4:name7:Orcharde")}},
		{
			name: "a search that runs",
			res:  &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_Searching{Searching: &dlv1.Searching{SinceUnixMs: since.UnixMilli(), Peers: 3}}},
			want: dc.Searching{Since: since, Peers: 3},
		},
		{
			name: "a search that timed out",
			res: &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_Unresolved{Unresolved: &dlv1.Unresolved{
				Reason: dlv1.UnresolvedReason_UNRESOLVED_REASON_TIMED_OUT, MostPeers: 2, TrackerMessage: "unregistered torrent",
			}}},
			want: dc.Unresolved{Reason: dc.TimedOut, MostPeers: 2, TrackerMessage: "unregistered torrent"},
		},
		{
			name: "no peer source",
			res:  &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_Unresolved{Unresolved: &dlv1.Unresolved{Reason: dlv1.UnresolvedReason_UNRESOLVED_REASON_NO_PEER_SOURCE}}},
			want: dc.Unresolved{Reason: dc.NoPeerSource},
		},
		{
			name: "an engine error",
			res: &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_Unresolved{Unresolved: &dlv1.Unresolved{
				Reason: dlv1.UnresolvedReason_UNRESOLVED_REASON_ENGINE_ERROR, Error: "libtorrent error 2: no such file",
			}}},
			want: dc.Unresolved{Reason: dc.EngineError, Error: "libtorrent error 2: no such file"},
		},
		{
			name: "an info dictionary over the .torrent cap",
			res:  &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_Unresolved{Unresolved: &dlv1.Unresolved{Reason: dlv1.UnresolvedReason_UNRESOLVED_REASON_INFO_TOO_LARGE}}},
			want: dc.Unresolved{Reason: dc.InfoTooLarge},
		},
		{name: "a held torrent", res: &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_Held{Held: &dlv1.Held{}}}, is: dc.ErrHeld},
		{name: "a client taking no new torrent", res: &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_NotTaking{NotTaking: &dlv1.NotTaking{}}}, is: dc.ErrNotTaking},
	} {
		t.Run(row.name, func(t *testing.T) {
			node := &resolver{res: row.res}
			c := dc.New(dc.Config{Remote: func(role.NodeID) downloadingv1connect.DownloadingServiceHandler { return node }})
			got, err := c.Resolve(t.Context(), role.NodeID{UUID: storage.NewUUID()}, link, endsAt)
			if row.is != nil {
				if !errors.Is(err, row.is) {
					t.Fatalf("Resolve = %+v, %v; want %v", got, err, row.is)
				}
			} else if err != nil || !equalResolution(got, row.want) {
				t.Fatalf("Resolve = %+v, %v; want %+v", got, err, row.want)
			}
			want := &dlv1.ResolveRequest{InfoHash: link.InfoHash[:], Name: link.Name, Trackers: link.Trackers, EndsAtUnixMs: endsAt.UnixMilli()}
			if !proto.Equal(node.asked, want) {
				t.Errorf("the Node was asked %v, want %v", node.asked, want)
			}
		})
	}
}

func equalResolution(a, b dc.Resolution) bool {
	if ra, ok := a.(dc.Resolved); ok {
		rb, ok := b.(dc.Resolved)
		return ok && bytes.Equal(ra.Info, rb.Info)
	}
	return a == b
}

func TestKnownInfoReadsTheInfoDictionaryOfAnyTorrentOfTheHash(t *testing.T) {
	lib := open(t)
	node := addNode(t, lib, "seedbox")
	m, raw := dotTorrent(t, "Orchard")
	known := func(h bencode.InfoHash) ([]byte, bool) {
		t.Helper()
		var info []byte
		var ok bool
		err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			info, ok, err = dc.KnownInfo(ctx, r, h)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return info, ok
	}
	if _, ok := known(m.InfoHash); ok {
		t.Fatal("KnownInfo found an info dictionary before any Torrent of the hash")
	}
	id, _ := mustAttach(t, lib, dc.Want{Node: node, Metainfo: m, Raw: raw, ContentType: "movies", WrittenOff: noneWrittenOff})
	if info, ok := known(m.InfoHash); !ok || !bytes.Equal(info, m.RawInfo) {
		t.Errorf("KnownInfo of a live Torrent's hash = %d bytes, %v; want its info dictionary", len(info), ok)
	}
	removeTorrent(t, lib, id, dc.ByAdmin)
	partTorrent(t, lib, id)
	if info, ok := known(m.InfoHash); !ok || !bytes.Equal(info, m.RawInfo) {
		t.Errorf("KnownInfo of a removed Torrent's hash = %d bytes, %v; want its info dictionary", len(info), ok)
	}
}
