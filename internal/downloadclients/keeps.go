package downloadclients

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Keep is what one Node's Downloading intent keeps beyond its lists, read
// in the derivation's Reader so the intent stays a function of the
// library.
type Keep struct {
	// All keeps every torrent the Node's clients hold.
	All bool
	// Hashes keeps these torrents, in order.
	Hashes []bencode.InfoHash
	// Round is the Holdings round the Node is asked for, 0 for none.
	Round uint64
	// PauseAll keeps and pauses every torrent.
	PauseAll bool
}

// HoldingsPage is one page of a Node's Holdings, checked and clamped.
// Refused is why the whole page was refused, "" when it wasn't: its
// Entries are then empty.
type HoldingsPage struct {
	Round       uint64
	Page, Pages int
	Entries     []Holding
	Refused     string
}

// Holding is one torrent a Node's client holds, as it reported it.
type Holding struct {
	InfoHash bencode.InfoHash
	// Client is the external client that holds it, nil for the built-in
	// one.
	Client *ClientID
	// Content is nil for a torrent the Node saved no resume data for.
	Content *HoldingContent
	// Done is the share of its pieces the Node has, 0 to 1.
	Done    float64
	Private bool
	Seed    time.Duration
	// Hosts is its trackers' hosts, lowercased, once each.
	Hosts  []string
	Folder string
}

type HoldingContent struct {
	Name string
	Size int64
}

// Keeps is what keeps a Node's torrents while SlipStream adopts them.
type Keeps interface {
	KeepOf(ctx context.Context, r storage.Reader[file], node role.NodeID) (Keep, error)
	// Blocked: no Node's torrent may be marked gone while it holds.
	Blocked(ctx context.Context, r storage.Reader[file]) (bool, error)
	// Holdings takes in one page of node's Holdings in tx.
	Holdings(ctx context.Context, tx *storage.Tx[file], node role.NodeID, p HoldingsPage) error
}

// NoKeeps keeps nothing beyond the lists, asks no Node for its Holdings,
// drops every page and never blocks.
var NoKeeps Keeps = noKeeps{}

type noKeeps struct{}

func (noKeeps) KeepOf(context.Context, storage.Reader[file], role.NodeID) (Keep, error) {
	return Keep{}, nil
}

func (noKeeps) Blocked(context.Context, storage.Reader[file]) (bool, error) { return false, nil }

func (noKeeps) Holdings(context.Context, *storage.Tx[file], role.NodeID, HoldingsPage) error {
	return nil
}

// Derive derives each Node's Downloading intent.
type Derive struct {
	Keeps Keeps
	// HoldingsPage is how many torrents a Holdings page lists at most.
	HoldingsPage int
}

// Intent is node's Downloading intent: its live Torrents, with a Retry
// waiting for its Node's check, and its discards, the Removing ones, each
// in added order and naming its client, its settings with defaults, the
// thresholds, the disk guard's hold, how often to sample the downloads
// volume, its external clients, on or off, with every setting but their
// secrets, and what Keeps keeps. The same library gives the same bytes.
func (d Derive) Intent(ctx context.Context, r storage.Reader[file], node role.NodeID) (*dlv1.Intent, error) {
	in, err := listed(ctx, r, node)
	if err != nil {
		return nil, err
	}
	k, err := d.Keeps.KeepOf(ctx, r, node)
	if err != nil {
		return nil, err
	}
	in.KeepAll, in.PauseAll = k.All, k.PauseAll
	for _, h := range k.Hashes {
		in.KeepHashes = append(in.KeepHashes, h[:])
	}
	if k.Round != 0 {
		in.Holdings = &dlv1.HoldingsRequest{Round: k.Round, PageSize: uint32(d.HoldingsPage)} //nolint:gosec // holdings_page is at most 10,000
	}
	return in, nil
}

