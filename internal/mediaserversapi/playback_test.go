package mediaserversapi_test

import (
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// The files the report's worlds play.
var (
	// plain is 1080p H.264 SDR in MKV at 8 Mb/s with 5.1 E-AC-3.
	plain = media.Record{
		Version: 1, Container: "matroska,webm", Bitrate: 8_000_000,
		Video: media.Video{Codec: "h264", Profile: "High", Width: 1920, Height: 1080, BitDepth: 8, HDR: []release.HDR{"SDR"}},
		Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "en", Default: true}},
	}
	// hdr10 is 1080p HEVC HDR10 in MKV at 10 Mb/s with 5.1 E-AC-3.
	hdr10 = media.Record{
		Version: 1, Container: "matroska,webm", Bitrate: 10_000_000,
		Video: media.Video{Codec: "hevc", Profile: "Main 10", Width: 1920, Height: 1080, BitDepth: 10, HDR: []release.HDR{"HDR10"}},
		Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "en", Default: true}},
	}
	// dolby is 2160p HEVC Dolby Vision 8.1, so DV and HDR10, in MKV at 40
	// Mb/s with 7.1 TrueHD and English PGS subtitles.
	dolby = media.Record{
		Version: 1, Container: "matroska,webm", Bitrate: 40_000_000,
		Video: media.Video{
			Codec: "hevc", Profile: "Main 10", Width: 3840, Height: 2160, BitDepth: 10,
			HDR: []release.HDR{"DV", "HDR10"}, DolbyVision: &media.DolbyVision{Profile: 8, Compatibility: 1},
		},
		Audio:     []media.Audio{{Codec: "truehd", Channels: 8, Language: "en", Default: true}},
		Subtitles: []media.Subtitle{{Codec: "hdmv_pgs_subtitle", Image: true, Language: "en"}},
	}
	// dolbyHD is dolby at 1080p with 5.1 E-AC-3 and no subtitles.
	dolbyHD = media.Record{
		Version: 1, Container: "matroska,webm", Bitrate: 12_000_000,
		Video: media.Video{
			Codec: "hevc", Profile: "Main 10", Width: 1920, Height: 1080, BitDepth: 10,
			HDR: []release.HDR{"DV", "HDR10"}, DolbyVision: &media.DolbyVision{Profile: 8, Compatibility: 1},
		},
		Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "en", Default: true}},
	}
	library = map[string]media.Record{"plain": plain, "hdr10": hdr10, "dolby": dolby, "dolbyHD": dolbyHD}
)

// heard is one play a server reported, of a library file by name.
type heard struct {
	player, name, file string
	video, audio       link.StreamDecision
	subtitles          link.SubtitleDecision
	// shown is whether the file's first subtitle track showed.
	shown    bool
	reasons  []string
	received link.Received
	// oneLook is a play stopped on the look that found it.
	oneLook bool
}

// direct is a direct play of file on the player tv.
func direct(file string) heard {
	return heard{player: "tv", file: file, video: link.StreamDirect, audio: link.StreamDirect, subtitles: link.SubtitlesNone}
}

// transcoded is file's video transcoded on the player tv for reasons.
func transcoded(file string, reasons ...string) heard {
	return heard{player: "tv", file: file, video: link.StreamTranscode, audio: link.StreamDirect, subtitles: link.SubtitlesNone, reasons: reasons}
}

// reportWorld is Den with its holder stopped, the library's files indexed
// at their names, and plays reported straight from seedbox ten minutes
// apart.
type reportWorld struct {
	rig
	id     string
	server link.ServerID
	next   time.Time
	// played is the play ids in report order.
	played []link.PlayID
}

