package playback

// reasons is the reason table: each transcode reason Jellyfin and Emby
// name, and the dimension it blocks. Stream is the stream an unreadable
// reason names.
var reasons = []struct {
	Name      string
	Dimension Dimension
	Stream    string
}{
	{"ContainerNotSupported", Container, ""},
	{"VideoCodecNotSupported", VideoCodec, ""},
	{"AudioCodecNotSupported", AudioCodec, ""},
	{"SubtitleCodecNotSupported", Subtitles, ""},
	{"SubtitleContentOptionsEnabled", Subtitles, ""},
	{"AudioIsExternal", AudioTrack, ""},
	{"ExternalAudioNotSupported", AudioTrack, ""},
	{"SecondaryAudioNotSupported", AudioTrack, ""},
	{"AudioDelayNotSupported", AudioTrack, ""},
	{"StreamCountExceedsLimit", StreamCount, ""},
	{"VideoProfileNotSupported", VideoProfile, ""},
	{"VideoLevelNotSupported", VideoProfile, ""},
	{"VideoCodecTagNotSupported", VideoProfile, ""},
	{"RefFramesNotSupported", VideoProfile, ""},
	{"VideoRangeTypeNotSupported", HDR, ""},
	{"VideoRangeNotSupported", HDR, ""},
	{"VideoResolutionNotSupported", Resolution, ""},
	{"VideoBitDepthNotSupported", BitDepth, ""},
	{"VideoFramerateNotSupported", FrameRate, ""},
	{"VideoRotationNotSupported", VideoShape, ""},
	{"AnamorphicVideoNotSupported", VideoShape, ""},
	{"InterlacedVideoNotSupported", VideoShape, ""},
	{"AudioChannelsNotSupported", Channels, ""},
	{"AudioProfileNotSupported", AudioFormat, ""},
	{"AudioSampleRateNotSupported", AudioFormat, ""},
	{"AudioBitDepthNotSupported", AudioFormat, ""},
	{"ContainerBitrateExceedsLimit", Bitrate, ""},
	{"VideoBitrateNotSupported", Bitrate, ""},
	{"AudioBitrateNotSupported", Bitrate, ""},
	{"UnknownVideoStreamInfo", Unreadable, "video"},
	{"UnknownAudioStreamInfo", Unreadable, "audio"},
	{"DirectPlayError", PlayerError, ""},
}

// reasonOf is the dimension the reason name blocks and the stream it
// names, Unknown for a name outside the table.
func reasonOf(name string) (d Dimension, stream string) {
	for _, r := range reasons {
		if r.Name == name {
			return r.Dimension, r.Stream
		}
	}
	return Unknown, ""
}
