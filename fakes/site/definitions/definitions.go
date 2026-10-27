// Package definitions holds the Cardigann definitions of fixture-site's
// fictional sites, which fake-catalog serves as its Indexer catalog. Each
// file is SlipStream's own writing. A file named <id>-v<n>.yml is a later
// version of <id>.yml that a cue swaps in.
package definitions

import "embed"

//go:embed *.yml
var FS embed.FS
