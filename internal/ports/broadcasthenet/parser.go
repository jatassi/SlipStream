// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BroadcastheNet/BroadcastheNetParser.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package broadcasthenet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
)

var regexProtocol = regexp.MustCompile(`^https?:`)

// jsonRPCResponse is NzbDrone.Common's JsonRpcResponse<T>.
type jsonRPCResponse[T any] struct {
	ID     string
	Result *T
	Error  json.RawMessage
}

// SlipStream: the messages are SlipStream's own, per WRITING.md. A 401 is
// a key the site refused, which SlipStream names account_refused.
func parseResponse(ix *indexer, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	switch indexerResponse.StatusCode {
	case http.StatusUnauthorized:
		return nil, port.IndexerAuthError("")
	case http.StatusNotFound:
		return nil, port.IndexerError("the Indexer's API responded with status 404 and may have changed")
	case http.StatusServiceUnavailable:
		return nil, port.RequestLimitReached("the hourly API limit")
	default:
		if indexerResponse.StatusCode != http.StatusOK {
			return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
		}
	}

	if strings.Contains(indexerResponse.Headers.Get("Content-Type"), "text/html") {
		return nil, port.IndexerError("the Indexer responded with HTML where JSON was expected")
	}

	if strings.Contains(strings.ToLower(indexerResponse.Content), "call limit exceeded") {
		return nil, port.RequestLimitReached("the hourly API limit")
	}

	if indexerResponse.Content == "Query execution was interrupted" {
		return nil, port.IndexerError("the Indexer's API responded with an internal server error")
	}

	var jsonResponse jsonRPCResponse[broadcastheNetTorrents]
	if err := json.Unmarshal(indexerResponse.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	if len(jsonResponse.Error) > 0 && string(jsonResponse.Error) != "null" {
		return nil, port.IndexerError("the Indexer's API responded with the error %s", compact(jsonResponse.Error))
	}
	if jsonResponse.Result == nil {
		return nil, port.IndexerError("the Indexer's API responded with no result")
	}

	if jsonResponse.Result.Results == 0 {
		return releaseInfos, nil
	}
	torrents, err := inOrder(jsonResponse.Result.Torrents)
	if err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}
	if torrents == nil {
		return releaseInfos, nil
	}

	protocol := indexerResponse.Request.URL.Scheme + ":"

	for _, torrent := range torrents {
		var flags []port.SiteFlag

		if strings.EqualFold(torrent.Origin, "INTERNAL") {
			flags = append(flags, port.Internal)
		}

		// SlipStream: IndexerBase adds Scene from TorrentInfo.Scene, which the
		// Go TorrentInfo doesn't have. It has no TvRageId, Origin, Source,
		// Container, Codec or Resolution either, because a Release carries
		// none of them.
		if strings.EqualFold(torrent.Origin, "SCENE") {
			flags = append(flags, port.Scene)
		}

		var minimumSeedTime int64 = 86400 // 120 hours for seasons and 24 hours for episodes
		if strings.EqualFold(torrent.Category, "SEASON") {
			minimumSeedTime = 432000
		}

		releaseInfo := port.TorrentInfo{
			GUID:                 fmt.Sprintf("BTN-%d", torrent.TorrentID),
			InfoURL:              fmt.Sprintf("%s//broadcasthe.net/torrents.php?id=%d&torrentid=%d", protocol, torrent.GroupID, torrent.TorrentID),
			DownloadURL:          regexProtocol.ReplaceAllLiteralString(torrent.DownloadURL, protocol),
			Title:                getTitle(torrent),
			Categories:           ix.Categories.FromTracker(torrent.Resolution),
			InfoHash:             torrent.InfoHash,
			Size:                 int64(torrent.Size),
			Grabs:                torrent.Snatched.value(),
			Seeders:              torrent.Seeders.value(),
			Peers:                torrent.Leechers.value() + torrent.Seeders.value(),
			PublishDate:          time.Unix(int64(torrent.Time), 0).UTC(),
			IndexerFlags:         flags,
			DownloadVolumeFactor: 0,
			UploadVolumeFactor:   1,
			MinimumRatio:         1,
			MinimumSeedTime:      minimumSeedTime,
		}

		if torrent.TvdbID.value() > 0 {
			releaseInfo.TvdbID = torrent.TvdbID.value()
		}

		if imdbID, err := strconv.Atoi(strings.TrimSpace(torrent.ImdbID)); strings.TrimSpace(torrent.ImdbID) != "" && err == nil {
			releaseInfo.ImdbID = imdbID
		}

		// Default to TV if category could not be mapped
		if len(releaseInfo.Categories) == 0 {
			releaseInfo.Categories = []newznab.Category{newznab.TV}
		}

		releaseInfos = append(releaseInfos, releaseInfo)
	}

	return releaseInfos, nil
}

var (
	h265 = regexp.MustCompile(`\b(H\.?265)\b`)
	h264 = regexp.MustCompile(`\b(H\.?264)\b`)
)

func getTitle(torrent broadcastheNetTorrent) string {
	releaseName := strings.ReplaceAll(torrent.ReleaseName, "\\", "")

	if c := strings.ToUpper(torrent.Container); c == "M2TS" || c == "ISO" {
		releaseName = h265.ReplaceAllLiteralString(releaseName, "HEVC")
		releaseName = h264.ReplaceAllLiteralString(releaseName, "AVC")
	}

	return releaseName
}

// inOrder is the Dictionary<int, BroadcastheNetTorrent>'s Values in the
// order the site sent them, nil when the site sent no object.
func inOrder(raw json.RawMessage) ([]broadcastheNetTorrent, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("read the torrents: %w", err)
	}
	torrents := []broadcastheNetTorrent{}
	for dec.More() {
		if _, err := dec.Token(); err != nil {
			return nil, fmt.Errorf("read the torrents: %w", err)
		}
		var t broadcastheNetTorrent
		if err := dec.Decode(&t); err != nil {
			return nil, fmt.Errorf("read the torrents: %w", err)
		}
		torrents = append(torrents, t)
	}
	return torrents, nil
}

func compact(raw json.RawMessage) string {
	var b bytes.Buffer
	if json.Compact(&b, raw) != nil {
		return string(raw)
	}
	return b.String()
}
