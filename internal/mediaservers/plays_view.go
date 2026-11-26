package mediaservers

import (
	"time"

	"github.com/jatassi/slipstream/internal/api"
)

// PlayView is one play as GET /{id}/plays lists it.
type PlayView struct {
	ID         string                  `json:"id" format:"uuid"`
	PlayerKey  string                  `json:"playerKey"`
	Player     string                  `json:"player"`
	Product    string                  `json:"product"`
	Platform   string                  `json:"platform"`
	User       string                  `json:"user"`
	Path       string                  `json:"path" doc:"The file as the server names it"`
	Work       *PlayWorkView           `json:"work,omitempty" doc:"Absent while the play has no snapshot"`
	Decision   string                  `json:"decision" enum:"direct_play,direct_stream,transcode"`
	Video      string                  `json:"video" enum:"direct,transcode,none"`
	Audio      string                  `json:"audio" enum:"direct,transcode,none"`
	Subtitles  string                  `json:"subtitles" enum:"none,direct,burned"`
	Hardware   bool                    `json:"hardware"`
	Reasons    []string                `json:"reasons" nullable:"false"`
	Received   ReceivedView            `json:"received"`
	StartedAt  time.Time               `json:"startedAt"`
	LastSeenAt time.Time               `json:"lastSeenAt"`
	EndedAt    api.Nullable[time.Time] `json:"endedAt" doc:"null while the play is open"`
}

// PlayWorkView is the file a play started on, from its snapshot.
type PlayWorkView struct {
	WorkID string `json:"workId" format:"uuid"`
	Title  string `json:"title"`
	Label  string `json:"label"`
}

// ReceivedView is what the server reports the player received.
type ReceivedView struct {
	Container  api.Nullable[string] `json:"container" doc:"null when unreported"`
	VideoCodec api.Nullable[string] `json:"videoCodec" doc:"null when unreported"`
	Height     api.Nullable[int]    `json:"height" doc:"null when unreported"`
	Bitrate    api.Nullable[int64]  `json:"bitrate" doc:"null when unreported, in bits a second"`
	AudioCodec api.Nullable[string] `json:"audioCodec" doc:"null when unreported"`
	Channels   api.Nullable[int]    `json:"channels" doc:"null when unreported"`
}

type Plays struct {
	Plays []PlayView `json:"plays" nullable:"false" doc:"Newest first"`
	Next  string     `json:"next,omitempty" doc:"The cursor of the next page; absent on the last"`
}