// Limits on what one Holdings entry carries.
const (
	maxName   = 1024
	maxFolder = 4096
	maxSize   = 1 << 50
	maxHosts  = 16
	maxHost   = 253
	// maxSeed is the most seconds a time.Duration holds.
	maxSeed = math.MaxInt64 / int64(time.Second)
)

// parseHoldings checks a Holdings page node sent and clamps its entries.
// It refuses the whole page when an entry names a client that isn't
// node's or an info hash that isn't one, or when the page lies outside a
// round of at most limit torrents in pages of page.
func parseHoldings(ctx context.Context, q *db.Queries, node role.NodeID, h *dlv1.Holdings, page, limit int) (HoldingsPage, error) {
	p := HoldingsPage{Round: h.GetRound(), Page: int(h.GetPage()), Pages: int(h.GetPages())}
	entries := h.GetEntries()
	switch {
	case p.Page >= p.Pages:
		p.Refused = fmt.Sprintf("The Node sent page %d of a list that has %d pages.", p.Page+1, p.Pages)
	case int64(p.Pages-1)*int64(page) >= int64(limit):
		p.Refused = fmt.Sprintf("The Node sent a list of %d pages, which is over the limit of %d torrents.", p.Pages, limit)
	case len(entries) > page:
		p.Refused = fmt.Sprintf("A page lists %d torrents, more than the %d a page holds.", len(entries), page)
	case int64(p.Page)*int64(page)+int64(len(entries)) > int64(limit):
		p.Refused = fmt.Sprintf("The Node's list is over the limit of %d torrents.", limit)
	}
	if p.Refused != "" {
		return p, nil
	}
	clients, err := q.ListNodeClients(ctx, node)
	if err != nil {
		return HoldingsPage{}, err
	}
	for _, e := range entries {
		held, why := holdingOf(e, clients)
		if why != "" {
			return HoldingsPage{Round: p.Round, Page: p.Page, Pages: p.Pages, Refused: why}, nil
		}
		p.Entries = append(p.Entries, held)
	}
	return p, nil
}

func holdingOf(e *dlv1.Holding, clients []db.DownloadclientsClient) (h Holding, refused string) {
	if len(e.GetInfoHash()) != len(bencode.InfoHash{}) {
		return Holding{}, fmt.Sprintf("An info hash is %d bytes, not 20.", len(e.GetInfoHash()))
	}
	h = Holding{
		InfoHash: bencode.InfoHash(e.GetInfoHash()), Done: share(e.GetDone()), Private: e.GetPrivate(),
		Seed:   time.Duration(min(max(e.GetSeedSeconds(), 0), maxSeed)) * time.Second,
		Folder: cutUTF8(e.GetSaveFolder(), maxFolder),
	}
	if id := e.GetClientId(); len(id) > 0 {
		i := slices.IndexFunc(clients, func(c db.DownloadclientsClient) bool { return bytes.Equal(c.ID.Bytes(), id) })
		if i < 0 {
			return Holding{}, "A torrent names a Download Client that isn't on this Node."
		}
		h.Client = &clients[i].ID
	}
	if c := e.GetContent(); c != nil {
		h.Content = &HoldingContent{Name: cutUTF8(c.GetName(), maxName), Size: min(max(c.GetSize(), 0), maxSize)}
	}
	for _, host := range e.GetHosts() {
		host = cutUTF8(strings.ToLower(host), maxHost)
		if len(h.Hosts) < maxHosts && host != "" && !slices.Contains(h.Hosts, host) {
			h.Hosts = append(h.Hosts, host)
		}
	}
	return h, ""
}

// share is d clamped to 0 through 1, 0 for NaN.
func share(d float64) float64 {
	if math.IsNaN(d) {
		return 0
	}
	return min(max(d, 0), 1)
}

// cutUTF8 is s's valid UTF-8 cut to at most n bytes on a rune boundary.
func cutUTF8(s string, n int) string {
	s = strings.ToValidUTF8(s, "")
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
