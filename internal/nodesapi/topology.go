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
	Self       bool       `json:"self" doc:"The Node answering, which is the Controller"`
	Controller bool       `json:"controller"`
	State      string     `json:"state" enum:"online,reconnecting,offline,lapsed,mismatched"`
	Since      time.Time  `json:"since" doc:"When it came in step; while away, its last Sync frame"`
	LastSeenAt *time.Time `json:"lastSeenAt,omitempty" doc:"Its last Sync frame; absent while it is online"`
	Roles      []roleView `json:"roles" nullable:"false" doc:"Every Role the Node stands in, in table order"`
	Resources  resources  `json:"resources"`
}

type resources struct {
	Downloading *downloadingResource `json:"downloading,omitempty" doc:"Its torrent client; absent unless the Node stands in Downloading and is in step"`
}

type downloadingResource struct {
	Torrents []torrentView `json:"torrents" nullable:"false" doc:"Each Grab whose torrent the client keeps"`
}

type torrentView struct {
	GrabID string `json:"grabId" format:"uuid"`
	Title  string `json:"title"`
	Done   int64  `json:"done" doc:"Bytes downloaded"`
	Wanted int64  `json:"wanted" doc:"Bytes the torrent's selected files hold"`
	Peers  int    `json:"peers"`
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
			State: nodeStates[n.Standing], Since: n.Since, Roles: rolesOf(n.Roles),
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
					GrabID: tr.Grab.String(), Title: tr.Title, Done: tr.Done, Wanted: tr.Wanted, Peers: tr.Peers,
				})
			}
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

func (h handlers) getTopology(ctx context.Context, _ identity.Admin, _ *struct{}) (*topologyOutput, error) {
	t, err := open(h.roster).Topology(ctx)
	if err != nil {
		return nil, err
	}
	return &topologyOutput{Body: topologyOf(t)}, nil
}
