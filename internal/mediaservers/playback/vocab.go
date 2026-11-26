package playback

// Dimension is what can stop a player playing a file directly.
type Dimension string

const (
	Resolution   Dimension = "resolution"
	VideoCodec   Dimension = "video_codec"
	HDR          Dimension = "hdr"
	AudioCodec   Dimension = "audio_codec"
	Channels     Dimension = "channels"
	Container    Dimension = "container"
	Subtitles    Dimension = "subtitles"
	AudioTrack   Dimension = "audio_track"
	StreamCount  Dimension = "stream_count"
	VideoProfile Dimension = "video_profile"
	BitDepth     Dimension = "bit_depth"
	FrameRate    Dimension = "frame_rate"
	VideoShape   Dimension = "video_shape"
	AudioFormat  Dimension = "audio_format"
	Bitrate      Dimension = "bitrate"
	Unreadable   Dimension = "unreadable"
	PlayerError  Dimension = "player_error"
	Unknown      Dimension = "unknown"
)

// FactKind is a fact of a played file that a play can prove the player
// received unchanged.
type FactKind string

const (
	FactContainer     FactKind = "container"
	FactBitrate       FactKind = "bitrate"
	FactResolution    FactKind = "resolution"
	FactVideoCodec    FactKind = "video_codec"
	FactVideoProfile  FactKind = "video_profile"
	FactBitDepth      FactKind = "bit_depth"
	FactHDR           FactKind = "hdr"
	FactDolbyVision   FactKind = "dolby_vision"
	FactAudioCodec    FactKind = "audio_codec"
	FactChannels      FactKind = "channels"
	FactSubtitleCodec FactKind = "subtitle_codec"
	FactSubtitleImage FactKind = "subtitle_image"
)

func FactKinds() []FactKind {
	return []FactKind{
		FactContainer, FactBitrate, FactResolution, FactVideoCodec, FactVideoProfile, FactBitDepth,
		FactHDR, FactDolbyVision, FactAudioCodec, FactChannels, FactSubtitleCodec, FactSubtitleImage,
	}
}

// OutcomeKind is what would avoid a group's transcodes.
type OutcomeKind string

const (
	OutcomeVersion  OutcomeKind = "version"
	OutcomeSuggest  OutcomeKind = "suggest"
	OutcomeExisting OutcomeKind = "existing"
	OutcomeNoTarget OutcomeKind = "no_target"
	OutcomeUnproven OutcomeKind = "unproven"
)

func OutcomeKinds() []OutcomeKind {
	return []OutcomeKind{OutcomeVersion, OutcomeSuggest, OutcomeExisting, OutcomeNoTarget, OutcomeUnproven}
}

// WorkState is why an existing Target doesn't yet give a Work a file the
// player plays directly.
type WorkState string

const (
	WorkNoFile       WorkState = "no_file"
	WorkNotInProfile WorkState = "not_in_profile"
	WorkNotListed    WorkState = "not_listed"
	WorkNotProven    WorkState = "not_proven"
)

func WorkStates() []WorkState {
	return []WorkState{WorkNoFile, WorkNotInProfile, WorkNotListed, WorkNotProven}
}

// Choose is the Target rule that can choose a dimension, "" for none.
type Choose string

const (
	ChooseQualities  Choose = "qualities"
	ChooseVideoCodec Choose = "video_codec"
	ChooseHDR        Choose = "hdr"
	ChooseAudioCodec Choose = "audio_codec"
	ChooseChannels   Choose = "channels"
)

// dimensions is the dimension table, in its order. Fact is the snapshot
// fact a version must differ on, "" where none can (no version clears it).
// Lower is whether a version must be lower, not only other.
var dimensions = []struct {
	Dimension Dimension
	Choose    Choose
	Fact      FactKind
	Lower     bool
}{
	{Resolution, ChooseQualities, FactResolution, true},
	{VideoCodec, ChooseVideoCodec, FactVideoCodec, false},
	{HDR, ChooseHDR, FactHDR, false},
	{AudioCodec, ChooseAudioCodec, FactAudioCodec, false},
	{Channels, ChooseChannels, FactChannels, true},
	{Container, "", FactContainer, false},
	{Subtitles, "", FactSubtitleCodec, false},
	{AudioTrack, "", "", false},
	{StreamCount, "", "", false},
	{VideoProfile, "", FactVideoProfile, false},
	{BitDepth, "", FactBitDepth, false},
	{FrameRate, "", "", false},
	{VideoShape, "", "", false},
	{AudioFormat, "", "", false},
	{Bitrate, "", FactBitrate, true},
	{Unreadable, "", "", false},
	{PlayerError, "", "", false},
	{Unknown, "", "", false},
}

func Dimensions() []Dimension {
	out := make([]Dimension, len(dimensions))
	for i, d := range dimensions {
		out[i] = d.Dimension
	}
	return out
}
