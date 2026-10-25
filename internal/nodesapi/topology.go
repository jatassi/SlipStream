package nodesapi

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
)

type topology struct {
	At       time.Time      `json:"at"`
	Sentence string         `json:"sentence" doc:"What stands out across the Nodes, in plain words"`
	Nodes    []topologyNode `json:"nodes" nullable:"false" doc:"Every Node, in id order"`
	Links    []topologyLink `json:"links" nullable:"false" doc:"The link between the Controller and each Worker, in the Workers' id order"`
}

type topologyOutput struct {
	Body topology
}

type topologyNode struct {
	ID         string     `json:"id" format:"uuid"`
	Name       string     `json:"name"`
	Self       bool       `json:"self" doc:"The Node that responded, which is the Controller"`
	Controller bool       `json:"controller"`
	State      string     `json:"state" enum:"online,reconnecting,offline,lapsed,mismatched"`
	Since      time.Time  `json:"since" doc:"When it came in step; while away, its last Sync frame"`
	LastSeenAt *time.Time `json:"lastSeenAt,omitempty" doc:"Its last Sync frame; absent while it is online"`
	Roles      []roleView `json:"roles" nullable:"false" doc:"Every Role the Node stands in, in table order"`
	Probes     []probe    `json:"probes" nullable:"false" doc:"The last verdict of each probe the Node ran, kept while it is away"`
	Resources  resources  `json:"resources"`
}

type resources struct {
	Downloading *downloadingResource `json:"downloading,omitempty" doc:"Its torrent client; absent unless the Node stands in Downloading and is in step"`
	Storage     *storageResource     `json:"storage,omitempty" doc:"Its library roots and the volumes they sit on; absent unless the Node stands in Storage. While it is away, as it last reported them"`
}

type storageResource struct {
	Roots   []storageRoot   `json:"roots" nullable:"false" doc:"The library roots on the Node, by Content Type, oldest first in each"`
	Volumes []storageVolume `json:"volumes" nullable:"false" doc:"The volumes its roots sit on"`
}

type storageRoot struct {
	ID          string `json:"id" format:"uuid"`
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Path        string `json:"path" doc:"The folder as its Node cleaned it"`
	State       string `json:"state" enum:"ready,missing,unmounted,foreign,unwritable,unresponsive,checking,offline,written_off" doc:"As listRoots names it"`
	VolumeID    string `json:"volumeId,omitempty" doc:"The volume its Node last found it ready on; absent before it did"`
}

type storageVolume struct {
	ID        string    `json:"id" doc:"The filesystem id, or the volume serial on Windows"`
	Name      string    `json:"name" doc:"The volume as its Node names it, such as D: (Media) or /mnt/media"`
	Mount     string    `json:"mount"`
	Kind      string    `json:"kind" enum:"local,removable,network"`
	Total     int64     `json:"total"`
	Free      int64     `json:"free" doc:"The bytes SlipStream may write"`
	SampledAt time.Time `json:"sampledAt" doc:"When its Node read these figures"`
}

type downloadingResource struct {
	Torrents []torrentView `json:"torrents" nullable:"false" doc:"Each Torrent the client keeps, in added order"`
}

type torrentView struct {
	TorrentID string `json:"torrentId" format:"uuid"`
	Title     string `json:"title" doc:"Its newest Grab's release, or its name once no Grab is left"`
	Done      int64  `json:"done" doc:"Bytes downloaded, live while an Admin listens"`
	Wanted    int64  `json:"wanted" doc:"Bytes the torrent's selected files hold"`
	Peers     int    `json:"peers" doc:"0 while its Node isn't sampled"`
}

type topologyLink struct {
	A      string `json:"a" format:"uuid" doc:"The Controller"`
	B      string `json:"b" format:"uuid" doc:"The Worker"`
	Dialer string `json:"dialer,omitempty" format:"uuid" doc:"The Node that dialed the live session; absent while the link is down"`
	State  string `json:"state" enum:"up,down,mismatched"`
	RTTMs  int64  `json:"rttMs,omitempty" doc:"The live session's round trip; absent until its first ping is answered"`
}

var nodeStates = map[nodes.Standing]string{
	nodes.InStep:         "online",
	nodes.Reconnecting:   "reconnecting",
	nodes.Offline:        "offline",
	nodes.LapsedNode:     "lapsed",
	nodes.MismatchedNode: "mismatched",
}

func topologyOf(t nodes.Topology) topology {
	out := topology{At: t.At, Sentence: t.Sentence, Nodes: []topologyNode{}, Links: []topologyLink{}}
	var controller string
	for _, n := range t.Nodes {
		v := topologyNode{
			ID: n.ID.String(), Name: n.Name, Self: n.Controller, Controller: n.Controller,
			State: nodeStates[n.Standing], Since: n.Since, Roles: rolesOf(n.Roles), Probes: probesOf(n.Probes),
		}
		if n.Controller {
			controller = v.ID
		}
		if n.Standing != nodes.InStep {
			v.LastSeenAt = &n.Since
		}
		if d := n.Downloading; d != nil {
			v.Resources.Downloading = &downloadingResource{Torrents: []torrentView{}}
			for _, tr := range d.Torrents {
				v.Resources.Downloading.Torrents = append(v.Resources.Downloading.Torrents, torrentView{
					TorrentID: tr.ID.String(), Title: tr.Title, Done: tr.Done, Wanted: tr.Wanted, Peers: tr.Peers,
				})
			}
		}
		if st := n.Storage; st != nil {
			v.Resources.Storage = storageOf(*st)
		}
		out.Nodes = append(out.Nodes, v)
	}
	for _, l := range t.Links {
		v := topologyLink{A: controller, B: l.Worker.String(), State: l.State.String(), RTTMs: l.RTT.Milliseconds()}
		if !l.Dialer.IsZero() {
			v.Dialer = l.Dialer.String()
		}
		out.Links = append(out.Links, v)
	}
	return out
}

func storageOf(f nodes.StorageFigures) *storageResource {
	out := &storageResource{Roots: []storageRoot{}, Volumes: []storageVolume{}}
	for _, r := range f.Roots {
		out.Roots = append(out.Roots, storageRoot{ID: r.ID.String(), ContentType: r.ContentType, Path: r.Path, State: r.State, VolumeID: r.VolumeID})
	}
	for _, v := range f.Volumes {
		out.Volumes = append(out.Volumes, storageVolume{
			ID: v.ID, Name: v.Name, Mount: v.Mount, Kind: v.Kind.String(), Total: v.Total, Free: v.Free, SampledAt: v.SampledAt,
		})
	}
	return out
}

func (h handlers) getTopology(ctx context.Context, _ identity.Admin, _ *struct{}) (*topologyOutput, error) {
	t, err := open(h.roster).Topology(ctx)
	if err != nil {
		return nil, err
	}
	return &topologyOutput{Body: topologyOf(t)}, nil
}
