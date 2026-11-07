// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/PassThePopcorn/PassThePopcornParser.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package passthepopcorn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

// SlipStream: the messages are SlipStream's own, per WRITING.md. A 401 is
// a key the site refused: Prowlarr's HttpClient throws on it before the
// parser runs, and SlipStream names it account_refused.
func parseResponse(ix *indexer, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	if indexerResponse.StatusCode != http.StatusOK {
		switch {
		case indexerResponse.StatusCode == http.StatusForbidden:
			return nil, port.RequestLimitReached("the query limit")
		case indexerResponse.StatusCode == http.StatusTooManyRequests && strings.Contains(indexerResponse.Content, "We are not a TV indexer"):
			return nil, port.IndexerError("the site responded that it has no TV search")
		case indexerResponse.StatusCode == http.StatusTooManyRequests:
			return nil, port.RequestLimitReached("the request limit")
		case indexerResponse.StatusCode == http.StatusUnauthorized:
			return nil, port.IndexerAuthError("")
		}
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	if contentType := indexerResponse.Headers.Get("Content-Type"); contentType != "application/json" {
		return nil, port.IndexerError("the Indexer responded with %q where application/json was expected", contentType)
	}

	var jsonResponse passThePopcornResponse
	if err := json.Unmarshal(indexerResponse.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	if jsonResponse.TotalResults == "0" ||
		strings.TrimSpace(jsonResponse.TotalResults) == "" ||
		jsonResponse.Movies == nil {
		return torrentInfos, nil
	}

	for _, result := range jsonResponse.Movies {
		for _, torrent := range result.Torrents {
			var downloadVolumeFactor float64
			switch strings.ToUpper(torrent.FreeleechType) {
			case "FREELEECH", "NEUTRAL LEECH":
				downloadVolumeFactor = 0
			case "HALF LEECH":
				downloadVolumeFactor = 0.5
			default:
				downloadVolumeFactor = 1
			}

			if ix.Settings.FreeleechOnly && downloadVolumeFactor != 0.0 {
				continue
			}

			id := torrent.ID
			title := torrent.ReleaseName

			var flags []port.SiteFlag

			if torrent.GoldenPopcorn {
				flags = append(flags, port.Golden)
			}

			if torrent.Checked {
				flags = append(flags, port.Approved)
			}

			// SlipStream: IndexerBase adds Scene from TorrentInfo.Scene, which the
			// Go TorrentInfo doesn't have.
			if torrent.Scene {
				flags = append(flags, port.Scene)
			}

			var uploadVolumeFactor float64
			switch strings.ToUpper(torrent.FreeleechType) {
			case "NEUTRAL LEECH":
				uploadVolumeFactor = 0
			default:
				uploadVolumeFactor = 1
			}

			var f fields
			size := f.long("Size", torrent.Size)
			grabs := f.int("Snatched", torrent.Snatched)
			seeders := f.int("Seeders", torrent.Seeders)
			leechers := f.int("Leechers", torrent.Leechers)
			publishDate := f.date(ix, torrent.UploadTime)
			var imdbID int
			if strings.TrimSpace(result.ImdbID) != "" {
				imdbID = f.int("ImdbId", result.ImdbID)
			}
			if f.err != nil {
				return nil, port.IndexerError("read torrent %d of the Indexer's search response: %v", id, f.err)
			}

			// SlipStream: int.Parse is fields. TorrentInfo has no Year, Genres or
			// PosterUrl, because a Release carries none of them.
			torrentInfos = append(torrentInfos, port.TorrentInfo{
				GUID:                 fmt.Sprintf("PassThePopcorn-%d", id),
				Title:                title,
				InfoURL:              getInfoURL(ix, result.GroupID, id),
				DownloadURL:          getDownloadURL(ix, id),
				Categories:           ix.Categories.FromTracker(result.CategoryID),
				Size:                 size,
				Grabs:                grabs,
				Seeders:              seeders,
				Peers:                leechers + seeders,
				PublishDate:          publishDate,
				ImdbID:               imdbID,
				IndexerFlags:         flags,
				DownloadVolumeFactor: downloadVolumeFactor,
				UploadVolumeFactor:   uploadVolumeFactor,
				MinimumRatio:         1,
				MinimumSeedTime:      345600,
			})
		}
	}

	return torrentInfos, nil
}

func getDownloadURL(ix *indexer, torrentID int) string {
	return port.NewURI(ix.BaseURL).
		CombinePath("/torrents.php").
		AddQueryParam("action", "download").
		AddQueryParam("id", strconv.Itoa(torrentID)).
		FullURI()
}

func getInfoURL(ix *indexer, groupID string, torrentID int) string {
	return port.NewURI(ix.BaseURL).
		CombinePath("/torrents.php").
		AddQueryParam("id", groupID).
		AddQueryParam("torrentid", strconv.Itoa(torrentID)).
		FullURI()
}

// fields is C#'s int.Parse, long.Parse and DateTime.Parse, keeping the
// first failure.
type fields struct{ err error }

func (f *fields) long(name, s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	f.fail(name, s, "whole number", err)
	return v
}

func (f *fields) int(name, s string) int {
	v, err := strconv.Atoi(s)
	f.fail(name, s, "whole number", err)
	return v
}

func (f *fields) date(ix *indexer, uploadTime string) time.Time {
	v, err := datetimeutil.Parse(uploadTime+" +0000", datetimeutil.AdjustToUniversal, ix.Clock)
	f.fail("UploadTime", uploadTime, "date", err)
	return v
}

func (f *fields) fail(name, s, kind string, err error) {
	if err != nil && f.err == nil {
		f.err = fmt.Errorf("%s %q isn't a %s", name, s, kind)
	}
}