func newReportWorld(t *testing.T) *reportWorld {
	t.Helper()
	r := newRig(t)
	r.w.Jellyfin(t, host, den(floorVer))
	w := &reportWorld{rig: r, id: r.add(t)}
	w.server = serverOf(t, w.id)
	synctest.Wait()
	r.w.Stop(r.w.Seedbox)
	wait(day)
	w.next = time.Now().Add(-23 * time.Hour)
	for name, rec := range library {
		file := r.w.File(t, mstest.File{Work: works.WorkID{UUID: storage.NewUUID()}, Item: works.ItemID{UUID: storage.NewUUID()}, Label: name, Media: rec})
		path, err := link.ParsePath(shelves + "/" + name + ".mkv")
		if err != nil {
			t.Fatal(err)
		}
		r.w.Index(t, w.server, path, file, name)
	}
	return w
}

func (w *reportWorld) hear(t *testing.T, hs ...heard) {
	t.Helper()
	for _, h := range hs {
		started := w.next
		w.next = w.next.Add(10 * time.Minute)
		seen := started.Add(5 * time.Minute)
		if h.oneLook {
			seen = started
		}
		var sub *link.Track
		if h.shown {
			sub = &link.Track{Position: 0}
		}
		name := h.name
		if name == "" {
			name = h.player
		}
		w.played = append(w.played, w.w.Play(t, w.w.Seedbox, mstest.Play{
			Server: w.server, Session: strconv.Itoa(len(w.played)), Path: shelves + "/" + h.file + ".mkv", Item: h.file,
			PlayerKey: h.player, Player: name, User: "ada",
			Streams: link.Streams{Video: h.video, Audio: h.audio, Subtitles: h.subtitles},
			Reasons: h.reasons, Received: h.received, Subtitle: sub, Started: started, Seen: seen,
		}))
	}
}

func (w *reportWorld) report(t *testing.T) playback.Report {
	t.Helper()
	status, body := w.o.Call(t, http.MethodGet, servers+"/"+w.id+"/playback", "")
	if status != http.StatusOK {
		t.Fatalf("GET playback = %d %s", status, body)
	}
	return decode[playback.Report](t, body)
}

func (w *reportWorld) player(t *testing.T, key string) playback.Player {
	t.Helper()
	rep := w.report(t)
	i := slices.IndexFunc(rep.Players, func(p playback.Player) bool { return p.Key == key })
	if i < 0 {
		t.Fatalf("the report has no player %s: %+v", key, rep.Players)
	}
	return rep.Players[i]
}

// lines is each of the player's groups as "<dims> [<cause>]: <plays>
// <kind> <what it names>, ...", a play named by its place in report
// order from 1.
func (w *reportWorld) lines(t *testing.T, key string) []string {
	t.Helper()
	var out []string
	for _, g := range w.player(t, key).Groups {
		var os []string
		for _, o := range g.Outcomes {
			s := fmt.Sprintf("%d %s", o.Plays, o.Kind)
			switch {
			case o.Missing != nil:
				s += " " + dims(*o.Missing)
			case o.Suggestion != nil:
				var b strings.Builder
				fmt.Fprintf(&b, " %q from", o.Suggestion.Name)
				for _, e := range o.Evidence {
					fmt.Fprintf(&b, " %s=%s#%d", e.Fact, e.Value, w.place(e.PlayID))
				}
				s += b.String()
			}
			os = append(os, s)
		}
		out = append(out, dims(g.Dimensions...)+" ["+dims(g.Cause...)+"]: "+strings.Join(os, ", "))
	}
	return out
}

func (w *reportWorld) place(id string) int {
	return slices.IndexFunc(w.played, func(p link.PlayID) bool { return p.String() == id }) + 1
}

func dims(bs ...playback.Blocked) string {
	out := make([]string, len(bs))
	for i, b := range bs {
		out[i] = string(b.Dimension) + "=" + b.Value
	}
	return strings.Join(out, " ")
}

