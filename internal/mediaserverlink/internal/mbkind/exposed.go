package mbkind

import (
	"context"
	"encoding/json"
	"net/netip"
	"net/url"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// exposedFacts are the route and field names Exposed reads past
// /System/Info/Public. An empty name is a fact the product doesn't hold.
type exposedFacts struct {
	// Measured is false until a recorded run against the product confirms
	// the names. Until then Exposed reads only /System/Info/Public, its
	// other facts read Unknown, and the conformance case skips.
	Measured bool
	Route    string
	// Remote and IPv6 are boolean fields.
	Remote, IPv6 string
	// Custom are fields holding a published address or a list of them.
	Custom []string
}

// jellyfinExposed and embyExposed are named from Jellyfin's
// NetworkConfiguration at v12.4 and the Emby SDK's SystemInfo, which no
// run has confirmed yet.
var (
	jellyfinExposed = exposedFacts{
		Route: "/System/Configuration/network", Remote: "EnableRemoteAccess", IPv6: "EnableIPv6", Custom: []string{"PublishedServerUriBySubnet"},
	}
	embyExposed = exposedFacts{Route: "/System/Info", Custom: []string{"WanAddress", "RemoteAddresses"}}
)

// ExposedMeasured is whether a run confirmed k's names.
func ExposedMeasured(k link.Kind) bool {
	f, ok := k.(family)
	return ok && f.exposed.Measured
}

// Exposed reads the server's local address from /System/Info/Public and,
// once measured, the rest from the kind's route.
func (c *conn) Exposed(ctx context.Context) (link.Exposed, error) {
	x, err := c.exposed(ctx)
	return x, problem(ctx, err)
}

func (c *conn) exposed(ctx context.Context) (link.Exposed, error) {
	info, err := c.c.PublicInfo(ctx)
	if err != nil {
		return link.Exposed{}, err
	}
	id, err := identityOf(info)
	if err != nil {
		return link.Exposed{}, err
	}
	if err := c.saved(id); err != nil {
		return link.Exposed{}, err
	}
	x := link.Exposed{Local: localOf(info.LocalAddress)}
	f := c.f.exposed
	if !f.Measured {
		return x, nil
	}
	fields, err := c.c.Fields(ctx, f.Route)
	if err != nil {
		return link.Exposed{}, err
	}
	for _, b := range []struct {
		name string
		to   *link.Tri
	}{{f.Remote, &x.Remote}, {f.IPv6, &x.IPv6}} {
		var v bool
		if raw, ok := fields[b.name]; ok && json.Unmarshal(raw, &v) == nil {
			*b.to = link.TriOf(v)
		}
	}
	for _, name := range f.Custom {
		var one string
		var many []string
		switch raw := fields[name]; {
		case json.Unmarshal(raw, &one) == nil && one != "":
			x.Custom = append(x.Custom, one)
		case json.Unmarshal(raw, &many) == nil:
			x.Custom = append(x.Custom, many...)
		}
	}
	return x, nil
}

// localOf is LocalAddress's host and port when its host is an address.
func localOf(address string) netip.AddrPort {
	u, err := url.Parse(address)
	if err != nil {
		return netip.AddrPort{}
	}
	ap, err := netip.ParseAddrPort(u.Host)
	if err != nil {
		return netip.AddrPort{}
	}
	return netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())
}
