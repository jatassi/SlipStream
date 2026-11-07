// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/FileList/FileList.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package filelist

import (
	_ "embed"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

// Port is FileList.io.
//
// SlipStream: the Music and Book search params are gone, because
// SlipStream has neither mode.
var Port = port.Define(port.Entry{
	ID:          "filelist",
	Name:        "FileList.io",
	Description: "FileList is a private Romanian tracker for movies, TV and more.",
	Privacy:     port.Private,
	IndexerURLs: []string{"https://filelist.io/", "https://thefl.org/"},
	LegacyURLs: []string{
		"https://filelist.io",
		"https://filelist.ro/",
		"http://filelist.ro/",
		"http://flro.org/",
		"https://flro.org/",
	},
	QueryLimit: port.PerHour(150),
	Capabilities: port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.ImdbID, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(24, newznab.TVAnime, "Anime"),
			port.Map(11, newznab.Audio, "Music"),
			port.Map(29, newznab.Books, "Books"),
			port.Map(30, newznab.Other, "Courses"),
			port.Map(15, newznab.TV, "Cartoons"),
			port.Map(18, newznab.Other, "Various"),
			port.Map(16, newznab.Books, "Docs"),
			port.Map(25, newznab.Movies3D, "Movies 3D"),
			port.Map(6, newznab.MoviesUHD, "Movies 4K"),
			port.Map(26, newznab.MoviesBluRay, "Movies 4K Blu-Ray"),
			port.Map(20, newznab.MoviesBluRay, "Movies Blu-Ray"),
			port.Map(2, newznab.MoviesDVD, "Movies DVD"),
			port.Map(3, newznab.MoviesForeign, "Movies DVD-RO"),
			port.Map(4, newznab.MoviesHD, "Movies HD"),
			port.Map(19, newznab.MoviesForeign, "Movies HD-RO"),
			port.Map(1, newznab.MoviesSD, "Movies SD"),
			port.Map(5, newznab.AudioLossless, "FLAC"),
			port.Map(10, newznab.Console, "Games Console"),
			port.Map(9, newznab.PCGames, "Games PC"),
			port.Map(31, newznab.TVForeign, "K-Drama"),
			port.Map(17, newznab.PC, "Linux"),
			port.Map(22, newznab.PCMobileOther, "Mobile"),
			port.Map(8, newznab.PC, "Apps"),
			port.Map(28, newznab.MoviesForeign, "RO Dubbed"),
			port.Map(28, newznab.TVForeign, "RO Dubbed"),
			port.Map(27, newznab.TVUHD, "TV-Series 4K"),
			port.Map(21, newznab.TVHD, "TV-Series HD"),
			port.Map(23, newznab.TVSD, "TV-Series SD"),
			port.Map(13, newznab.TVSport, "Sport"),
			port.Map(12, newznab.AudioVideo, "Videoclips"),
			port.Map(7, newznab.XXX, "XXX"),
		),
	},
	Upstream: port.ReadUpstream(upstream),
}, port.Impl[fileListSettings]{
	Login:          port.NoLogin{},
	Movie:          getMovieSearchRequests,
	TV:             getTvSearchRequests,
	Basic:          getBasicSearchRequests,
	Parse:          parseResponse,
	FilterReleases: cleanupReleases,
})

func cleanupReleases(_ *port.Indexer[fileListSettings], searchCriteria port.SearchCriteria, releases []port.TorrentInfo) []port.TorrentInfo {
	return port.FilterReleasesByQuery(releases, searchCriteria)
}
