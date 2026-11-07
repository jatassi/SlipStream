package transfers

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers/internal/ledger"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

const gb = setting.GB

var (
	t0      = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	seedbox = nodeID(1)
	nas     = nodeID(2)
	home    = nodeID(3)
	vps     = nodeID(4)
	onNas   = rootID(1)
	onHome  = rootID(2)
	onVps   = rootID(3)
	names   = map[role.NodeID]string{seedbox: "seedbox", nas: "nas", home: "home", vps: "vps"}
)

func uuidOf(prefix, n byte) storage.UUID {
	b := make([]byte, 16)
	b[0], b[6], b[8], b[15] = prefix, 0x70, 0x80, n
	u, err := storage.UUIDFromBytes(b)
	if err != nil {
		panic(err)
	}
	return u
}

func nodeID(n byte) role.NodeID { return role.NodeID{UUID: uuidOf(1, n)} }
func rootID(n byte) roots.ID    { return roots.ID{UUID: uuidOf(2, n)} }
func tid(n byte) ID             { return ID{UUID: uuidOf(3, n)} }

type world struct{ s snapshot }

func newWorld() *world {
	s := snapshot{
		now:        t0,
		limits:     map[role.NodeID]int64{},
		pairs:      map[pairKey]ledger.Pair{},
		dests:      map[roots.ID]roots.Destination{},
		sources:    map[sourceAt]source.Readiness{},
		peers:      map[role.NodeID]nodes.Peer{},
		ends:       map[role.NodeID]End{},
		facts:      map[role.NodeID]*tv1.Facts{},
		fullAt:     map[ID]time.Time{},
		routeRetry: time.Hour,
	}
	for n, name := range names {
		var key link.Key
		key[0] = n.Bytes()[15]
		s.peers[n] = nodes.Peer{Name: name, Key: key, Addresses: []string{name + ":8485"}}
		s.ends[n] = End{Controller: n == home}
	}
	for id, node := range map[roots.ID]role.NodeID{onNas: nas, onHome: home, onVps: vps} {
		s.dests[id] = roots.Destination{
			Root: roots.Root{ID: id, Node: node, Place: 1}, State: roots.Ready,
			Volume: &storing.Volume{ID: "v-" + names[node], Name: "Media", Mount: "/media", Total: 1000 * gb, Free: 100 * gb, SampledAt: t0.Add(-time.Minute)},
		}
	}
	return &world{s}
}

// add appends a Transfer of one file per size from src into root, line
// order following n.
func (w *world) add(n byte, src role.NodeID, root roots.ID, sizes ...int64) *world {
	t := Transfer{ID: tid(n), Purpose: Grab, Job: strconv.Itoa(int(n)), Source: src, Destination: Root{ID: root}, Created: t0.Add(time.Duration(n) * time.Second)}
	for i, size := range sizes {
		t.Files = append(t.Files, File{Position: i, Name: fmt.Sprintf("file %d.%d.mkv", n, i), Source: torrent(n, i), Size: size})
	}
	w.s.transfers = append(w.s.transfers, t)
	return w
}

func torrent(n byte, file int) source.Torrent {
	var h bencode.InfoHash
	h[0] = n
	return source.Torrent{Hash: h, File: file}
}

func (w *world) file(n byte, position int, change func(*File)) *world {
	i := slices.IndexFunc(w.s.transfers, func(t Transfer) bool { return t.ID == tid(n) })
	change(&w.s.transfers[i].Files[position])
	return w
}

func (w *world) end(node role.NodeID, change func(*End)) *world {
	e := w.s.ends[node]
	change(&e)
	w.s.ends[node] = e
	return w
}

func (w *world) dest(id roots.ID, change func(*roots.Destination)) *world {
	d := w.s.dests[id]
	change(&d)
	w.s.dests[id] = d
	return w
}

func (w *world) receiving(node role.NodeID, r *tv1.Receiving) *world {
	f := w.s.facts[node]
	if f == nil {
		f = &tv1.Facts{}
		w.s.facts[node] = f
	}
	f.Receiving = append(f.Receiving, r)
	return w
}

