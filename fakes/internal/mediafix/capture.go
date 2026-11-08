package mediafix

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jatassi/slipstream/fakes/scenario"
)

// Capture is one file whose ffprobe output internal/media's tests parse,
// kept in internal/media/testdata/ffprobe as <Name>.streams.json,
// <Name>.frame.json and <Name>.cmd.
type Capture struct {
	Name string
	File scenario.MediaFile
	// Atmos is the profile written by hand into the first audio stream:
	// FFmpeg has no Atmos encoder, so the capture is of a plain track.
	Atmos string
	Notes []string
}

func video(hdr string) scenario.MediaVideo {
	return scenario.MediaVideo{Width: 320, Height: 180, Seconds: 1, HDR: hdr}
}

var surround = []scenario.MediaAudio{{Codec: "eac3", Channels: 6, Language: "eng"}}

// Captures are the files internal/media's Parse tests read.
var Captures = []Capture{
	{Name: "sdr-h264", File: scenario.MediaFile{Path: "sdr-h264.mkv", Video: video("sdr"), Audio: []scenario.MediaAudio{{Codec: "aac", Channels: 2, Language: "eng"}}}},
	{Name: "hdr10", File: scenario.MediaFile{Path: "hdr10.mkv", Video: video("hdr10")}},
	{Name: "hdr10plus", File: scenario.MediaFile{Path: "hdr10plus.mkv", Video: video("hdr10plus")}},
	{Name: "hlg", File: scenario.MediaFile{Path: "hlg.mkv", Video: video("hlg")}},
	{Name: "dv5", File: scenario.MediaFile{Path: "dv5.mkv", Video: video("dv5")}},
	{Name: "dv8", File: scenario.MediaFile{Path: "dv8.mkv", Video: video("dv8")}},
	{Name: "multi-audio-subs", File: scenario.MediaFile{
		Path: "multi-audio-subs.mkv", Video: video("sdr"),
		Audio: []scenario.MediaAudio{
			{Codec: "eac3", Channels: 6, Language: "eng", Default: true, Title: "Surround"},
			{Codec: "aac", Channels: 2, Language: "jpn"},
			{Codec: "truehd", Channels: 6, Language: "fre"},
		},
		Subtitles: []scenario.MediaSubtitle{{Language: "spa", Forced: true}, {Language: "eng"}},
	}},
	{
		Name: "no-duration", File: scenario.MediaFile{Path: "no-duration.h264", Video: video("sdr")},
		Notes: []string{"No container this ffmpeg writes (.ts or .m2ts included, truncated or not) leaves format.duration out, so the case is an H.264 elementary stream."},
	},
	{Name: "eac3-atmos", File: scenario.MediaFile{Path: "eac3-atmos.mkv", Video: video("sdr"), Audio: surround}, Atmos: "Dolby Digital Plus + Dolby Atmos"},
	{Name: "truehd-atmos", File: scenario.MediaFile{Path: "truehd-atmos.mkv", Video: video("sdr"), Audio: []scenario.MediaAudio{{Codec: "truehd", Channels: 6, Language: "eng"}}}, Atmos: "Dolby TrueHD + Dolby Atmos"},
}

// probes are the two ffprobe runs media.FFprobe makes, as the capture's
// file suffix and arguments before the file.
var probes = []struct{ suffix, args string }{
	{".streams.json", "-v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams"},
	{".frame.json", "-v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1"},
}

// Capture writes c's three files into dir: the bundled ffprobe's two runs
// on the file Build makes, and the commands that made them.
func (b *Builder) Capture(ctx context.Context, c Capture, dir string) error {
	built, err := b.Build(ctx, c.File)
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "mediafix-capture-")
	if err != nil {
		return fmt.Errorf("capture %s: %w", c.Name, err)
	}
	defer func() { _ = os.RemoveAll(work) }()
	name := c.Name + path.Ext(c.File.Path)
	data, err := os.ReadFile(built)
	if err != nil {
		return fmt.Errorf("capture %s: %w", c.Name, err)
	}
	if err := os.WriteFile(filepath.Join(work, name), data, 0o600); err != nil {
		return fmt.Errorf("capture %s: %w", c.Name, err)
	}
	for _, p := range probes {
		cmd := exec.CommandContext(ctx, b.tools.FFprobe, append(strings.Fields(p.args), name)...)
		cmd.Dir = work
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("capture %s: ffprobe: %w", c.Name, err)
		}
		if p.suffix == ".streams.json" && c.Atmos != "" {
			if out, err = withAtmos(out, c.Atmos); err != nil {
				return fmt.Errorf("capture %s: %w", c.Name, err)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, c.Name+p.suffix), out, 0o644); err != nil {
			return fmt.Errorf("capture %s: %w", c.Name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, c.Name+".cmd"), b.commands(c), 0o644); err != nil {
		return fmt.Errorf("capture %s: %w", c.Name, err)
	}
	return nil
}

// longName is the codec_long_name line of a stream in ffprobe's JSON.
var longName = regexp.MustCompile(`(?m)^( *)"codec_long_name": .*\n`)

// withAtmos writes profile into the audio stream after the video, after
// its codec_long_name, where ffprobe prints a profile.
func withAtmos(streams []byte, profile string) ([]byte, error) {
	at := longName.FindAllSubmatchIndex(streams, 2)
	if len(at) < 2 {
		return nil, errors.New("ffprobe printed no audio stream")
	}
	line := string(streams[at[1][2]:at[1][3]]) + `"profile": "` + profile + "\",\n"
	return bytes.Join([][]byte{streams[:at[1][1]], []byte(line), streams[at[1][1]:]}, nil), nil
}

// commands is the .cmd beside c's captures: a shell script of what made
// them, run in an empty folder with the bundled tools.
func (b *Builder) commands(c Capture) []byte {
	var s strings.Builder
	fmt.Fprintf(&s, "# %s: made by go run ./fakes/internal/mediafix/capture with the tools SlipStream bundles (native/lock.json):\n", c.Name)
	fmt.Fprintf(&s, "# %s, %s.\n", strings.Join(strings.Fields(b.ffmpeg)[:3], " "), strings.TrimSpace(b.dovi))
	for _, n := range c.Notes {
		fmt.Fprintf(&s, "# %s\n", n)
	}
	if c.Atmos != "" {
		fmt.Fprintf(&s, "# FFmpeg has no Atmos encoder (E-AC-3 JOC or TrueHD with Atmos), so the capture of a plain %s track\n", c.File.Audio[0].Codec)
		fmt.Fprintf(&s, "# has only streams[1].profile set by hand to %q, the string the bundled ffprobe prints for it:\n", c.Atmos)
		s.WriteString("# strings ffprobe | grep Atmos  ->  Dolby Digital Plus + Dolby Atmos, Dolby TrueHD + Dolby Atmos\n")
	}
	r := recipeOf(c.Name, c.File)
	for _, f := range r.files {
		fmt.Fprintf(&s, "cat > %s <<'EOF'\n%sEOF\n", f.name, f.text)
	}
	for _, st := range r.steps {
		s.WriteString(shell(append([]string{st.tool}, st.args...)) + "\n")
	}
	for _, p := range probes {
		fmt.Fprintf(&s, "ffprobe %s %s > %s%s\n", p.args, r.out, c.Name, p.suffix)
	}
	return []byte(s.String())
}

var plain = regexp.MustCompile(`^[A-Za-z0-9_./:=,+%#@-]+$`)

func shell(args []string) string {
	out := make([]string, len(args))
	for i, a := range args {
		if plain.MatchString(a) {
			out[i] = a
			continue
		}
		out[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	return strings.Join(out, " ")
}