// TestEachTranscodeGetsTheOutcomeItsEvidenceDecides reads tv's groups
// from GET /playback. A dimension no Target chooses is no_target and the
// lead names it; with every dimension choosable, a dimension with no
// proven value is unproven; with all proven, the play suggests a Target.
func TestEachTranscodeGetsTheOutcomeItsEvidenceDecides(t *testing.T) {
	audioTranscoded := direct("plain")
	audioTranscoded.audio, audioTranscoded.reasons = link.StreamTranscode, []string{"AudioCodecNotSupported"}
	bothTranscoded := transcoded("dolby", "VideoRangeTypeNotSupported")
	bothTranscoded.audio = link.StreamTranscode
	oneLook := direct("plain")
	oneLook.oneLook = true
	for _, tc := range []struct {
		name  string
		plays []heard
		want  []string
	}{
		{
			name:  "a container no Target can choose is no_target",
			plays: []heard{direct("plain"), transcoded("dolby", "ContainerNotSupported")},
			want:  []string{"container=MKV [container=MKV]: 1 no_target"},
		},
		{
			name:  "the lead names the dimension no Target fixes",
			plays: []heard{direct("plain"), transcoded("dolby", "VideoRangeTypeNotSupported", "ContainerNotSupported")},
			want:  []string{"hdr=DV container=MKV [container=MKV]: 1 no_target"},
		},
		{
			name:  "with no evidence the first dimension is unproven",
			plays: []heard{transcoded("dolby", "VideoRangeTypeNotSupported")},
			want:  []string{"hdr=DV [hdr=DV]: 1 unproven resolution=2160p"},
		},
		{
			name:  "a play stopped on its first look is no evidence",
			plays: []heard{oneLook, transcoded("dolby", "VideoRangeTypeNotSupported")},
			want:  []string{"hdr=DV [hdr=DV]: 1 unproven resolution=2160p"},
		},
		{
			name:  "a play seen twice proves what it received unchanged",
			plays: []heard{direct("plain"), transcoded("dolby", "VideoRangeTypeNotSupported")},
			want: []string{
				`hdr=DV [hdr=DV]: 1 suggest "1080p SDR" from resolution=1080p#1 video_codec=x264#1 hdr=SDR#1 audio_codec=TrueHD#2 audio_codec=DD+#1 channels=7.1#2`,
			},
		},
		{
			name:  "a player that never received audio unchanged is unproven on audio",
			plays: []heard{audioTranscoded, bothTranscoded},
			want: []string{
				"hdr=DV [hdr=DV]: 1 unproven audio_codec=TrueHD",
				"audio_codec=DD+ [audio_codec=DD+]: 1 unproven audio_codec=DD+",
			},
		},
		{
			name: "plays blocked alike group, largest first, and the lead names at most two",
			plays: []heard{
				direct("plain"),
				transcoded("dolby", "VideoRangeTypeNotSupported", "AudioCodecNotSupported", "VideoResolutionNotSupported"),
				transcoded("dolby", "VideoRangeTypeNotSupported"),
				transcoded("dolby", "VideoRangeTypeNotSupported"),
			},
			want: []string{
				`hdr=DV [hdr=DV]: 2 suggest "1080p SDR" from resolution=1080p#1 video_codec=x264#1 hdr=SDR#1 audio_codec=TrueHD#4 audio_codec=DD+#1 channels=7.1#4`,
				`resolution=2160p hdr=DV audio_codec=TrueHD [resolution=2160p hdr=DV]: 1 suggest "1080p SDR" from resolution=1080p#1 video_codec=x264#1 hdr=SDR#1 audio_codec=DD+#1 channels=7.1#4`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := newReportWorld(t)
				w.hear(t, tc.plays...)
				if got := w.lines(t, "tv"); !slices.Equal(got, tc.want) {
					t.Errorf("tv's groups are\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tc.want, "\n"))
				}
			})
		})
	}
}

// each is one Blocked a Target can choose per value.
func each(d playback.Dimension, values ...string) []playback.Blocked {
	out := make([]playback.Blocked, len(values))
	for i, v := range values {
		out[i] = playback.Blocked{Dimension: d, Value: v, Choosable: true}
	}
	return out
}

