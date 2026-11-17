// Package notifications sends History's entries to the Admin's
// notification connections and the bell. It queues each entry in the Write
// that records it, and fans it out at release.
package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifying"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

type Config struct {
	Library *storage.DB[storage.Library]
	// Catalog is History's kinds, which the node seals before the
	// dispatcher starts.
	Catalog   *entry.Catalog
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
	cfg Config
	// doorbell wakes the dispatcher. A Write that queues an entry rings it
	// before it commits; SQLite runs one Write at a time, so the
	// dispatcher's next Write reads the pending row.
	doorbell chan struct{}
}

// New panics on a provider table with two rows of one id.
func New(cfg Config) *Service {
	seen := map[providers.ID]bool{}
	for _, p := range cfg.Providers {
		if seen[p.ID] {
			panic(fmt.Sprintf("notifications: two providers have the id %s", p.ID))
		}
		seen[p.ID] = true
	}
	return &Service{cfg: cfg, doorbell: make(chan struct{}, 1)}
}

func (s *Service) ring() {
	select {
	case s.doorbell <- struct{}{}:
	default:
	}
}
