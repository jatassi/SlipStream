// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/AlphaRatio.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package alpharatio is AlphaRatio, on the Gazelle family.
package alpharatio

import (
	_ "embed"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/internal/gazelle"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[alphaRatioSettings]

var Port = gazelle.Define(port.Entry{
	ID:           "alpharatio",
	Name:         "AlphaRatio",
	IndexerURLs:  []string{"https://alpharatio.cc/"},
	Description:  "AlphaRatio is a private Tracker for 0day and general Releases.",
	Privacy:      port.Private,
	PageSize:     50,
	RateLimit:    3 * time.Second,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, gazelle.Site[alphaRatioSettings]{
	ImdbInTags:               true,
	GetBasicSearchParameters: getBasicSearchParameters,
	GetDownloadURL:           getDownloadURL,
})

func setCapabilities() port.Capabilities {
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q},
		Categories: port.Categories(
			port.Map(1, newznab.TVSD, "TvSD"),
			port.Map(2, newznab.TVHD, "TvHD"),
			port.Map(3, newznab.TVUHD, "TvUHD"),
			port.Map(4, newznab.TVSD, "TvDVDRip"),
			port.Map(5, newznab.TVSD, "TvPackSD"),
			port.Map(6, newznab.TVHD, "TvPackHD"),
			port.Map(7, newznab.TVUHD, "TvPackUHD"),
			port.Map(8, newznab.MoviesSD, "MovieSD"),
			port.Map(9, newznab.MoviesHD, "MovieHD"),
			port.Map(10, newznab.MoviesUHD, "MovieUHD"),
			port.Map(11, newznab.MoviesSD, "MoviePackSD"),
			port.Map(12, newznab.MoviesHD, "MoviePackHD"),
			port.Map(13, newznab.MoviesUHD, "MoviePackUHD"),
			port.Map(14, newznab.XXX, "MovieXXX"),
			port.Map(15, newznab.MoviesBluRay, "Bluray"),
			port.Map(16, newznab.TVAnime, "AnimeSD"),
			port.Map(17, newznab.TVAnime, "AnimeHD"),
			port.Map(18, newznab.PCGames, "GamesPC"),
			port.Map(19, newznab.ConsoleXBox, "GamesxBox"),
			port.Map(20, newznab.ConsolePS4, "GamesPS"),
			port.Map(21, newznab.ConsoleWii, "GamesNin"),
			port.Map(22, newznab.PC0day, "AppsWindows"),
			port.Map(23, newznab.PCMac, "AppsMAC"),
			port.Map(24, newznab.PC0day, "AppsLinux"),
			port.Map(25, newznab.PCMobileOther, "AppsMobile"),
			port.Map(26, newznab.XXX, "0dayXXX"),
			port.Map(27, newznab.Books, "eBook"),
			port.Map(28, newznab.AudioAudiobook, "AudioBook"),
			port.Map(29, newznab.AudioOther, "Music"),
			port.Map(30, newznab.Other, "Misc"),
		),
	}
}

// getBasicSearchParameters is AlphaRatioRequestGenerator's.
func getBasicSearchParameters(ix *ix, searchCriteria port.SearchCriteria, term string) stringutil.NameValueCollection {
	parameters := gazelle.GetBasicSearchParameters(ix, searchCriteria, term)

	if ix.Settings.FreeleechOnly {
		parameters.Set("freetorrent", "1")
	}

	if ix.Settings.ExcludeScene {
		parameters.Set("scene", "0")
	}

	if searchCriteria.Limit > 0 && searchCriteria.Offset > 0 {
		page := searchCriteria.Offset/searchCriteria.Limit + 1
		parameters.Set("page", strconv.Itoa(page))
	}

	return parameters
}

// getDownloadURL is AlphaRatioParser's GetDownloadUrl.
func getDownloadURL(ix *ix, torrentID int, canUseToken bool) string {
	url := port.NewURI(ix.BaseURL).
		CombinePath("/torrents.php").
		AddQueryParam("action", "download").
		AddQueryParam("id", strconv.Itoa(torrentID))

	if t := ix.Settings.UseFreeleechToken; (t == gazelle.Preferred || t == gazelle.Required) && canUseToken {
		url = url.AddQueryParam("usetoken", "1")
	}

	return url.FullURI()
}

type alphaRatioSettings struct {
	gazelle.Settings
	FreeleechOnly bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
	ExcludeScene  bool `port:"excludeScene" label:"Exclude scene Releases" help:"Leave scene Releases out of every search."`
}