// TestASuggestionHoldsOnlyProvenValues reads the suggestion of tv's one
// group: every rule stays inside what tv received unchanged, and the
// values it leaves out are listed as blocked or unproven.
func TestASuggestionHoldsOnlyProvenValues(t *testing.T) {
	flagship := transcoded("dolbyHD")
	flagship.received = link.Received{Height: 1080, VideoCodec: "x265"}
	noTrueHD := transcoded("dolby", "VideoRangeTypeNotSupported", "AudioCodecNotSupported")
	noTrueHD.audio = link.StreamTranscode
	for _, tc := range []struct {
		name  string
		plays []heard
		dims  string
		want  playback.Suggestion
	}{
		{
			name:  "proven HDR10 and x265 stay, the blocked DV and TrueHD and the unproven values leave",
			plays: []heard{direct("plain"), direct("hdr10"), noTrueHD},
			dims:  "hdr=DV audio_codec=TrueHD",
			want: playback.Suggestion{
				Name: "1080p HDR10", Highest: "1080p", VideoCodecs: []string{"x264", "x265"},
				NotHDR:         []string{"DV", "HDR10+", "HLG"},
				NotAudioCodecs: []string{"TrueHD", "DTS-HD MA", "DTS", "DD", "AAC", "FLAC", "PCM", "Opus", "MP3"},
				NotChannels:    []string{"7.1"},
				Blocked:        slices.Concat(each(playback.HDR, "DV"), each(playback.AudioCodec, "TrueHD")),
				Unproven: slices.Concat(
					each(playback.Resolution, "2160p"), each(playback.VideoCodec, "AV1", "VP9", "XviD", "MPEG2"),
					each(playback.HDR, "HDR10+", "HLG"), each(playback.AudioCodec, "DTS-HD MA", "DTS", "DD", "AAC", "FLAC", "PCM", "Opus", "MP3"),
					each(playback.Channels, "7.1"),
				),
			},
		},
		{
			name:  "a Dolby Vision 8.1 file on a player with HDR10 proven is blocked on DV alone",
			plays: []heard{direct("hdr10"), transcoded("dolbyHD", "VideoRangeTypeNotSupported")},
			dims:  "hdr=DV",
			want:  flagshipSuggestion,
		},
		{
			name:  "so is one the server gives no reason for",
			plays: []heard{direct("hdr10"), flagship},
			dims:  "hdr=DV",
			want:  flagshipSuggestion,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := newReportWorld(t)
				w.hear(t, tc.plays...)
				groups := w.player(t, "tv").Groups
				if len(groups) != 1 || len(groups[0].Outcomes) != 1 || groups[0].Outcomes[0].Suggestion == nil {
					t.Fatalf("tv's groups are %+v, want one suggesting a Target", groups)
				}
				if got := dims(groups[0].Dimensions...); got != tc.dims {
					t.Errorf("the group is blocked on %s, want %s", got, tc.dims)
				}
				if got := *groups[0].Outcomes[0].Suggestion; !reflect.DeepEqual(got, tc.want) {
					t.Errorf("the suggestion is\n%+v\nwant\n%+v", got, tc.want)
				}
			})
		})
	}
}

var flagshipSuggestion = playback.Suggestion{
	Name: "1080p HDR10", Highest: "1080p", VideoCodecs: []string{"x265"},
	NotHDR:         []string{"DV", "HDR10+", "HLG", "SDR"},
	NotAudioCodecs: []string{"TrueHD", "DTS-HD MA", "DTS", "DD", "AAC", "FLAC", "PCM", "Opus", "MP3"},
	NotChannels:    []string{"7.1"},
	Blocked:        each(playback.HDR, "DV"),
	Unproven: slices.Concat(
		each(playback.Resolution, "2160p"), each(playback.VideoCodec, "x264", "AV1", "VP9", "XviD", "MPEG2"),
		each(playback.HDR, "HDR10+", "HLG", "SDR"), each(playback.AudioCodec, "TrueHD", "DTS-HD MA", "DTS", "DD", "AAC", "FLAC", "PCM", "Opus", "MP3"),
		each(playback.Channels, "7.1"),
	),
}