func (w *world) sending(node role.NodeID, r *tv1.Sending) *world {
	f := w.s.facts[node]
	if f == nil {
		f = &tv1.Facts{}
		w.s.facts[node] = f
	}
	f.Sending = append(f.Sending, r)
	return w
}

func (w *world) ready(src role.NodeID, n byte, state source.ReadyState) *world {
	w.s.sources[sourceAt{src, torrent(n, 0)}] = source.Readiness{State: state, Name: fmt.Sprintf("torrent %d", n)}
	return w
}

func (w *world) pair(src, dest role.NodeID, route tv1.Route, at time.Time) *world {
	w.s.pairs[pairKey{src, dest}] = ledger.Pair{SourceNode: src, DestinationNode: dest, Rate: 1000, RateAt: storage.FromTime(at), Route: new(int64(route)), RouteAt: storage.FromTime(at)}
	return w
}

// state is a view in one line: its state, its wait or need and the Node
// it names, and its route while active.
func state(v View) string {
	var b strings.Builder
	b.WriteString(v.State.String())
	switch v.State {
	case Queued:
		fmt.Fprintf(&b, " %d", v.Ahead)
	case Waiting:
		fmt.Fprintf(&b, " %s %s", v.Wait.Kind, names[v.Wait.Node])
	case NeedsSpace:
		fmt.Fprintf(&b, " %d GB", v.Space.Need/gb)
	case Failed:
		fmt.Fprintf(&b, " %s", v.Failure.Code)
	case Copying:
	}
	if v.Route != tv1.Route_ROUTE_UNSPECIFIED {
		fmt.Fprintf(&b, " route %d", v.Route)
	}
	return b.String()
}

// actives is each Node's active Transfers, "send 1" or "receive 1".
func actives(j judgment) map[string][]string {
	out := map[string][]string{}
	for node, in := range j.intents {
		for _, a := range in.GetActive() {
			side := map[tv1.Side]string{tv1.Side_SIDE_SEND: "send", tv1.Side_SIDE_RECEIVE: "receive"}[a.GetSide()]
			out[names[node]] = append(out[names[node]], fmt.Sprintf("%s %d", side, a.GetTransferId()[15]))
		}
	}
	return out
}

