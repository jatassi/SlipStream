package mediaservers

import (
	"context"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// world is what a server's view reads beside its row.
type world struct {
	names    map[role.NodeID]nodes.Named
	roots    []roots.Root
	words    map[roots.ID]string
	mappings map[link.ServerID]map[roots.ID]db.MediaserversMapping
	plex     account
}

func (m *MediaServers) world(ctx context.Context, r storage.Reader[storage.Library]) (world, error) {
	var w world
	var err error
	if w.names, err = m.names(ctx, r); err != nil {
		return w, err
	}
	if w.roots, err = roots.All(ctx, r); err != nil {
		return w, err
	}
	if w.words, err = m.cfg.RootWords(ctx, r); err != nil {
		return w, err
	}
	if w.plex, _, err = m.readAccount(ctx, db.New(r)); err != nil {
		return w, err
	}
	all, err := db.New(r).ListAllMappings(ctx)
	w.mappings = map[link.ServerID]map[roots.ID]db.MediaserversMapping{}
	for _, mp := range all {
		if w.mappings[mp.ServerID] == nil {
			w.mappings[mp.ServerID] = map[roots.ID]db.MediaserversMapping{}
		}
		w.mappings[mp.ServerID][mp.RootID] = mp
	}
	return w, err
}

// needed is each root a server would show but has no confirmed mapping
// for, in root order.
func (w world) needed(id link.ServerID) []roots.ID {
	var out []roots.ID
	for _, root := range w.roots {
		mp, ok := w.mappings[id][root.ID]
		if !ok || (mp.Shown && mp.ConfirmedAt.IsZero()) {
			out = append(out, root.ID)
		}
	}
	return out
}

func (s service) List(ctx context.Context) ([]ServerView, error) {
	out := []ServerView{}
	err := s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		w, err := s.m.world(ctx, r)
		if err != nil {
			return err
		}
		rows, err := db.New(r).ListServers(ctx)
		for _, row := range rows {
			out = append(out, s.m.viewOf(row, w))
		}
		return err
	})
	return out, err
}

func (s service) Get(ctx context.Context, id link.ServerID) (ServerView, error) {
	var v ServerView
	err := s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		w, err := s.m.world(ctx, r)
		if err != nil {
			return err
		}
		sv, err := s.m.load(ctx, r, id)
		v = s.m.viewOf(sv.row, w)
		return err
	})
	return v, err
}

func timeOrNil(t storage.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	v := t.Time()
	return &v
}

func (m *MediaServers) viewOf(row db.MediaserversServer, w world) ServerView {
	k, _ := m.cfg.Kinds.Of(row.Kind)
	v := ServerView{
		ID: row.ID.String(), Kind: KindOf(row.Kind), Name: row.Name, Address: row.Address,
		FallbackAddress: deref(row.FallbackAddress), BrowserAddress: deref(row.BrowserAddress),
		Version: row.Version, Identity: row.Identity,
		ReachingNode: NodeRef{ID: row.ReachingNodeID.String(), Name: w.names[row.ReachingNodeID].Name},
		Fallback:     row.Fallback, CheckedAt: timeOrNil(row.CheckedAt),
		Contact: string(row.Contact), ContactSince: timeOrNil(row.ContactSince), Sends: row.Sends,
		LastContactAt: timeOrNil(row.LastContactAt), LastSendAt: timeOrNil(row.LastSendAt),
	}
	if !row.RunsOnNodeID.IsZero() {
		v.RunsOnNode = &NodeRef{ID: row.RunsOnNodeID.String(), Name: w.names[row.RunsOnNodeID].Name}
	}
	if row.UnhealthyReason != nil {
		v.Unhealthy = &Unhealthy{Reason: *row.UnhealthyReason, Since: row.UnhealthySince.Time()}
	}
	if k.About().Form == link.SignInPlexPIN {
		v.PlexPass = &w.plex.plexPass
	}
	needed := w.needed(row.ID)
	v.MappingsNeeded = len(needed)
	v.State = m.stateOf(row, k, w, needed)
	return v
}

