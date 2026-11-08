// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/FileList/FileListParser.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package filelist

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

// SlipStream: the messages are SlipStream's own, per WRITING.md. A 401
// is the Basic credential the site refused, which SlipStream names
// account_refused with FileList's own words.
func parseResponse(ix *indexer, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	if indexerResponse.StatusCode == http.StatusUnauthorized {
		return nil, port.IndexerAuthError(siteError(indexerResponse))
	}

	if indexerResponse.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	if message := siteError(indexerResponse); message != "" {
		return nil, port.IndexerError("the Indexer responded with the error %q", message)
	}

	if contentType := indexerResponse.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q where application/json was expected", contentType)
	}

	var releaseInfos []port.TorrentInfo

	var results []fileListTorrent
	if err := json.Unmarshal(indexerResponse.Body, &results); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	for _, row := range results {
		// skip non-freeleech results when freeleech only is set
		if ix.Settings.FreeleechOnly && !bool(row.FreeLeech) {
			continue
		}

		id := row.ID

		var flags []port.SiteFlag
		if bool(row.Internal) {
			flags = append(flags, port.Internal)
		}

		imdbID := 0
		if len(row.ImdbID) > 2 {
			imdbID, _ = strconv.Atoi(strings.TrimLeft(row.ImdbID, "t"))
		}

		downloadVolumeFactor := 1.0
		if bool(row.FreeLeech) {
			downloadVolumeFactor = 0
		}
		uploadVolumeFactor := 1.0
		if bool(row.DoubleUp) {
			uploadVolumeFactor = 2
		}

		publishDate, err := datetimeutil.Parse(row.UploadDate+" +0200", datetimeutil.AdjustToUniversal, ix.Clock)
		if err != nil {
			return nil, port.IndexerError("read torrent %d of the Indexer's search response: %v", id, err)
		}

		// SlipStream: TorrentInfo has no Description or Genres, because a
		// Release carries neither.
		releaseInfos = append(releaseInfos, port.TorrentInfo{
			GUID:                 fmt.Sprintf("FileList-%d", id),
			Title:                row.Name,
			Size:                 row.Size,
			Categories:           ix.Categories.FromDesc(row.Category),
			DownloadURL:          getDownloadURL(ix, id),
			InfoURL:              getInfoURL(ix, id),
			Seeders:              row.Seeders,
			Peers:                row.Leechers + row.Seeders,
			PublishDate:          publishDate,
			ImdbID:               imdbID,
			IndexerFlags:         flags,
			Files:                int(row.Files),
			Grabs:                int(row.TimesCompleted),
			DownloadVolumeFactor: downloadVolumeFactor,
			UploadVolumeFactor:   uploadVolumeFactor,
			MinimumRatio:         1,
			MinimumSeedTime:      172800, // 48 hours
		})
	}

	return releaseInfos, nil
}

// siteError is the error a {"error": ...} body carries, "" for none.
func siteError(indexerResponse *port.IndexerResponse) string {
	var errorResponse fileListErrorResponse
	if !strings.HasPrefix(indexerResponse.Content, `{"error"`) || json.Unmarshal(indexerResponse.Body, &errorResponse) != nil {
		return ""
	}
	return errorResponse.Error
}

func getDownloadURL(ix *indexer, torrentID uint) string {
	return port.NewURI(ix.BaseURL).
		CombinePath("/download.php").
		AddQueryParam("id", strconv.FormatUint(uint64(torrentID), 10)).
		AddQueryParam("passkey", strings.TrimSpace(ix.Settings.Passkey)).
		FullURI()
}

func getInfoURL(ix *indexer, torrentID uint) string {
	return port.NewURI(ix.BaseURL).
		CombinePath("/details.php").
		AddQueryParam("id", strconv.FormatUint(uint64(torrentID), 10)).
		FullURI()
}
