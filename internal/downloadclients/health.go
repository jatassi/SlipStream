package downloadclients

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
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

// HealthSource answers what is wrong with each downloading Node's
// clients. Its built-in client fails while its listener won't bind, and
// warns while the disk guard holds new downloads. Each external client
// fails while it is away. A Node out of step is unsure of its listener
// and its external clients, and one not reported since the Controller
// started is unsure of its listener, so a blip neither raises nor clears
// them. Check is Check now on the external clients.
func (c *Clients) HealthSource() health.Source {
	return health.Source{Answer: c.healthAnswer, Check: c.healthCheck}
}

func (c *Clients) healthAnswer(ctx context.Context) (health.Answer, error) {
	var (
		guards    map[role.NodeID]Guard
		names     map[role.NodeID]string
		externals []External
	)
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		if guards, err = guardsIn(ctx, rd); err != nil {
			return err
		}
		if externals, err = externalsIn(ctx, rd); err != nil {
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
				Text: fmt.Sprintf("The torrent client on %s can't listen on %s, and no peer can connect to it until it does. If another program is using the port, free it or choose another, then restart %s.",
					name, f.GetListen(), name),
				Detail: []health.Fact{{Label: "Listen address", Text: f.GetListen()}, {Label: "Error", Text: f.GetListenError()}},
				Action: clientLink(node),
			}})
		}
		if g, on := guards[node]; on {
			a.Items = append(a.Items, guardItem(node, name, g))
		}
		for _, e := range externals {
			if e.Node != node {
				continue
			}
			switch reach := ReachOf(e, c.cfg.Roster.InStep(node)); {
			case reach == ReachNodeOffline:
				a.Unsure = append(a.Unsure, health.Unsure{Key: ClientHealthKey(e.ID), Says: health.Says{Subject: e.Name + " on " + name, Text: name + " is offline."}})
			case reach.Away():
				a.Items = append(a.Items, clientItem(e, name))
			}
		}
	}
	return a, nil
}

// ClientHealthKey is external client id's Health item.
func ClientHealthKey(id ClientID) health.Key { return health.KeyOf(id.String(), "client") }

// clientItem is the Health error of away client e on its Node named node:
// what happened and what to do first, and since when.
func clientItem(e External, node string) health.Item {
	p := e.Poll
	text := Says(*p.Problem, e.Subject(), node)
	if p.State == clients.SignInRefused && e.Kind.Facts().Lockout && p.Problem.Cause == clients.WrongCredentials {
		text = Stopped(e.Name)
	}
	detail := []health.Fact{{Label: "Address", Text: e.Address.String()}}
	if p.Version != "" {
		detail = append(detail, health.Fact{Label: "Version", Text: p.Version})
	}
	detail = append(detail, health.Fact{Label: "Since", At: p.Since})
	return health.Item{Key: ClientHealthKey(e.ID), Level: health.Fail, Says: health.Says{
		Subject: e.Name + " on " + node,
		Text:    text,
		Detail:  detail,
		Action:  health.LinkAction("/settings/download-clients?client="+e.ID.String(), "Edit "+e.Name),
	}}
}

// externalsIn is every external client, without its secrets.
func externalsIn(ctx context.Context, rd storage.Reader[file]) ([]External, error) {
	rows, err := db.New(rd).ListClients(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]External, len(rows))
	for i, row := range rows {
		if out[i], err = externalOf(row, nil); err != nil {
			return nil, err
		}
	}
	return out, nil
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
