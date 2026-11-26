package mbkind

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/jellyfin"
	"github.com/jatassi/slipstream/internal/release"
)

// mbNames are Jellyfin's and Emby's codec and container names, which are
// ffmpeg's, in the release vocabulary. The recorded runs of 2026-11-26 read
// mkv, mp4 and ts, h264 and hevc, aac and dts, and PGSSUB.
var mbNames = link.Names{
	Containers: map[string]release.Container{"mkv": release.MKV, "mp4": release.MP4, "ts": release.TS},
	Video:      map[string]release.VideoCodec{"h264": "x264", "hevc": "x265", "av1": "AV1", "vp9": "VP9", "mpeg2video": "MPEG2", "mpeg4": "XviD"},
	Audio:      map[string]release.AudioCodec{"truehd": "TrueHD", "dts": "DTS", "eac3": "DD+", "ac3": "DD", "aac": "AAC", "flac": "FLAC", "opus": "Opus", "mp3": "MP3"},
	Subtitles:  map[string]release.SubtitleCodec{"subrip": release.SRT, "ass": release.ASS, "pgssub": release.PGS, "dvdsub": release.VobSub, "webvtt": release.WebVTT, "mov_text": release.MovText},
}

// burned is the reason both products give for subtitles burned into the
// video, which a burn re-encodes.
const burned = "SubtitleCodecNotSupported"

// observed is a session as its server lists it, before its version is
// read: item is NowPlayingItem, whose sources name the version, and audio
// and subtitle are the chosen streams' Index in the version's
// MediaStreams.
type observed struct {
	link.Session
	item            string
	audio, subtitle *int
}

// read is s as both products list it, false when it plays nothing. The
// streams come from TranscodingInfo, never from PlayMethod alone: a
// browser's remux reads Transcode with both streams direct, and Emby's
// direct play of a file reads DirectStream with no TranscodingInfo
// (recorded runs 2026-11-26). A Transcode with no TranscodingInfo is an
// Emby player that hasn't reported its playback (#71), so only the
// container is known to change. Neither product sends DeviceType, so the
// platform is unknown.
func read(s jellyfin.Session) (observed, bool) {
	if s.NowPlayingItem == nil {
		return observed{}, false
	}
	o := observed{item: s.NowPlayingItem.ID, Session: link.Session{
		Key: s.ID, PlayerKey: s.DeviceID, Player: s.DeviceName, Product: s.Client, User: s.UserName,
		Streams: link.Streams{Video: link.StreamDirect, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone},
	}}
	ps := s.PlayState
	if ps == nil {
		return o, true
	}
	o.VersionID, o.audio = ps.MediaSourceID, ps.AudioStreamIndex
	if i := ps.SubtitleStreamIndex; i != nil && *i >= 0 {
		o.subtitle, o.Streams.Subtitles = i, link.SubtitlesDirect
	}
	o.Streams.ContainerChanged = ps.PlayMethod == jellyfin.Transcode
	ti := s.TranscodingInfo
	if ti == nil || ps.PlayMethod == jellyfin.DirectPlay {
		return o, true
	}
	o.Streams.ContainerChanged = true
	o.Streams.Video, o.Streams.Audio = direct(ti.IsVideoDirect), direct(ti.IsAudioDirect)
	o.Reasons = slices.Clone(ti.TranscodeReasons)
	if o.subtitle != nil && !ti.IsVideoDirect && slices.Contains(o.Reasons, burned) {
		o.Streams.Subtitles = link.SubtitlesBurned
	}
	o.Hardware = ti.VideoEncoderIsHardware || (ti.HardwareAccelerationType != "" && ti.HardwareAccelerationType != "none")
	o.Received = mbNames.Received(ti.Container, ti.VideoCodec, ti.AudioCodec, ti.Height, ti.AudioChannels, ti.Bitrate)
	return o, true
}

func direct(is bool) link.StreamDecision {
	if is {
		return link.StreamDirect
	}
	return link.StreamTranscode
}

// session is o playing v, its tracks found among v's streams.
func (o observed) session(v link.Version) link.Session {
	s := o.Session
	s.Version, s.Audio, s.Subtitle = v, track(o.audio, v.Audio, v.External), track(o.subtitle, v.Subtitles, v.External)
	return s
}

