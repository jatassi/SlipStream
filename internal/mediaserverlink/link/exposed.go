package link

import "net/netip"

// Tri is a fact a server may not report. Its zero value is Unknown, so a
// field no reader filled never reads as Yes.
type Tri uint8

const (
	Unknown Tri = iota
	Yes
	No
)

func TriOf(b bool) Tri {
	if b {
		return Yes
	}
	return No
}

// Exposed is what a server reports about its own use away from home, read
// on its reaching Node.
type Exposed struct {
	// Local is the address and port the server reports it listens on, zero
	// when it reports none.
	Local netip.AddrPort
	// Remote is Plex's Remote Access, PublishServerOnPlexOnlineKey, or
	// Jellyfin's EnableRemoteAccess.
	Remote Tri
	// Mapping is Plex's own check of its public address, nil on Jellyfin
	// and Emby.
	Mapping *PlexMapping
	// MappingRoot is GET /'s myPlexMappingState as Plex sent it.
	MappingRoot string
	// Custom are the addresses the server publishes beside its own: Plex's
	// customConnections, Jellyfin's published server addresses, Emby's WAN
	// and remote addresses.
	Custom []string
	IPv6   Tri
}

// PlexMapping is /myplex/account's mapping state.
type PlexMapping struct {
	// State is mappingState as Plex sent it.
	State string
	// Error is mappingError, empty when Plex reports none.
	Error string
	// Pending is reachabilityPending: Plex hasn't finished checking.
	Pending bool
	// Public is publicAddress and publicPort, zero when Plex knows none.
	Public netip.AddrPort
}
