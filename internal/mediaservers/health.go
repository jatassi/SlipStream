package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

var HealthAbout = health.About{
	ID:     health.MediaServersID,
	Label:  "Media Servers",
	Phrase: "Media Servers",
	Events: []live.Event{ChangedEvent, UnhealthyEvent, RecoveredEvent, roots.ChangedEvent, nodes.ChangedEvent},
}

// HealthSource answers each server's health as the sender judged it, and
// each root it would show with no confirmed mapping. An unhealthy server
// is an Error. While its reaching Node is away the server is a Warning
// when healthy, and an Error item keeps its level, as its Node can't
// check it.
func (m *MediaServers) HealthSource() health.Source {
	return health.Source{Answer: m.healthAnswer}
}

// serverHealthKey is server id's Health item.
func serverHealthKey(id link.ServerID) health.Key { return health.KeyOf("server", id.String()) }

// mappingHealthKey is the Health item of root without a confirmed mapping
// on server id.
func mappingHealthKey(id link.ServerID, root roots.ID) health.Key {
	return health.KeyOf("mapping", id.String(), root.String())
}

func (m *MediaServers) healthAnswer(ctx context.Context) (health.Answer, error) {
	var (
		w    world
		rows []db.MediaserversServer
	)
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if w, err = m.world(ctx, r); err != nil {
			return err
		}
		rows, err = db.New(r).ListServers(ctx)
		return err
	})
	if err != nil {
		return health.Answer{}, err
	}
	var a health.Answer
	for _, row := range rows {
		k, _ := m.cfg.Kinds.Of(row.Kind)
		key := serverHealthKey(row.ID)
		node := w.names[row.ReachingNodeID].Name
		offline := health.Says{Subject: row.Name, Text: offlineWords(row, node), Detail: m.serverFacts(row, node, ""), Action: serverLink(row)}
		switch held := heldOf(row); {
		case held.reason != "" && m.away(row.ReachingNodeID):
			a.Unsure = append(a.Unsure, health.Unsure{Key: key, Says: offline})
		case held.reason != "":
			a.Items = append(a.Items, health.Item{Key: key, Level: health.Fail, Says: health.Says{
				Subject: row.Name, Text: m.unhealthyWords(row, k, w, held.reason), Detail: m.serverFacts(row, node, held.reason), Action: serverLink(row),
			}})
		case m.away(row.ReachingNodeID):
			a.Items = append(a.Items, health.Item{Key: key, Level: health.Warn, Says: offline})
		}
		for _, root := range w.needed(row.ID) {
			a.Items = append(a.Items, health.Item{Key: mappingHealthKey(row.ID, root), Level: health.Warn, Says: health.Says{
				Subject: w.words[root] + " on " + row.Name,
				Text:    noMappingWords(row.Name, w.words[root]),
				Action:  health.LinkAction("/settings/media-servers/"+row.ID.String()+"/mappings", "Edit Path mappings"),
			}})
		}
	}
	return a, nil
}

func (m *MediaServers) unhealthyWords(row db.MediaserversServer, k link.Kind, w world, reason string) string {
	switch reason {
	case reasonOtherServer:
		return otherServerWords(row, k)
	case reasonSignInRefused:
		return signInRefusedWords(row, k)
	case reasonSendsFailing:
		return m.sendsFailingWords(row, k)
	}
	return m.notRespondingWords(row, k, w)
}

// serverFacts are the server's facts under Details, with those of reason
// when it is unhealthy.
func (m *MediaServers) serverFacts(row db.MediaserversServer, node, reason string) []health.Fact {
	out := []health.Fact{{Label: "Address", Text: row.Address}, {Label: "Reaching Node", Text: node}, {Label: "Version", Text: row.Version}}
	at := func(label string, t storage.Time) {
		if !t.IsZero() {
			out = append(out, health.Fact{Label: label, At: t.Time()})
		}
	}
	switch reason {
	case reasonOtherServer:
		out = append(out, health.Fact{Label: "Expected server", Text: row.Identity}, health.Fact{Label: "Responding server", Text: deref(row.ContactFound)})
	case reasonNotResponding:
		at("Last response", row.LastContactAt)
	case reasonSendsFailing:
		at("Last change accepted", row.LastSendAt)
	}
	at("Unhealthy since", row.UnhealthySince)
	return out
}

func serverLink(row db.MediaserversServer) health.Action {
	return health.LinkAction("/settings/media-servers/"+row.ID.String(), "Open "+row.Name)
}
