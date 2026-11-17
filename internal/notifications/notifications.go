// Package notifications records what happened in the Write that made it
// true and sends it to the Admin's notification connections and the bell.
package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifying"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

type Config struct {
	Library   *storage.DB[storage.Library]
	Sealer    *secret.Sealer
	Providers []providers.Provider
	Courier   Courier
	Roster    Roster
	Mail      MailServer
	Artwork   Artwork
	Endpoints providers.Endpoints
	Changed   ChangedPublisher
	Seen      SeenPublisher
	Timings   Timings
	Rand      *rand.Rand
	Zone      *time.Location
	Log       *slog.Logger
	// BeforePass runs before each dispatcher pass, and its error fails the
	// pass as a storage error would: the test tier's seam. Nil in a Node.
	BeforePass func(ctx context.Context) error
}

// Courier reaches the Node that sends: notifying.Client. An error means
// the Node didn't respond; a provider's or a script's failure is an
// answer.
type Courier interface {
	Post(ctx context.Context, node role.NodeID, r notifying.Request) (notifying.Response, error)
	Run(ctx context.Context, node role.NodeID, s notifying.Script) (notifying.Ran, error)
	Send(ctx context.Context, node role.NodeID, s mail.Server, m mail.Message) (mail.Outcome, error)
}

// Roster is the Nodes now: *nodes.Service.
type Roster interface {
	Self() role.NodeID
	Changed() <-chan struct{}
	Pick(r role.Name, prefer role.NodeID) (role.NodeID, bool)
	Holds(r role.Name, node role.NodeID) bool
	InStep(node role.NodeID) bool
	Name(node role.NodeID) string
	WrittenOff(r role.Name, node role.NodeID) bool
}

// MailServer is the SMTP server Email connections send through:
// *mail.Mail. false while mail isn't set up.
type MailServer interface {
	Server(ctx context.Context) (mail.Server, bool, error)
}

// Artwork reads posters: *metadata.Metadata.
type Artwork interface {
	Image(ctx context.Context, ref artwork.Ref, size artwork.Size) (artwork.Image, error)
	PublicURL(ref artwork.Ref, size artwork.Size) (string, error)
}

// Changed is published after a commit that showed bell rows or changed
// what a connection's row says.
type Changed struct {
	Feed        bool `json:"feed"`
	Connections bool `json:"connections"`
}

// Seen is published to one Account after its bell cursor moves.
type Seen struct{}

var (
	ChangedEvent = live.AdminEvent[Changed]("notifications.changed")
	SeenEvent    = live.AccountEvent[Seen]("notifications.seen")
)

// ChangedPublisher is the notifications.changed event bound to the hub.
type ChangedPublisher interface {
	Publish(ctx context.Context, v Changed)
}

// SeenPublisher is the notifications.seen event bound to the hub.
type SeenPublisher interface {
	Publish(ctx context.Context, account storage.UUID, v Seen)
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
