// Package notifications records what happened in the Write that made it
// true and sends it to the Admin's notification connections and the bell.
package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

type Config struct {
	Library   *storage.DB[storage.Library]
	Sealer    *secret.Sealer
	Providers []providers.Provider
	Timings   Timings
	Rand      *rand.Rand
	Zone      *time.Location
	Log       *slog.Logger
}

type Service struct {
	cfg     Config
	catalog *catalog
	// doorbell wakes the dispatcher. A Write that records an event rings it
	// before it commits; SQLite runs one Write at a time, so the
	// dispatcher's next Write reads the event.
	doorbell chan struct{}
}

// New panics on a provider table with two rows of one id. Its catalog
// starts with New Health item and Health item cleared.
func New(cfg Config) *Service {
	seen := map[providers.ID]bool{}
	for _, p := range cfg.Providers {
		if seen[p.ID] {
			panic(fmt.Sprintf("notifications: two providers have the id %s", p.ID))
		}
		seen[p.ID] = true
	}
	return &Service{cfg: cfg, catalog: newCatalog(), doorbell: make(chan struct{}, 1)}
}

// Recorder registers owner's rows in the catalog and answers the Recorder
// they are recorded through. It panics on a kind already registered, and
// once the dispatcher has started.
func (s *Service) Recorder(owner string, rows []notice.Entry) notice.Recorder {
	s.catalog.register(owner, rows)
	return notice.Sink(owner, rows, s.put)
}

// put inserts an event due now and rings the doorbell.
func (s *Service) put(ctx context.Context, tx *storage.Tx[storage.Library], k notice.Kind, subject []byte) error {
	now := storage.Now()
	if _, err := db.New(tx).InsertEvent(ctx, db.InsertEventParams{Kind: k, At: now, DueAt: now, Subject: string(subject)}); err != nil {
		return fmt.Errorf("record a %s event: %w", k, err)
	}
	s.ring()
	return nil
}

func (s *Service) ring() {
	select {
	case s.doorbell <- struct{}{}:
	default:
	}
}
