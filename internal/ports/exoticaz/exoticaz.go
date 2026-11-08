// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/ExoticaZ.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package exoticaz is ExoticaZ, on the Avistaz family.
package exoticaz

import (
	_ "embed"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/newznab"
	avistazbase "github.com/jatassi/slipstream/internal/ports/internal/avistaz"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

var Port = avistazbase.Define(port.Entry{
	ID:           "exoticaz",
	Name:         "ExoticaZ",
	IndexerURLs:  []string{"https://exoticaz.to/"},
	Description:  "ExoticaZ, once called YourExotic, is a private tracker for adult videos.",
	Privacy:      port.Private,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, avistazbase.Site{
	ParseCategories: parseCategories,
})

func setCapabilities() port.Capabilities {
	return port.Capabilities{
		Categories: port.Categories(
			port.Map(1, newznab.XXXx264, "Video Clip"),
			port.Map(2, newznab.XXXPack, "Video Pack"),
			port.Map(3, newznab.XXXPack, "Siterip Pack"),
			port.Map(4, newznab.XXXPack, "Pornstar Pack"),
			port.Map(5, newznab.XXXDVD, "DVD"),
			port.Map(6, newznab.XXXx264, "BluRay"),
			port.Map(7, newznab.XXXImageSet, "Photo Pack"),
			port.Map(8, newznab.XXXImageSet, "Books & Magazines"),
		),
	}
}

// parseCategories is ExoticaZParser's.
// SlipStream: a Go map has no order, so the ids are read sorted.
func parseCategories(ix *port.Indexer[avistazbase.Settings], row avistazbase.AvistazRelease) ([]newznab.Category, error) {
	cat := row.Category

	var categories []newznab.Category
	for _, c := range slices.Sorted(maps.Keys(cat)) {
		categories = append(categories, ix.Categories.FromTracker(c)...)
	}
	return categories, nil
}
