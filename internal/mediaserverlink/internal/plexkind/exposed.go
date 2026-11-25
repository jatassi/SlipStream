package plexkind

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// exposedFacts are the /:/prefs ids Exposed reads.
type exposedFacts struct {
	// Measured is false until a recorded run confirms the ids. The
	// conformance case skips until then.
	Measured bool
	// Custom is the comma-separated custom server access URLs.
	Custom string
	// Remote is Remote Access.
	Remote string
	IPv6   string
}

// plexExposed is what Plex 1.43.5.11079 answered in the recorded run of
// 2026-11-24 (conformance/plex/recorded/2026-11-24-plex-reachability.md).
var plexExposed = exposedFacts{Measured: true, Custom: "customConnections", Remote: "PublishServerOnPlexOnlineKey", IPv6: "EnableIPv6"}

func ExposedMeasured() bool { return plexExposed.Measured }

// Exposed reads GET / for the server's identity and mapping state,
// /myplex/account for Plex's own check, and /:/prefs for Remote Access,
// the custom URLs and IPv6.
func (c *conn) Exposed(ctx context.Context) (link.Exposed, error) {
	if c.base != nil {
		return link.Exposed{}, unreachable(c.base)
	}
	s, err := c.c.Server(ctx)
	if err != nil {
		return link.Exposed{}, fail(ctx, err)
	}
	if _, err := c.identity(s.MachineIdentifier, s.FriendlyName, s.Version); err != nil {
		return link.Exposed{}, err
	}
	a, err := c.c.Account(ctx)
	if err != nil {
		return link.Exposed{}, fail(ctx, err)
	}
	prefs, err := c.c.Prefs(ctx)
	if err != nil {
		return link.Exposed{}, fail(ctx, err)
	}
	x := link.Exposed{
		Local:       addrPort(a.PrivateAddress, a.PrivatePort),
		MappingRoot: s.MyPlexMappingState,
		Mapping: &link.PlexMapping{
			State: a.MappingState, Error: a.MappingError, Pending: a.ReachabilityPending, Public: addrPort(a.PublicAddress, a.PublicPort),
		},
	}
	var custom string
	for _, read := range []struct {
		id string
		to any
	}{{plexExposed.Remote, &x.Remote}, {plexExposed.IPv6, &x.IPv6}, {plexExposed.Custom, &custom}} {
		if err := pref(prefs, read.id, read.to); err != nil {
			return link.Exposed{}, &link.ServerError{Problem: link.Problem{Reason: link.ReasonFailed, Detail: err.Error()}}
		}
	}
	for u := range strings.SplitSeq(custom, ",") {
		if u = strings.TrimSpace(u); u != "" {
			x.Custom = append(x.Custom, u)
		}
	}
	return x, nil
}

// pref decodes the setting id into to, a *link.Tri or a *string. A setting
// Plex doesn't list leaves to as it is.
func pref(prefs []plex.Setting, id string, to any) error {
	i := slices.IndexFunc(prefs, func(s plex.Setting) bool { return s.ID == id })
	if i < 0 {
		return nil
	}
	var err error
	switch to := to.(type) {
	case *link.Tri:
		var b bool
		err = json.Unmarshal(prefs[i].Value, &b)
		*to = link.TriOf(b)
	case *string:
		err = json.Unmarshal(prefs[i].Value, to)
	}
	if err != nil {
		return fmt.Errorf("read the Plex setting %s: %w", id, err)
	}
	return nil
}

// addrPort is an address and port Plex reported, zero when either is
// missing.
func addrPort(address string, port uint16) netip.AddrPort {
	a, err := netip.ParseAddr(address)
	if err != nil || port == 0 {
		return netip.AddrPort{}
	}
	return netip.AddrPortFrom(a.Unmap(), port)
}
