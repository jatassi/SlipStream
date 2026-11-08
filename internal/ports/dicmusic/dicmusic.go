// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/DICMusic.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package dicmusic is DICMusic, on the Gazelle family.
package dicmusic

import (
	_ "embed"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/internal/gazelle"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

// Port is DICMusic.
// SlipStream: the settings are gazelle.UserPassOrCookieSettings, since
// DICMusicSettings adds nothing to it.
var Port = gazelle.Define(port.Entry{
	ID:           "dicmusic",
	Name:         "DICMusic",
	IndexerURLs:  []string{"https://dicmusic.com/"},
	LegacyURLs:   []string{"https://dicmusic.club/"},
	Description:  "DICMusic is a private Chinese Tracker for music.",
	Language:     "zh-CN",
	Privacy:      port.Private,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, gazelle.Site[gazelle.UserPassOrCookieSettings]{})

// SlipStream: no MusicSearchParams, since SlipStream has no music search
// mode. Music releases still come back from Basic.
func setCapabilities() port.Capabilities {
	return port.Capabilities{
		Categories: port.Categories(
			port.Map(1, newznab.Audio, "Music"),
			port.Map(2, newznab.PC, "Applications"),
		),
	}
}
