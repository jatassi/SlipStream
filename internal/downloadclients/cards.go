package downloadclients

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// ClientsChanged names the Nodes whose built-in client card or settings
// moved.
type ClientsChanged struct {
	NodeIDs []string `json:"nodeIds" nullable:"false" doc:"The Nodes whose built-in client's card or settings changed"`
}

var ClientsChangedEvent = live.AdminEvent[ClientsChanged]("download-clients.changed")

// Card is one downloading Node's built-in client.
type Card struct {
	Node role.NodeID
	// InStep: the Node's Sync session is attached. Facts is nil while it
	// isn't.
	InStep bool
	// Facts is nil until the Node's client first reports them.
	Facts    *ClientFacts
	Settings NodeSettings
	// Totals add up the Node's live Torrents', and Torrents counts them.
	Totals   Totals
	Torrents int
	// Guard is set while the disk guard holds the Node's new downloads.
	Guard *Guard
}

// ClientFacts is a Node's client as it reports itself.
type ClientFacts struct {
	Libtorrent, PeerIDPrefix, UserAgent string
	// Listen is the configured listen address.
	Listen string
	// Bound is the address it listens on, "" while it has no listener.
	Bound string
	// ListenError is why it has no listener.
	ListenError string
	// ReachedAt is when a peer that proves the listen port first connected
	// in to where it listens now, zero while none has.
	ReachedAt time.Time
}

// Reach is the listen-port check's verdict. Nothing here can see a closed
// port, so it is never closed.
type Reach string

const (
	Reachable Reach = "reachable"
	Unproven  Reach = "unproven"
)

func (f ClientFacts) Reach() Reach {
	if f.ReachedAt.IsZero() {
		return Unproven
	}
	return Reachable
}

func factsFrom(f *dlv1.Facts) *ClientFacts {
	return &ClientFacts{
		Libtorrent: f.GetLibtorrent(), PeerIDPrefix: f.GetPeerIdPrefix(), UserAgent: f.GetUserAgent(),
		Listen: f.GetListen(), Bound: f.GetBound(), ListenError: f.GetListenError(), ReachedAt: atMillis(f.GetIncomingAtUnixMs()),
	}
}

// Cards is the built-in client of every Node assigned Downloading or
// draining it, in Node order.
func (c *Clients) Cards(ctx context.Context) ([]Card, error) {
	downloaders := c.cfg.Roster.Serving(role.Downloading)
	cards := make([]Card, len(downloaders))
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		guards, err := guardsIn(ctx, r)
		if err != nil {
			return err
		}
		for i, node := range downloaders {
			o, err := nodeOverridesOf(ctx, r, node)
			if err != nil {
				return err
			}
			t, err := db.New(r).GetNodeTotals(ctx, node)
			if err != nil {
				return err
			}
			cards[i] = Card{
				Node: node, InStep: c.cfg.Roster.InStep(node), Settings: o.Settings(), Torrents: int(t.Torrents),
				Totals: Totals{
					Uploaded: t.Uploaded, Downloaded: t.Downloaded,
					Active: time.Duration(t.ActiveSeconds) * time.Second, Seeding: time.Duration(t.SeedingSeconds) * time.Second,
				},
			}
			if f, ok := c.cfg.Facts(node); ok {
				cards[i].Facts = factsFrom(f)
			}
			if g, on := guards[node]; on {
				cards[i].Guard = &g
			}
		}
		return nil
	})
	return cards, err
}

// Arrived is nodes.Config.Facts: a Node's Downloading facts change its
// card, and their downloads volume sample may move the disk guard.
func (c *Clients) Arrived(node role.NodeID, scope string) {
	if scope != string(role.Downloading) {
		return
	}
	ctx := context.Background()
	if f, ok := c.cfg.Facts(node); ok && f.GetDownloads() != nil {
		err := c.record(ctx, node, f)
		if err == nil {
			_, err = c.guard(ctx, []role.NodeID{node})
		}
		if err != nil {
			c.cfg.Log.Error("judge a Node's downloads volume", "node", node.String(), "err", err.Error())
		}
	}
	c.cardsChanged(node)
}

func (c *Clients) cardsChanged(nodes ...role.NodeID) {
	ch := ClientsChanged{NodeIDs: make([]string, len(nodes))}
	for i, n := range nodes {
		ch.NodeIDs[i] = n.String()
	}
	c.cfg.CardsChanged.Publish(context.Background(), ch)
}
