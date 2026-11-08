// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BroadcastheNet/BroadcastheNetTorrentQuery.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package broadcasthenet

// SlipStream: the json tags are what Prowlarr's ToJson writes, camelCase
// with defaults left out.
type broadcastheNetTorrentQuery struct {
	ID         string   `json:"id,omitempty"`
	Category   string   `json:"category,omitempty"`
	Name       string   `json:"name,omitempty"`
	Search     string   `json:"search,omitempty"`
	Codec      []string `json:"codec,omitempty"`
	Container  []string `json:"container,omitempty"`
	Source     []string `json:"source,omitempty"`
	Resolution []string `json:"resolution,omitempty"`
	Origin     []string `json:"origin,omitempty"`
	Hash       string   `json:"hash,omitempty"`
	Tvdb       string   `json:"tvdb,omitempty"`
	Imdb       string   `json:"imdb,omitempty"`
	Tvrage     string   `json:"tvrage,omitempty"`
	Age        string   `json:"age,omitempty"`
}
