package downloadclients_test

import (
	"context"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/bencode"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
)

// keeps keeps keep, blocks while blocked and records each Holdings page.
type keeps struct {
	keep    dc.Keep
	blocked bool
	pages   []dc.HoldingsPage
}

func (k *keeps) KeepOf(context.Context, storage.Reader[storage.Library], role.NodeID) (dc.Keep, error) {
	return k.keep, nil
}

func (k *keeps) Blocked(context.Context, storage.Reader[storage.Library]) (bool, error) {
	return k.blocked, nil
}

func (k *keeps) Holdings(_ context.Context, _ *storage.Tx[storage.Library], _ role.NodeID, p dc.HoldingsPage) error {
	k.pages = append(k.pages, p)
	return nil
}

// keeping is Media server with k as its Keeps, in pages of page torrents
// and at most limit in a round.
func keeping(t *testing.T, k *keeps, page, limit int) *mediaServer {
	t.Helper()
	return newMediaServer(t, func(cfg *dc.Config) {
		cfg.Keeps, cfg.HoldingsPage, cfg.HoldingsMax = k, page, limit
	})
}

func holdings(round uint64, page, pages uint32, entries ...*dlv1.Holding) *dlv1.Report {
	return &dlv1.Report{Kind: &dlv1.Report_Holdings{Holdings: &dlv1.Holdings{Round: round, Page: page, Pages: pages, Entries: entries}}}
}

func entry(b byte) *dlv1.Holding {
	return &dlv1.Holding{InfoHash: append(make([]byte, 19), b), Content: &dlv1.HoldingContent{Name: "Orchard", Size: 1 << 30}, Done: 1}
}

// last is the last page k took in, failing when it took none.
func (k *keeps) last(t *testing.T) dc.HoldingsPage {
	t.Helper()
	if len(k.pages) == 0 {
		t.Fatal("Keeps took in no Holdings page")
	}
	return k.pages[len(k.pages)-1]
}

// TestHoldingsClampNamesSizesFoldersAndHosts sends pages of random
// entries, many past every limit and some not valid UTF-8: each entry
// comes out within its limits, as a prefix of what was sent.
func TestHoldingsClampNamesSizesFoldersAndHosts(t *testing.T) {
	k := &keeps{}
	s := keeping(t, k, 100, 1000)
	rng := rand.New(rand.NewPCG(7, 8)) //nolint:gosec // seeded on purpose so the test replays
	for round := range uint64(20) {
		sent := make([]*dlv1.Holding, 50)
		for i := range sent {
			sent[i] = randomEntry(rng, byte(i))
		}
		s.report(t, s.node, holdings(round+1, 0, 1, sent...))
		got := k.last(t)
		if got.Refused != "" || len(got.Entries) != len(sent) {
			t.Fatalf("round %d was refused (%q) or took in %d of %d entries, want all", round+1, got.Refused, len(got.Entries), len(sent))
		}
		for i, e := range got.Entries {
			checkClamped(t, sent[i], e)
		}
	}
}

func checkClamped(t *testing.T, sent *dlv1.Holding, got dc.Holding) {
	t.Helper()
	valid := func(s string, n int, from string) bool {
		return utf8.ValidString(s) && len(s) <= n && strings.HasPrefix(strings.ToValidUTF8(from, ""), s)
	}
	if !valid(got.Content.Name, 1024, sent.GetContent().GetName()) || got.Content.Size < 0 || got.Content.Size > 1<<50 {
		t.Errorf("the content is %q of %d bytes, want a valid prefix of at most 1,024 bytes and 0 to 2^50", got.Content.Name, got.Content.Size)
	}
	if !valid(got.Folder, 4096, sent.GetSaveFolder()) {
		t.Errorf("the folder is %d bytes, want a valid prefix of at most 4,096", len(got.Folder))
	}
	if got.Done < 0 || got.Done > 1 || math.IsNaN(got.Done) || got.Seed < 0 {
		t.Errorf("done is %v and seed %v, want 0 to 1 and 0 or more", got.Done, got.Seed)
	}
	if len(got.Hosts) > 16 {
		t.Errorf("%d hosts, want at most 16", len(got.Hosts))
	}
	for i, h := range got.Hosts {
		if h == "" || len(h) > 253 || h != strings.ToLower(h) || slices.Contains(got.Hosts[:i], h) {
			t.Errorf("host %q, want 1 to 253 bytes, lowercase and once", h)
		}
	}
}

func randomEntry(rng *rand.Rand, b byte) *dlv1.Holding {
	text := func(n int) string {
		out := make([]byte, rng.IntN(n))
		for i := range out {
			out[i] = "aZé\xff\xc3/\\."[rng.IntN(9)]
		}
		return string(out)
	}
	e := &dlv1.Holding{
		InfoHash:    append(make([]byte, 19), b),
		Content:     &dlv1.HoldingContent{Name: text(3000), Size: rng.Int64() - math.MaxInt64/2},
		Done:        []float64{-1, 0.5, 2, math.NaN(), math.Inf(1)}[rng.IntN(5)],
		SeedSeconds: rng.Int64() - math.MaxInt64/2, SaveFolder: text(9000),
	}
	for range rng.IntN(40) {
		e.Hosts = append(e.Hosts, text(400), "Tracker.Example")
	}
	return e
}

