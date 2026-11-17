// Package history is the one log of what happened to the library: every
// fact is recorded as an entry in the Write that makes it true. History
// lists the entries, and notifications sends them on to the bell and the
// Admin's connections.
package history

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
)

type EventID = db.EventID

type Config struct {
	Library *storage.DB[storage.Library]
	Catalog *entry.Catalog
	// Outbox is notifications: it queues in each capture's Write and keeps
	// what it still owes.
	Outbox  Outbox
	Changed ChangedPublisher
	// Batch is housekeeping.Timings.PruneBatch: rows per sweep Write.
	Batch int
	Log   *slog.Logger
}

// Outbox is what sends entries on. notifications implements it.
type Outbox interface {
	// Queue runs in the capture's Write after the entry is stored. It does
	// row work only.
	Queue(ctx context.Context, tx *storage.Tx[storage.Library], c Captured) error
	// Owed is which of ids a sweep must leave: a pending row, a due
	// delivery, or the newer entry such a message renders a key from.
	Owed(ctx context.Context, r storage.Reader[storage.Library], ids []EventID) (map[EventID]bool, error)
	// Told is the words a target was last sent for a Health key. A clear
	// whose item History no longer holds uses them.
	Told(ctx context.Context, r storage.Reader[storage.Library], key string) (Told, bool, error)
}

// Captured is one stored entry as the Outbox sees it.
type Captured struct {
	ID    EventID
	Kind  entry.Kind
	At    time.Time
	Quiet bool
	// Keys are a Health entry's "<category>.<key>"s, nil for other kinds.
	Keys []string
}

type Told struct {
	Subject string
	Since   time.Time
}

// Changed is published after a commit that changed History.
type Changed struct{}

var ChangedEvent = live.AdminEvent[Changed]("history.changed")

// ChangedPublisher is the history.changed event bound to the hub.
type ChangedPublisher interface {
	Publish(ctx context.Context, v Changed)
}

type Service struct {
	cfg Config
	// bell wakes Run. A capture rings it before its Write commits, and a
	// sweep after its Write deleted rows.
	bell chan struct{}
	// swept is set by a sweep that deleted rows, so Run publishes even
	// though no newer entry was recorded.
	swept chan struct{}
	// clear wakes Run's sweeper after Clear raised the floor.
	clear chan struct{}
}

// New registers the Health rows first in cfg.Catalog.
func New(cfg Config) *Service {
	cfg.Catalog.Register("history", []entry.Entry{NewItemsRow, ClearedRow})
	return &Service{cfg: cfg, bell: make(chan struct{}, 1), swept: make(chan struct{}, 1), clear: make(chan struct{}, 1)}
}

// Recorder registers owner's rows in the catalog and answers the Recorder
// they are recorded through. It panics on a kind already registered, a row
// with no History group, and once the catalog is sealed.
func (s *Service) Recorder(owner string, rows []entry.Entry) entry.Recorder {
	s.cfg.Catalog.Register(owner, rows)
	return entry.Sink(owner, rows, s.put)
}

func (s *Service) put(ctx context.Context, tx *storage.Tx[storage.Library], k entry.Kind, subject []byte, a entry.About) error {
	return s.store(ctx, tx, k, subject, a, nil)
}

// store inserts one entry with its Works and Health items, hands it to the
// Outbox and rings Run. at is the capture's time, so ids and times rise
// together.
func (s *Service) store(ctx context.Context, tx *storage.Tx[storage.Library], k entry.Kind, subject []byte, a entry.About, items []Item) error {
	q := db.New(tx)
	at := storage.Now()
	p := db.InsertEventParams{Kind: k, At: at, GrabID: a.Grab, ImportID: a.Import, Subject: string(subject)}
	if a.Type != "" {
		p.ContentType = &a.Type
	}
	if len(a.Works) == 1 {
		p.WorkID = a.Works[0]
	}
	id, err := q.InsertEvent(ctx, p)
	if err != nil {
		return fmt.Errorf("record a %s entry: %w", k, err)
	}
	for _, w := range a.Works {
		if err := q.InsertWork(ctx, db.InsertWorkParams{EventID: id, WorkID: w}); err != nil {
			return fmt.Errorf("record a %s entry: %w", k, err)
		}
	}
	keys := make([]string, 0, len(items))
	for _, it := range items {
		if err := q.InsertItem(ctx, it.params(id)); err != nil {
			return fmt.Errorf("record a %s entry: %w", k, err)
		}
		keys = append(keys, it.Key)
	}
	c := Captured{ID: id, Kind: k, At: at.Time(), Quiet: a.Quiet}
	if items != nil {
		c.Keys = keys
	}
	if err := s.cfg.Outbox.Queue(ctx, tx, c); err != nil {
		return err
	}
	ring(s.bell)
	return nil
}

func ring(c chan struct{}) {
	select {
	case c <- struct{}{}:
	default:
	}
}
