package grabs_test

import (
	"context"
	"crypto/sha1" //nolint:gosec // the v1 info hash
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

const hold = 20 * time.Second

// asked is one Resolve call a Downloading Node received.
type asked struct {
	node   role.NodeID
	link   magnet.Link
	endsAt time.Time
	at     time.Time
}

// swarm is the Downloading Role's Controller half as a Grab of a magnet
// link calls it: each Resolve gets the next of answers, the last one
// again once they run out. A Searching or Resolved answer comes after the
// hold, as the Node's long poll does.
type swarm struct {
	noDownloads
	mu      sync.Mutex
	answers []answer
	calls   []asked
}

type answer struct {
	res downloadclients.Resolution
	err error
}

func (s *swarm) Resolve(ctx context.Context, node role.NodeID, link magnet.Link, endsAt time.Time) (downloadclients.Resolution, error) {
	s.mu.Lock()
	s.calls = append(s.calls, asked{node: node, link: link, endsAt: endsAt, at: time.Now()})
	a := s.answers[min(len(s.calls), len(s.answers))-1]
	s.mu.Unlock()
	switch a.res.(type) {
	case downloadclients.Searching, downloadclients.Resolved:
		select {
		case <-time.After(hold):
		case <-ctx.Done():
			return nil, fmt.Errorf("resolve: %w", ctx.Err())
		}
	}
	return a.res, a.err
}

func (s *swarm) asked() []asked {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.calls)
}

// roster is one Node holding every Role, whose next change the test
// makes.
type roster struct {
	node    role.NodeID
	mu      sync.Mutex
	changed chan struct{}
}

func (r *roster) Self() role.NodeID                                 { return r.node }
func (r *roster) Pick(role.Name, role.NodeID) (role.NodeID, bool)   { return r.node, true }
func (r *roster) Route(role.Name, role.NodeID) (role.NodeID, error) { return r.node, nil }
func (*roster) Away(role.Name, role.NodeID) (nodes.Absence, bool)   { return nodes.Absence{}, false }
func (*roster) InStep(role.NodeID) bool                             { return true }
func (*roster) WrittenOff(role.Name, role.NodeID) bool              { return false }

func (r *roster) Changed() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.changed
}

func (r *roster) change() {
	r.mu.Lock()
	defer r.mu.Unlock()
	close(r.changed)
	r.changed = make(chan struct{})
}

type published struct {
	mu  sync.Mutex
	ids []string
}

func (p *published) Publish(_ context.Context, c grabs.Changed) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ids = append(p.ids, c.GrabID)
}

func (p *published) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.ids)
}

// pipeline is a Controller holding every Role, with one Work, its library
// root and one Indexer, whose Grabs ask swarm for metadata.
type pipeline struct {
	t       *testing.T
	lib     library
	g       *grabs.Grabs
	swarm   *swarm
	roster  *roster
	changed *published
	node    role.NodeID
	work    works.WorkID
	indexer storage.UUID
}

