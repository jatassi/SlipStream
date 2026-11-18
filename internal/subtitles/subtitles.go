// Package subtitles is the Controller's Subtitles: what each library file
// wants, the Speech Maps and Sidecars that meet it, and the Keeper, the
// one caller of SidecarService that settles every Sidecar byte effect
// from rows. Its Node halves are internal/storing (SidecarService) and
// internal/subtitling.
package subtitles

import (
	"context"
	"embed"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is the Library set subtitles, after files.
func Schema() storage.Schema[storage.Library] {
	return storage.NewSchema[storage.Library]("subtitles", schemaFiles)
}

// Files is the library files as wants read them.
type Files interface {
	Subjects(ctx context.Context, r storage.Reader[storage.Library], after files.ID, n int, among works.Among[works.WorkID]) ([]files.Subject, error)
	SubjectsOf(ctx context.Context, r storage.Reader[storage.Library], ids []files.ID) ([]files.Subject, error)
}

// Targets is each Profile's Wanted Languages.
type Targets interface {
	LanguagesOf(ctx context.Context, r storage.Reader[storage.Library]) (map[targets.ProfileID][]targets.WantedLanguage, error)
}

// Roots is which Node holds each root.
type Roots interface {
	StateOf(ctx context.Context, id roots.ID) (roots.Root, roots.State, error)
}

type Config struct {
	Library *storage.DB[storage.Library]
	Files   Files
	Targets Targets
	Roots   Roots
	// Types is each Content Type's subtitle facet.
	Types map[works.TypeID]Facet
	// Storage is SidecarService on each Storage Node.
	Storage storing.SidecarRemote
	// Videos is the video extensions a survey lists companions of.
	Videos func(ctx context.Context, r storage.Reader[storage.Library]) (media.Videos, error)
	// Providers is every Subtitle Provider's Spec, in default order.
	Providers []subtitling.Spec
	Tasks     Cadences
	Roster    Roster
	// Reconciling is whether a restored Controller is Reconciling (ADR
	// 0019), read in the caller's snapshot. Nothing writes, carries or
	// removes a Sidecar while it is.
	Reconciling func(ctx context.Context, r storage.Reader[storage.Library]) (bool, error)
	History     entry.Recorder
	Timings     Timings
	Clock       func() time.Time
	Rand        *rand.Rand
	Log         *slog.Logger
}

type Subtitles struct {
	cfg    Config
	keeper *keeper
	pass   chan struct{}
}

// Open seeds a provider row for each Spec without one, after the last,
// and clears every lease the previous process left: only one Controller
// process ever holds one. Run settles what the rows say is pending.
func Open(ctx context.Context, cfg Config) (*Subtitles, error) {
	s := &Subtitles{cfg: cfg, pass: make(chan struct{}, 1)}
	s.keeper = newKeeper(s)
	err := cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		for _, p := range cfg.Providers {
			if err := q.SeedProvider(ctx, string(p.ID)); err != nil {
				return err
			}
		}
		return ClearLeases(ctx, tx)
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

// ClearLeases clears every want's lease, in tx: the wants keep their
// gaps, because a crash isn't an empty search.
func ClearLeases(ctx context.Context, tx *storage.Tx[storage.Library]) error {
	_, err := db.New(tx).ClearLeases(ctx)
	return err
}

// Run runs the Keeper and the bulk pass, once at start and again soon
// after each Write that asks for one, until ctx ends.
func (s *Subtitles) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Go(func() {
		if err := s.keeper.run(ctx); err != nil {
			s.cfg.Log.ErrorContext(ctx, "The subtitles Keeper stopped", "error", err)
		}
	})
	defer wg.Wait()
	for {
		if _, err := s.Reconcile(ctx); err != nil && ctx.Err() == nil {
			s.cfg.Log.WarnContext(ctx, "The subtitles pass failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-s.pass:
		}
	}
}

// soonReconcile asks for a bulk pass Timings.WakeDelay from now, by when
// the Write that asked has committed.
func (s *Subtitles) soonReconcile() {
	time.AfterFunc(s.cfg.Timings.WakeDelay, func() {
		select {
		case s.pass <- struct{}{}:
		default:
		}
	})
}

func (s *Subtitles) spec(id subtitling.ProviderID) subtitling.Spec {
	for _, p := range s.cfg.Providers {
		if p.ID == id {
			return p
		}
	}
	return subtitling.Spec{}
}
