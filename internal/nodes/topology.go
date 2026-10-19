package nodes

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/link"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// Topology is every Node and the link between the Controller and each
// Worker at one instant, as the canvas draws them. Rates travel apart,
// in nodes.traffic.
type Topology struct {
	At time.Time
	// Sentence says what stands out, in plain words.
	Sentence string
	Nodes    []TopologyNode
	Links    []TopologyLink
}

type TopologyNode struct {
	NodeView
	Roles []RoleView
	// Probes is the Node's last probe verdicts, kept while it is away.
	Probes []role.ProbeResult
	// Downloading is its client's torrents, nil unless it is in step and
	// stands in Downloading.
	Downloading *Downloading
	// Storage is its roots and their volumes, nil unless it stands in
	// Storage; while it is away, as it last reported them.
	Storage *StorageFigures
}

// StorageFigures is what a Storage Node holds: its roots, and the volumes
// they sit on as it last reported them.
type StorageFigures struct {
	Roots   []StorageRoot
	Volumes []storing.Volume
}

type StorageRoot struct {
	ID          storage.UUID
	ContentType string
	Path        string
	// State is the root's state as the Library roots section names it.
	State string
	// VolumeID is empty before its Node found it ready at its place.
	VolumeID string
}

type Downloading struct {
	Torrents []Torrent
}

// Torrent is one Grab whose torrent a Node's client keeps.
type Torrent struct {
	Grab   storage.UUID
	Title  string
	Done   int64
	Wanted int64
	Peers  int
}

// TopologyLink is the link between the Controller and Worker.
type TopologyLink struct {
	Worker ID
	// Dialer is the Node that dialed the live session, zero while down.
	Dialer ID
	// State is Up, Down or Mismatched.
	State LinkState
	// RTT is zero until the live session's first ping is answered.
	RTT time.Duration
}

func (s *Service) Topology(ctx context.Context) (Topology, error) {
	board := s.Board()
	t := Topology{At: time.Now(), Sentence: sentence(board)}
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		for _, v := range board.nodes {
			n, err := s.topologyNode(ctx, r, board, v)
			if err != nil {
				return err
			}
			t.Nodes = append(t.Nodes, n)
			if !v.Controller {
				t.Links = append(t.Links, s.linkTo(v.ID))
			}
		}
		return nil
	})
	if err != nil {
		return Topology{}, fmt.Errorf("read the topology: %w", err)
	}
	return t, nil
}

func (s *Service) topologyNode(ctx context.Context, r storage.Reader[file], board *Board, v NodeView) (TopologyNode, error) {
	held, _, err := s.work(ctx, r, v.ID)
	if err != nil {
		return TopologyNode{}, err
	}
	n := TopologyNode{NodeView: v, Roles: board.Roles(v.ID, held), Probes: s.probesOf(v.ID)}
	if v.Standing == InStep && board.Who(role.Downloading).Serves(v.ID) {
		kept, err := s.cfg.Kept(ctx, r, v.ID)
		if err != nil {
			return TopologyNode{}, fmt.Errorf("read the Grabs node %s keeps: %w", v.ID, err)
		}
		f, _ := s.cfg.Downloads.Facts(v.ID)
		n.Downloading = &Downloading{Torrents: torrentsOf(f, kept)}
	}
	if board.Who(role.Storage).Serves(v.ID) {
		f, err := s.cfg.Storage(ctx, r, v.ID)
		if err != nil {
			return TopologyNode{}, fmt.Errorf("read node %s's Storage: %w", v.ID, err)
		}
		n.Storage = &f
	}
	return n, nil
}

// torrentsOf is each Grab of kept whose torrent f reports, in f's order.
func torrentsOf(f *dlv1.Facts, kept []Grabbed) []Torrent {
	out := []Torrent{}
	for _, t := range f.GetTorrents() {
		h := bencode.InfoHash(t.GetInfoHash())
		for _, g := range kept {
			if g.InfoHash == h {
				out = append(out, Torrent{Grab: g.Grab, Title: g.Title, Done: t.GetDone(), Wanted: t.GetWanted(), Peers: int(t.GetPeers())})
			}
		}
	}
	return out
}