// TestJudge has one case per row of the transitions table in candidate 1's
// design (section 2.4) that judge decides, and one per derived State and
// Wait.
func TestJudge(t *testing.T) {
	cases := []struct {
		name    string
		world   *world
		want    map[byte]string
		actives map[string][]string
	}{
		{
			name:    "a lone Transfer with both ends in step is active on route 1",
			world:   newWorld().add(1, seedbox, onNas, gb),
			want:    map[byte]string{1: "waiting connecting seedbox route 1"},
			actives: map[string][]string{"seedbox": {"send 1"}, "nas": {"receive 1"}},
		},
		{
			name:    "bytes moving on the judged route copy",
			world:   newWorld().add(1, seedbox, onNas, gb).receiving(nas, &tv1.Receiving{TransferId: tid(1).Bytes(), Route: tv1.Route_ROUTE_RECEIVER_DIALS, Active: true, Moving: true, Rate: 1 << 20}),
			want:    map[byte]string{1: "copying route 1"},
			actives: map[string][]string{"seedbox": {"send 1"}, "nas": {"receive 1"}},
		},
		{
			name:    "one active Transfer per pair, the rest queue in line order",
			world:   newWorld().add(1, seedbox, onNas, gb).add(2, seedbox, onNas, gb).add(3, seedbox, onNas, gb).add(4, seedbox, onVps, gb),
			want:    map[byte]string{1: "waiting connecting seedbox route 1", 2: "queued 1", 3: "queued 2", 4: "waiting connecting seedbox route 1"},
			actives: map[string][]string{"seedbox": {"send 1", "send 4"}, "nas": {"receive 1"}, "vps": {"receive 4"}},
		},
		{
			name:  "an offline end holds the pair and goes in neither intent",
			world: newWorld().add(1, seedbox, onNas, gb).add(2, seedbox, onNas, gb).end(seedbox, func(e *End) { e.Standing = nodes.Offline }),
			want:  map[byte]string{1: "waiting node_away seedbox", 2: "queued 1"},
		},
		{
			name:  "a lapsed end holds the pair",
			world: newWorld().add(1, seedbox, onNas, gb).end(nas, func(e *End) { e.Standing = nodes.LapsedNode }),
			want:  map[byte]string{1: "waiting node_away nas"},
		},
		{
			name:  "an end missing from the board reads away",
			world: newWorld().add(1, seedbox, onNas, gb).with(func(s *snapshot) { delete(s.ends, nas) }),
			want:  map[byte]string{1: "waiting node_away nas"},
		},
		{
			name:  "an end on another build holds the pair",
			world: newWorld().add(1, seedbox, onNas, gb).end(seedbox, func(e *End) { e.Standing = nodes.MismatchedNode }),
			want:  map[byte]string{1: "waiting node_other_build seedbox"},
		},
		{
			name:    "a reconnecting end stops nothing",
			world:   newWorld().add(1, seedbox, onNas, gb).end(seedbox, func(e *End) { e.Standing = nodes.Reconnecting }),
			want:    map[byte]string{1: "waiting connecting seedbox route 1"},
			actives: map[string][]string{"seedbox": {"send 1"}, "nas": {"receive 1"}},
		},
		{
			name:    "a root not ready steps aside and the next copies",
			world:   newWorld().add(1, seedbox, onNas, gb).add(2, seedbox, onNas, gb).dest(onNas, func(d *roots.Destination) { d.State = roots.Unmounted }),
			want:    map[byte]string{1: "waiting root_not_ready nas", 2: "waiting root_not_ready nas"},
			actives: map[string][]string{},
		},
		{
			name:    "a draining Storage Node steps aside",
			world:   newWorld().add(1, seedbox, onNas, gb).end(nas, func(e *End) { e.DrainsStorage = true }),
			want:    map[byte]string{1: "waiting storage_draining nas"},
			actives: map[string][]string{},
		},
		{
			name:    "a checking source steps aside and the next on the pair copies",
			world:   newWorld().add(1, seedbox, onNas, gb).add(2, seedbox, onNas, gb).ready(seedbox, 1, source.Checking),
			want:    map[byte]string{1: "waiting source_checking seedbox", 2: "waiting connecting seedbox route 1"},
			actives: map[string][]string{"seedbox": {"send 2"}, "nas": {"receive 2"}},
		},
		{
			name:  "an incomplete source steps aside",
			world: newWorld().add(1, seedbox, onNas, gb).add(2, seedbox, onNas, gb).ready(seedbox, 2, source.Incomplete),
			want:  map[byte]string{1: "waiting connecting seedbox route 1", 2: "waiting source_checking seedbox"},
		},
		{
			name:  "a ready source copies",
			world: newWorld().add(1, seedbox, onNas, gb).ready(seedbox, 1, source.Ready),
			want:  map[byte]string{1: "waiting connecting seedbox route 1"},
		},
		{
			name:    "a receiver's step-aside wait goes to the back of its line",
			world:   newWorld().add(1, seedbox, onNas, gb).add(2, seedbox, onNas, gb).receiving(nas, &tv1.Receiving{TransferId: tid(1).Bytes(), Wait: &tv1.Wait{Kind: tv1.Wait_KIND_SOURCE_NOT_READY, Detail: "permission denied"}}),
			want:    map[byte]string{1: "waiting source_not_ready seedbox", 2: "waiting connecting seedbox route 1"},
			actives: map[string][]string{"seedbox": {"send 2"}, "nas": {"receive 2"}},
		},
		{
			name:    "alone on its pair, a receiver's step-aside wait stays active",
			world:   newWorld().add(1, seedbox, onNas, gb).receiving(nas, &tv1.Receiving{TransferId: tid(1).Bytes(), Wait: &tv1.Wait{Kind: tv1.Wait_KIND_ROOT_NOT_READY}}),
			want:    map[byte]string{1: "waiting root_not_ready nas route 1"},
			actives: map[string][]string{"seedbox": {"send 1"}, "nas": {"receive 1"}},
		},
		{
			name:  "space is first come, first served per volume",
			world: newWorld().add(1, seedbox, onNas, 50*gb).add(2, vps, onNas, 40*gb).add(3, home, onNas, 30*gb),
			want:  map[byte]string{1: "waiting connecting seedbox route 1", 2: "needs_space 10 GB", 3: "waiting connecting home route 1"},
		},
		{
			name:  "verified bytes leave the volume's reserve",
			world: newWorld().add(1, seedbox, onNas, 90*gb).receiving(nas, &tv1.Receiving{TransferId: tid(1).Bytes(), VerifiedBytes: 20 * gb}),
			want:  map[byte]string{1: "waiting connecting seedbox route 1"},
		},
		{
			name:  "a full disk needs space until a newer sample",
			world: newWorld().add(1, seedbox, onNas, gb).receiving(nas, &tv1.Receiving{TransferId: tid(1).Bytes(), Wait: &tv1.Wait{Kind: tv1.Wait_KIND_DISK_FULL}}),
			want:  map[byte]string{1: "needs_space 1 GB"},
		},
		{
			name: "a sample newer than the full disk lets it go on",
			world: newWorld().add(1, seedbox, onNas, gb).receiving(nas, &tv1.Receiving{TransferId: tid(1).Bytes(), Wait: &tv1.Wait{Kind: tv1.Wait_KIND_DISK_FULL}}).
				dest(onNas, func(d *roots.Destination) { d.Volume.SampledAt = t0 }).with(func(s *snapshot) { s.fullAt[tid(1)] = t0.Add(-time.Second) }),
			want: map[byte]string{1: "waiting connecting seedbox route 1"},
		},
		{
			name:  "a receiver that gave up on route 1 flips to route 2",
			world: newWorld().add(1, seedbox, onNas, gb).receiving(nas, &tv1.Receiving{TransferId: tid(1).Bytes(), Route: tv1.Route_ROUTE_RECEIVER_DIALS, Active: true, GaveUp: []tv1.Route{tv1.Route_ROUTE_RECEIVER_DIALS}}),
			want:  map[byte]string{1: "waiting connecting seedbox route 2"},
		},
		{
			name: "a facts route before the flip can't skip the next route",
			world: newWorld().add(1, seedbox, onNas, gb).
				receiving(nas, &tv1.Receiving{TransferId: tid(1).Bytes(), Route: tv1.Route_ROUTE_RECEIVER_DIALS, Active: true, GaveUp: []tv1.Route{tv1.Route_ROUTE_RECEIVER_DIALS}}).
				sending(seedbox, &tv1.Sending{TransferId: tid(1).Bytes(), Route: tv1.Route_ROUTE_RECEIVER_DIALS, GaveUp: []tv1.Route{tv1.Route_ROUTE_SENDER_DIALS}}),
			want: map[byte]string{1: "waiting connecting seedbox route 2"},
		},
		{
			name: "a Worker and the Controller fall back to the Node link",
			world: newWorld().add(1, seedbox, onHome, gb).
				receiving(home, &tv1.Receiving{TransferId: tid(1).Bytes(), Route: tv1.Route_ROUTE_SENDER_DIALS, Active: true, GaveUp: []tv1.Route{tv1.Route_ROUTE_RECEIVER_DIALS}}).
				sending(seedbox, &tv1.Sending{TransferId: tid(1).Bytes(), Route: tv1.Route_ROUTE_SENDER_DIALS, GaveUp: []tv1.Route{tv1.Route_ROUTE_SENDER_DIALS}}),
			want:    map[byte]string{1: "waiting connecting seedbox route 3"},
			actives: map[string][]string{"seedbox": {"send 1"}, "home": {"receive 1"}},
		},
		{
			name: "two Workers that gave up both routes have none",
			world: newWorld().add(1, seedbox, onNas, gb).
				receiving(nas, &tv1.Receiving{TransferId: tid(1).Bytes(), Route: tv1.Route_ROUTE_SENDER_DIALS, Active: true, GaveUp: []tv1.Route{tv1.Route_ROUTE_RECEIVER_DIALS}}).
				sending(seedbox, &tv1.Sending{TransferId: tid(1).Bytes(), Route: tv1.Route_ROUTE_SENDER_DIALS, GaveUp: []tv1.Route{tv1.Route_ROUTE_SENDER_DIALS}}),
			want:    map[byte]string{1: "waiting no_route nas route 4"},
			actives: map[string][]string{"seedbox": {"send 1"}, "nas": {"receive 1"}},
		},
		{
			name:  "a pair's recent route starts the next Transfer",
			world: newWorld().add(1, seedbox, onNas, gb).pair(seedbox, nas, tv1.Route_ROUTE_SENDER_DIALS, t0.Add(-time.Minute)),
			want:  map[byte]string{1: "waiting connecting seedbox route 2"},
		},
		{
			name:  "a pair's route older than the retry starts from route 1",
			world: newWorld().add(1, seedbox, onNas, gb).pair(seedbox, nas, tv1.Route_ROUTE_SENDER_DIALS, t0.Add(-2*time.Hour)),
			want:  map[byte]string{1: "waiting connecting seedbox route 1"},
		},
		{
			name:    "verified but not placed frees the pair and reads copying at the full size",
			world:   newWorld().add(1, seedbox, onNas, gb).add(2, seedbox, onNas, gb).file(1, 0, func(f *File) { f.Verified = true }),
			want:    map[byte]string{1: "copying", 2: "waiting connecting seedbox route 1"},
			actives: map[string][]string{"seedbox": {"send 2"}, "nas": {"receive 2"}},
		},
		{
			name:  "a stored failure keeps its staging and holds nothing",
			world: newWorld().add(1, seedbox, onNas, gb).add(2, seedbox, onNas, gb).with(func(s *snapshot) { s.transfers[0].Failure = &Failure{Code: Damaged, Position: 0} }),
			want:  map[byte]string{1: "failed damaged", 2: "waiting connecting seedbox route 1"},
		},
		{
			name:    "a root re-homed onto the source reroutes",
			world:   newWorld().add(1, seedbox, onNas, gb).dest(onNas, func(d *roots.Destination) { d.Node = seedbox }),
			want:    map[byte]string{1: "failed rerouted"},
			actives: map[string][]string{},
		},
		{
			name:    "a source written off fails",
			world:   newWorld().add(1, seedbox, onNas, gb).ready(seedbox, 1, source.WrittenOff),
			want:    map[byte]string{1: "failed source_written_off"},
			actives: map[string][]string{},
		},
		{
			name:    "a root re-homed to a third Node receives there",
			world:   newWorld().add(1, seedbox, onNas, gb).dest(onNas, func(d *roots.Destination) { d.Node, d.Place = vps, 2 }),
			want:    map[byte]string{1: "waiting connecting seedbox route 1"},
			actives: map[string][]string{"seedbox": {"send 1"}, "vps": {"receive 1"}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			j := judge(c.world.s)
			got := map[byte]string{}
			for _, v := range j.views {
				got[v.ID.Bytes()[15]] = state(v)
			}
			if !maps.Equal(got, c.want) {
				t.Errorf("states = %v, want %v", got, c.want)
			}
			if c.actives != nil {
				if a := actives(j); !maps.EqualFunc(a, c.actives, slices.Equal) {
					t.Errorf("active = %v, want %v", a, c.actives)
				}
			}
		})
	}
}

func (w *world) with(change func(*snapshot)) *world {
	change(&w.s)
	return w
}
