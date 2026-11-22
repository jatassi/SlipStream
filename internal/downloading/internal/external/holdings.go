package external

import (
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/bencode"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// ErrNoExport is a client kind that can't hand back a torrent's metainfo.
var ErrNoExport = errors.New("this kind of Download Client can't export a torrent's metainfo")

// Holdings is every torrent the client lists under the share's label, and
// each other torrent the share records there, since an attached torrent
// keeps its own label. It fails whole when the client can't be read.
func (a *Actor) Holdings(ctx context.Context) ([]*dlv1.Holding, error) {
	var out []*dlv1.Holding
	err := errStopped
	a.ask(func(s *state) { out, err = s.holdings(ctx) })
	return out, err
}

// Metainfo is the client's own copy of h's metainfo, ErrNoExport for a
// kind that keeps none it can hand back.
func (a *Actor) Metainfo(ctx context.Context, h bencode.InfoHash) ([]byte, error) {
	var out []byte
	err := errStopped
	a.ask(func(s *state) { out, err = s.metainfo(ctx, h) })
	return out, err
}

func (s *state) holdings(ctx context.Context) ([]*dlv1.Holding, error) {
	if err := s.session(ctx); err != nil {
		return nil, err
	}
	held, err := s.client.List(ctx, s.label)
	if err != nil {
		return nil, fmt.Errorf("list the client's torrents: %w", err)
	}
	var recorded []bencode.InfoHash
	for _, t := range s.share.Torrents {
		if _, ok := held[bencode.InfoHash(t.GetInfoHash())]; !ok {
			recorded = append(recorded, bencode.InfoHash(t.GetInfoHash()))
		}
	}
	for _, d := range s.share.Discards {
		if _, ok := held[bencode.InfoHash(d.GetInfoHash())]; !ok {
			recorded = append(recorded, bencode.InfoHash(d.GetInfoHash()))
		}
	}
	if len(recorded) > 0 {
		found, err := s.client.Poll(ctx, recorded)
		if err != nil {
			return nil, fmt.Errorf("look up the client's torrents: %w", err)
		}
		for h, st := range found {
			held[h] = st
		}
	}
	out := make([]*dlv1.Holding, 0, len(held))
	for h, st := range held {
		out = append(out, s.holding(h, st))
	}
	return out, nil
}

func (s *state) holding(h bencode.InfoHash, st clients.Status) *dlv1.Holding {
	var done float64
	if st.Size > 0 {
		done = float64(st.Done) / float64(st.Size)
	}
	return &dlv1.Holding{
		InfoHash: h[:], ClientId: s.cfg.ID.Bytes(), Content: &dlv1.HoldingContent{Name: st.Name, Size: st.Size},
		Done: done, Private: st.Private, SeedSeconds: int64(st.Counters.Seeding.Seconds()), Hosts: st.Hosts,
		SaveFolder: st.Folder,
	}
}

func (s *state) metainfo(ctx context.Context, h bencode.InfoHash) ([]byte, error) {
	if err := s.session(ctx); err != nil {
		return nil, err
	}
	e, ok := s.client.(clients.Exporter)
	if !ok {
		return nil, ErrNoExport
	}
	return e.Export(ctx, h)
}
