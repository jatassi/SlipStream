package playback

import "time"

// Window is the plays a report covers.
type Window struct {
	// Since is the later of now minus Keep and the server's plays_since.
	Since time.Time
	// Keep is the Keep play history for setting.
	Keep time.Duration
}

// Report is why each player of one server transcoded, and what would
// avoid it.
type Report struct {
	Since      time.Time `json:"since" doc:"The report covers plays that started at or after this"`
	Window     int64     `json:"window" minimum:"1" doc:"Keep play history for, in seconds"`
	Reachable  bool      `json:"reachable" doc:"false while the server's Connect from Node is offline, when nothing is recorded"`
	Plex       bool      `json:"plex" doc:"The server is Plex, which gives no reasons"`
	Plays      int       `json:"plays" minimum:"0"`
	Transcoded int       `json:"transcoded" minimum:"0"`
	Unmatched  int       `json:"unmatched" minimum:"0" doc:"Transcoded plays with no snapshot"`
	Players    []Player  `json:"players" nullable:"false" doc:"Worst first"`
}

type Player struct {
	Key           string    `json:"key"`
	Name          string    `json:"name"`
	Product       string    `json:"product"`
	Platform      string    `json:"platform"`
	LastPlayedAt  time.Time `json:"lastPlayedAt"`
	Plays         int       `json:"plays" minimum:"0"`
	DirectPlays   int       `json:"directPlays" minimum:"0"`
	DirectStreams int       `json:"directStreams" minimum:"0"`
	Transcoded    int       `json:"transcoded" minimum:"0"`
	Unmatched     int       `json:"unmatched" minimum:"0"`
	Groups        []Group   `json:"groups" nullable:"false" doc:"Largest first"`
}

// Group is a player's transcodes that the same dimensions blocked.
type Group struct {
	Dimensions []Blocked   `json:"dimensions" nullable:"false" doc:"In dimension table order"`
	Cause      []Blocked   `json:"cause" nullable:"false" doc:"The dimensions the player's line names: the first no Target fixes, else at most two"`
	Reasons    []string    `json:"reasons" nullable:"false" doc:"As the server gave them"`
	Targets    []TargetRef `json:"targets" nullable:"false" doc:"The Targets the transcoded files met when they played"`
	Plays      int         `json:"plays" minimum:"1"`
	Outcomes   []Outcome   `json:"outcomes" nullable:"false" doc:"Largest first"`
}

// Blocked is a dimension and the played file's value on it.
type Blocked struct {
	Dimension Dimension `json:"dimension" enum:"resolution,video_codec,hdr,audio_codec,channels,container,subtitles,audio_track,stream_count,video_profile,bit_depth,frame_rate,video_shape,audio_format,bitrate,unreadable,player_error,unknown"`
	Value     string    `json:"value" doc:"The played file's value in release words; the reason's name for audio_track, video_shape and audio_format; empty where none applies"`
	Choosable bool      `json:"choosable" doc:"A Target can choose this dimension"`
}

// Outcome is what would avoid some of a group's transcodes.
type Outcome struct {
	Kind       OutcomeKind `json:"kind" enum:"version,suggest,existing,no_target,unproven"`
	Plays      int         `json:"plays" minimum:"1"`
	Version    *Version    `json:"version,omitempty" doc:"Absent unless kind is version"`
	Suggestion *Suggestion `json:"suggestion,omitempty" doc:"Absent unless kind is suggest"`
	Target     *TargetRef  `json:"target,omitempty" doc:"Absent unless kind is existing"`
	Works      []WorkLine  `json:"works,omitempty" nullable:"false" doc:"Absent unless kind is existing"`
	Missing    *Blocked    `json:"missing,omitempty" doc:"Absent unless kind is unproven: the first dimension with no proven value"`
	Evidence   []Evidence  `json:"evidence,omitempty" nullable:"false" doc:"Absent unless kind is version, suggest or existing: one per proven value"`
}

type Version struct {
	FileID       string `json:"fileId" format:"uuid"`
	Label        string `json:"label"`
	SeparateItem bool   `json:"separateItem" doc:"The server lists the file as a separate item, which the player can't choose as a version"`
}

// Suggestion is a Target that would give the player files it plays
// directly, holding only values the player was proven to receive.
type Suggestion struct {
	Name           string    `json:"name"`
	Highest        string    `json:"highest" enum:"2160p,1080p,720p,480p"`
	VideoCodecs    []string  `json:"videoCodecs" nullable:"false" enum:"x264,x265,AV1,VP9,XviD,MPEG2"`
	NotHDR         []string  `json:"notHdr" nullable:"false" enum:"DV,HDR10+,HDR10,HLG,SDR"`
	NotAudioCodecs []string  `json:"notAudioCodecs" nullable:"false" enum:"TrueHD,DTS-HD MA,DTS,DD+,DD,AAC,FLAC,PCM,Opus,MP3"`
	NotChannels    []string  `json:"notChannels" nullable:"false" enum:"7.1,5.1,2.0,1.0"`
	Blocked        []Blocked `json:"blocked" nullable:"false" doc:"Values left out because they blocked direct play"`
	Unproven       []Blocked `json:"unproven" nullable:"false" doc:"Values left out because the player hasn't received them unchanged in the window"`
}

// WorkLine is one Work an existing Target covers, and why it doesn't yet
// play directly.
type WorkLine struct {
	WorkID      string     `json:"workId" format:"uuid"`
	ItemID      string     `json:"itemId" format:"uuid" doc:"The snapshot's Item, which Search for one scopes to"`
	Title       string     `json:"title"`
	ContentType string     `json:"contentType" doc:"The Work's Content Type, whose noun the line uses"`
	State       WorkState  `json:"state" enum:"no_file,not_in_profile,not_listed,not_proven"`
	Profile     *TargetRef `json:"profile,omitempty" doc:"Absent unless state is not_in_profile"`
	Fact        *FactValue `json:"fact,omitempty" doc:"Absent unless state is not_proven: the first tracked fact not proven"`
}

type FactValue struct {
	Fact  FactKind `json:"fact" enum:"container,bitrate,resolution,video_codec,video_profile,bit_depth,hdr,dolby_vision,audio_codec,channels,subtitle_codec,subtitle_image"`
	Value string   `json:"value"`
}

// Evidence is the newest play that proves a value.
type Evidence struct {
	FactValue
	PlayID      string    `json:"playId" format:"uuid"`
	WorkID      string    `json:"workId" format:"uuid"`
	Title       string    `json:"title"`
	ContentType string    `json:"contentType"`
	StartedAt   time.Time `json:"startedAt"`
}
