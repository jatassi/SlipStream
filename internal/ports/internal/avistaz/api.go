// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Avistaz/AvistazApi.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package avistaz

type AvistazRelease struct {
	URL      string            `json:"url"`
	Download string            `json:"download"`
	Category map[string]string `json:"category"`

	MovieTvinfo *AvistazIDInfo `json:"movie_tv"`

	CreatedAtIso string `json:"created_at_iso"`

	FileName string `json:"file_name"`

	ReleaseTitle string `json:"release_title"`

	InfoHash string `json:"info_hash"`

	Leech     int `json:"leech"`
	Completed int `json:"completed"`
	Seed      int `json:"seed"`

	FileSize int64 `json:"file_size"`

	FileCount int `json:"file_count"`

	DownloadMultiply *float64 `json:"download_multiply"`

	UploadMultiply *float64 `json:"upload_multiply"`

	VideoQuality string `json:"video_quality"`

	Type string `json:"type"`

	Format string `json:"format"`

	Audio    []AvistazLanguage `json:"audio"`
	Subtitle []AvistazLanguage `json:"subtitle"`
}

type AvistazLanguage struct {
	ID       int    `json:"id"`
	Language string `json:"language"`
}

type AvistazResponse struct {
	Data []AvistazRelease `json:"data"`
}

type AvistazErrorResponse struct {
	Message string `json:"message"`
}

type AvistazIDInfo struct {
	Tmdb string `json:"tmdb"`
	Tvdb string `json:"tvdb"`
	Imdb string `json:"imdb"`
}

type AvistazAuthResponse struct {
	Token string `json:"token"`
}
