package trackers_test

import (
	"context"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/works"
)

type published struct{ trackers []*string }

func (p *published) Publish(_ context.Context, v trackers.SeedingChanged) {
	p.trackers = append(p.trackers, v.TrackerID)
}

type identities struct{ trackers []string }

func (p *identities) Publish(_ context.Context, v indexers.TrackerChanged) {
	p.trackers = append(p.trackers, v.TrackerID)
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

// roster is every Node in step unless offline, and drains nothing.
type roster struct{ offline *bool }

func (r roster) InStep(role.NodeID) bool              { return !*r.offline }
func (roster) WrittenOff(role.Name, role.NodeID) bool { return false }
func (roster) Changed() <-chan struct{}               { return nil }
func (roster) Serving(role.Name) []role.NodeID        { return nil }
func (roster) Draining(role.Name, role.NodeID) bool   { return false }

// late is the Trackers' seeder once they are made, after the clients.
type late struct{ x **trackers.Trackers }

func (l late) Judge(ctx context.Context, r storage.Reader[storage.Library], rows []downloadclients.Row) (map[downloadclients.ID]downloadclients.Judged, error) {
	return (*l.x).Seeder().Judge(ctx, r, rows)
}

func (l late) EndlessUnits(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) (int64, error) {
	return (*l.x).Seeder().EndlessUnits(ctx, r, node)
}

type library struct {
	t               *testing.T
	lib             *storage.DB[storage.Library]
	x               *trackers.Trackers
	clients         *downloadclients.Clients
	changed         *published
	trackersChanged *identities
	soon            int
	// offline takes the library's Node out of step.
	offline bool
	// node, work and item are what every planted Grab names.
	node, work, item storage.UUID
}

func newLibrary(t *testing.T) *library {
	t.Helper()
	l := &library{
		t: t, changed: &published{}, trackersChanged: &identities{},
		lib: storagetest.Library(t, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(),
			grabs.Schema(), indexers.Schema(), trackers.Schema(), files.Schema()),
		node: storage.NewUUID(), work: storage.NewUUID(), item: storage.NewUUID(),
	}
	l.clients = downloadclients.New(downloadclients.Config{Library: l.lib, Roster: roster{&l.offline}, Uses: grabs.TorrentUses{}, Seeding: late{&l.x}})
	l.x = trackers.New(trackers.Config{
		Library: l.lib, Clients: l.clients, Grabs: grabs.TrackerFacts{}, Roster: roster{&l.offline},
		Soon: func() { l.soon++ }, Reconciling: trackers.NotReconciling, Changed: l.changed, TrackersChanged: l.trackersChanged,
	})
	l.exec(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, l.node)
	l.exec(`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, l.work)
	l.exec(`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, l.item, l.work)
	return l
}

func (l *library) exec(sql string, args ...any) {
	l.t.Helper()
	err := l.lib.Write(l.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, sql, args...)
		return err
	})
	if err != nil {
		l.t.Fatal(err)
	}
}

// tracker adds a Tracker named name.
func (l *library) tracker(name string, privacy indexers.Privacy) indexers.TrackerID {
	l.t.Helper()
	id := indexers.TrackerID{UUID: storage.NewUUID()}
	l.exec(`INSERT INTO indexers_tracker (id, name, privacy, created_at) VALUES (?, ?, ?, 0)`, id, name, string(privacy))
	return id
}

// grab plants a Grab of tracker's made at, whose Release stated ratio and
// seed, each nil for none.
func (l *library) grab(tracker indexers.TrackerID, at time.Time, ratio *float64, seed *time.Duration) {
	l.t.Helper()
	var seconds *int64
	if seed != nil {
		s := int64(*seed / time.Second)
		seconds = &s
	}
	l.exec(`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, tracker_id, node_id, release_title, release_size, release_magnet,
		min_ratio, min_seed_seconds, stage, created_at, updated_at)
		VALUES (?, ?, ?, 'any', 'search', ?, ?, 'Orchard.1080p', 1, 'magnet:?xt=urn:btih:00', ?, ?, 'fetching', ?, ?)`,
		storage.NewUUID(), l.work, l.item, tracker, l.node, ratio, seconds, storage.FromTime(at), storage.FromTime(at))
}

// saved fails unless the saves since the last call published one
// trackers.seeding-changed each, naming want (nil for none), and asked the
// Seeding goals task for a run each.
func (l *library) saved(saves int, want *indexers.TrackerID) {
	l.t.Helper()
	if len(l.changed.trackers) != saves || l.soon != saves {
		l.t.Fatalf("%d saves published %d trackers.seeding-changed and asked for %d runs, want %d of each", saves, len(l.changed.trackers), l.soon, saves)
	}
	for _, got := range l.changed.trackers {
		if (got == nil) != (want == nil) || got != nil && *got != want.String() {
			l.t.Fatalf("trackers.seeding-changed named %v, want %v", got, want)
		}
	}
	l.changed.trackers, l.soon = nil, 0
}

func ptr[T any](v T) *T { return &v }
