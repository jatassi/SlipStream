// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleApi.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package gazelle

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// String is a C# string property as Newtonsoft fills it: from a JSON
// string, or from a number's text. Gazelle sites send size, seeders and
// groupId as either.
type String string

func (s *String) UnmarshalJSON(b []byte) error {
	switch {
	case bytes.Equal(b, []byte("null")):
	case bytes.HasPrefix(b, []byte(`"`)):
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return fmt.Errorf("read a JSON string: %w", err)
		}
		*s = String(v)
	default:
		*s = String(b)
	}
	return nil
}

type GazelleArtist struct {
	Name    String `json:"name"`
	ID      String `json:"id"`
	Aliasid String `json:"aliasid"`
}

type GazelleTorrent struct {
	TorrentID     int             `json:"torrentId"`
	EditionID     int             `json:"editionId"`
	Artists       []GazelleArtist `json:"artists"`
	Remastered    bool            `json:"remastered"`
	RemasterYear  String          `json:"remasterYear"`
	RemasterTitle String          `json:"remasterTitle"`
	ReleaseName   String          `json:"releaseName"`
	Media         String          `json:"media"`
	Encoding      String          `json:"encoding"`
	Format        String          `json:"format"`
	HasLog        bool            `json:"hasLog"`
	LogScore      int             `json:"logScore"`
	HasCue        bool            `json:"hasCue"`
	Scene         bool            `json:"scene"`
	VanityHouse   bool            `json:"vanityHouse"`
	FileCount     int             `json:"fileCount"`
	// SlipStream: Time is the site's text, read as UTC where the parser
	// uses it, as Newtonsoft's DateTimeZoneHandling.Utc reads a DateTime.
	Time                String `json:"time"`
	Size                String `json:"size"`
	Snatches            int    `json:"snatches"`
	Seeders             String `json:"seeders"`
	Leechers            String `json:"leechers"`
	Category            String `json:"category"`
	IsFreeLeech         bool   `json:"isFreeLeech"`
	IsNeutralLeech      bool   `json:"isNeutralLeech"`
	IsFreeload          bool   `json:"isFreeload"`
	IsPersonalFreeLeech bool   `json:"isPersonalFreeLeech"`
	CanUseToken         bool   `json:"canUseToken"`
}

type GazelleRelease struct {
	GroupID             String           `json:"groupId"`
	GroupName           String           `json:"groupName"`
	TorrentID           int              `json:"torrentId"`
	Size                String           `json:"size"`
	FileCount           int              `json:"fileCount"`
	Snatches            int              `json:"snatches"`
	Seeders             String           `json:"seeders"`
	Leechers            String           `json:"leechers"`
	Category            String           `json:"category"`
	Artist              String           `json:"artist"`
	GroupYear           String           `json:"groupYear"`
	Cover               String           `json:"cover"`
	Tags                []String         `json:"tags"`
	ReleaseType         String           `json:"releaseType"`
	TotalLeechers       int              `json:"totalLeechers"`
	TotalSeeders        int              `json:"totalSeeders"`
	TotalSnatched       int              `json:"totalSnatched"`
	MaxSize             int64            `json:"maxSize"`
	GroupTime           String           `json:"groupTime"`
	Torrents            []GazelleTorrent `json:"torrents"`
	IsFreeLeech         bool             `json:"isFreeLeech"`
	IsNeutralLeech      bool             `json:"isNeutralLeech"`
	IsFreeload          bool             `json:"isFreeload"`
	IsPersonalFreeLeech bool             `json:"isPersonalFreeLeech"`
	CanUseToken         bool             `json:"canUseToken"`
}

type GazelleResponse struct {
	Status   String                 `json:"status"`
	Response *GazelleBrowseResponse `json:"response"`
}

type GazelleBrowseResponse struct {
	Results     []GazelleRelease `json:"results"`
	CurrentPage String           `json:"currentPage"`
	Pages       String           `json:"pages"`
}

type GazelleAuthResponse struct {
	Status   String                `json:"status"`
	Response *GazelleIndexResponse `json:"response"`
}

type GazelleIndexResponse struct {
	Username String `json:"username"`
	ID       String `json:"id"`
	Authkey  String `json:"authkey"`
	Passkey  String `json:"passkey"`
}

type GazelleErrorResponse struct {
	Error String `json:"error"`
}