// TestAPlayWithNoReasonIsBlockedOnWhatItsStreamsShow transcodes dolby on
// tv with no reason, as Plex reports every play, after a direct play of
// plain, and reads the dimensions of its group.
func TestAPlayWithNoReasonIsBlockedOnWhatItsStreamsShow(t *testing.T) {
	video := func(r link.Received, burned, shown bool) heard {
		h := transcoded("dolby")
		h.received, h.shown = r, shown
		if burned {
			h.subtitles = link.SubtitlesBurned
		}
		return h
	}
	audio := direct("dolby")
	audio.audio, audio.received = link.StreamTranscode, link.Received{AudioCodec: "AAC", Channels: 2}
	lower := link.Received{Height: 1080, VideoCodec: "x264"}
	same := link.Received{Height: 2160, VideoCodec: "x264"}
	for _, tc := range []struct {
		name  string
		plays []heard
		want  string
	}{
		{name: "a lower height, another codec and an unproven HDR format", plays: []heard{video(lower, false, false)}, want: "resolution=2160p video_codec=x265 hdr=DV"},
		{name: "a codec the player received unchanged drops out", plays: []heard{direct("hdr10"), video(lower, false, false)}, want: "resolution=2160p hdr=DV"},
		{name: "another audio codec and fewer channels", plays: []heard{audio}, want: "audio_codec=TrueHD channels=7.1"},
		{name: "a burn drops the video codec and HDR", plays: []heard{video(same, true, true)}, want: "subtitles=PGS"},
		{name: "a burn with no shown track", plays: []heard{video(same, true, false)}, want: "subtitles="},
		{name: "a burn keeps a lower height", plays: []heard{video(lower, true, true)}, want: "resolution=2160p subtitles=PGS"},
		{
			name:  "with nothing left a lower bitrate is bitrate",
			plays: []heard{direct("dolby"), video(link.Received{Height: 2160, VideoCodec: "x265", Bitrate: 20_000_000}, false, false)},
			want:  "bitrate=",
		},
		{
			name:  "with nothing left and no lower bitrate it is unknown",
			plays: []heard{direct("dolby"), video(link.Received{Height: 2160, VideoCodec: "x265"}, false, false)},
			want:  "unknown=",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := newReportWorld(t)
				w.hear(t, direct("plain"))
				w.hear(t, tc.plays...)
				groups := w.player(t, "tv").Groups
				if len(groups) != 1 {
					t.Fatalf("tv has groups %+v, want one", groups)
				}
				if got := dims(groups[0].Dimensions...); got != tc.want {
					t.Errorf("the play is blocked on %s, want %s", got, tc.want)
				}
			})
		})
	}
}

