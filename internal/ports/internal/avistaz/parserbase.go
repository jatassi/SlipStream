// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Avistaz/AvistazParserBase.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package avistaz

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
)

var hdResolutions = []string{"1080p", "1080i", "720p"}

type parserBase struct{ site Site }

func (p parserBase) parseResponse(ix *port.Indexer[Settings], indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	if indexerResponse.StatusCode == http.StatusNotFound {
		return releaseInfos, nil
	}

	if indexerResponse.StatusCode == http.StatusTooManyRequests {
		// SlipStream: the outcome has no text, so upstream's words go unused.
		return nil, port.RequestLimitReached("")
	}

	if indexerResponse.StatusCode == http.StatusUnauthorized {
		var errorResponse AvistazErrorResponse
		_ = json.Unmarshal(indexerResponse.Body, &errorResponse)
		// SlipStream: the outcome's sentence already says the credentials
		// were refused, so there is no upstream fallback message.
		return nil, port.IndexerAuthError(errorResponse.Message)
	}

	if indexerResponse.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	if contentType := indexerResponse.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer's response is %q, which isn't JSON", contentType)
	}

	var jsonResponse AvistazResponse
	if err := json.Unmarshal(indexerResponse.Body, &jsonResponse); err != nil {
		return nil, fmt.Errorf("read the Indexer's JSON: %w", err)
	}

	for _, row := range jsonResponse.Data {
		detailsURL := row.URL

		categories, err := p.site.ParseCategories(ix, row)
		if err != nil {
			return nil, err
		}

		publishDate, err := datetimeutil.Parse(row.CreatedAtIso, datetimeutil.AdjustToUniversal, ix.Clock)
		if err != nil {
			return nil, fmt.Errorf("read a release's date: %w", err)
		}

		// SlipStream: no Languages or Subs, which a Release doesn't carry.
		release := port.TorrentInfo{
			GUID:                 detailsURL,
			InfoURL:              detailsURL,
			Title:                p.site.ParseTitle(row),
			DownloadURL:          row.Download,
			Categories:           categories,
			InfoHash:             row.InfoHash,
			Size:                 row.FileSize,
			Files:                row.FileCount,
			Grabs:                row.Completed,
			Seeders:              row.Seed,
			Peers:                row.Leech + row.Seed,
			PublishDate:          publishDate,
			DownloadVolumeFactor: volumeFactor(row.DownloadMultiply),
			UploadVolumeFactor:   volumeFactor(row.UploadMultiply),
			MinimumRatio:         1,
			MinimumSeedTime:      259200, // 72 hours
		}

		if row.FileSize > 0 {
			sizeGigabytes := float64(row.FileSize) / (1024 * 1024 * 1024)

			if sizeGigabytes > 50.0 {
				release.MinimumSeedTime = int64(100*math.Log(sizeGigabytes)-219.2023) * 3600
			} else {
				release.MinimumSeedTime = 259200 + int64(sizeGigabytes*7200)
			}
		}

		if row.MovieTvinfo != nil {
			release.ImdbID, _ = parseutil.GetImdbID(row.MovieTvinfo.Imdb)
			release.TmdbID, _ = parseutil.TryCoerceInt(row.MovieTvinfo.Tmdb)
			release.TvdbID, _ = parseutil.TryCoerceInt(row.MovieTvinfo.Tvdb)
		}

		releaseInfos = append(releaseInfos, release)
	}

	slices.SortStableFunc(releaseInfos, func(a, b port.TorrentInfo) int { return b.PublishDate.Compare(a.PublishDate) })
	return releaseInfos, nil
}

// volumeFactor is a factor the response leaves out as 1.
// SlipStream: Prowlarr keeps it null, which its consumers read as 1, and
// a Go zero is freeleech.
func volumeFactor(multiply *float64) float64 {
	if multiply == nil {
		return 1
	}
	return *multiply
}

func parseCategories(_ *port.Indexer[Settings], row AvistazRelease) ([]newznab.Category, error) {
	var categories []newznab.Category
	videoQuality := row.VideoQuality

	switch strings.ToUpper(row.Type) {
	case "MOVIE":
		switch {
		case slices.Contains(hdResolutions, videoQuality):
			categories = append(categories, newznab.MoviesHD)
		case videoQuality == "2160p":
			categories = append(categories, newznab.MoviesUHD)
		default:
			categories = append(categories, newznab.MoviesSD)
		}
	case "TV-SHOW":
		switch {
		case slices.Contains(hdResolutions, videoQuality):
			categories = append(categories, newznab.TVHD)
		case videoQuality == "2160p":
			categories = append(categories, newznab.TVUHD)
		default:
			categories = append(categories, newznab.TVSD)
		}
	case "MUSIC":
		categories = append(categories, newznab.Audio)
	default:
		return nil, fmt.Errorf("the Indexer lists a release of type %q, which maps to no category", row.Type)
	}

	return categories, nil
}

func parseTitle(row AvistazRelease) string {
	return row.FileName
}
