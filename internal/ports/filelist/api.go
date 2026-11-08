// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/FileList/FileListApi.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package filelist

import "fmt"

type fileListTorrent struct {
	ID               uint   `json:"id"`
	Name             string `json:"name"`
	Size             int64  `json:"size"`
	Leechers         int    `json:"leechers"`
	Seeders          int    `json:"seeders"`
	TimesCompleted   uint   `json:"times_completed"`
	Files            uint   `json:"files"`
	ImdbID           string `json:"imdb"`
	Internal         stBool `json:"internal"`
	FreeLeech        stBool `json:"freeleech"`
	DoubleUp         stBool `json:"doubleup"`
	UploadDate       string `json:"upload_date"`
	Category         string `json:"category"`
	SmallDescription string `json:"small_description"`
}

type fileListErrorResponse struct {
	Error string `json:"error"`
}

// stBool is Prowlarr's STJson BooleanConverter: true, false, 1 or 0.
type stBool bool

func (b *stBool) UnmarshalJSON(raw []byte) error {
	switch string(raw) {
	case "true", "1":
		*b = true
	case "false", "0", "null":
		*b = false
	default:
		return fmt.Errorf("%s isn't a boolean", raw)
	}
	return nil
}