// stateOf is the server's one line on the list, leading with what to do:
// what stops changes first, then what the reaching Node last saw.
func (m *MediaServers) stateOf(row db.MediaserversServer, k link.Kind, w world, needed []roots.ID) State {
	node := w.names[row.ReachingNodeID].Name
	switch {
	case row.Contact == link.ContactOtherServer:
		return State{Level: "fail", Words: otherServerWords(row, k)}
	case row.Contact == link.ContactSignInRefused || (k.About().Form == link.SignInPlexPIN && w.plex.refused):
		return State{Level: "fail", Words: signInRefusedWords(row, k)}
	case row.UnhealthyReason != nil && *row.UnhealthyReason == reasonSendsFailing:
		return State{Level: "fail", Words: m.sendsFailingWords(row, k)}
	case m.away(row.ReachingNodeID):
		return State{Level: "warn", Words: offlineWords(row, node)}
	case row.Contact == link.ContactNotResponding:
		return State{Level: "fail", Words: m.notRespondingWords(row, k, w)}
	case len(needed) == 1:
		return State{Level: "warn", Words: noMappingWords(row.Name, w.words[needed[0]])}
	case len(needed) > 1:
		return State{Level: "warn", Words: fmt.Sprintf("%s won't get changes for files in %d library roots. Add their Path mappings, or mark them as not shown on this server.", row.Name, len(needed))}
	case row.Contact == link.ContactResponding:
		return State{Level: "ok", Words: fmt.Sprintf("%s responds to %s.", row.Name, node)}
	}
	return State{Level: "ok", Words: fmt.Sprintf("%s hasn't checked %s yet.", node, row.Name)}
}

// away is a reaching Node's link down. The Controller is never away.
func (m *MediaServers) away(node role.NodeID) bool { return !m.up(node) }

func (m *MediaServers) clock(t storage.Time) string { return t.Time().In(m.cfg.Zone).Format("15:04") }

func otherServerWords(row db.MediaserversServer, k link.Kind) string {
	words, _ := Words(&OtherServerError{Label: k.About().Label, Address: row.Address})
	return words
}

func signInRefusedWords(row db.MediaserversServer, k link.Kind) string {
	with := RefusedKey
	if k.About().Form == link.SignInPlexPIN {
		with = RefusedPlex
	}
	return refusedWords(&SignInRefusedError{Server: row.Name, With: with})
}

func (m *MediaServers) sendsFailingWords(row db.MediaserversServer, k link.Kind) string {
	return fmt.Sprintf("%s hasn't accepted changes since %s. SlipStream retries automatically. Check %s's log for the cause.",
		row.Name, m.clock(row.UnhealthySince), k.About().Label)
}

func (m *MediaServers) notRespondingWords(row db.MediaserversServer, k link.Kind, w world) string {
	node := w.names[row.ReachingNodeID].Name
	running := k.About().Label + " is running"
	if !row.RunsOnNodeID.IsZero() {
		running += " on " + w.names[row.RunsOnNodeID].Name
	}
	return fmt.Sprintf("%s hasn't responded to %s since %s. Check that %s and that %s can reach %s.",
		row.Name, node, m.clock(row.ContactSince), running, node, hostOf(row.Address))
}

func offlineWords(row db.MediaserversServer, node string) string {
	if row.Fallback == "available" {
		return fmt.Sprintf("%s can't be checked while %s is offline. SlipStream sends changes through the Controller until %s is back.", row.Name, node, node)
	}
	return fmt.Sprintf("SlipStream holds changes for %s until %s is back.", row.Name, node)
}

func noMappingWords(server, root string) string {
	return fmt.Sprintf("%s won't get changes for files in %s. Add a Path mapping, or mark the root as not shown on this server.", server, root)
}
