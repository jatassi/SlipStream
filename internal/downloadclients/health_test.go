package downloadclients_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/health"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
)

func answer(t *testing.T, c *dc.Clients) health.Answer {
	t.Helper()
	a, err := c.HealthSource().Answer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func itemFor(a health.Answer, node role.NodeID) (health.Item, bool) {
	for _, it := range a.Items {
		if strings.HasPrefix(string(it.Key), node.String()+".") {
			return it, true
		}
	}
	return health.Item{}, false
}

func TestABindFailureIsAHealthError(t *testing.T) {
	lib := open(t)
	seedbox, home, attic, fresh := addNode(t, lib, "seedbox"), addNode(t, lib, "home"), addNode(t, lib, "attic"), addNode(t, lib, "fresh")
	facts := map[role.NodeID]*dlv1.Facts{
		seedbox: {Listen: "0.0.0.0:6881", ListenError: "Address already in use"},
		home:    {Listen: "0.0.0.0:6881", Bound: "192.0.2.4:6881"},
		attic:   {Listen: "0.0.0.0:6881", ListenError: "Address already in use"},
	}
	c := dc.New(dc.Config{
		Library: lib, Touch: func(...role.NodeID) {}, Changed: published{},
		Roster: &roster{serving: []role.NodeID{seedbox, home, attic, fresh}, away: map[role.NodeID]bool{attic: true}},
		Facts:  func(n role.NodeID) (*dlv1.Facts, bool) { f, ok := facts[n]; return f, ok },
	})

	a := answer(t, c)
	if len(a.Items) != 1 {
		t.Fatalf("items = %+v, want seedbox's bind failure alone", a.Items)
	}
	it := a.Items[0]
	if it.Level != health.Fail || it.Subject != "Torrent client on seedbox" ||
		!strings.HasPrefix(it.Text, "The torrent client on seedbox can't listen on 0.0.0.0:6881, so no peer can connect in.") {
		t.Errorf("seedbox's item = %+v, want an error that leads with why no peer connects in", it)
	}
	if it.Action.Kind() != health.ActionLink || it.Action.Href() != "/nodes/"+seedbox.String()+"/torrent-client" {
		t.Errorf("seedbox's action = %v %q, want a link to its Torrent client", it.Action.Kind(), it.Action.Href())
	}
	if !factSays(it.Detail, "Error", "Address already in use") {
		t.Errorf("seedbox's details = %+v, want the listen error", it.Detail)
	}
	unsure := map[string]string{}
	for _, u := range a.Unsure {
		unsure[u.Subject] = u.Text
	}
	if unsure["Torrent client on attic"] != "attic is offline." || unsure["Torrent client on fresh"] != "fresh's torrent client hasn't reported since SlipStream started." || len(unsure) != 2 {
		t.Errorf("unsure = %v, want attic offline and fresh unreported, so neither raises nor clears", unsure)
	}
}

func factSays(detail []health.Fact, label, text string) bool {
	for _, f := range detail {
		if f.Label == label && f.Text == text {
			return true
		}
	}
	return false
}

// sampler is a Node's client that samples its downloads volume and sends
// its facts as Sample asks.
type sampler struct {
	downloadingv1connect.UnimplementedDownloadingServiceHandler
	sample func(ctx context.Context) (*dlv1.SampleResponse, error)
}

func (s sampler) Sample(ctx context.Context, _ *dlv1.SampleRequest) (*dlv1.SampleResponse, error) {
	return s.sample(ctx)
}

func TestTheDiskGuardIsAHealthItem(t *testing.T) {
	g := newGuarded(t)
	r := &roster{serving: []role.NodeID{g.node}}
	at := time.Now().Truncate(time.Millisecond)
	g.c = dc.New(dc.Config{
		Library: g.lib, Touch: func(...role.NodeID) {}, Changed: published{}, CardsChanged: &told{}, Roster: r,
		Facts: func(role.NodeID) (*dlv1.Facts, bool) {
			g.mu.Lock()
			defer g.mu.Unlock()
			return g.facts, g.facts != nil
		},
		Remote: func(role.NodeID) downloadingv1connect.DownloadingServiceHandler {
			return sampler{sample: func(context.Context) (*dlv1.SampleResponse, error) {
				at = at.Add(time.Minute)
				go g.sample(25*setting.GB, at)
				return &dlv1.SampleResponse{SampledAtUnixMs: at.UnixMilli()}, nil
			}}
		},
		Log: slog.New(slog.DiscardHandler),
	})

	g.sample(12*setting.GB, at)
	it, ok := itemFor(answer(t, g.c), g.node)
	if !ok || it.Level != health.Warn || it.Subject != "New downloads on seedbox" ||
		!strings.HasPrefix(it.Text, "New downloads on seedbox wait for disk space. /data has 12 GB left, below its error level of 20 GB.") {
		t.Fatalf("seedbox's item = %+v (found %v), want a warning that new downloads wait for disk space", it, ok)
	}
	if it.Action.Kind() != health.ActionLink || it.Action.Href() != "/nodes/"+g.node.String()+"/torrent-client" {
		t.Errorf("the guard's action = %v %q, want a link to seedbox's Torrent client", it.Action.Kind(), it.Action.Href())
	}

	vols, err := g.c.HealthVolumes().Volumes(t.Context())
	if err != nil || len(vols) != 1 || vols[0].Node != g.node || vols[0].NodeName != "seedbox" || vols[0].Mount != "/data" || vols[0].Free != 12*setting.GB || !vols[0].Live {
		t.Fatalf("Storage's downloads volumes = %+v, %v; want seedbox's /data with 12 GB free, live", vols, err)
	}

	if err := g.c.HealthVolumes().Sample(t.Context(), g.node); err != nil {
		t.Fatalf("Check now on seedbox's downloads volume: %v", err)
	}
	if _, ok := itemFor(answer(t, g.c), g.node); ok {
		t.Error("after Check now found 25 GB free, the guard's item stays, want it cleared")
	}
	if vols, _ := g.c.HealthVolumes().Volumes(t.Context()); len(vols) != 1 || vols[0].Free != 25*setting.GB {
		t.Errorf("after Check now Storage reads %+v, want the fresh 25 GB", vols)
	}

	r.away = map[role.NodeID]bool{g.node: true}
	if err := g.c.HealthVolumes().Sample(t.Context(), g.node); !errors.Is(err, health.ErrUnreachable) {
		t.Errorf("Check now on an offline Node = %v, want unreachable", err)
	}
	if vols, _ := g.c.HealthVolumes().Volumes(t.Context()); len(vols) != 1 || vols[0].Live {
		t.Errorf("an offline Node's downloads volume reads %+v, want its last figures as of their sample", vols)
	}
	r.serving = nil
	if vols, _ := g.c.HealthVolumes().Volumes(t.Context()); len(vols) != 0 {
		t.Errorf("a Node that stopped downloading still reports %+v, want none", vols)
	}
}
