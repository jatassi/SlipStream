package mediafix

import (
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/internal/speech"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/converting/clip"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
)

// recipe is the commands that make one file in an empty folder: write
// files, then run steps, which leave out.
type recipe struct {
	files []file
	steps []clip.Step
	out   string
}

// file is one file a recipe writes. key stands for text in the cache key
// when set, so a long file keys on what makes it.
type file struct{ name, text, key string }

var bitexact = []string{"-fflags", "+bitexact", "-flags:v", "+bitexact", "-flags:a", "+bitexact", "-map_metadata", "-1"}

// classes are the clip classes of the scenario's HDR kinds but HDR10+.
var classes = map[string]encode.Class{
	"sdr": encode.ClassH264x8, "hdr10": encode.ClassHDR10, "hlg": encode.ClassHLG, "dv8": encode.ClassDV81, "dv5": encode.ClassDV5,
}

// recipeOf is the recipe that makes f, naming every file it writes after
// stem. The bytes it makes don't depend on stem. A salt other than ""
// goes in the container's comment tag, so files of one description made
// with two salts differ in their bytes and read alike.
func recipeOf(stem string, f scenario.MediaFile, salt string) recipe {
	ext := path.Ext(f.Path)
	size := clip.Size{W: f.Video.Width, H: f.Video.Height, Seconds: f.Video.Seconds}
	v := clip.HDR10Plus(size, stem)
	if f.Video.HDR != "hdr10plus" {
		v = clip.VideoOf(classes[f.Video.HDR], size, stem)
	}
	r := recipe{out: stem + ext, steps: v.Before}
	for _, vf := range v.Files {
		r.files = append(r.files, file{name: vf.Name, text: vf.Text})
	}
	args := append([]string{"-v", "error", "-y"}, v.Input...)
	maps := []string{"-map", "0:v"}
	var tracks []string
	inputs := 1
	if len(f.Audio) > 0 {
		var audio []string
		args, audio, inputs = r.audio(stem, f, args)
		maps = append(maps, audio...)
		tracks = append(tracks, audioTracks(f.Audio)...)
	}
	subtitles := inputs
	for i, s := range f.Subtitles {
		srt := fmt.Sprintf("%s.%d.srt", stem, i+1)
		r.files = append(r.files, file{name: srt, text: fmt.Sprintf("1\n00:00:00,000 --> 00:00:01,000\nSubtitle %d\n", i+1)})
		args = append(args, "-i", srt)
		maps = append(maps, "-map", strconv.Itoa(subtitles+i))
		tracks = append(tracks, subtitleTrack(i, s)...)
	}
	if len(f.Subtitles) > 0 {
		tracks = append(tracks, "-c:s", "srt")
	}
	args = slices.Concat(args, maps, v.Codec, tracks, bitexact)
	if salt != "" {
		args = append(args, "-metadata", "comment="+salt)
	}
	if ext == ".h264" {
		args = append(args, "-f", "h264")
	}
	r.steps = append(r.steps, clip.Step{Tool: clip.FFmpeg, Args: append(args, r.out)})
	return r
}

// layouts are the channel layouts a track of each channel count gets.
// FFmpeg's AAC encoder writes 5.1(side) as an unknown layout, so AAC's
// six channels are 5.1.
var layouts = map[int]string{1: "mono", 2: "stereo", 6: "5.1(side)", 8: "7.1"}

func layoutOf(a scenario.MediaAudio) string {
	if a.Codec == "aac" && a.Channels == 6 {
		return "5.1"
	}
	return layouts[a.Channels]
}

// audio adds the inputs that make f's audio, after the video's: one 440
// Hz tone split into a pan per tone track, and each speech track's PCM,
// written in Go so ffmpeg only remuxes it. It answers the args, each
// track's -map in track order, and the next input's index.
func (r *recipe) audio(stem string, f scenario.MediaFile, args []string) (out, maps []string, next int) {
	var split, pans strings.Builder
	next, tones := 1, 0
	for _, a := range f.Audio {
		if a.Speech == nil {
			tones++
		}
	}
	if tones > 0 {
		args = append(args, "-f", "lavfi", "-i", fmt.Sprintf("sine=frequency=440:sample_rate=48000:duration=%d", f.Video.Seconds))
		split.WriteString("[1:a]asplit=" + strconv.Itoa(tones))
		next++
	}
	tone := 0
	for i, a := range f.Audio {
		if a.Speech != nil {
			pcm := fmt.Sprintf("%s.a%d.pcm", stem, i+1)
			key := fmt.Sprintf("speech seed %d density %g length %v", a.Speech.Seed, a.Speech.Density, a.Speech.Length)
			r.files = append(r.files, file{name: pcm, text: string(speech.PCM(*a.Speech)), key: key})
			args = append(args, "-f", "s16le", "-ar", strconv.Itoa(speechtest.Rate), "-ac", "1", "-i", pcm)
			maps = append(maps, "-map", strconv.Itoa(next)+":a")
			next++
			continue
		}
		fmt.Fprintf(&split, "[s%d]", tone)
		fmt.Fprintf(&pans, ";[s%d]pan=%s", tone, layoutOf(a))
		for c := range a.Channels {
			fmt.Fprintf(&pans, "|c%d=c0", c)
		}
		fmt.Fprintf(&pans, "[a%d]", tone)
		maps = append(maps, "-map", fmt.Sprintf("[a%d]", tone))
		tone++
	}
	if tones > 0 {
		args = append(args, "-filter_complex", split.String()+pans.String())
	}
	return args, maps, next
}

func audioTracks(audio []scenario.MediaAudio) []string {
	var args []string
	experimental := false
	for i, a := range audio {
		n := strconv.Itoa(i)
		args = append(args, "-c:a:"+n, a.Codec)
		if a.Language != "" {
			args = append(args, "-metadata:s:a:"+n, "language="+a.Language)
		}
		if a.Title != "" {
			args = append(args, "-metadata:s:a:"+n, "title="+a.Title)
		}
		args = append(args, "-disposition:a:"+n, disposition(a.Default, false))
		experimental = experimental || a.Codec == "truehd"
	}
	if experimental {
		args = append(args, "-strict", "-2")
	}
	return args
}

func subtitleTrack(i int, s scenario.MediaSubtitle) []string {
	n := strconv.Itoa(i)
	var args []string
	if s.Language != "" {
		args = append(args, "-metadata:s:s:"+n, "language="+s.Language)
	}
	return append(args, "-disposition:s:"+n, disposition(s.Default, s.Forced))
}

func disposition(def, forced bool) string {
	switch {
	case def && forced:
		return "default+forced"
	case def:
		return "default"
	case forced:
		return "forced"
	}
	return "0"
}
