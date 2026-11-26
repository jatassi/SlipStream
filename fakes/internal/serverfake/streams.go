package serverfake

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// Streamed is a file the Scenario lists, as the server names it, and its
// streams in the product's names. Unlisted is a file the server's scans
// skip.
type Streamed struct {
	Path     link.ServerPath
	Streams  []mediaserver.Stream
	Unlisted bool
}

// subtitleCodecs are each kind's name for SubRip.
var subtitleCodecs = map[string]string{"plex": "srt", "jellyfin": "subrip", "emby": "subrip"}

// streamed is every MediaFile in the library folders p holds.
func streamed(s scenario.Scenario, p Played) ([]Streamed, error) {
	var out []Streamed
	folders := unrooted(s, p.MediaServerAt)
	for _, r := range s.HeldRoots(p.MediaServerAt) {
		folders = append(folders, r.Folder)
	}
	for _, lf := range s.Library.Folders {
		if !slices.Contains(folders, lf.Folder) {
			continue
		}
		for _, m := range lf.Files {
			path, err := p.Path(lf.Folder + "/" + m.Path)
			if err != nil {
				return nil, err
			}
			file := lf.Folder + "/" + m.Path
			out = append(out, Streamed{Path: path, Streams: streamsOf(m, subtitleCodecs[p.Kind]), Unlisted: slices.Contains(p.Unlisted, file)})
		}
	}
	return out, nil
}

// streamsOf is m's streams: H.264 for SDR and 10-bit HEVC for the HDR
// formats, as the world encodes them.
func streamsOf(m scenario.MediaFile, subrip string) []mediaserver.Stream {
	video := mediaserver.Stream{Type: "video", Codec: "h264", Height: m.Video.Height}
	if m.Video.HDR != "sdr" {
		video.Codec, video.Profile = "hevc", "Main 10"
	}
	out := []mediaserver.Stream{video}
	for _, a := range m.Audio {
		out = append(out, mediaserver.Stream{Type: "audio", Codec: a.Codec, Channels: a.Channels, Language: a.Language, Default: a.Default})
	}
	for _, sub := range m.Subtitles {
		out = append(out, mediaserver.Stream{Type: "subtitle", Codec: subrip, Language: sub.Language, Default: sub.Default})
	}
	return out
}

// SetStreams is a <kind>-streams cue's arguments.
type SetStreams struct {
	Path    string               `json:"path"`
	Streams []mediaserver.Stream `json:"streams"`
}

// Skip is a <kind>-skip cue's arguments.
type Skip struct {
	Paths []string `json:"paths"`
}

// Streams gives core files' streams, leaves the unlisted ones out of its
// scans and takes the <kind>-streams and <kind>-skip cues.
func Streams(env fake.Env, kind string, core *mediaserver.Server, files []Streamed) error {
	var skipped []link.ServerPath
	for _, f := range files {
		core.SetStreams(f.Path, f.Streams)
		if f.Unlisted {
			skipped = append(skipped, f.Path)
		}
	}
	core.Skip(skipped)
	if err := fake.OnCue(env, kind+"-streams", func(_ context.Context, a SetStreams) (any, error) {
		p, err := link.ParsePath(a.Path)
		if err != nil {
			return nil, err
		}
		core.SetStreams(p, a.Streams)
		return map[string]int{"streams": len(a.Streams)}, nil
	}); err != nil {
		return err
	}
	return fake.OnCue(env, kind+"-skip", func(_ context.Context, a Skip) (any, error) {
		paths := make([]link.ServerPath, len(a.Paths))
		for i, s := range a.Paths {
			p, err := link.ParsePath(s)
			if err != nil {
				return nil, err
			}
			paths[i] = p
		}
		core.Skip(paths)
		return map[string]int{"skipped": len(paths)}, nil
	})
}
