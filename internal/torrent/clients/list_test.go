package clients_test

import (
	"log/slog"
	"maps"
	"slices"
	"testing"
	"time"

	fakedeluge "github.com/jatassi/slipstream/fakes/deluge"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	fakeqbittorrent "github.com/jatassi/slipstream/fakes/qbittorrent"
	fakertorrent "github.com/jatassi/slipstream/fakes/rtorrent"
	"github.com/jatassi/slipstream/fakes/scenario"
	faketransmission "github.com/jatassi/slipstream/fakes/transmission"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/clients/deluge"
	"github.com/jatassi/slipstream/internal/torrent/clients/qbittorrent"
	"github.com/jatassi/slipstream/internal/torrent/clients/rtorrent"
	"github.com/jatassi/slipstream/internal/torrent/clients/transmission"
)

const (
	listNode = "Media server"
	password = "orchard"
)

// played is one version of one kind's fake, and what that version reports.
type played struct {
	kind    clients.Kind
	version string
	spec    func(dir string) fake.Spec
	open    func(clients.Session) clients.Client
	// private and hosts are whether the version reports the private flag
	// and its trackers' hosts.
	private, hosts bool
}

func everyFake() []played {
	var out []played
	for _, v := range fakeqbittorrent.Versions {
		out = append(out, played{kind: clients.QBittorrent, version: v, open: qbittorrent.Open, private: v != "4.6", hosts: true,
			spec: func(dir string) fake.Spec {
				return fakeqbittorrent.Playing(fakeqbittorrent.Client{Node: listNode, Version: v, Username: "admin", Password: password, Dir: dir, Folder: "/downloads", Rate: 100})
			}})
	}
	for _, v := range faketransmission.Versions {
		out = append(out, played{kind: clients.Transmission, version: v, open: transmission.Open, private: true, hosts: true,
			spec: func(dir string) fake.Spec {
				return faketransmission.Playing(faketransmission.Daemon{Node: listNode, Version: v, Username: "admin", Password: password, Dir: dir, Folder: "/downloads", Rate: 100})
			}})
	}
	for _, v := range fakedeluge.Versions {
		out = append(out, played{kind: clients.Deluge, version: v, open: deluge.Open, private: true, hosts: true,
			spec: func(dir string) fake.Spec {
				return fakedeluge.Playing(fakedeluge.Client{Node: listNode, Version: v, Password: password, Daemons: []fakedeluge.Daemon{
					{Host: "127.0.0.1", Port: 58846, Dir: dir, Folder: "/downloads", Rate: 100},
				}})
			}})
	}
	for _, v := range fakertorrent.Versions {
		out = append(out, played{kind: clients.RTorrent, version: v, open: rtorrent.Open, private: true,
			spec: func(dir string) fake.Spec {
				return fakertorrent.Playing(fakertorrent.Client{Node: listNode, Version: v, Username: "admin", Password: password, Dir: dir, Folder: "/downloads", Rate: 100})
			}})
	}
	return out
}

func open(t *testing.T, p played, served faketest.Served) clients.Client {
	t.Helper()
	a, err := clients.ParseAddress(served.Address.URL(p.kind.Facts().Path), p.kind)
	if err != nil {
		t.Fatal(err)
	}
	s := clients.Session{
		HTTP: served.Client, Address: a, Secrets: clients.Secrets{Password: password}, Call: 30 * time.Second,
		Log: slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
	if p.kind != clients.Deluge {
		s.Username = "admin"
	}
	return p.open(s)
}

// TestEachClientListsByLabel adds orchard and lantern under SlipStream's
// label and harbor under another to each fake. List answers orchard and
// lantern with their names, sizes and labels, orchard private where the
// kind reports it and each with its tracker's host where the kind reports
// hosts, and never harbor.
func TestEachClientListsByLabel(t *testing.T) {
	for _, p := range everyFake() {
		t.Run(string(p.kind)+" "+p.version, func(t *testing.T) { listsByLabel(t, p) })
	}
}

func listsByLabel(t *testing.T, p played) {
	t.Helper()
	cat := faketest.CatalogPublic(t, 1000, []string{"lantern", "harbor"}, "orchard", "lantern", "harbor")
	rels := map[string]clients.Add{}
	for _, r := range cat.Releases {
		rels[r.Name] = clients.Add{Metainfo: r.Metainfo, Hash: r.InfoHash}
	}
	served := faketest.InMemory(t, p.spec(t.TempDir()), scenario.Scenario{Seed: string(p.kind)}, faketest.WithReleases(cat), faketest.As(listNode))
	c := open(t, p, served)
	mine, other := label(t, clients.DefaultLabel, p.kind), label(t, "other", p.kind)
	for name, l := range map[string]clients.Label{"orchard": mine, "lantern": mine, "harbor": other} {
		a := rels[name]
		a.Label = l
		if err := c.Add(t.Context(), a); err != nil {
			t.Fatalf("Add(%s): %v", name, err)
		}
	}

	got, err := c.List(t.Context(), mine)
	if err != nil {
		t.Fatal(err)
	}
	want := []bencode.InfoHash{rels["orchard"].Hash, rels["lantern"].Hash}
	if keys := slices.Collect(maps.Keys(got)); !sameHashes(keys, want) {
		t.Fatalf("List lists %v, want orchard %s and lantern %s alone", keys, want[0], want[1])
	}
	var hosts []string
	if p.hosts {
		hosts = []string{"127.0.0.1"}
	}
	for name, private := range map[string]bool{"orchard": true, "lantern": false} {
		s := got[rels[name].Hash]
		if s.Name != name+".mkv" || s.Size != 1000 || s.Label != clients.DefaultLabel {
			t.Errorf("%s lists as %q of %d bytes under %q, want %q of 1000 under %q", name, s.Name, s.Size, s.Label, name+".mkv", clients.DefaultLabel)
		}
		if want := private && p.private; s.Private != want {
			t.Errorf("%s lists private %t, want %t", name, s.Private, want)
		}
		if !slices.Equal(s.Hosts, hosts) {
			t.Errorf("%s lists hosts %v, want %v", name, s.Hosts, hosts)
		}
	}
}

func label(t *testing.T, raw string, kind clients.Kind) clients.Label {
	t.Helper()
	l, err := clients.ParseLabel(raw, kind)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func sameHashes(a, b []bencode.InfoHash) bool {
	cmp := func(x, y bencode.InfoHash) int { return slices.Compare(x[:], y[:]) }
	return slices.Equal(slices.SortedFunc(slices.Values(a), cmp), slices.SortedFunc(slices.Values(b), cmp))
}