// TestEveryReasonBlocksItsDimension transcodes dolby with its English PGS
// track showing once per reason, each on its own player, and reads the
// one dimension each blocks with dolby's value on it. The rows are the
// spec's reason table.
func TestEveryReasonBlocksItsDimension(t *testing.T) {
	rows := []struct{ reason, dimension, value string }{
		{"ContainerNotSupported", "container", "MKV"},
		{"VideoCodecNotSupported", "video_codec", "x265"},
		{"AudioCodecNotSupported", "audio_codec", "TrueHD"},
		{"SubtitleCodecNotSupported", "subtitles", "PGS"},
		{"SubtitleContentOptionsEnabled", "subtitles", "PGS"},
		{"AudioIsExternal", "audio_track", "AudioIsExternal"},
		{"ExternalAudioNotSupported", "audio_track", "ExternalAudioNotSupported"},
		{"SecondaryAudioNotSupported", "audio_track", "SecondaryAudioNotSupported"},
		{"AudioDelayNotSupported", "audio_track", "AudioDelayNotSupported"},
		{"StreamCountExceedsLimit", "stream_count", ""},
		{"VideoProfileNotSupported", "video_profile", "Main 10"},
		{"VideoLevelNotSupported", "video_profile", "Main 10"},
		{"VideoCodecTagNotSupported", "video_profile", "Main 10"},
		{"RefFramesNotSupported", "video_profile", "Main 10"},
		{"VideoRangeTypeNotSupported", "hdr", "DV"},
		{"VideoRangeNotSupported", "hdr", "DV"},
		{"VideoResolutionNotSupported", "resolution", "2160p"},
		{"VideoBitDepthNotSupported", "bit_depth", "10"},
		{"VideoFramerateNotSupported", "frame_rate", ""},
		{"VideoRotationNotSupported", "video_shape", "VideoRotationNotSupported"},
		{"AnamorphicVideoNotSupported", "video_shape", "AnamorphicVideoNotSupported"},
		{"InterlacedVideoNotSupported", "video_shape", "InterlacedVideoNotSupported"},
		{"AudioChannelsNotSupported", "channels", "7.1"},
		{"AudioProfileNotSupported", "audio_format", "AudioProfileNotSupported"},
		{"AudioSampleRateNotSupported", "audio_format", "AudioSampleRateNotSupported"},
		{"AudioBitDepthNotSupported", "audio_format", "AudioBitDepthNotSupported"},
		{"ContainerBitrateExceedsLimit", "bitrate", ""},
		{"VideoBitrateNotSupported", "bitrate", ""},
		{"AudioBitrateNotSupported", "bitrate", ""},
		{"UnknownVideoStreamInfo", "unreadable", "video"},
		{"UnknownAudioStreamInfo", "unreadable", "audio"},
		{"DirectPlayError", "player_error", ""},
		{"SomethingNew", "unknown", ""},
	}
	choosable := []string{"resolution", "video_codec", "hdr", "audio_codec", "channels"}
	synctest.Test(t, func(t *testing.T) {
		w := newReportWorld(t)
		for _, row := range rows {
			h := transcoded("dolby", row.reason)
			h.player, h.shown = row.reason, true
			w.hear(t, h)
		}
		rep := w.report(t)
		for _, row := range rows {
			i := slices.IndexFunc(rep.Players, func(p playback.Player) bool { return p.Key == row.reason })
			if i < 0 || len(rep.Players[i].Groups) != 1 {
				t.Errorf("%s: the report has no one group for its player: %+v", row.reason, rep.Players)
				continue
			}
			want := []playback.Blocked{{Dimension: playback.Dimension(row.dimension), Value: row.value, Choosable: slices.Contains(choosable, row.dimension)}}
			if got := rep.Players[i].Groups[0].Dimensions; !reflect.DeepEqual(got, want) {
				t.Errorf("%s blocks %+v, want %+v", row.reason, got, want)
			}
			if got := rep.Players[i].Groups[0].Reasons; !slices.Equal(got, []string{row.reason}) {
				t.Errorf("%s's group gives the reasons %q, want it as the server gave it", row.reason, got)
			}
		}
	})
}

// TestAPlayerIsItsKey reads the players of each world: a player is its
// key, named as its latest play names it, worst first, and a transcode
// with no file counts as unmatched in no group.
func TestAPlayerIsItsKey(t *testing.T) {
	as := func(h heard, key, name string) heard {
		h.player, h.name = key, name
		return h
	}
	unfiled := as(transcoded("lost"), "a", "TV")
	for _, tc := range []struct {
		name  string
		plays []heard
		want  []string
	}{
		{
			name:  "a renamed player is one player",
			plays: []heard{as(direct("plain"), "tv", "Living room"), as(direct("plain"), "tv", "Den TV")},
			want:  []string{"tv Den TV: 2 plays, 2 direct, 0 transcoded, 0 unmatched, 0 groups"},
		},
		{
			name:  "two players with one name are two",
			plays: []heard{as(direct("plain"), "a", "TV"), as(transcoded("dolby", "ContainerNotSupported"), "a", "TV"), as(transcoded("dolby", "ContainerNotSupported"), "b", "TV")},
			want: []string{
				"b TV: 1 plays, 0 direct, 1 transcoded, 0 unmatched, 1 groups",
				"a TV: 2 plays, 1 direct, 1 transcoded, 0 unmatched, 1 groups",
			},
		},
		{
			name:  "a transcode with no file is unmatched",
			plays: []heard{unfiled, as(direct("plain"), "a", "TV")},
			want:  []string{"a TV: 2 plays, 1 direct, 1 transcoded, 1 unmatched, 0 groups"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := newReportWorld(t)
				w.hear(t, tc.plays...)
				rep := w.report(t)
				var got []string
				plays, transcodes, unmatched := 0, 0, 0
				for _, p := range rep.Players {
					got = append(got, fmt.Sprintf("%s %s: %d plays, %d direct, %d transcoded, %d unmatched, %d groups",
						p.Key, p.Name, p.Plays, p.DirectPlays, p.Transcoded, p.Unmatched, len(p.Groups)))
					plays, transcodes, unmatched = plays+p.Plays, transcodes+p.Transcoded, unmatched+p.Unmatched
				}
				if !slices.Equal(got, tc.want) {
					t.Errorf("the players are\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tc.want, "\n"))
				}
				if rep.Plays != plays || rep.Transcoded != transcodes || rep.Unmatched != unmatched {
					t.Errorf("the report counts %d plays, %d transcoded, %d unmatched, want its players' %d, %d, %d",
						rep.Plays, rep.Transcoded, rep.Unmatched, plays, transcodes, unmatched)
				}
			})
		})
	}
}

