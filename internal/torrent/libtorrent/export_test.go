package libtorrent

import (
	"net/netip"

	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// OpenRouted opens a session as Open does on an unspecified address, with
// route standing for the host's default route.
func OpenRouted(cfg engine.Config, route func() (netip.Addr, error)) (*Session, error) {
	return openRouted(cfg, route)
}
