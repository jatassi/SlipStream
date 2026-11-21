package downloading

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// resumed is what a torrent's resume data says of it. Resume data is
// libtorrent's: the info dictionary under "info", the trackers' tiers
// under "trackers", the seeding seconds under "seeding_time" and one byte
// a piece under "pieces", its low bit set for a piece the Node has.
type resumed struct {
	meta bencode.Metainfo
	// metainfo is rebuilt from the info dictionary and the trackers, so
	// its info hash is the torrent's.
	metainfo []byte
	done     float64
	seeding  time.Duration
}

func readResume(data []byte) (resumed, error) {
	inner, err := bencode.DecodeMetainfo(data)
	if err != nil {
		return resumed{}, fmt.Errorf("resume data: %w", err)
	}
	v, err := bencode.Decode(data)
	if err != nil {
		return resumed{}, fmt.Errorf("resume data: %w", err)
	}
	d, _ := v.(map[string]any)
	var tiers []any
	var announce string
	for _, tier := range trackerTiers(d["trackers"]) {
		urls := make([]any, len(tier))
		for i, u := range tier {
			urls[i] = u
		}
		tiers = append(tiers, urls)
		if announce == "" {
			announce = tier[0]
		}
	}
	top := map[string]any{"info": bencode.Raw(inner.RawInfo)}
	if announce != "" {
		top["announce"], top["announce-list"] = announce, tiers
	}
	metainfo, err := bencode.Encode(top)
	if err != nil {
		return resumed{}, fmt.Errorf("rebuild the metainfo: %w", err)
	}
	m, err := bencode.DecodeMetainfo(metainfo)
	if err != nil {
		return resumed{}, fmt.Errorf("rebuild the metainfo: %w", err)
	}
	r := resumed{meta: m, metainfo: metainfo}
	if s, ok := d["seeding_time"].(int64); ok && s > 0 {
		r.seeding = time.Duration(s) * time.Second
	}
	if pieces, ok := d["pieces"].(string); ok {
		total := len(m.Info.Pieces) / 20
		have := 0
		for i := range min(len(pieces), total) {
			if pieces[i]&1 == 1 {
				have++
			}
		}
		if total > 0 {
			r.done = float64(have) / float64(total)
		}
	}
	return r, nil
}

func trackerTiers(v any) [][]string {
	list, _ := v.([]any)
	var out [][]string
	for _, t := range list {
		urls, _ := t.([]any)
		var tier []string
		for _, u := range urls {
			if s, ok := u.(string); ok && s != "" {
				tier = append(tier, s)
			}
		}
		if tier != nil {
			out = append(out, tier)
		}
	}
	return out
}

func (r resumed) size() int64 {
	var n int64
	for _, f := range r.meta.Info.Files {
		n += f.Length
	}
	return n
}

// hosts is every tracker's host, lowercased, once each, in tracker order.
func (r resumed) hosts() []string {
	var out []string
	for _, u := range r.meta.Trackers() {
		if h := strings.ToLower(hostOf(u)); h != "" && !slices.Contains(out, h) {
			out = append(out, h)
		}
	}
	return out
}

// builtinHoldings is every torrent the built-in client keeps a row for, in
// info hash order, read page rows at a time: one the Node never saved
// resume data for by hash, privacy and folder alone.
func builtinHoldings(ctx context.Context, local *storage.DB[storage.Local], page int) ([]*dlv1.Holding, error) {
	var out []*dlv1.Holding
	after := []byte{}
	for {
		rows, err := db.New(local).ListTorrentsAfter(ctx, db.ListTorrentsAfterParams{After: after, Page: int64(page)})
		if err != nil {
			return nil, fmt.Errorf("list the torrents this Node holds: %w", err)
		}
		for _, row := range rows {
			h := &dlv1.Holding{InfoHash: row.InfoHash, Private: row.Private, SaveFolder: row.SaveFolder}
			if row.Resume != nil {
				r, err := readResume(row.Resume)
				if err != nil {
					return nil, fmt.Errorf("torrent %s: %w", bencode.InfoHash(row.InfoHash), err)
				}
				h.Content = &dlv1.HoldingContent{Name: r.meta.Info.Name, Size: r.size()}
				h.Done, h.SeedSeconds, h.Hosts = r.done, int64(r.seeding/time.Second), r.hosts()
			}
			out = append(out, h)
		}
		if len(rows) < page {
			return out, nil
		}
		after = rows[len(rows)-1].InfoHash
	}
}

// sendHoldings queues the round req asks for: every torrent the Node
// holds, in pages of req's size, in one Write. A round with no torrents is
// one empty page.
func sendHoldings(ctx context.Context, local *storage.DB[storage.Local], h Holder, req *dlv1.HoldingsRequest) error {
	size := int(req.GetPageSize())
	entries, err := builtinHoldings(ctx, local, size)
	if err != nil {
		return err
	}
	pages := max(1, (len(entries)+size-1)/size)
	return local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		for p := range pages {
			page := &dlv1.Holdings{
				Round: req.GetRound(), Page: uint32(p), Pages: uint32(pages), //nolint:gosec // holdings_max bounds the pages
				Entries: entries[p*size : min((p+1)*size, len(entries))],
			}
			if err := h.Report(ctx, tx, &dlv1.Report{Kind: &dlv1.Report_Holdings{Holdings: page}}); err != nil {
				return err
			}
		}
		return nil
	})
}

// Metainfo is a built-in torrent's metainfo, rebuilt from its resume data.
// An external client's torrent answers Unimplemented.
func (n *Node) Metainfo(ctx context.Context, req *dlv1.MetainfoRequest) (*dlv1.MetainfoResponse, error) {
	if len(req.GetClientId()) > 0 {
		return nil, rpc.Errorf(rpc.CodeUnimplemented, "this Node's external clients can't export a torrent's metainfo")
	}
	n.mu.Lock()
	local := n.local
	n.mu.Unlock()
	if local == nil {
		return nil, rpc.Errorf(rpc.CodeUnavailable, "the torrent client hasn't started")
	}
	hash := req.GetInfoHash()
	data, err := db.New(local).GetResume(ctx, hash)
	switch {
	case errors.Is(err, storage.ErrNoRows), err == nil && data == nil:
		return nil, rpc.Errorf(rpc.CodeNotFound, "this Node holds no resume data for torrent %x", hash)
	case err != nil:
		return nil, fmt.Errorf("read the torrent's resume data: %w", err)
	}
	r, err := readResume(data)
	if err != nil {
		return nil, err
	}
	return &dlv1.MetainfoResponse{Metainfo: r.metainfo}, nil
}