// track is the stream whose Index is index: a position among embedded,
// or one of external. A stream the version doesn't list is no track.
func track(index *int, embedded, external []link.Stream) *link.Track {
	if index == nil {
		return nil
	}
	id := func(s link.Stream) bool { return s.ID == strconv.Itoa(*index) }
	if i := slices.IndexFunc(embedded, id); i >= 0 {
		return &link.Track{Position: i}
	}
	if i := slices.IndexFunc(external, id); i >= 0 {
		return &link.Track{External: &link.ExternalTrack{Codec: external[i].Codec, Language: external[i].Language}}
	}
	return nil
}

// versionOf is the version m is: its file, and its audio, subtitle and
// external streams in Index order, each named by its Index.
func versionOf(m jellyfin.MediaSource) (link.Version, error) {
	p, err := link.ParsePath(m.Path)
	if err != nil {
		return link.Version{}, err
	}
	v := link.Version{Path: p}
	for _, st := range slices.SortedStableFunc(slices.Values(m.MediaStreams), func(a, b jellyfin.Stream) int { return cmp.Compare(a.Index, b.Index) }) {
		s := link.Stream{ID: strconv.Itoa(st.Index), Language: st.Language}
		switch st.Type {
		case jellyfin.StreamAudio:
			s.Codec = mbNames.AudioCodec(st.Codec)
		case jellyfin.StreamSubtitle:
			s.Codec = mbNames.SubtitleCodec(st.Codec)
		default:
			continue
		}
		switch {
		case st.IsExternal:
			v.External = append(v.External, s)
		case st.Type == jellyfin.StreamAudio:
			v.Audio = append(v.Audio, s)
		default:
			v.Subtitles = append(v.Subtitles, s)
		}
	}
	return v, nil
}

// sessions are the sessions that play an item with their versions, capped,
// the keys of those that name no version, and whether any session
// re-encodes its video, counted before the cap. A version no live play
// resolved is read from the item NowPlayingItem names, which on both
// products lists every version of its group (recorded runs 2026-11-26). A
// session whose read fails is left out of both lists, so nothing is
// recorded under the wrong file.
func (c *conn) sessions(ctx context.Context, all []jellyfin.Session) (out []link.Session, pending []string, transcoding bool) {
	for _, s := range all {
		o, ok := read(s)
		if !ok {
			continue
		}
		transcoding = transcoding || o.Streams.Video == link.StreamTranscode
		switch {
		case o.VersionID == "":
			pending = append(pending, o.Key)
			continue
		case len(out) == c.d.Timings.SessionCap:
			continue
		}
		v, known := c.d.Versions.Version(o.VersionID)
		if !known {
			var err error
			if v, err = c.version(ctx, s.UserID, o.item, o.VersionID); err != nil {
				c.d.Log.Warn("read the version a session plays", "server", c.d.Server.ID.String(), "item", o.item, "err", err.Error())
				continue
			}
		}
		out = append(out, o.session(v))
	}
	return out, pending, transcoding
}

// version is the media source whose Id is source among item's, with the
// items that hold item's sources. It never takes NowPlayingItem for the
// played file's item: a Jellyfin alternate plays as its own item, which
// the library lists only within its primary, and Emby's is the item the
// player opened (recorded runs 2026-11-26).
func (c *conn) version(ctx context.Context, user, item, source string) (link.Version, error) {
	it, found, err := c.c.Item(ctx, user, item)
	switch {
	case err != nil:
		return link.Version{}, err
	case !found:
		return link.Version{}, fmt.Errorf("the server has no item %s", item)
	}
	i := slices.IndexFunc(it.MediaSources, func(m jellyfin.MediaSource) bool { return m.ID == source })
	if i < 0 {
		return link.Version{}, fmt.Errorf("item %s lists no media source %s", item, source)
	}
	v, err := versionOf(it.MediaSources[i])
	if err != nil {
		return link.Version{}, err
	}
	for _, m := range it.MediaSources {
		if h := c.f.holder(m); !slices.Contains(v.Items, h) {
			v.Items = append(v.Items, h)
		}
	}
	return v, nil
}

// holder is the item that holds m's file: Emby names it, and a Jellyfin
// source's Id is its item's (recorded runs 2026-11-26).
func (f family) holder(m jellyfin.MediaSource) string {
	if f.flavor == jellyfin.Emby {
		return m.ItemID
	}
	return m.ID
}
