// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleParser.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package gazelle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
)

type parser[S settings] struct{ site Site[S] }

// SlipStream: the messages are SlipStream's own, per WRITING.md, with
// the site's words quoted. CookiesUpdater(null, null) has no Go: a
// logged-out response already brings a sign-in.
func (p parser[S]) parseResponse(ix *port.Indexer[S], indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo
	settings := ix.Settings.gazelle()

	if indexerResponse.StatusCode != http.StatusOK {
		var errorResponse GazelleErrorResponse
		_ = json.Unmarshal(indexerResponse.Body, &errorResponse)
		if errorResponse.Error != "" {
			return nil, port.IndexerError("the Indexer responded with status %d and the error %q", indexerResponse.StatusCode, errorResponse.Error)
		}
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	if contentType := indexerResponse.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q where application/json was expected", contentType)
	}

	var jsonResponse GazelleResponse
	if err := json.Unmarshal(indexerResponse.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}
	if jsonResponse.Status != "success" ||
		strings.TrimSpace(string(jsonResponse.Status)) == "" ||
		jsonResponse.Response == nil {
		return torrentInfos, nil
	}

	for _, result := range jsonResponse.Response.Results {
		// SlipStream: no PosterUrl, which a Release doesn't carry.

		if result.Torrents != nil {
			for _, torrent := range result.Torrents {
				isFreeLeech := torrent.IsFreeLeech || torrent.IsNeutralLeech || torrent.IsPersonalFreeLeech

				// skip releases that cannot be used with freeleech tokens when the option is enabled
				if settings.UseFreeleechToken == Required && !torrent.CanUseToken && !isFreeLeech {
					continue
				}

				id := torrent.TorrentID

				title := fmt.Sprintf("%s - %s (%s) [%s %s] [%s]", result.Artist, result.GroupName, result.GroupYear, torrent.Format, torrent.Encoding, torrent.Media)
				if torrent.HasCue {
					title += " [Cue]"
				}

				infoURL := GetInfoURL(ix, string(result.GroupID), id)

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
				publishDate, err := datetimeutil.Parse(string(torrent.Time), datetimeutil.AssumeUniversal|datetimeutil.AdjustToUniversal, ix.Clock)
				if err != nil {
					return nil, fmt.Errorf("read a release's date: %w", err)
				}

				// SlipStream: no Container or Codec, which a Release doesn't
				// carry. IndexerBase adds Scene from TorrentInfo.Scene, which
				// the Go TorrentInfo carries as a flag.
				release := port.TorrentInfo{
					GUID:                 infoURL,
					InfoURL:              infoURL,
					DownloadURL:          p.site.GetDownloadURL(ix, id, torrent.CanUseToken && !isFreeLeech),
					Title:                encoding.HTMLDecode(title),
					Files:                torrent.FileCount,
					Grabs:                torrent.Snatches,
					Size:                 size,
					Seeders:              seeders,
					Peers:                leechers + seeders,
					PublishDate:          publishDate,
					DownloadVolumeFactor: factor(isFreeLeech),
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

				torrentInfos = append(torrentInfos, release)
			}
		} else {
			isFreeLeech := result.IsFreeLeech || result.IsNeutralLeech || result.IsPersonalFreeLeech

			// skip releases that cannot be used with freeleech tokens when the option is enabled
			if settings.UseFreeleechToken == Required && !result.CanUseToken && !isFreeLeech {
				continue
			}

			id := result.TorrentID
			groupName := encoding.HTMLDecode(string(result.GroupName))
			infoURL := GetInfoURL(ix, string(result.GroupID), id)

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
			var publishDate time.Time
			if num, err := strconv.ParseInt(string(result.GroupTime), 10, 64); err == nil {
				publishDate = time.Unix(num, 0).UTC()
			} else if publishDate, err = datetimeutil.FromFuzzyTime(string(result.GroupTime), ix.Clock); err != nil {
				return nil, fmt.Errorf("read a release's date: %w", err)
			}

			release := port.TorrentInfo{
				GUID:                 infoURL,
				InfoURL:              infoURL,
				DownloadURL:          p.site.GetDownloadURL(ix, id, result.CanUseToken && !isFreeLeech),
				Title:                groupName,
				Size:                 size,
				Seeders:              seeders,
				Peers:                leechers + seeders,
				Files:                result.FileCount,
				Grabs:                result.Snatches,
				PublishDate:          publishDate,
				DownloadVolumeFactor: factor(isFreeLeech),
				UploadVolumeFactor:   factor(result.IsNeutralLeech),
			}

			category := string(result.Category)
			if category == "" || strings.Contains(category, "Select Category") {
				release.Categories = ix.Categories.FromTracker("1")
			} else {
				release.Categories = ix.Categories.FromDesc(category)
			}

			torrentInfos = append(torrentInfos, release)
		}
	}

	// order by date
	slices.SortStableFunc(torrentInfos, func(a, b port.TorrentInfo) int { return b.PublishDate.Compare(a.PublishDate) })
	return torrentInfos, nil
}

// factor is C#'s free ? 0 : 1.
func factor(free bool) float64 {
	if free {
		return 0
	}
	return 1
}

// GetDownloadURL is the parser's GetDownloadUrl, which a site's own may
// replace.
func GetDownloadURL[S settings](ix *port.Indexer[S], torrentID int, canUseToken bool) string {
	url := port.NewURI(ix.BaseURL).
		CombinePath("/torrents.php").
		AddQueryParam("action", "download").
		AddQueryParam("id", strconv.Itoa(torrentID))

	if t := ix.Settings.gazelle().UseFreeleechToken; (t == Preferred || t == Required) && canUseToken {
		url = url.AddQueryParam("usetoken", "1")
	}

	return url.FullURI()
}

// SlipStream: no GetPosterUrl, since a Release carries no poster.

// GetInfoURL is the parser's GetInfoUrl.
func GetInfoURL[S any](ix *port.Indexer[S], groupID string, torrentID int) string {
	url := port.NewURI(ix.BaseURL).
		CombinePath("/torrents.php").
		AddQueryParam("id", groupID).
		AddQueryParam("torrentid", strconv.Itoa(torrentID))

	return url.FullURI()
}
