// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BroadcastheNet/BroadcastheNetTorrent.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package broadcasthenet

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type broadcastheNetTorrent struct {
	GroupName      string
	GroupID        looseInt
	TorrentID      looseInt
	SeriesID       looseInt
	Series         string
	SeriesBanner   string
	SeriesPoster   string
	YoutubeTrailer string
	Category       string
	Snatched       *looseInt
	Seeders        *looseInt
	Leechers       *looseInt
	Source         string
	Container      string
	Codec          string
	Resolution     string
	Origin         string
	ReleaseName    string
	Size           looseInt
	Time           looseInt
	TvdbID         *looseInt
	TvrageID       *looseInt
	ImdbID         string
	InfoHash       string
	DownloadURL    string
}

// looseInt is a C# int or long property, which Newtonsoft fills from a
// JSON number or a string of digits alike: BroadcasTheNet sends numbers
// as strings.
type looseInt int64

func (n *looseInt) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	s := string(b)
	if len(b) > 0 && b[0] == '"' {
		if err := json.Unmarshal(b, &s); err != nil {
			return fmt.Errorf("read a number: %w", err)
		}
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("read a number: %w", err)
	}
	*n = looseInt(v)
	return nil
}

// value is the nullable's GetValueOrDefault().
func (n *looseInt) value() int {
	if n == nil {
		return 0
	}
	return int(*n)
}
