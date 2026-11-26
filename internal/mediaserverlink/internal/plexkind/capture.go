package plexkind

import (
	"context"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
	"github.com/jatassi/slipstream/internal/release"
)

// decisions are the TranscodeSession's words in SlipStream's. The
// recorded run of 2026-11-26 read copy and transcode for video and audio,
// burn for burned subtitles and no subtitleDecision otherwise. A word
// these lack is the zero value, which the Controller refuses.
var decisions = struct {
	streams   map[string]link.StreamDecision
	subtitles map[string]link.SubtitleDecision
}{
	streams:   map[string]link.StreamDecision{"copy": link.StreamDirect, "transcode": link.StreamTranscode},
	subtitles: map[string]link.SubtitleDecision{"copy": link.SubtitlesDirect, "transcode": link.SubtitlesDirect, "burn": link.SubtitlesBurned},
}

// names are Plex's codec and container names in the release vocabulary.
// The recorded run of 2026-11-26 read mkv and mp4, h264 and hevc, aac and
// dca for DTS, and pgs.
var names = link.Names{
	Containers: map[string]release.Container{"mkv": release.MKV, "mp4": release.MP4, "mpegts": release.TS},
	Video:      map[string]release.VideoCodec{"h264": "x264", "hevc": "x265", "av1": "AV1", "vp9": "VP9", "mpeg2video": "MPEG2", "mpeg4": "XviD"},
	Audio:      map[string]release.AudioCodec{"truehd": "TrueHD", "dca-ma": "DTS-HD MA", "dca": "DTS", "eac3": "DD+", "ac3": "DD", "aac": "AAC", "flac": "FLAC", "pcm": "PCM", "opus": "Opus", "mp3": "MP3"},
	Subtitles:  map[string]release.SubtitleCodec{"srt": release.SRT, "ass": release.ASS, "pgs": release.PGS, "vobsub": release.VobSub, "webvtt": release.WebVTT, "mov_text": release.MovText},
}

// observed is a session as Plex lists it, before its version is read:
// item is its ratingKey, and audio and subtitle the ids of the streams it
// plays.
type observed struct {
	link.Session
	item            string
	audio, subtitle plex.ID
}

// read is x in SlipStream's words. The version is the selected Media, or
// the only one; a session lists every Media of its item. Its
// TranscodeSession counts only when the Part's decision is transcode: a
// direct play can carry one left from the player's earlier play. While
// Plex transcodes, the selected Media is what it sends, so the received
// height and bitrate are its own: TranscodeSession has no height once the
// video is re-encoded, and Media.bitrate is in kbps (recorded run
// 2026-11-26). Plex gives no reasons.
func read(x plex.Session) observed {
	o := observed{item: string(x.RatingKey), Session: link.Session{
		Key: string(x.SessionKey), Streams: link.Streams{Video: link.StreamDirect, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone},
	}}
	if x.User != nil {
		o.User = x.User.Title
	}
	if p := x.Player; p != nil {
		o.PlayerKey, o.Player, o.Product, o.Platform = p.MachineIdentifier, p.Title, p.Product, p.Platform
	}
	m, ok := selected(x.Media)
	if !ok {
		return o
	}
	o.VersionID = string(m.ID)
	var part plex.Part
	if len(m.Parts) > 0 {
		part = m.Parts[0]
	}
	for _, st := range part.Streams {
		switch {
		case st.StreamType == plex.StreamAudio && o.audio == "":
			o.audio = st.ID
		case st.StreamType == plex.StreamSubtitle && o.subtitle == "":
			o.subtitle, o.Streams.Subtitles = st.ID, link.SubtitlesDirect
		}
	}
	t := x.Transcode
	if t == nil || part.Decision != "transcode" {
		return o
	}
	o.Streams.ContainerChanged = true
	o.Streams.Video, o.Streams.Audio = decisions.streams[t.VideoDecision], decisions.streams[t.AudioDecision]
	if t.SubtitleDecision != "" {
		o.Streams.Subtitles = decisions.subtitles[t.SubtitleDecision]
	}
	o.Hardware = t.TranscodeHwFullPipeline || t.TranscodeHwEncoding != ""
	o.Received = names.Received(t.Container, t.VideoCodec, t.AudioCodec, m.Height, t.AudioChannels, m.Bitrate*1000)
	return o
}

