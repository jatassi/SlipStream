// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/SecretCinema.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package secretcinema is Secret Cinema, on the Gazelle family with its
// own parser.
package secretcinema

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/internal/gazelle"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[gazelle.Settings]

var Port = gazelle.Define(port.Entry{
	ID:           "secretcinema",
	Name:         "Secret Cinema",
	IndexerURLs:  []string{"https://secret-cinema.pw/"},
	Description:  "Secret Cinema is a private Tracker for rare movies, with no ratio to keep.",
	Privacy:      port.Private,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, gazelle.Site[gazelle.Settings]{
	ParseResponse: parseResponse,
})

// SlipStream: no MusicSearchParams, since SlipStream has no music search
// mode. Music releases still come back from Basic.
func setCapabilities() port.Capabilities {
	return port.Capabilities{
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(1, newznab.Movies, "Movies"),
			port.Map(2, newznab.Audio, "Music"),
		),
	}
}

var yearRegex = regexp.MustCompile(`(\b|[-._ ])((?:19|20)\d{2})(\b|[-._ ])`)

var (
	bdmvRegex = regexp.MustCompile(`(?i)\bBDMV\b`)
	sdRegex   = regexp.MustCompile(`(?i)\bSD\b`)
)

// parseResponse is SecretCinemaParser's ParseResponse.
// SlipStream: the messages are SlipStream's own, per WRITING.md, with
// the site's words quoted. CookiesUpdater(null, null) has no Go: a
// logged-out response already brings a sign-in.
func parseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	if indexerResponse.StatusCode != http.StatusOK {
		var errorResponse gazelle.GazelleErrorResponse
		_ = json.Unmarshal(indexerResponse.Body, &errorResponse)
		if errorResponse.Error != "" {
			return nil, port.IndexerError("the Indexer responded with status %d and the error %q", indexerResponse.StatusCode, errorResponse.Error)
		}
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	if contentType := indexerResponse.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q where application/json was expected", contentType)
	}

	var jsonResponse gazelle.GazelleResponse
	if err := json.Unmarshal(indexerResponse.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}
	if jsonResponse.Status != "success" ||
		strings.TrimSpace(string(jsonResponse.Status)) == "" ||
		jsonResponse.Response == nil {
		return releaseInfos, nil
	}

	for _, result := range jsonResponse.Response.Results {
		if result.Torrents != nil {
			for _, torrent := range result.Torrents {
				id := torrent.TorrentID
				infoURL := getInfoURL(ix, string(result.GroupID), id)

				// in SC movies, artist=director and GroupName=title
				artist := encoding.HTMLDecode(string(result.Artist))
				title := encoding.HTMLDecode(string(result.GroupName))
				// SlipStream: wallTime is C#'s time, the site's text read as
				// a wall time in its +02:00 zone.
				wallTime, err := datetimeutil.Parse(string(torrent.Time), datetimeutil.AssumeUniversal|datetimeutil.AdjustToUniversal, ix.Clock)
				if err != nil {
					return nil, fmt.Errorf("read a release's date: %w", err)
				}

				size, err := strconv.ParseInt(string(torrent.Size), 10, 64)
				if err != nil {
					return nil, fmt.Errorf("read a release's size: %w", err)
				}
				seeders, err := strconv.Atoi(string(torrent.Seeders))
				if err != nil {
					return nil, fmt.Errorf("read a release's seeders: %w", err)
				}
				leechers, err := strconv.Atoi(string(torrent.Leechers))
				if err != nil {
					return nil, fmt.Errorf("read a release's leechers: %w", err)
				}

				// SlipStream: no Container or Codec, which a Release doesn't
				// carry. Scene is a flag.
				release := port.TorrentInfo{
					GUID:                 infoURL,
					InfoURL:              infoURL,
					DownloadURL:          getDownloadURL(ix, id),
					Title:                title,
					Files:                torrent.FileCount,
					Grabs:                torrent.Snatches,
					Size:                 size,
					Seeders:              seeders,
					Peers:                leechers + seeders,
					PublishDate:          wallTime.Add(-2 * time.Hour),
					DownloadVolumeFactor: factor(torrent.IsFreeLeech || torrent.IsNeutralLeech || torrent.IsPersonalFreeLeech),
					UploadVolumeFactor:   factor(torrent.IsNeutralLeech),
				}
				if torrent.Scene {
					release.IndexerFlags = []port.SiteFlag{port.Scene}
				}

				// SlipStream: a category the response leaves out reads as "".
				category := string(torrent.Category)
				if category == "" || strings.Contains(category, "Select Category") {
					release.Categories = ix.Categories.FromTracker("1")
				} else {
					release.Categories = ix.Categories.FromDesc(category)
				}

				if isAnyMovieCategory(release.Categories) {
					if releaseName := string(torrent.ReleaseName); strings.TrimSpace(releaseName) != "" && yearRegex.MatchString(releaseName) {
						release.Title = strings.TrimSpace(encoding.HTMLDecode(releaseName))
					} else {
						release.Title = strings.TrimSpace(fmt.Sprintf("%s (%s) %s", title, result.GroupYear, torrent.Media))

						if remasterTitle := string(torrent.RemasterTitle); strings.TrimSpace(remasterTitle) != "" {
							release.Title += " / " + strings.TrimSpace(encoding.HTMLDecode(remasterTitle))
						}

						// Replace media formats with standards
						release.Title = bdmvRegex.ReplaceAllString(release.Title, "COMPLETE BLURAY")
						release.Title = sdRegex.ReplaceAllString(release.Title, "DVDRip")
					}
				} else {
					// SC API currently doesn't return anything but title.
					// SlipStream: no Artist or Album, which a Release doesn't
					// carry.
					release.Title = fmt.Sprintf("%s - %s (%s) [%s %s] [%s]", artist, title, result.GroupYear, torrent.Format, torrent.Encoding, torrent.Media)
				}

				if torrent.HasCue {
					release.Title += " [Cue]"
				}

				releaseInfos = append(releaseInfos, release)
			}
		} else {
			id := result.TorrentID
			infoURL := getInfoURL(ix, string(result.GroupID), id)
			groupName := encoding.HTMLDecode(string(result.GroupName))

			size, err := strconv.ParseInt(string(result.Size), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("read a release's size: %w", err)
			}
			seeders, err := strconv.Atoi(string(result.Seeders))
			if err != nil {
				return nil, fmt.Errorf("read a release's seeders: %w", err)
			}
			leechers, err := strconv.Atoi(string(result.Leechers))
			if err != nil {
				return nil, fmt.Errorf("read a release's leechers: %w", err)
			}
			groupTime, err := parseutil.CoerceLong(string(result.GroupTime))
			if err != nil {
				return nil, fmt.Errorf("read a release's date: %w", err)
			}

			release := port.TorrentInfo{
				GUID:                 infoURL,
				InfoURL:              infoURL,
				DownloadURL:          getDownloadURL(ix, id),
				Title:                groupName,
				Size:                 size,
				Seeders:              seeders,
				Peers:                leechers + seeders,
				Files:                result.FileCount,
				Grabs:                result.Snatches,
				PublishDate:          time.Unix(groupTime, 0).UTC(),
				DownloadVolumeFactor: factor(result.IsFreeLeech || result.IsNeutralLeech || result.IsPersonalFreeLeech),
				UploadVolumeFactor:   factor(result.IsNeutralLeech),
			}

			category := string(result.Category)
			if category == "" || strings.Contains(category, "Select Category") {
				release.Categories = ix.Categories.FromTracker("1")
			} else {
				release.Categories = ix.Categories.FromDesc(category)
			}

			releaseInfos = append(releaseInfos, release)
		}
	}

	// order by date
	slices.SortStableFunc(releaseInfos, func(a, b port.TorrentInfo) int { return b.PublishDate.Compare(a.PublishDate) })
	return releaseInfos, nil
}

func isAnyMovieCategory(category []newznab.Category) bool {
	movies := newznab.Family(newznab.Movies.ID)
	return slices.ContainsFunc(category, func(c newznab.Category) bool { return slices.Contains(movies, c.ID) })
}

func getDownloadURL(ix *ix, torrentID int) string {
	url := port.NewURI(ix.BaseURL).
		CombinePath("/torrents.php").
		AddQueryParam("action", "download").
		AddQueryParam("id", strconv.Itoa(torrentID))

	if t := ix.Settings.UseFreeleechToken; t == gazelle.Preferred || t == gazelle.Required {
		url = url.AddQueryParam("useToken", "1")
	}

	return url.FullURI()
}

func getInfoURL(ix *ix, groupID string, torrentID int) string {
	url := port.NewURI(ix.BaseURL).
		CombinePath("/torrents.php").
		AddQueryParam("id", groupID).
		AddQueryParam("torrentid", strconv.Itoa(torrentID))

	return url.FullURI()
}

// factor is C#'s free ? 0 : 1.
func factor(free bool) float64 {
	if free {
		return 0
	}
	return 1
}
