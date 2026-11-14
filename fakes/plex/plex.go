// Package plex is fake-plex: one Plex Media Server over the shared Media
// Server core, the same server at every Node's front. It copies Plex Media
// Server 1.43.4.10903 (#71, #101) and 1.43.5.11029 (the recorded runs of
// 2026-11-14): sections with their Locations, partial and section
// refreshes with refreshing and scannedAt, the item refresh, items with
// Media[].Part[].file and their Sidecar streams, the 404 HTML body of a
// gone item, /status/sessions and /transcode/sessions with videoDecision,
// and the notification socket. A partial refresh takes forward slashes or
// backslashes and stores backslashes on Windows. A same-folder rename
// keeps the ratingKey, a destination-first move carries the play count
// and collections to a new item, and a source-first refresh with
// autoEmptyTrash purges the item. Editions are their own items only under
// Plex Pass. Every route but /identity answers 401 without a token the
// server issued, the account's JWT included. It answers JSON whatever the
// Accept header asks for.
package plex

import (
	"context"
	"net"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/fakes/internal/plexaccount"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/node"
)

const name = "plex"

// Versions are the versions this fake plays.
var Versions = []string{"1.43.4.10903", "1.43.5.11029"}

// Fake is the Plex an Admin picks from plex.tv's resources, at every
// Node's front. Its address is typed, so Endpoints sets nothing.
var Fake = fake.Spec{
	Name:      name,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.OnlineSuite("plex"),
	Cues:      []fake.CueSpec{{Name: "plex-play"}, {Name: "plex-stop"}, {Name: "plex-restart"}, {Name: "plex-scan"}},
	Captures:  []fake.CaptureSpec{Refreshes, Items},
}

var (
	// Refreshes records each refresh a front took: partial, section and
	// item.
	Refreshes = fake.CaptureSpec{Name: "plex-refreshes", Kind: "plex.refresh"}
	// Items records each item the server added, changed or removed.
	Items = fake.CaptureSpec{Name: "plex-items", Kind: "plex.item"}
)

// Refresh is a Refreshes record's data. Path is the folder as the server
// stores it, empty for a section refresh, and Item is set for an item
// refresh.
type Refresh struct {
	Section string `json:"section,omitempty"`
	Path    string `json:"path,omitempty"`
	Item    string `json:"item,omitempty"`
}

// ItemRecord is an Items record's data.
type ItemRecord struct {
	Event     string   `json:"event"`
	RatingKey string   `json:"ratingKey"`
	Section   string   `json:"section"`
	Files     []string `json:"files"`
}

// Server is the one Plex a test or the world plays.
type Server struct {
	Name       string
	Version    string
	Identifier string
	// Tokens are server tokens it accepts beside those fake-plex-tv issues
	// for it.
	Tokens []string
	// Owner is the account owner, whose watch data the token reads.
	Owner    string
	PlexPass bool
	// KeepTrash turns autoEmptyTrash off, which is on by default.
	KeepTrash bool
	// Root is the folder the media dir is, as Plex names it, such as /media
	// or C:\Media.
	Root     string
	Sections []Section
}

// Section is one library section. Folders are its Locations, under Root.
type Section struct {
	Key     string
	Title   string
	Type    string
	Folders []string
	// Scanned is a section that was scanned with its content before the
	// fake started.
	Scanned bool
}

// Playing is Fake playing srv, for a test that names it.
func Playing(srv Server) fake.Spec {
	spec := Fake
	spec.New = func(_ scenario.Scenario, env fake.Env) (fake.Server, error) { return serve(env, &srv) }
	return spec
}

// newServer plays no Plex until a Scenario names one.
func newServer(_ scenario.Scenario, env fake.Env) (fake.Server, error) { return serve(env, nil) }

type server struct {
	env     fake.Env
	srv     *Server
	account *plexaccount.Account
	core    *mediaserver.Server
	http    fake.Server
	items   *fake.Capture
}

func serve(env fake.Env, srv *Server) (fake.Server, error) {
	s := &server{env: env, srv: srv, account: plexaccount.Of(env.Shared)}
	if srv != nil {
		s.account.Claim(plexaccount.Server{Target: fault.Target(env.Name), Name: srv.Name, Identifier: srv.Identifier, Version: srv.Version})
		core, err := newCore(env, srv)
		if err != nil {
			return nil, err
		}
		s.core = core
		if s.items, err = env.Capture(Items.Name); err != nil {
			return nil, err
		}
		if err := s.cues(); err != nil {
			return nil, err
		}
	}
	refreshes, err := env.Capture(Refreshes.Name)
	if err != nil {
		return nil, err
	}
	s.http = fake.HTTP(env, s.routes(refreshes))
	return s, nil
}

func newCore(env fake.Env, srv *Server) (*mediaserver.Server, error) {
	root, err := link.ParsePath(srv.Root)
	if err != nil {
		return nil, err
	}
	libs := make([]mediaserver.Library, len(srv.Sections))
	for i, sec := range srv.Sections {
		libs[i] = mediaserver.Library{Key: sec.Key, Name: sec.Title, Type: sec.Type, Scanned: sec.Scanned}
		for _, f := range sec.Folders {
			p, err := link.ParsePath(f)
			if err != nil {
				return nil, err
			}
			libs[i].Folders = append(libs[i].Folders, p)
		}
	}
	return mediaserver.New(mediaserver.Behavior{
		FreshSectionIgnores: true, StableKeyInFolder: true, PurgesVacated: !srv.KeepTrash, Editions: srv.PlexPass,
		ScanTime: 250 * time.Millisecond,
	}, mediaserver.Disk{Media: env.Media, Root: root}, libs), nil
}

func (s *server) Serve(ctx context.Context, l net.Listener) error {
	if s.core != nil {
		go s.record(ctx)
	}
	return s.http.Serve(ctx, l)
}

// record writes each item event to the Items capture until ctx ends.
func (s *server) record(ctx context.Context) {
	for {
		sub := s.core.Subscribe()
		for done := false; !done; {
			select {
			case <-ctx.Done():
				sub.Cancel()
				return
			case <-sub.Done:
				done = true
			case e := <-sub.C:
				s.recordItem(ctx, e)
			}
		}
	}
}

func (s *server) recordItem(ctx context.Context, e mediaserver.Event) {
	events := map[mediaserver.EventKind]string{mediaserver.ItemAdded: "added", mediaserver.ItemChanged: "changed", mediaserver.ItemRemoved: "removed"}
	event, ok := events[e.Kind]
	if !ok {
		return
	}
	rec := ItemRecord{Event: event, RatingKey: e.Item.Key, Section: e.Library, Files: []string{}}
	for _, p := range e.Item.Parts {
		rec.Files = append(rec.Files, p.File.String())
	}
	s.items.Record(ctx, rec)
}