// TestAPageNamingAnotherNodesClientIsRefused sends a page naming
// Seedbox's client from Media server, and one naming no client at all:
// each is refused whole. A page naming Media server's own client is taken
// in, the entry carrying it.
func TestAPageNamingAnotherNodesClientIsRefused(t *testing.T) {
	k := &keeps{}
	s := keeping(t, k, 100, 1000)
	seedbox := addNode(t, s.lib, "Seedbox")
	theirs := mustImport(t, s.lib, secrettest.Sealer(t), transmission(t, seedbox, "Their Transmission", "http://192.168.1.30:9091"))
	for name, id := range map[string][]byte{"Seedbox's client": theirs.Bytes(), "no client": storage.NewUUID().Bytes()} {
		named := entry(2)
		named.ClientId = id
		s.report(t, s.node, holdings(1, 0, 1, entry(1), named))
		if got := k.last(t); got.Refused == "" || len(got.Entries) != 0 {
			t.Errorf("a page naming %s was taken in with %d entries, want it refused whole", name, len(got.Entries))
		}
	}
	own := entry(3)
	own.ClientId = s.client.Bytes()
	s.report(t, s.node, holdings(1, 0, 1, own))
	if got := k.last(t); got.Refused != "" || len(got.Entries) != 1 || got.Entries[0].Client == nil || *got.Entries[0].Client != s.client {
		t.Errorf("a page naming Media server's own client reads %+v, want its entry with the client", got)
	}
}

// TestHoldingsPastTheCapAreRefusedWhole bounds a round at 5 torrents in
// pages of 2, so 3 pages at most: a page past that bound, a page longer
// than 2, a page whose torrents pass 5, a page number past its count and a
// short info hash are each refused whole, while a last page of 1 is taken
// in.
func TestHoldingsPastTheCapAreRefusedWhole(t *testing.T) {
	k := &keeps{}
	s := keeping(t, k, 2, 5)
	short := entry(9)
	short.InfoHash = short.GetInfoHash()[:12]
	for _, c := range []struct {
		name string
		rep  *dlv1.Report
		fact string
	}{
		{"4 pages", holdings(1, 0, 4, entry(1)), "4 pages"},
		{"a page of 3", holdings(1, 0, 3, entry(1), entry(2), entry(3)), "3 torrents"},
		{"a last page of 2", holdings(1, 2, 3, entry(5), entry(6)), "5 torrents"},
		{"page 3 of 3 counted from 0", holdings(1, 3, 3), "page 4"},
		{"a short info hash", holdings(1, 0, 3, entry(1), short), "12 bytes"},
	} {
		s.report(t, s.node, c.rep)
		if got := k.last(t); !strings.Contains(got.Refused, c.fact) || len(got.Entries) != 0 || got.Round != 1 {
			t.Errorf("%s: refused %q with %d entries, want a refusal naming %q with none", c.name, got.Refused, len(got.Entries), c.fact)
		}
	}
	s.report(t, s.node, holdings(1, 2, 3, entry(5)))
	if got := k.last(t); got.Refused != "" || got.Page != 2 || got.Pages != 3 || len(got.Entries) != 1 || got.Entries[0].InfoHash != bencode.InfoHash(entry(5).GetInfoHash()) {
		t.Errorf("a last page of 1 reads %+v, want it taken in", got)
	}
}

// TestAVanishWhileBlockedMarksNothing reports Orchard gone from
// Transmission while Keeps blocks, live and again as the outbox replays
// it: Orchard stays pending and nothing is touched. Once Keeps no longer
// blocks, the replay marks it gone.
func TestAVanishWhileBlockedMarksNothing(t *testing.T) {
	k := &keeps{blocked: true}
	s := keeping(t, k, 100, 1000)
	orchard := s.send(t, "Orchard")
	for range 2 {
		s.report(t, s.node, vanished(orchard))
	}
	if got := outcomeOf(t, s.lib, orchard).Outcome; got != (dc.Pending{}) {
		t.Errorf("while Keeps blocks the vanished torrent reads %#v, want pending", got)
	}
	if got := s.heard.touchedNodes(); len(got) != 0 {
		t.Errorf("while Keeps blocks the vanish touched %v, want no intent", got)
	}
	k.blocked = false
	s.report(t, s.node, vanished(orchard))
	if got := outcomeOf(t, s.lib, orchard).Outcome; got != (dc.Gone{Reason: dc.ByClient, Client: "Transmission"}) {
		t.Errorf("once Keeps no longer blocks the replay reads %#v, want gone, removed by Transmission", got)
	}
}

// TestTheIntentCarriesWhatKeepsKeeps derives Media server's intent while
// Keeps keeps everything, two hashes and asks round 4: the intent says so
// with the Controller's page size. With no round it asks for none.
func TestTheIntentCarriesWhatKeepsKeeps(t *testing.T) {
	s := newMediaServer(t)
	k := dc.Keep{All: true, Hashes: []bencode.InfoHash{{1}, {2}}, Round: 4, PauseAll: true}
	in := derive(t, s, dc.Derive{Keeps: &keeps{keep: k}, HoldingsPage: 1000})
	if !in.GetKeepAll() || !in.GetPauseAll() || len(in.GetKeepHashes()) != 2 || in.GetKeepHashes()[1][0] != 2 ||
		in.GetHoldings().GetRound() != 4 || in.GetHoldings().GetPageSize() != 1000 {
		t.Errorf("the intent keeps all %v, pauses all %v, keeps %x and asks %v; want everything kept and paused, both hashes and round 4 in pages of 1,000",
			in.GetKeepAll(), in.GetPauseAll(), in.GetKeepHashes(), in.GetHoldings())
	}
	k.Round = 0
	if in := derive(t, s, dc.Derive{Keeps: &keeps{keep: k}, HoldingsPage: 1000}); in.GetHoldings() != nil {
		t.Errorf("with no round the intent asks %v, want no Holdings", in.GetHoldings())
	}
}

func derive(t *testing.T, s *mediaServer, d dc.Derive) *dlv1.Intent {
	t.Helper()
	var in *dlv1.Intent
	err := s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		in, err = d.Intent(ctx, r, s.node)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return in
}
