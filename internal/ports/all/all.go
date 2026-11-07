// Package all is the binary's one port Registry. Each port adds itself
// from its own file here (hdbits.go: func init() { add(hdbits.Port) }),
// so adding a port touches no shared line and two port changes never
// conflict.
package all

import "github.com/jatassi/slipstream/internal/ports/port"

var ports []port.Port

func add(p port.Port) { ports = append(ports, p) }

// Registry is built once: internal/node hands the same value to the
// Controller, as a port.Catalog, and to the Node, through port.Driver.
func Registry() port.Registry { return port.NewRegistry(ports, retired, moved) }

// retired are ports deleted because their site died. The change that
// deletes a port's package adds it here.
var retired = []port.Retired{}

// moved are ports whose site moved to the catalog, for the release that
// migrates their Indexers.
var moved = []port.Moved{}
