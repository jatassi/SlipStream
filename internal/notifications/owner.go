package notifications

import (
	"context"
	"maps"
	"net/url"
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Owner is the Notifying Role's owner. It keeps no work. A Node that can't
// reach the mail server can't gain the Role, one that can't reach a
// connection's host gains it with a warning, and a Worker a Custom Script
// runs on keeps the Role until the script moves. The Controller runs its
// own scripts whatever Roles it has, since the Role lapses to it.
func (s *Service) Owner() nodes.Owner {
	return nodes.Owner{
		Role: role.Notifying, Held: nodes.HoldsNothing, Unassigned: nodes.MovesNothing,
		Reach: s.reachTargets, Keeps: s.scriptsOn,
	}
}

// reachTargets is the mail server, then the host of each connection that
// is on, one target per host and port. A host the mail server shares
// blocks; every other one only warns.
func (s *Service) reachTargets(ctx context.Context, r storage.Reader[storage.Library]) ([]nodes.Target, error) {
	var out []nodes.Target
	m, err := mail.ReadSettings(ctx, r)
	if err != nil {
		return nil, err
	}
	if m.Host != "" {
		out = append(out, nodes.Target{Target: nodesystem.Target{Host: m.Host, Port: m.Port, TLS: m.Security == mail.TLS}, What: "the mail server", Block: true})
	}
	conns, err := connectionsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	for _, c := range conns {
		if !c.On {
			continue
		}
		t, ok := s.hostOf(c)
		if !ok {
			continue
		}
		if i := slices.IndexFunc(out, func(o nodes.Target) bool { return o.Target == t }); i >= 0 {
			if !out[i].Block {
				out[i].What += " and " + s.label(c)
			}
			continue
		}
		out = append(out, nodes.Target{Target: t, What: s.label(c)})
	}
	return out, nil
}

// hostOf is the host and port c sends to. A connection whose secrets don't
// open has none: its line is paused until the Admin enters them again.
func (s *Service) hostOf(c Connection) (nodesystem.Target, bool) {
	secrets, err := s.open(c.ID, c.Sealed)
	if err != nil {
		return nodesystem.Target{}, false
	}
	values := maps.Clone(c.Settings)
	maps.Copy(values, secrets)
	u, ok := s.provider(c.Provider).Host(values, s.cfg.Endpoints)
	if !ok {
		return nodesystem.Target{}, false
	}
	return targetOf(u)
}

// targetOf is u's host and port, the scheme's port when u names none.
func targetOf(u *url.URL) (nodesystem.Target, bool) {
	t := nodesystem.Target{Host: u.Hostname(), TLS: u.Scheme == "https"}
	switch p := u.Port(); {
	case p != "":
		n, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return nodesystem.Target{}, false
		}
		t.Port = uint16(n)
	case t.TLS:
		t.Port = 443
	default:
		t.Port = 80
	}
	return t, t.Host != ""
}

// scriptsOn names the Custom Script connections that run on node, by
// name, and none on the Controller.
func (s *Service) scriptsOn(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) ([]string, error) {
	if node == s.cfg.Roster.Self() {
		return nil, nil
	}
	rows, err := db.New(r).ListScriptsOn(ctx, node)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i, row := range rows {
		c, err := connectionOf(row)
		if err != nil {
			return nil, err
		}
		out[i] = s.label(c)
	}
	return out, nil
}
