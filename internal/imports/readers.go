package imports

import "github.com/jatassi/slipstream/internal/grabs"

// Uses is downloadclients.Uses: grabs' Uses of each Torrent. Stateless.
type Uses struct{ grabs.TorrentUses }
