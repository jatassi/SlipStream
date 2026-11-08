package scenario

import (
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/release"
)

// LibraryFolder is media on disk the world writes under the run's media
// dir before it makes any root. A root whose folder names it holds its
// files from its first scan; a folder no root names waits for an Admin to
// add it as a root.
type LibraryFolder struct {
	Folder string      `jsonschema:"required,pattern=^[a-z0-9][a-z0-9._-]*$" yaml:"folder"`
	Files  []MediaFile `jsonschema:"required,minItems=1"                   yaml:"files"`
}

// MediaFile is a Matroska video file at Path inside its folder: any case,
// spaces and brackets, '/' separated. The process world encodes it with
// the bundled tools; the simulation tier writes a few bytes in its place
// and probes them as the encode would read.
type MediaFile struct {
	Path      string          `jsonschema:"required" yaml:"path"`
	Video     MediaVideo      `jsonschema:"required" yaml:"video"`
	Audio     []MediaAudio    `yaml:"audio,omitempty"`
	Subtitles []MediaSubtitle `yaml:"subtitles,omitempty"`
}

// MediaVideo is ffmpeg's testsrc2 pattern at 24 frames a second: H.264 for
// SDR, 10-bit HEVC for the HDR formats.
type MediaVideo struct {
	Width   int    `jsonschema:"required,minimum=64,maximum=7680,multipleOf=2" yaml:"width"`
	Height  int    `jsonschema:"required,minimum=64,maximum=4320,multipleOf=2" yaml:"height"`
	Seconds int    `jsonschema:"required,minimum=1,maximum=10"                  yaml:"seconds"`
	HDR     string `jsonschema:"required,enum=sdr,enum=hdr10,enum=hdr10plus,enum=hlg,enum=dv5,enum=dv8" yaml:"hdr"`
}

// MediaAudio is a sine tone in Codec. Language is an ISO 639 code as
// Matroska tags it, such as eng; empty leaves the track untagged.
type MediaAudio struct {
	Codec    string `jsonschema:"required,enum=aac,enum=ac3,enum=eac3,enum=truehd,enum=flac" yaml:"codec"`
	Channels int    `jsonschema:"required,enum=1,enum=2,enum=6,enum=8"                      yaml:"channels"`
	Language string `yaml:"language,omitempty"`
	Default  bool   `yaml:"default,omitempty"`
	Title    string `yaml:"title,omitempty"`
}

// MediaSubtitle is a SubRip track of one line.
type MediaSubtitle struct {
	Language string `yaml:"language,omitempty"`
	Forced   bool   `yaml:"forced,omitempty"`
	Default  bool   `yaml:"default,omitempty"`
}

// HDRKinds are the values MediaVideo.HDR takes.
var HDRKinds = []string{"sdr", "hdr10", "hdr10plus", "hlg", "dv5", "dv8"}

// AudioChannels are the channel counts each audio codec encodes.
var AudioChannels = map[string][]int{
	"aac":    {1, 2, 6, 8},
	"ac3":    {1, 2, 6},
	"eac3":   {1, 2, 6},
	"truehd": {1, 2, 6},
	"flac":   {1, 2, 6, 8},
}

// unportable are the characters a Windows file name can't hold.
const unportable = `\:*?"<>|`

func (l Library) checkFolders(add func(format string, args ...any)) {
	var folders []string
	for _, f := range l.Folders {
		switch {
		case !folderName.MatchString(f.Folder):
			add("library folder %q does not match %s", f.Folder, folderName)
		case slices.Contains(folders, f.Folder):
			add("library has two folders named %s", f.Folder)
		}
		folders = append(folders, f.Folder)
		if len(f.Files) == 0 {
			add("library folder %s has no files", f.Folder)
		}
		var paths []string
		for _, m := range f.Files {
			at := "library folder " + f.Folder + " file " + m.Path
			switch {
			case !fs.ValidPath(m.Path) || m.Path == "." || strings.ContainsAny(m.Path, unportable):
				add("%s is not a relative path of '/' separated names without %s", at, unportable)
			case path.Ext(m.Path) != ".mkv":
				add("%s is not a .mkv file", at)
			case slices.Contains(paths, strings.ToLower(m.Path)):
				add("%s is listed twice", at)
			}
			paths = append(paths, strings.ToLower(m.Path))
			m.check(add, at)
		}
	}
}

func (m MediaFile) check(add func(format string, args ...any), at string) {
	v := m.Video
	if v.Width < 64 || v.Width > 7680 || v.Height < 64 || v.Height > 4320 || v.Width%2 != 0 || v.Height%2 != 0 {
		add("%s video is %dx%d; width and height are even, from 64x64 up to 7680x4320", at, v.Width, v.Height)
	}
	if v.Seconds < 1 || v.Seconds > 10 {
		add("%s video lasts %d seconds; it lasts 1 to 10", at, v.Seconds)
	}
	if !slices.Contains(HDRKinds, v.HDR) {
		add("%s video hdr is %q, not one of %s", at, v.HDR, strings.Join(HDRKinds, ", "))
	}
	for i, a := range m.Audio {
		channels, ok := AudioChannels[a.Codec]
		switch {
		case !ok:
			add("%s audio %d codec is %q, not aac, ac3, eac3, truehd or flac", at, i+1, a.Codec)
		case !slices.Contains(channels, a.Channels):
			add("%s audio %d is %s with %d channels; %s encodes %v", at, i+1, a.Codec, a.Channels, a.Codec, channels)
		}
		checkLanguage(add, at, "audio", i, a.Language)
	}
	for i, s := range m.Subtitles {
		checkLanguage(add, at, "subtitle", i, s.Language)
	}
}

func checkLanguage(add func(format string, args ...any), at, kind string, i int, lang string) {
	if lang == "" {
		return
	}
	if _, err := release.ParseLanguage(lang); err != nil {
		add("%s %s %d language: %v", at, kind, i+1, err)
	}
}
