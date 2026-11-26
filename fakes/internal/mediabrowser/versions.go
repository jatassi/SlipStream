package mediabrowser

import (
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// Received is what a transcode sends the player, in the products' names.
type Received = mediaserver.Received

// source is one version an item lists, and the item that holds its file.
type source struct {
	id   string
	item mediaserver.Item
	part mediaserver.Part
}

// sources are the versions it lists, as the recorded runs of 2026-11-26
// read them. Jellyfin lists its files, the primary's Id being the item's
// and an alternate's being the alternate's own item. Emby lists the file
// of every item that shares its PresentationUniqueKey, each Id mediasource_
// and that item's.
func (s *server) sources(it mediaserver.Item) []source {
	group := []mediaserver.Item{it}
	if s.p.Flavor == Emby {
		group = nil
		for _, other := range s.allItems() {
			if other.Library == it.Library && presentation(other) == presentation(it) {
				group = append(group, other)
			}
		}
	}
	var out []source
	for _, g := range group {
		for _, p := range g.Parts {
			if p.Missing {
				continue
			}
			src := source{id: id("source " + p.File.String()), item: g, part: p}
			switch {
			case s.p.Flavor == Emby:
				src.id = "mediasource_" + g.Key
			case len(out) == 0:
				src.id = g.Key
			}
			out = append(out, src)
		}
	}
	return out
}

func presentation(it mediaserver.Item) string { return it.GUID + "/" + it.Edition }

// alternate is the Jellyfin item an alternate version is: the item whose
// source has the id, with that source first.
func (s *server) alternate(sid string) (mediaserver.Item, []source, bool) {
	if s.p.Flavor != Jellyfin {
		return mediaserver.Item{}, nil, false
	}
	for _, it := range s.allItems() {
		srcs := s.sources(it)
		for i, src := range srcs {
			if i > 0 && src.id == sid {
				return it, append([]source{src}, append(srcs[:i:i], srcs[i+1:]...)...), true
			}
		}
	}
	return mediaserver.Item{}, nil, false
}

var streamTypes = map[string]string{"video": "Video", "audio": "Audio", "subtitle": "Subtitle"}

// renderStreams are p's MediaStreams: the streams SetStreams gave, then a
// subtitle stream per Sidecar, each at its Index.
func renderStreams(p mediaserver.Part) []map[string]any {
	var out []map[string]any
	for _, st := range embedded(p) {
		m := map[string]any{
			"Index": len(out), "Type": streamTypes[st.Type], "Codec": st.Codec, "IsDefault": st.Default, "IsExternal": false,
		}
		for k, v := range map[string]any{"Profile": st.Profile, "Language": st.Language} {
			if v != "" {
				m[k] = v
			}
		}
		for k, v := range map[string]int{"Channels": st.Channels, "Height": st.Height} {
			if v != 0 {
				m[k] = v
			}
		}
		out = append(out, m)
	}
	for _, sc := range p.Sidecars {
		name := filepath.Base(filepath.FromSlash(strings.ReplaceAll(sc.String(), `\`, "/")))
		ext := filepath.Ext(name)
		m := map[string]any{"Index": len(out), "Type": "Subtitle", "Codec": sidecarCodec(strings.TrimPrefix(ext, ".")), "IsExternal": true, "Path": sc.String()}
		if lang := strings.TrimPrefix(filepath.Ext(strings.TrimSuffix(name, ext)), "."); len(lang) == 2 || len(lang) == 3 {
			m["Language"] = lang
		}
		out = append(out, m)
	}
	return out
}

func embedded(p mediaserver.Part) []mediaserver.Stream {
	var out []mediaserver.Stream
	for _, st := range p.Streams {
		if !st.External {
			out = append(out, st)
		}
	}
	return out
}

// sidecarCodec is the product's name for a Sidecar's extension: ffmpeg's.
func sidecarCodec(ext string) string {
	if ext == "srt" {
		return "subrip"
	}
	return ext
}

// streamIndex is the Index of p's n-th embedded stream of typ, or for a
// subtitle n of -1 its first Sidecar's, false when p has none.
func streamIndex(p mediaserver.Part, typ string, n int) (int, bool) {
	all := embedded(p)
	if typ == "subtitle" && n == -1 {
		return len(all), len(p.Sidecars) > 0
	}
	for i, st := range all {
		if st.Type != typ {
			continue
		}
		if n == 0 {
			return i, true
		}
		n--
	}
	return 0, false
}

// productNames are the products' names for the release words a
// conformance play receives.
var productNames = map[string]string{"MKV": "mkv", "MP4": "mp4", "TS": "ts", "x264": "h264", "x265": "hevc", "AAC": "aac", "DTS": "dts", "DD": "ac3", "DD+": "eac3"}

// ReceivedOf is r, in release words, in the products' names.
func ReceivedOf(r link.Received) Received {
	name := func(s string) string { return either(productNames[s], s) }
	return Received{
		Container: name(r.Container), VideoCodec: name(r.VideoCodec), AudioCodec: name(r.AudioCodec),
		Height: r.Height, Bitrate: r.Bitrate, Channels: r.Channels,
	}
}
