package shellapi_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/shellapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type shell struct {
	g *grabs.Grabs
	n *nodes.Service
	c *downloadclients.Clients
	h *health.Health
}

func (s shell) Grabs() (*grabs.Grabs, bool)                       { return s.g, true }
func (s shell) Nodes() (*nodes.Service, bool)                     { return s.n, true }
func (s shell) DownloadClients() (*downloadclients.Clients, bool) { return s.c, true }
func (s shell) Health() (*health.Health, bool)                    { return s.h, true }

type unlinked struct{}

func (unlinked) State(link.Key) link.State                    { return link.Down{} }
func (unlinked) Subscribe(context.Context) <-chan link.Change { return nil }
func (unlinked) Advertised() (link.Addresses, bool)           { return link.Addresses{}, false }
func (unlinked) Reset(link.Key)                               {}
func (unlinked) Keep(link.Target)                             {}
func (unlinked) Drop(link.Key)                                {}
func (unlinked) Invite(context.Context, link.Target) error    { return nil }
func (unlinked) Stats(link.Key) (link.Stats, bool)            { return link.Stats{}, false }

// seed inserts one Grab in stage, with the columns the stage's CHECKs ask
// for, against the one Item in the library.
const seed = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id,
	release_title, release_size, release_link, stage, torrent_id, path, failure_code, created_at, updated_at)
SELECT ?, work_id, id, 'standard', 'search', ?, ?, 'The.Attic.1080p.WEB', 4096, 'https://site.test/dl/1', ?,
	CASE WHEN ? = 'fetching' THEN NULL ELSE ? END,
	CASE WHEN ? IN ('transferring', 'importing', 'imported') THEN 'The Attic.mkv' END,
	CASE WHEN ? = 'failed' THEN 'download_failed' END,
	0, 0
FROM works_item`

func TestActivityCountsTheGrabsNeitherImportedNorFailed(t *testing.T) {
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema(), works.Schema(), targets.Schema(),
		roots.Schema(), indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema(), health.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	var admin identity.Admin
	if _, _, err := w.Add(ctx, admin, contenttypetest.ID, "w1", works.Adding{Monitor: true}); err != nil {
		t.Fatal(err)
	}
	roster, err := nodes.Open(ctx, nodes.Config{Library: lib, Name: "home", Link: unlinked{}, Owners: nodestest.Unowned(), Log: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	watch := openHealth(t, lib, roster)
	s := ownertest.Serve(t, lib, func(r *api.Router) {
		shellapi.Routes(r, shell{grabs.New(grabs.Config{Library: lib, Transfers: transfers.New(transfers.ControllerConfig{Library: lib})}), roster, downloadclients.New(downloadclients.Config{Library: lib}), watch})
	})

	s.Expect(t, http.MethodGet, "/api/v1/counts", "", http.StatusOK, `{"activity":0,"downloads":{"done":0,"wanted":0,"allPaused":false},"alerts":{"nodes":null,"health":null,"tasks":null}}`)
	home, indexer, torrent := roster.Self(), storage.NewUUID(), storage.NewUUID()
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, indexer, home); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers,
			content_type, wanted, added_order, added_at) VALUES (?, ?, randomblob(20), x'00', 'The.Attic.1080p.WEB', 4096, 1, '[]', 'fixture', 'started', 1, 0)`,
			torrent, home); err != nil {
			return err
		}
		for _, stage := range []string{"fetching", "fetching", "downloading", "probing", "transferring", "importing", "imported", "imported", "failed"} {
			if _, err := tx.ExecContext(ctx, seed, storage.NewUUID(), indexer, home, stage, stage, torrent, stage, stage); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Expect(t, http.MethodGet, "/api/v1/counts", "", http.StatusOK, `{"activity":6,"downloads":{"done":0,"wanted":4096,"allPaused":false},"alerts":{"nodes":null,"health":null,"tasks":null}}`)
}

var quietClients = health.Category{
	About:  health.About{ID: health.DownloadClientsID, Label: "Download Clients", Phrase: "download clients"},
	Source: health.Source{Answer: func(context.Context) (health.Answer, error) { return health.Answer{}, nil }},
}

var failedTask = health.Category{
	About: health.About{ID: health.ScheduledTasksID, Label: "Scheduled tasks", Phrase: "scheduled tasks"},
	Source: health.Source{Answer: func(context.Context) (health.Answer, error) {
		return health.Answer{Items: []health.Item{{Key: "housekeeping", Level: health.Warn, Says: health.Says{Subject: "Housekeeping", Text: "Housekeeping failed."}}}}, nil
	}},
}

func openHealth(t *testing.T, lib *storage.DB[storage.Library], roster *nodes.Service) *health.Health {
	t.Helper()
	hub := live.NewHub([]live.Event{nodes.ChangedEvent})
	t.Cleanup(hub.Close)
	h, err := health.Open(t.Context(), health.Config{
		Library:    lib,
		Hub:        hub,
		Changed:    discard[health.Changed]{},
		Categories: []health.Category{{About: nodes.HealthAbout, Source: roster.HealthSource()}, quietClients, failedTask},
		Timings:    health.DefaultTimings(),
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestTheAlertsCarryHealthsWorstLevels(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema(), works.Schema(), downloadclients.Schema(), grabs.Schema(), health.Schema())
	roster, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Name: "home", Link: unlinked{}, Owners: nodestest.Unowned(), Log: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	watch := openHealth(t, lib, roster)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = watch.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	if _, err := watch.Report(t.Context()); err != nil {
		t.Fatal(err)
	}
	s := ownertest.Serve(t, lib, func(r *api.Router) {
		shellapi.Routes(r, shell{grabs.New(grabs.Config{Library: lib, Transfers: transfers.New(transfers.ControllerConfig{Library: lib})}), roster, downloadclients.New(downloadclients.Config{Library: lib}), watch})
	})
	s.Expect(t, http.MethodGet, "/api/v1/counts", "", http.StatusOK, `{"activity":0,"downloads":{"done":0,"wanted":0,"allPaused":false},"alerts":{"nodes":null,"health":"warn","tasks":"warn"}}`)
}
