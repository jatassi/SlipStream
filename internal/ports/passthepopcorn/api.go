// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/PassThePopcorn/PassThePopcornApi.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package passthepopcorn

type passThePopcornResponse struct {
	TotalResults string
	Movies       []passThePopcornMovie
	Page         string
}

type passThePopcornMovie struct {
	GroupID    string `json:"GroupId"`
	CategoryID string `json:"CategoryId"`
	Title      string
	Year       string
	Cover      string
	Tags       []string
	ImdbID     string `json:"ImdbId"`
	Torrents   []passThePopcornTorrent
}

type passThePopcornTorrent struct {
	ID            int `json:"Id"`
	Quality       string
	Source        string
	Container     string
	Codec         string
	Resolution    string
	Scene         bool
	Size          string
	UploadTime    string
	RemasterTitle string
	Snatched      string
	Seeders       string
	Leechers      string
	ReleaseName   string
	Checked       bool
	GoldenPopcorn bool
	FreeleechType string
}