func (s *Service) linkTo(worker ID) TopologyLink {
	l := TopologyLink{Worker: worker, State: Down}
	key, ok := s.Key(worker)
	if !ok {
		return l
	}
	up, live := s.cfg.Link.State(key).(link.Live)
	if !live {
		return l
	}
	l.State, l.Dialer = Up, worker
	if !up.Matched {
		l.State = Mismatched
	}
	if up.Dialed {
		l.Dialer = s.self
	}
	if st, ok := s.cfg.Link.Stats(key); ok {
		l.RTT = st.RTT
	}
	return l
}

// sentence says what stands out on b: each Node out of step or settling,
// the Roles the Controller covers in their place, and each Node draining
// or writing off a Role. With nothing to say, it says every Node is
// online.
func sentence(b *Board) string {
	var parts []string
	controller := ""
	for _, v := range b.nodes {
		if v.Controller {
			controller = v.Name
			continue
		}
		switch {
		case v.Settling:
			parts = append(parts, v.Name+" is back and settling.")
		case v.Standing == Reconnecting:
			parts = append(parts, v.Name+" is reconnecting.")
		case v.Standing == Offline, v.Standing == LapsedNode:
			parts = append(parts, v.Name+" is offline.")
		case v.Standing == MismatchedNode:
			parts = append(parts, v.Name+" runs another version.")
		case v.Standing == InStep:
		}
	}
	if covered := b.covering(); covered != 0 {
		parts = append(parts, controller+" is "+doingAll(covered)+" in "+placeOf(b, covered)+" place.")
	}
	for i, v := range b.nodes {
		var draining, writingOff []role.Name
		for _, r := range role.Names() {
			switch b.states[i][r] {
			case Draining:
				draining = append(draining, r)
			case WritingOff:
				writingOff = append(writingOff, r)
			case NoState, Holding, Away, Lapsed, Covering:
			}
		}
		if len(draining) > 0 {
			parts = append(parts, v.Name+" is draining "+labels(draining)+".")
		}
		if len(writingOff) > 0 {
			parts = append(parts, v.Name+" is writing off "+labels(writingOff)+".")
		}
	}
	switch {
	case len(parts) > 0:
		return strings.Join(parts, " ")
	case len(b.nodes) == 1:
		return controller + " is the only Node."
	case len(b.nodes) == 2:
		return "Both Nodes are online."
	}
	return "All " + strconv.Itoa(len(b.nodes)) + " Nodes are online."
}

// placeOf is "its" when the Roles the Controller covers belong to one
// Node, else "their".
func placeOf(b *Board, covered role.Set) string {
	var holder ID
	for i, v := range b.nodes {
		for _, r := range covered.Names() {
			if st := b.states[i][r]; st != Holding && st != Lapsed {
				continue
			}
			if !holder.IsZero() && holder != v.ID {
				return "their"
			}
			holder = v.ID
		}
	}
	return "its"
}

func doingAll(set role.Set) string {
	var out []string
	for _, r := range set.Names() {
		out = append(out, doing(r))
	}
	return and(out)
}

func labels(rs []role.Name) string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Label()
	}
	return and(out)
}

func and(words []string) string {
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

// doing is what a holder of r is doing, from its purpose: "searches"
// becomes "searching", "stores the library" "storing the library".
func doing(r role.Name) string {
	verb, rest, _ := strings.Cut(r.Purpose(), " ")
	switch {
	case strings.HasSuffix(verb, "ches"), strings.HasSuffix(verb, "shes"):
		verb = strings.TrimSuffix(verb, "es")
	default:
		verb = strings.TrimSuffix(verb, "s")
	}
	verb = strings.TrimSuffix(verb, "e")
	if rest != "" {
		rest = " " + rest
	}
	return verb + "ing" + rest
}