func newPipeline(t *testing.T, answers ...answer) *pipeline {
	t.Helper()
	lib := storagetest.Library(t, schemas()...)
	p := &pipeline{
		t: t, lib: lib, swarm: &swarm{answers: answers}, changed: &published{},
		node: role.NodeID{UUID: storage.NewUUID()}, indexer: storage.NewUUID(),
	}
	p.roster = &roster{node: p.node, changed: make(chan struct{})}
	run(t, lib,
		sql(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, p.node),
		sql(`INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, p.indexer, p.node),
	)
	rootstest.Root(t, lib, p.node, contenttypetest.ID, t.TempDir())
	fx := contenttypetest.New()
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	assign := func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
		return roots.Assign(ctx, tx, w, nil)
	}
	tree, _, err := w.Add(t.Context(), identity.Admin{}, contenttypetest.ID, "w1", works.Adding{Monitor: true, Also: assign})
	if err != nil {
		t.Fatal(err)
	}
	p.work = tree.ID
	p.g = grabs.New(grabs.Config{
		Library: lib, Works: w,
		Indexers: indexers.New(indexers.Config{Library: lib, Roster: p.roster, Changed: discard[indexers.Changed]{}}),
		Roster:   p.roster, Downloads: p.swarm,
		Transfers: transfers.NewReceiver(transfers.Config{Local: storagetest.Local(t, transfers.Schema())}),
		Timings:   grabs.DefaultTimings(), Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Changed: p.changed, Log: slog.New(slog.DiscardHandler),
	})
	return p
}

func schemas() []storage.Schema[storage.Library] {
	return []storage.Schema[storage.Library]{
		secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), health.Schema(), roots.Schema(),
		indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema(),
	}
}

// grab adds a fetching Grab of link, with the Indexer's info hash hint
// when hint isn't nil.
func (p *pipeline) grab(link string, hint *bencode.InfoHash) grabs.GrabID {
	p.t.Helper()
	id := storage.NewUUID()
	var h []byte
	if hint != nil {
		h = hint[:]
	}
	run(p.t, p.lib, sql(`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size,
		release_link, release_info_hash, stage, created_at, updated_at)
		SELECT ?, work_id, id, 'any', 'search', ?, ?, 'The.Attic.1080p.WEB', 4096, ?, ?, 'fetching', ?, ? FROM works_item`,
		id, p.indexer, p.node, link, h, storage.Now(), storage.Now()))
	return grabs.GrabID{UUID: id}
}

// running runs the pipeline until the test ends.
func (p *pipeline) running() {
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error)
	go func() { done <- p.g.Run(ctx) }()
	p.t.Cleanup(func() {
		stop()
		if err := <-done; err != nil {
			p.t.Error(err)
		}
	})
	synctest.Wait()
}

func (p *pipeline) view(id grabs.GrabID) grabs.View {
	p.t.Helper()
	synctest.Wait()
	v, err := p.g.Get(p.t.Context(), identity.Admin{}, id)
	if err != nil {
		p.t.Fatal(err)
	}
	return v
}

func (p *pipeline) failed(id grabs.GrabID, code grabs.Code) grabs.View {
	p.t.Helper()
	v := p.view(id)
	if v.Stage != grabs.Failed || v.Failed.Stage != grabs.Fetching || v.Failed.Code != code {
		p.t.Fatalf("the Grab is %s with %+v, want failed at fetching with %s", v.Stage, v.Failed, code)
	}
	return v
}

// release is a torrent's info dictionary and its magnet link.
type release struct {
	info []byte
	link magnet.Link
}

func newRelease(t *testing.T, trackers ...string) release {
	t.Helper()
	info, err := bencode.Encode(map[string]any{"name": "The.Attic.1080p.WEB", "piece length": int64(16384), "pieces": "aaaaaaaaaaaaaaaaaaaa", "length": int64(4096)})
	if err != nil {
		t.Fatal(err)
	}
	return release{info: info, link: magnet.Link{InfoHash: sha1.Sum(info), Name: "The.Attic.1080p.WEB", Trackers: trackers}} //nolint:gosec // the v1 info hash
}

func searchingWith(peers int) answer {
	return answer{res: downloadclients.Searching{Since: time.Now(), Peers: peers}}
}

func TestAMagnetLinksGrabShowsItsSearchAndDownloadsTheTorrentAPeerSent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel := newRelease(t, "https://tracker.test/announce/secret")
		p := newPipeline(t, searchingWith(0), searchingWith(3), answer{res: downloadclients.Resolved{Info: rel.info}})
		id := p.grab(rel.link.String(), &rel.link.InfoHash)
		started := time.Now()
		p.running()

		if m := p.view(id).Metadata; m == nil || m.Peers != 0 || !m.EndsAt.Equal(started.Add(downloadclients.MetadataWait.Default)) {
			t.Fatalf("while the first call waits the Grab shows %+v, want no peers and an end 30 minutes from its first ask", m)
		}
		time.Sleep(2 * hold)
		if m := p.view(id).Metadata; m == nil || m.Peers != 3 {
			t.Fatalf("after the Node responded with 3 peers the Grab shows %+v", m)
		}
		changes := p.changed.count()
		time.Sleep(hold)
		v := p.view(id)
		if v.Stage != grabs.Downloading || v.Torrent == nil || v.Torrent.InfoHash != rel.link.InfoHash || v.FetchNode != p.node || v.Metadata != nil {
			t.Fatalf("once a peer sent the metadata the Grab is %s with %+v from %v showing %+v, want downloading the link's torrent fetched by its Downloading Node",
				v.Stage, v.Torrent, v.FetchNode, v.Metadata)
		}
		if p.changed.count() <= changes {
			t.Error("grabs.changed went out for neither the peers moving nor the advance")
		}
		p.askedInTurn(rel.link, started.Add(downloadclients.MetadataWait.Default), 3)
	})
}

// askedInTurn checks that the Downloading Node was asked n times for link
// until endsAt, each call right after the one before held.
func (p *pipeline) askedInTurn(link magnet.Link, endsAt time.Time, n int) {
	p.t.Helper()
	calls := p.swarm.asked()
	if len(calls) != n {
		p.t.Fatalf("the Node was asked %d times, want %d", len(calls), n)
	}
	for i, c := range calls {
		if c.node != p.node || !slices.Equal(c.link.Trackers, link.Trackers) || !c.endsAt.Equal(endsAt) {
			p.t.Errorf("call %d asked %v for %+v until %v, want the Downloading Node, the link and the first ask's deadline", i, c.node, c.link, c.endsAt)
		}
		if i > 0 && c.at.Sub(calls[i-1].at) != hold {
			p.t.Errorf("call %d came %v after the one before, want right after its hold", i, c.at.Sub(calls[i-1].at))
		}
	}
}

func TestAMagnetLinkWhoseSearchEndsFailsItsGrabAtFetching(t *testing.T) {
	for _, row := range []struct {
		name  string
		end   downloadclients.Unresolved
		code  grabs.Code
		words string
	}{
		{"timed out", downloadclients.Unresolved{Reason: downloadclients.TimedOut, MostPeers: 2, TrackerMessage: "unregistered torrent"}, grabs.NoMetadata, "The tracker responded: unregistered torrent"},
		{"no peer source", downloadclients.Unresolved{Reason: downloadclients.NoPeerSource}, grabs.NoPeerSource, "DHT is off"},
		{"engine error", downloadclients.Unresolved{Reason: downloadclients.EngineError, Error: "libtorrent error 2"}, grabs.DownloadFailed, "libtorrent error 2"},
	} {
		t.Run(row.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				rel := newRelease(t)
				p := newPipeline(t, answer{res: row.end})
				id := p.grab(rel.link.String(), nil)
				p.running()
				if v := p.failed(id, row.code); !strings.Contains(v.Failed.Message, row.words) || v.Torrent != nil {
					t.Errorf("the Grab failed with %q and Torrent %+v, want %q in it and no Torrent", v.Failed.Message, v.Torrent, row.words)
				}
			})
		})
	}
}

func TestAGrabFailsAtItsDeadlineAfterAnyRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel := newRelease(t, "https://tracker.test/announce/secret")
		p := newPipeline(t, searchingWith(1))
		run(t, p.lib, sql(`UPDATE downloadclients_settings SET metadata_wait_seconds = 60 WHERE id = 1`))
		id := p.grab(rel.link.String(), nil)
		p.running()
		time.Sleep(time.Minute + hold)
		v := p.failed(id, grabs.NoMetadata)
		if want := "No peer sent the metadata in " + setting.Words(time.Minute); v.Failed.Message != want {
			t.Errorf("the Grab failed with %q, want %q", v.Failed.Message, want)
		}

		late := p.grab(rel.link.String(), nil)
		run(t, p.lib, sql(`UPDATE grabs_grab SET metadata_started_at = ? WHERE id = ?`, storage.FromTime(time.Now().Add(-2*time.Minute)), late))
		asked := len(p.swarm.asked())
		p.roster.change()
		p.failed(late, grabs.NoMetadata)
		if got := len(p.swarm.asked()); got != asked {
			t.Errorf("a Grab whose recorded search ran past its deadline asked the Node %d times, want none", got-asked)
		}
	})
}

func TestAMagnetLinkOfAKnownHashNeedsNoSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel := newRelease(t)
		p := newPipeline(t, answer{err: downloadclients.ErrHeld})
		id := p.grab(rel.link.String(), nil)
		p.running()
		if v := p.view(id); v.Stage != grabs.Fetching || len(p.swarm.asked()) != 1 {
			t.Fatalf("after the Node responded held the Grab is %s with %d calls, want fetching after one", v.Stage, len(p.swarm.asked()))
		}
		raw, err := rel.link.Torrent(rel.info)
		if err != nil {
			t.Fatal(err)
		}
		attachTorrent(t, p, raw)
		time.Sleep(time.Minute)
		if v := p.view(id); v.Stage != grabs.Downloading || v.Torrent.InfoHash != rel.link.InfoHash || len(p.swarm.asked()) != 1 {
			t.Errorf("with a Torrent of the hash recorded the Grab is %s after %d calls, want downloading it with no new call", v.Stage, len(p.swarm.asked()))
		}
	})
}

func TestAMagnetLinkThatIsntTheIndexersReleaseFailsBeforeAnyCall(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel, other := newRelease(t), newRelease(t, "https://tracker.test/announce/secret")
		other.info = append([]byte(nil), rel.info...)
		other.info[len(other.info)-2] = 'b'
		p := newPipeline(t, answer{res: downloadclients.Resolved{Info: other.info}})
		hint := bencode.InfoHash{0xee}
		mismatched := p.grab(rel.link.String(), &hint)
		p.running()
		p.failed(mismatched, grabs.InfohashMismatch)
		if n := len(p.swarm.asked()); n != 0 {
			t.Errorf("a magnet link whose hash isn't the Indexer's asked the Node %d times, want none", n)
		}
		wrong := p.grab(rel.link.String(), nil)
		p.roster.change()
		time.Sleep(hold)
		p.failed(wrong, grabs.InfohashMismatch)
	})
}

func TestAMagnetLinkSlipStreamCantReadFailsItsGrab(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPipeline(t)
		id := p.grab("magnet:?dn=The.Attic.1080p.WEB", nil)
		p.running()
		p.failed(id, grabs.MagnetInvalid)
	})
}

func TestANodeTakingNoNewTorrentHoldsTheGrabUntilTheRosterChanges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel := newRelease(t)
		p := newPipeline(t, answer{err: downloadclients.ErrNotTaking}, searchingWith(0))
		id := p.grab(rel.link.String(), nil)
		p.running()
		time.Sleep(10 * time.Minute)
		if v := p.view(id); v.Stage != grabs.Fetching || len(p.swarm.asked()) != 1 {
			t.Fatalf("with the Node taking no new torrent the Grab is %s after %d calls, want fetching after one", v.Stage, len(p.swarm.asked()))
		}
		p.roster.change()
		time.Sleep(grabs.DefaultTimings().RetryMin)
		synctest.Wait()
		if n := len(p.swarm.asked()); n != 2 {
			t.Errorf("after the roster changed the Node was asked %d times, want again", n)
		}
	})
}

func attachTorrent(t *testing.T, p *pipeline, raw []byte) {
	t.Helper()
	m, err := bencode.DecodeMetainfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	err = p.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, _, err := downloadclients.Attach(ctx, tx, downloadclients.Want{
			Node: p.node, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: func(role.NodeID) bool { return false },
		}, storage.Now())
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
