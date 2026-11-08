// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/SpeedApp.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package speedapp is SpeedApp, on the SpeedApp family.
package speedapp

import (
	_ "embed"

	"github.com/jatassi/slipstream/internal/newznab"
	speedappbase "github.com/jatassi/slipstream/internal/ports/internal/speedapp"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

var Port = speedappbase.Define(port.Entry{
	ID:           "speedapp",
	Name:         "SpeedApp.io",
	IndexerURLs:  []string{"https://speedapp.io/"},
	LegacyURLs:   []string{"https://speedapp.io"},
	Description:  "SpeedApp is a private Romanian Tracker for movies, TV and general Releases.",
	Language:     "ro-RO",
	Privacy:      port.Private,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, speedappbase.Site{})

// SlipStream: no MusicSearchParams or BookSearchParams, since SlipStream
// has no music or book search mode. Music and book releases still come
// back from Basic.
func setCapabilities() port.Capabilities {
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.ImdbID, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(38, newznab.Movies, "Movie Packs"),
			port.Map(10, newznab.MoviesSD, "Movies: SD"),
			port.Map(35, newznab.MoviesSD, "Movies: SD Ro"),
			port.Map(8, newznab.MoviesHD, "Movies: HD"),
			port.Map(29, newznab.MoviesHD, "Movies: HD Ro"),
			port.Map(7, newznab.MoviesDVD, "Movies: DVD"),
			port.Map(2, newznab.MoviesDVD, "Movies: DVD Ro"),
			port.Map(17, newznab.MoviesBluRay, "Movies: BluRay"),
			port.Map(24, newznab.MoviesBluRay, "Movies: BluRay Ro"),
			port.Map(59, newznab.Movies, "Movies: Ro"),
			port.Map(57, newznab.MoviesUHD, "Movies: 4K (2160p) Ro"),
			port.Map(61, newznab.MoviesUHD, "Movies: 4K (2160p)"),
			port.Map(41, newznab.TV, "TV Packs"),
			port.Map(66, newznab.TV, "TV Packs Ro"),
			port.Map(45, newznab.TVSD, "TV Episodes"),
			port.Map(46, newznab.TVSD, "TV Episodes Ro"),
			port.Map(43, newznab.TVHD, "TV Episodes HD"),
			port.Map(44, newznab.TVHD, "TV Episodes HD Ro"),
			port.Map(60, newznab.TV, "TV Ro"),
			port.Map(11, newznab.PCGames, "Games: PC-ISO"),
			port.Map(52, newznab.Console, "Games: Console"),
			port.Map(1, newznab.PC0day, "Applications"),
			port.Map(14, newznab.PC, "Applications: Linux"),
			port.Map(37, newznab.PCMac, "Applications: Mac"),
			port.Map(19, newznab.PCMobileOther, "Applications: Mobile"),
			port.Map(62, newznab.TV, "TV Cartoons"),
			port.Map(3, newznab.TVAnime, "TV Anime / Hentai"),
			port.Map(6, newznab.BooksEBook, "E-books"),
			port.Map(5, newznab.Audio, "Music"),
			port.Map(64, newznab.AudioVideo, "Music Video"),
			port.Map(18, newznab.Other, "Images"),
			port.Map(22, newznab.TVSport, "TV Sports"),
			port.Map(58, newznab.TVSport, "TV Sports Ro"),
			port.Map(9, newznab.TVDocumentary, "TV Documentary"),
			port.Map(63, newznab.TVDocumentary, "TV Documentary Ro"),
			port.Map(65, newznab.Other, "Tutorial"),
			port.Map(67, newznab.OtherMisc, "Miscellaneous"),
			port.Map(15, newznab.XXX, "XXX Movies"),
			port.Map(47, newznab.XXX, "XXX DVD"),
			port.Map(48, newznab.XXX, "XXX HD"),
			port.Map(49, newznab.XXXImageSet, "XXX Images"),
			port.Map(50, newznab.XXX, "XXX Packs"),
			port.Map(51, newznab.XXX, "XXX SD"),
		),
	}
}