// selected is the Media with selected set, else the only one, false for
// several with none selected.
func selected(ms []plex.Media) (plex.Media, bool) {
	if i := slices.IndexFunc(ms, func(m plex.Media) bool { return m.Selected }); i >= 0 {
		return ms[i], true
	}
	if len(ms) == 1 {
		return ms[0], true
	}
	return plex.Media{}, false
}

// session is o playing v, its tracks found among v's streams.
func (o observed) session(v link.Version) link.Session {
	s := o.Session
	s.Version, s.Audio, s.Subtitle = v, track(o.audio, v.Audio, v.External), track(o.subtitle, v.Subtitles, v.External)
	return s
}

// track is the stream whose id is id: a position among embedded, or one
// of external. A stream the version doesn't list is no track.
func track(id plex.ID, embedded, external []link.Stream) *link.Track {
	if id == "" {
		return nil
	}
	is := func(s link.Stream) bool { return s.ID == string(id) }
	if i := slices.IndexFunc(embedded, is); i >= 0 {
		return &link.Track{Position: i}
	}
	if i := slices.IndexFunc(external, is); i >= 0 {
		return &link.Track{External: &link.ExternalTrack{Codec: external[i].Codec, Language: external[i].Language}}
	}
	return nil
}

// versionOf is the Media of md whose id is media: its first Part's file,
// and that Part's audio, subtitle and Sidecar streams in order, each
// named by its id. A Sidecar's language is its short tag, as its file
// name gives it.
func versionOf(md plex.Metadata, media string) (link.Version, error) {
	i := slices.IndexFunc(md.Media, func(m plex.Media) bool { return string(m.ID) == media })
	if i < 0 || len(md.Media[i].Parts) == 0 {
		return link.Version{}, fmt.Errorf("item %s lists no media %s", md.RatingKey, media)
	}
	part := md.Media[i].Parts[0]
	p, err := link.ParsePath(part.File)
	if err != nil {
		return link.Version{}, err
	}
	v := link.Version{Path: p, Items: []string{string(md.RatingKey)}}
	for _, st := range part.Streams {
		s := link.Stream{ID: string(st.ID), Language: st.LanguageCode}
		switch {
		case st.Key != "":
			s.Codec, s.Language = names.SubtitleCodec(st.Codec), st.LanguageTag
			v.External = append(v.External, s)
		case st.StreamType == plex.StreamAudio:
			s.Codec = names.AudioCodec(st.Codec)
			v.Audio = append(v.Audio, s)
		case st.StreamType == plex.StreamSubtitle:
			s.Codec = names.SubtitleCodec(st.Codec)
			v.Subtitles = append(v.Subtitles, s)
		}
	}
	return v, nil
}

// sessions are the sessions that play a version, capped, and the keys of
// those that name none. A version no live play resolved is read from
// /library/metadata, as the session's Media describes what Plex sends,
// with no file while it transcodes. A session whose read fails is left
// out of both lists, so nothing is recorded under the wrong file.
func (c *conn) sessions(ctx context.Context, all []plex.Session) (out []link.Session, pending []string) {
	for _, x := range all {
		o := read(x)
		switch {
		case o.VersionID == "":
			pending = append(pending, o.Key)
			continue
		case len(out) == c.t.SessionCap:
			continue
		}
		v, known := c.versions.Version(o.VersionID)
		if !known {
			var err error
			if v, err = c.version(ctx, o.item, o.VersionID); err != nil {
				c.log.Warn("read the version a session plays", "server", c.server.ID.String(), "item", o.item, "err", err.Error())
				continue
			}
		}
		out = append(out, o.session(v))
	}
	return out, pending
}

func (c *conn) version(ctx context.Context, item, media string) (link.Version, error) {
	md, found, err := c.c.Item(ctx, plex.ID(item))
	switch {
	case err != nil:
		return link.Version{}, err
	case !found:
		return link.Version{}, fmt.Errorf("item %s is gone", item)
	}
	return versionOf(md, media)
}
