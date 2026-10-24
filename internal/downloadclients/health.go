package downloadclients

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

var HealthAbout = health.About{
	ID:     health.DownloadClientsID,
	Label:  "Download Clients",
	Phrase: "download clients",
	Events: []live.Event{ClientsChangedEvent},
}

// HealthVolumes is each downloading Node's downloads volume as its client
// last sampled it, which Storage judges like a root's. An offline Node's
// volume keeps its last figures.
func (c *Clients) HealthVolumes() health.VolumeSource {
	return health.VolumeSource{Volumes: c.healthVolumes, Sample: c.sampleVolume, Events: []live.Event{ClientsChangedEvent}, Remedy: "move the downloads folder to another volume"}
}

func (c *Clients) healthVolumes(ctx context.Context) ([]health.Volume, error) {
	serving := map[role.NodeID]bool{}
	for _, n := range c.cfg.Roster.Serving(role.Downloading) {
		serving[n] = true
	}
	var out []health.Volume
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		vols, err := volumesIn(ctx, rd)
		if err != nil {
			return err
		}
		names, err := namesIn(ctx, rd)
		if err != nil {
			return err
		}
		for _, v := range vols {
			if serving[v.Node] {
				out = append(out, health.Volume{
					Node: v.Node, NodeName: names[v.Node], Mount: v.Mount, Name: v.Name,
					Total: v.Total, Free: v.Free, SampledAt: v.SampledAt, Live: c.cfg.Roster.InStep(v.Node),
				})
			}
		}
		return nil
	})
	return out, err
}

// HealthSource answers what is wrong with each downloading Node's built-in
// client: a listener that won't bind fails, and the disk guard holding new
// downloads warns. A Node out of step, or not reported since the
// Controller started, is unsure of its listener, so a blip neither raises
// nor clears it.
func (c *Clients) HealthSource() health.Source {
	return health.Source{Answer: c.healthAnswer}
}

func (c *Clients) healthAnswer(ctx context.Context) (health.Answer, error) {
	var (
		guards map[role.NodeID]Guard
		names  map[role.NodeID]string
	)
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		if guards, err = guardsIn(ctx, rd); err != nil {
			return err
		}
		names, err = namesIn(ctx, rd)
		return err
	})
	if err != nil {
		return health.Answer{}, err
	}
	var a health.Answer
	for _, node := range c.cfg.Roster.Serving(role.Downloading) {
		name := names[node]
		key := health.KeyOf(node.String(), "listen")
		subject := "Torrent client on " + name
		f, reported := c.cfg.Facts(node)
		switch {
		case !c.cfg.Roster.InStep(node):
			a.Unsure = append(a.Unsure, health.Unsure{Key: key, Says: health.Says{Subject: subject, Text: name + " is offline."}})
		case !reported:
			a.Unsure = append(a.Unsure, health.Unsure{Key: key, Says: health.Says{Subject: subject, Text: name + "'s torrent client hasn't reported since SlipStream started."}})
		case f.GetBound() == "" && f.GetListenError() != "":
			a.Items = append(a.Items, health.Item{Key: key, Level: health.Fail, Says: health.Says{
				Subject: subject,
				Text: fmt.Sprintf("The torrent client on %s can't listen on %s, so no peer can connect in. If another program holds the port, free it or choose another, then restart %s.",
					name, f.GetListen(), name),
				Detail: []health.Fact{{Label: "Listen address", Text: f.GetListen()}, {Label: "Error", Text: f.GetListenError()}},
				Action: clientLink(node),
			}})
		}
		if g, on := guards[node]; on {
			a.Items = append(a.Items, guardItem(node, name, g))
		}
	}
	return a, nil
}

func guardItem(node role.NodeID, name string, g Guard) health.Item {
	v := g.Volume
	sentence, hint := g.Words(name)
	return health.Item{Key: health.KeyOf(node.String(), "disk-guard"), Level: health.Warn, Says: health.Says{
		Subject: "New downloads on " + name,
		Text:    sentence + " " + hint,
		Detail: []health.Fact{
			{Label: "Volume", Text: v.Name},
			{Label: "Free", Bytes: &v.Free},
			{Label: "Total", Bytes: &v.Total},
			{Label: "Error level", Bytes: &g.ErrorBelow.Bytes},
			{Label: "Sampled", At: v.SampledAt},
		},
		Action: clientLink(node),
	}}
}

func clientLink(node role.NodeID) health.Action {
	return health.LinkAction("/nodes/"+node.String()+"/torrent-client", "Open Torrent client")
}

func namesIn(ctx context.Context, rd storage.Reader[file]) (map[role.NodeID]string, error) {
	named, err := nodes.Names(ctx, rd)
	if err != nil {
		return nil, err
	}
	out := make(map[role.NodeID]string, len(named))
	for _, n := range named {
		out[n.ID] = n.Name
	}
	return out, nil
}