// TestTheWindowFollowsTheSettingAndPlaysSince plays once ten days after
// Den was added and once 35 days after: the report starts at Den's
// plays_since until the default 30 days reach past it, then 30 days back,
// then 5 days back once Keep play history for is lowered to that.
func TestTheWindowFollowsTheSettingAndPlaysSince(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newReportWorld(t)
		since := w.w.PlaysSince(t, w.server)
		wait(9 * day)
		w.next = time.Now()
		w.hear(t, direct("plain"))
		expectWindow(t, w, since, 30*day, 1)
		wait(25 * day)
		w.next = time.Now().Add(-time.Hour)
		w.hear(t, direct("plain"))
		expectWindow(t, w, time.Now().Add(-30*day), 30*day, 2)
		if status, body := w.o.Call(t, http.MethodPatch, servers+"/settings", `{"values":{"mediaServers.keepPlays":432000}}`); status != http.StatusOK {
			t.Fatalf("lower Keep play history for = %d %s", status, body)
		}
		expectWindow(t, w, time.Now().Add(-5*day), 5*day, 1)
	})
}

func expectWindow(t *testing.T, w *reportWorld, since time.Time, keep time.Duration, plays int) {
	t.Helper()
	rep := w.report(t)
	if !rep.Since.Equal(since) || rep.Window != int64(keep/time.Second) || rep.Plays != plays {
		t.Errorf("the report covers %d plays since %v over %d s, want %d since %v over %d s",
			rep.Plays, rep.Since, rep.Window, plays, since, int64(keep/time.Second))
	}
}

// TestTheReportSaysWhetherItsServerIsReachableAndPlex reads Den's report
// while seedbox is online and once it's stopped, and a Plex server's.
func TestTheReportSaysWhetherItsServerIsReachableAndPlex(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newWatched(t)
		header := func(id string) (bool, bool) {
			status, body := r.o.Call(t, http.MethodGet, servers+"/"+id+"/playback", "")
			if status != http.StatusOK {
				t.Fatalf("GET playback = %d %s", status, body)
			}
			rep := decode[playback.Report](t, body)
			return rep.Reachable, rep.Plex
		}
		if reachable, plex := header(r.id); !reachable || plex {
			t.Errorf("with seedbox online Den's report reads reachable %t and Plex %t, want reachable and not Plex", reachable, plex)
		}
		r.w.Stop(r.w.Seedbox)
		synctest.Wait()
		if reachable, _ := header(r.id); reachable {
			t.Error("with seedbox stopped Den's report reads reachable")
		}
	})
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		r.signIn(t)
		v := r.addPlex(t, r.w.Seedbox.String())
		rep := decode[playback.Report](t, r.expect(t, http.MethodGet, servers+"/"+v.ID+"/playback", "", http.StatusOK))
		if !rep.Plex {
			t.Error("a Plex server's report doesn't read Plex")
		}
	})
}
