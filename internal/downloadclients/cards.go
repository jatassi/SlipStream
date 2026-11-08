package downloadclients

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// ClientsChanged names the Nodes whose card or settings moved, and the
// external clients whose settings or poll moved.
type ClientsChanged struct {
	NodeIDs   []string `json:"nodeIds" nullable:"false" doc:"The Nodes whose card or settings changed"`
	ClientIDs []string `json:"clientIds" nullable:"false" doc:"The external clients whose settings or state changed"`
}

var ClientsChangedEvent = live.AdminEvent[ClientsChanged]("download-clients.changed")

// Card is one downloading Node's clients: its built-in client and its
// external ones.
type Card struct {
	Node role.NodeID
	// GrabsTo is the external client the Node's new Grabs go to, nil for
	// its built-in client.
	GrabsTo *ClientID
	// Externals are its external clients by name.
	Externals []ExternalCard
	// InStep: the Node's Sync session is attached. Facts is nil while it
	// isn't.
	InStep bool
	// Facts is nil until the Node's client first reports them.
	Facts    *ClientFacts
	Settings NodeSettings
	// Totals add up the Node's live Torrents', and Torrents counts them.
	Totals   Totals
	Torrents int
	// Volume is the Node's downloads folder volume as last sampled, nil
	// until the first sample.
	Volume *Volume
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
		vols, err := volumesIn(ctx, r)
		if err != nil {
			return err
		}
		externals, err := c.externals(ctx, r)
		if err != nil {
			return err
		}
		for i, node := range downloaders {
			if cards[i], err = c.cardIn(ctx, r, node); err != nil {
				return err
			}
			if g, on := guards[node]; on {
				cards[i].Guard = &g
			}
			if j := slices.IndexFunc(vols, func(v Volume) bool { return v.Node == node }); j >= 0 {
				cards[i].Volume = &vols[j]
			}
			cards[i].Externals = onNode(externals, node)
		}
		return nil
	})
	return cards, err
}

func (c *Clients) cardIn(ctx context.Context, r storage.Reader[file], node role.NodeID) (Card, error) {
	o, err := nodeOverridesOf(ctx, r, node)
	if err != nil {
		return Card{}, err
	}
	t, err := db.New(r).GetNodeTotals(ctx, node)
	if err != nil {
		return Card{}, err
	}
	card := Card{
		Node: node, GrabsTo: o.GrabsTo, InStep: c.cfg.Roster.InStep(node), Settings: o.Settings(), Torrents: int(t.Torrents),
		Totals: Totals{
			Uploaded: t.Uploaded, Downloaded: t.Downloaded,
			Active: time.Duration(t.ActiveSeconds) * time.Second, Seeding: time.Duration(t.SeedingSeconds) * time.Second,
		},
	}
	if f, ok := c.cfg.Facts(node); ok {
		card.Facts = factsFrom(f)
	}
	return card, nil
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

func (c *Clients) cardsChanged(nodes ...role.NodeID) { c.clientsChanged(nil, nodes...) }

func (c *Clients) clientsChanged(ids []ClientID, nodes ...role.NodeID) {
	ch := ClientsChanged{NodeIDs: make([]string, len(nodes)), ClientIDs: make([]string, len(ids))}
	for i, n := range nodes {
		ch.NodeIDs[i] = n.String()
	}
	for i, id := range ids {
		ch.ClientIDs[i] = id.String()
	}
	c.cfg.CardsChanged.Publish(context.Background(), ch)
}

// ExternalCard is one external client as its card shows it.
type ExternalCard struct {
	External
	Reach ClientReach
	// Torrents counts its live Torrents.
	Torrents int
}

// CheckEveryField and ImportDelayField are the client's settings as the
// form shows them.
func (e External) CheckEveryField() setting.DurationField  { return CheckEvery.Field(e.checkEvery) }
func (e External) ImportDelayField() setting.DurationField { return ImportDelay.Field(e.importDelay) }

// externals is every external client, with secrets only as Saved flags.
func (c *Clients) externals(ctx context.Context, r storage.Reader[file]) ([]ExternalCard, error) {
	q := db.New(r)
	rows, err := q.ListClients(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := q.CountLiveByClient(ctx)
	if err != nil {
		return nil, err
	}
	held := map[ClientID]int{}
	for _, n := range counts {
		held[*n.ClientID] = int(n.Torrents)
	}
	out := make([]ExternalCard, len(rows))
	for i, row := range rows {
		e, err := externalOf(row, c.cfg.Sealer)
		if err != nil {
			return nil, err
		}
		out[i] = ExternalCard{External: e, Reach: ReachOf(e, c.cfg.Roster.InStep(e.Node)), Torrents: held[e.ID]}
	}
	return out, nil
}

func onNode(all []ExternalCard, node role.NodeID) []ExternalCard {
	out := []ExternalCard{}
	for _, e := range all {
		if e.Node == node {
			out = append(out, e)
		}
	}
	return out
}

// Orphans are the external clients on Nodes that no longer serve
// Downloading, which an Admin can only delete.
func (c *Clients) Orphans(ctx context.Context) ([]ExternalCard, error) {
	serving := c.cfg.Roster.Serving(role.Downloading)
	var out []ExternalCard
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		all, err := c.externals(ctx, r)
		for _, e := range all {
			if !slices.Contains(serving, e.Node) {
				out = append(out, e)
			}
		}
		return err
	})
	return out, err
}
