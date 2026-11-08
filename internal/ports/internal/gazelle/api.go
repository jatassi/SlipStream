// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleApi.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package gazelle

import "github.com/jatassi/slipstream/internal/prowlarr/serializer"

type GazelleArtist struct {
	Name    serializer.String `json:"name"`
	ID      serializer.String `json:"id"`
	Aliasid serializer.String `json:"aliasid"`
}

type GazelleTorrent struct {
	TorrentID     int               `json:"torrentId"`
	EditionID     int               `json:"editionId"`
	Artists       []GazelleArtist   `json:"artists"`
	Remastered    bool              `json:"remastered"`
	RemasterYear  serializer.String `json:"remasterYear"`
	RemasterTitle serializer.String `json:"remasterTitle"`
	ReleaseName   serializer.String `json:"releaseName"`
	Media         serializer.String `json:"media"`
	Encoding      serializer.String `json:"encoding"`
	Format        serializer.String `json:"format"`
	HasLog        bool              `json:"hasLog"`
	LogScore      int               `json:"logScore"`
	HasCue        bool              `json:"hasCue"`
	Scene         bool              `json:"scene"`
	VanityHouse   bool              `json:"vanityHouse"`
	FileCount     int               `json:"fileCount"`
	// SlipStream: Time is the site's text, read as UTC where the parser
	// uses it, as Newtonsoft's DateTimeZoneHandling.Utc reads a DateTime.
	Time                serializer.String `json:"time"`
	Size                serializer.String `json:"size"`
	Snatches            int               `json:"snatches"`
	Seeders             serializer.String `json:"seeders"`
	Leechers            serializer.String `json:"leechers"`
	Category            serializer.String `json:"category"`
	IsFreeLeech         bool              `json:"isFreeLeech"`
	IsNeutralLeech      bool              `json:"isNeutralLeech"`
	IsFreeload          bool              `json:"isFreeload"`
	IsPersonalFreeLeech bool              `json:"isPersonalFreeLeech"`
	CanUseToken         bool              `json:"canUseToken"`
}

type GazelleRelease struct {
	GroupID             serializer.String   `json:"groupId"`
	GroupName           serializer.String   `json:"groupName"`
	TorrentID           int                 `json:"torrentId"`
	Size                serializer.String   `json:"size"`
	FileCount           int                 `json:"fileCount"`
	Snatches            int                 `json:"snatches"`
	Seeders             serializer.String   `json:"seeders"`
	Leechers            serializer.String   `json:"leechers"`
	Category            serializer.String   `json:"category"`
	Artist              serializer.String   `json:"artist"`
	GroupYear           serializer.String   `json:"groupYear"`
	Cover               serializer.String   `json:"cover"`
	Tags                []serializer.String `json:"tags"`
	ReleaseType         serializer.String   `json:"releaseType"`
	TotalLeechers       int                 `json:"totalLeechers"`
	TotalSeeders        int                 `json:"totalSeeders"`
	TotalSnatched       int                 `json:"totalSnatched"`
	MaxSize             int64               `json:"maxSize"`
	GroupTime           serializer.String   `json:"groupTime"`
	Torrents            []GazelleTorrent    `json:"torrents"`
	IsFreeLeech         bool                `json:"isFreeLeech"`
	IsNeutralLeech      bool                `json:"isNeutralLeech"`
	IsFreeload          bool                `json:"isFreeload"`
	IsPersonalFreeLeech bool                `json:"isPersonalFreeLeech"`
	CanUseToken         bool                `json:"canUseToken"`
}

type GazelleResponse struct {
	Status   serializer.String      `json:"status"`
	Response *GazelleBrowseResponse `json:"response"`
}

type GazelleBrowseResponse struct {
	Results     []GazelleRelease  `json:"results"`
	CurrentPage serializer.String `json:"currentPage"`
	Pages       serializer.String `json:"pages"`
}

type GazelleAuthResponse struct {
	Status   serializer.String     `json:"status"`
	Response *GazelleIndexResponse `json:"response"`
}

type GazelleIndexResponse struct {
	Username serializer.String `json:"username"`
	ID       serializer.String `json:"id"`
	Authkey  serializer.String `json:"authkey"`
	Passkey  serializer.String `json:"passkey"`
}

type GazelleErrorResponse struct {
	Error serializer.String `json:"error"`
}
