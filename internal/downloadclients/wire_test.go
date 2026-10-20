package downloadclients_test

import (
	"context"
	"slices"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/link"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// readers records what Clients tells the Downloading Control and the
// Admin's event stream.
type readers struct {
	touched   []role.NodeID
	published []dc.Changed
}

func (r *readers) touch(nodes ...role.NodeID) { r.touched = append(r.touched, nodes...) }

func (r *readers) Publish(_ context.Context, c dc.Changed) { r.published = append(r.published, c) }

func closed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func applyReport(t *testing.T, lib library, c *dc.Clients, node role.NodeID, rep *dlv1.Report) {
	t.Helper()
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return c.Apply(ctx, tx, node, rep) })
}

func TestAppliedReportsReachTheirReadersOnceCommitted(t *testing.T) {
	lib := open(t)
	a := addNode(t, lib, "seedbox")
	m, raw := dotTorrent(t, "Orchard")
	id, _ := mustAttach(t, lib, dc.Want{Node: a, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	r := &readers{}
	c := dc.New(dc.Config{Library: lib, Touch: r.touch, Changed: r})

	moved := c.Moved()
	applyReport(t, lib, c, a, &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{
		{Id: id.Bytes(), AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Done: 500, Wanted: 1024},
	}}}})
	if closed(moved) || len(r.published) != 0 {
		t.Fatalf("readers heard of a report before Applied: moved %v, published %v", closed(moved), r.published)
	}
	c.Applied(a, "probe")
	if closed(moved) || len(r.published) != 0 {
		t.Fatalf("readers heard of another Role's results: moved %v, published %v", closed(moved), r.published)
	}
	c.Applied(a, string(role.Downloading))
	if !closed(moved) || !slices.EqualFunc(r.published, []dc.Changed{{IDs: []string{id.String()}}}, func(a, b dc.Changed) bool { return slices.Equal(a.IDs, b.IDs) }) {
		t.Fatalf("once applied: moved %v, published %v; want moved and %s published", closed(moved), r.published, id)
	}
	if got := outcomeOf(t, lib, id).Progress; got == nil || *got != (dc.Progress{Done: 500, Size: 1024}) {
		t.Errorf("progress = %v, want 500 of 1024", got)
	}
	if len(r.touched) != 0 {
		t.Errorf("a report touched %v, want no intent changed", r.touched)
	}

	exec(t, lib, "UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = ?, delete_files = 1 WHERE id = ?", storage.Now(), dc.ByAdmin, id)
	var in *dlv1.Intent
	err := lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		in, err = dc.Intent(ctx, rd, a)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	discard := &dlv1.Discard{Id: id.Bytes(), InfoHash: m.InfoHash[:], DeleteFiles: true}
	if len(in.GetTorrents()) != 0 || len(in.GetDiscards()) != 1 || !proto.Equal(in.GetDiscards()[0], discard) {
		t.Fatalf("the intent of a Removing Torrent is %v, want only its discard %v", in, discard)
	}

	parted := &dlv1.Report{Kind: &dlv1.Report_Parted{Parted: &dlv1.Parted{Id: id.Bytes(), Freed: 1024}}}
	for range 2 {
		applyReport(t, lib, c, a, parted)
		c.Applied(a, string(role.Downloading))
	}
	if !slices.Equal(r.touched, []role.NodeID{a}) {
		t.Errorf("confirming the discard twice touched %v, want the Node's intent once", r.touched)
	}
	if got := intentOf(t, lib, a); len(got) != 0 {
		t.Errorf("the intent lists %v once the discard parted, want nothing", got)
	}
}

// callers maps a caller's link key to its Node, as the Controller's side
// of the link does.
type callers map[link.Key]role.NodeID

func (c callers) Caller(ctx context.Context) (role.NodeID, bool) {
	k, _ := rpc.CallerOf(ctx)
	n, ok := c[k]
	return n, ok
}

type types struct{}

func (types) Describe(t works.TypeID) (works.Description, bool) {
	return contenttypetest.Description, t == contenttypetest.ID
}

type noRoster struct{}

func (noRoster) Key(role.NodeID) (link.Key, bool)   { return link.Key{}, false }
func (noRoster) Serves(role.Name, role.NodeID) bool { return false }

func TestTheTorrentServiceAnswersOnlyTheTorrentsOwnNode(t *testing.T) {
	lib := open(t)
	a, b := addNode(t, lib, "home"), addNode(t, lib, "seedbox")
	m, raw := dotTorrent(t, "Orchard")
	id, _ := mustAttach(t, lib, dc.Want{Node: a, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	applyReport(t, lib, clients(), a, &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{
		{Id: id.Bytes(), AtUnixMs: reportedAt.UnixMilli(), SaveFolder: "/data/Movies/Orchard", Totals: &dlv1.TotalsRow{Id: id.Bytes(), Uploaded: 100, Downloaded: 1024, ActiveSeconds: 60}},
	}}}})

	key := link.Key{7}
	nodes := callers{key: a}
	d := rpc.NewDispatch(nil, key, key, func(role.Name) bool { return true })
	downloadingv1connect.DownloadingControlService.Mount(rpc.NewCluster(d, noRoster{}), dc.TorrentService{Library: lib, Nodes: nodes, Types: types{}})
	svc := downloadingv1connect.DownloadingControlService.Call(d)
	fetch := func() (*dlv1.TorrentResponse, error) {
		return svc.Torrent(t.Context(), &dlv1.TorrentRequest{Id: id.Bytes()})
	}

	res, err := fetch()
	want := &dlv1.TorrentResponse{
		Metainfo: raw, Baseline: &dlv1.TotalsRow{Id: id.Bytes(), Uploaded: 100, Downloaded: 1024, ActiveSeconds: 60},
		SaveFolder: "/data/Movies/Orchard", ContentType: contenttypetest.Description.Name,
	}
	if err != nil || !proto.Equal(res, want) {
		t.Fatalf("the Torrent's own Node fetched %v, %v; want %v", res, err, want)
	}

	nodes[key] = b
	if _, err := fetch(); rpc.CodeOf(err) != rpc.CodeNotFound {
		t.Errorf("another Node's fetch = %v, want NotFound", err)
	}
	delete(nodes, key)
	if _, err := fetch(); rpc.CodeOf(err) != rpc.CodeNotFound {
		t.Errorf("an unknown caller's fetch = %v, want NotFound", err)
	}
	nodes[key] = a
	removeTorrent(t, lib, id, dc.ByAdmin)
	if _, err := fetch(); rpc.CodeOf(err) != rpc.CodeNotFound {
		t.Errorf("a fetch of a removed Torrent = %v, want NotFound", err)
	}
}
