// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BroadcastheNet/BroadcastheNetTorrent.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package broadcasthenet

import (
	"github.com/jatassi/slipstream/internal/prowlarr/serializer"
)

type broadcastheNetTorrent struct {
	GroupName      string
	GroupID        serializer.Int
	TorrentID      serializer.Int
	SeriesID       serializer.Int
	Series         string
	SeriesBanner   string
	SeriesPoster   string
	YoutubeTrailer string
	Category       string
	Snatched       *serializer.Int
	Seeders        *serializer.Int
	Leechers       *serializer.Int
	Source         string
	Container      string
	Codec          string
	Resolution     string
	Origin         string
	ReleaseName    string
	Size           serializer.Int
	Time           serializer.Int
	TvdbID         *serializer.Int
	TvrageID       *serializer.Int
	ImdbID         string
	InfoHash       string
	DownloadURL    string
}

// value is the nullable's GetValueOrDefault().
func value(n *serializer.Int) int {
	if n == nil {
		return 0
	}
	return int(*n)
}
