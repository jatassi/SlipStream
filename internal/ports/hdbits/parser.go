// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsParser.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package hdbits

import (
	"bytes"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

var halfLeechMediums = []int{int(hdBitsMediumBluray), int(hdBitsMediumRemux), int(hdBitsMediumCapture)}

// SlipStream: the messages are SlipStream's own, per WRITING.md. A failed
// or missing authentication is a key the site refused, which SlipStream
// names account_refused with HDBits' own words.
func parseResponse(ix *indexer, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	if indexerResponse.StatusCode == http.StatusForbidden {
		return nil, port.RequestLimitReached("the query limit")
	}

	if indexerResponse.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	var jsonResponse hdBitsResponse
	if err := json.Unmarshal(indexerResponse.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}
	if jsonResponse.Status == nil {
		return nil, port.IndexerError("the Indexer's search response has no status")
	}

	switch *jsonResponse.Status {
	case statusSuccess:
	case statusAuthDataMissing, statusAuthFailed:
		return nil, port.IndexerAuthError(jsonResponse.Message)
	default:
		if jsonResponse.Message == "" {
			return nil, port.IndexerError("the Indexer's API responded with status %d", *jsonResponse.Status)
		}
		return nil, port.IndexerError("the Indexer's API responded with status %d and %q", *jsonResponse.Status, jsonResponse.Message)
	}

	if !bytes.HasPrefix(bytes.TrimSpace(jsonResponse.Data), []byte("[")) {
		return nil, port.IndexerError("the Indexer's search response has no list of results")
	}

	var releaseInfos []port.TorrentInfo

	var queryResults []torrentQueryResponse
	if err := json.Unmarshal(jsonResponse.Data, &queryResults); err != nil {
		return nil, port.IndexerError("read the Indexer's search results: %v", err)
	}

	for _, result := range queryResults {
		if ix.Settings.FreeleechOnly && result.FreeLeech != "yes" {
			continue
		}

		id := string(result.ID)

		publishDate, err := datetimeutil.Parse(result.Added, datetimeutil.AdjustToUniversal, ix.Clock)
		if err != nil {
			return nil, port.IndexerError("read torrent %s of the Indexer's search response: %v", id, err)
		}

		var imdbID, tvdbID int
		if result.ImdbInfo != nil {
			imdbID = result.ImdbInfo.ID
		}
		if result.TvdbInfo != nil && result.TvdbInfo.ID != nil {
			tvdbID = *result.TvdbInfo.ID
		}

		// SlipStream: TorrentInfo has no Year, because a Release carries none.
		releaseInfos = append(releaseInfos, port.TorrentInfo{
			GUID:                 "HDBits-" + id,
			Title:                getTitle(ix, result),
			Size:                 result.Size,
			Categories:           ix.Categories.FromTracker(strconv.Itoa(result.TypeCategory)),
			InfoHash:             result.Hash,
			DownloadURL:          getDownloadURL(ix, id),
			InfoURL:              getInfoURL(ix, id),
			Seeders:              result.Seeders,
			Grabs:                int(result.TimesCompleted),
			Files:                int(result.NumFiles),
			Peers:                result.Leechers + result.Seeders,
			PublishDate:          publishDate,
			ImdbID:               imdbID,
			TvdbID:               tvdbID,
			DownloadVolumeFactor: getDownloadVolumeFactor(result),
			UploadVolumeFactor:   getUploadVolumeFactor(result),
			IndexerFlags:         getIndexerFlags(result),
		})
	}

	return releaseInfos, nil
}

func getTitle(ix *indexer, item torrentQueryResponse) string {
	// Use release name for XXX content and full discs
	if item.TypeCategory != 7 && item.TypeMedium != 1 && ix.Settings.UseFilenames && strings.TrimSpace(item.FileName) != "" {
		return replaceFold(item.FileName, ".torrent", "")
	}
	return item.Name
}

// replaceFold is string.Replace with StringComparison.InvariantCultureIgnoreCase.
func replaceFold(s, old, replacement string) string {
	var b strings.Builder
	lower, lowerOld := strings.ToLower(s), strings.ToLower(old)
	for {
		i := strings.Index(lower, lowerOld)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		b.WriteString(replacement)
		s, lower = s[i+len(old):], lower[i+len(old):]
	}
}

func getIndexerFlags(item torrentQueryResponse) []port.SiteFlag {
	var flags []port.SiteFlag

	if item.TypeOrigin == 1 {
		flags = append(flags, port.Internal)
	}

	if item.TypeExclusive == 1 {
		flags = append(flags, port.Exclusive)
	}

	return flags
}

func getDownloadVolumeFactor(item torrentQueryResponse) float64 {
	if item.FreeLeech == "yes" {
		return 0
	}

	// 100% Neutral Leech: all XXX content.
	if item.TypeCategory == 7 {
		return 0
	}

	// 50% Free Leech: all full discs, remuxes, captures and all internal encodes, also all TV and Documentary content.
	if slices.Contains(halfLeechMediums, item.TypeMedium) || item.TypeOrigin == 1 || item.TypeCategory == 2 || item.TypeCategory == 3 {
		return 0.5
	}

	return 1
}

func getUploadVolumeFactor(item torrentQueryResponse) float64 {
	// 100% Neutral Leech: all XXX content.
	if item.TypeCategory == 7 {
		return 0
	}
	return 1
}

func getDownloadURL(ix *indexer, torrentID string) string {
	return port.NewURI(ix.BaseURL).
		CombinePath("/download.php").
		AddQueryParam("id", torrentID).
		AddQueryParam("passkey", ix.Settings.Passkey).
		FullURI()
}

func getInfoURL(ix *indexer, torrentID string) string {
	return port.NewURI(ix.BaseURL).
		CombinePath("/details.php").
		AddQueryParam("id", torrentID).
		FullURI()
}
