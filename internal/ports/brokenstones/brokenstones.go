// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BrokenStones.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package brokenstones is BrokenStones, on the Gazelle family.
package brokenstones

import (
	_ "embed"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/internal/gazelle"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

var Port = gazelle.Define(port.Entry{
	ID:           "brokenstones",
	Name:         "BrokenStones",
	IndexerURLs:  []string{"https://brokenstones.is/"},
	Description:  "BrokenStones is a private Tracker for macOS and iOS apps and games.",
	Privacy:      port.Private,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, gazelle.Site[gazelle.Settings]{})

func setCapabilities() port.Capabilities {
	return port.Capabilities{
		Categories: port.Categories(
			port.Map(1, newznab.PCMac, "MacOS Apps"),
			port.Map(2, newznab.PCMac, "MacOS Games"),
			port.Map(3, newznab.PCMobileiOS, "iOS Apps"),
			port.Map(4, newznab.PCMobileiOS, "iOS Games"),
			port.Map(5, newznab.Other, "Graphics"),
			port.Map(6, newznab.Audio, "Audio"),
			port.Map(7, newznab.Other, "Tutorials"),
			port.Map(8, newznab.Other, "Other"),
		),
	}
}
