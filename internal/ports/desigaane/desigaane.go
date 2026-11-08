// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/DesiGaane.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package desigaane is DesiGaane, on the Gazelle family.
package desigaane

import (
	_ "embed"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/internal/gazelle"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

var Port = gazelle.Define(port.Entry{
	ID:           "desigaane",
	Name:         "DesiGaane",
	IndexerURLs:  []string{"https://desigaane.rocks/"},
	Description:  "DesiGaane is a private Tracker for Desi music.",
	Privacy:      port.Private,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, gazelle.Site[gazelle.Settings]{})

func setCapabilities() port.Capabilities {
	return port.Capabilities{
		Categories: port.Categories(
			port.Map(1, newznab.Audio, "Music"),
		),
	}
}
