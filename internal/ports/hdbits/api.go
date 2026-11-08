// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsApi.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package hdbits

import (
	"encoding/json"

	"github.com/jatassi/slipstream/internal/prowlarr/serializer"
)

// SlipStream: the json tags are what Prowlarr's ToJson writes, camelCase
// with nulls left out.
type torrentQuery struct {
	Username      string    `json:"username"`
	Passkey       string    `json:"passkey"`
	Hash          string    `json:"hash,omitempty"`
	Search        string    `json:"search,omitempty"`
	Category      []int     `json:"category,omitempty"`
	Codec         []int     `json:"codec,omitempty"`
	Medium        []int     `json:"medium,omitempty"`
	Origin        []int     `json:"origin,omitempty"`
	ImdbInfo      *imdbInfo `json:"imdb,omitempty"`
	TvdbInfo      *tvdbInfo `json:"tvdb,omitempty"`
	FileInTorrent string    `json:"file_in_torrent,omitempty"`
	SnatchedOnly  *bool     `json:"snatched_only,omitempty"`
	Limit         *int      `json:"limit,omitempty"`
	Page          *int      `json:"page,omitempty"`
}

type hdBitsResponse struct {
	Status  *statusCode     `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type torrentQueryResponse struct {
	ID             serializer.String `json:"id"`
	Hash           string            `json:"hash"`
	Leechers       int               `json:"leechers"`
	Seeders        int               `json:"seeders"`
	Name           string            `json:"name"`
	TimesCompleted uint              `json:"times_completed"`
	Size           int64             `json:"size"`
	UtAdded        int64             `json:"utadded"`
	Added          string            `json:"added"`
	Comments       uint              `json:"comments"`
	NumFiles       uint              `json:"numfiles"`
	FileName       string            `json:"filename"`
	FreeLeech      string            `json:"freeleech"`
	TypeCategory   int               `json:"type_category"`
	TypeCodec      int               `json:"type_codec"`
	TypeMedium     int               `json:"type_medium"`
	TypeOrigin     int               `json:"type_origin"`
	TypeExclusive  int               `json:"type_exclusive"`
	ImdbInfo       *imdbInfo         `json:"imdb"`
	TvdbInfo       *tvdbInfo         `json:"tvdb"`
}

type imdbInfo struct {
	ID            int      `json:"id"`
	EnglishTitle  string   `json:"englishTitle,omitempty"`
	OriginalTitle string   `json:"originalTitle,omitempty"`
	Year          *int     `json:"year,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Rating        *float32 `json:"rating,omitempty"`
}

type tvdbInfo struct {
	ID      *int              `json:"id,omitempty"`
	Season  *int              `json:"season,omitempty"`
	Episode serializer.String `json:"episode,omitempty"`
}

type statusCode int

const (
	statusSuccess         statusCode = 0
	statusAuthDataMissing statusCode = 4
	statusAuthFailed      statusCode = 5
)
