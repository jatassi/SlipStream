// Package simconv is the simulation's Converting seams: a runner whose
// every run succeeds and proofs for the processor alone.
package simconv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
)

// Made is a world's prober: each file a run on any Node made, by its base
// name, and every other path from the prober it wraps. A file the wrapped
// prober doesn't know, such as a source's copy in a scratch folder, reads
// as the file it probed with the same first bytes.
type Made struct {
	base media.Prober
	mu   sync.Mutex
	of   map[string]media.Record
	// bytes is each probed file's record by its first bytes.
	bytes map[string]media.Record
}

func NewMade(base media.Prober) *Made {
	return &Made{base: base, of: map[string]media.Record{}, bytes: map[string]media.Record{}}
}

// headLen is how many first bytes tell two copies of a file apart from
// every other file the walk writes.
const headLen = 32

func (m *Made) Probe(ctx context.Context, path string) (media.Record, error) {
	m.mu.Lock()
	r, ok := m.of[filepath.Base(path)]
	m.mu.Unlock()
	if ok {
		return r, nil
	}
	head := headOf(path)
	r, err := m.base.Probe(ctx, path)
	m.mu.Lock()
	defer m.mu.Unlock()
	if err == nil && head != "" {
		m.bytes[head] = r
	}
	if copied, ok := m.bytes[head]; err != nil && head != "" && ok {
		return copied, nil
	}
	return r, err
}

// headOf is path's first bytes, "" when it has fewer or can't be read.
func headOf(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	b := make([]byte, headLen)
	_, err = io.ReadFull(f, b)
	if cerr := f.Close(); err != nil || cerr != nil {
		return ""
	}
	return string(b)
}

// Converting is node.Env.Converting over m: Succeeds registering what it
// makes with m, and ProcessorOnly.
func (m *Made) Converting(converting.RealConfig) (converting.Converter, converting.Proofs) {
	return Succeeds{Made: m}, ProcessorOnly{}
}

// Succeeds writes a placeholder of its own bytes at the output, ffmpeg's
// last argument, registers its record under the name the output is
// renamed to and under those bytes, so a copy reads as it, and exits 0 at
// once. The record is the input's with the streams the arguments map,
// and the codec, frame size, tone map and video bitrate they encode.
type Succeeds struct{ Made *Made }

func (s Succeeds) Start(ctx context.Context, args []string) (converting.Run, error) {
	out := args[len(args)-1]
	src, err := s.Made.Probe(ctx, args[slices.Index(args, "-i")+1])
	if err != nil {
		return nil, fmt.Errorf("probe the input: %w", err)
	}
	name := strings.TrimSuffix(filepath.Base(out), ".part")
	sum := sha256.Sum256([]byte(name))
	body := hex.EncodeToString(sum[:])
	if err := os.WriteFile(out, []byte(body), 0o600); err != nil {
		return nil, fmt.Errorf("write the output: %w", err)
	}
	r := made(src, args)
	s.Made.mu.Lock()
	s.Made.of[name], s.Made.bytes[body[:headLen]] = r, r
	s.Made.mu.Unlock()
	progress := make(chan converting.Progress)
	close(progress)
	return exited{progress}, nil
}

func made(src media.Record, args []string) media.Record {
	out := src
	out.Audio, out.Subtitles = nil, nil
	for i, a := range args {
		switch {
		case a == "-map" && strings.HasPrefix(args[i+1], "0:a:"):
			n, _ := strconv.Atoi(strings.TrimPrefix(args[i+1], "0:a:"))
			out.Audio = append(out.Audio, src.Audio[n])
		case a == "-map" && strings.HasPrefix(args[i+1], "0:s:"):
			n, _ := strconv.Atoi(strings.TrimPrefix(args[i+1], "0:s:"))
			out.Subtitles = append(out.Subtitles, src.Subtitles[n])
		case a == "-c:v" && args[i+1] != "copy":
			out.Codec = codecOf(args[i+1])
		case a == "-vf":
			filtered(&out, args[i+1])
		case a == "-b:v":
			kbps, _ := strconv.Atoi(strings.TrimSuffix(args[i+1], "k"))
			out.Bitrate = int64(kbps) * 1000
		}
	}
	return out
}

// filtered applies a -vf chain's scale and tone map to out.
func filtered(out *media.Record, chain string) {
	if m := scaled.FindStringSubmatch(chain); m != nil {
		out.Width, _ = strconv.Atoi(m[1])
		out.Height, _ = strconv.Atoi(m[2])
	}
	switch {
	case strings.Contains(chain, ":t=bt709:"):
		out.HDR, out.DolbyVision, out.BitDepth = []release.HDR{"SDR"}, nil, 8
	case strings.Contains(chain, ":t=smpte2084:"):
		out.HDR, out.DolbyVision = []release.HDR{"HDR10"}, nil
	}
}

var scaled = regexp.MustCompile(`scale=w=(\d+):h=(\d+)`)

// codecOf is the codec an ffmpeg encoder writes, as ffprobe names it.
func codecOf(encoder string) string {
	for _, c := range []encode.Codec{encode.H264, encode.HEVC, encode.AV1} {
		if strings.Contains(encoder, string(c)) {
			return string(c)
		}
	}
	return map[string]string{"libx264": "h264", "libx265": "hevc", "libsvtav1": "av1"}[encoder]
}

type exited struct{ progress chan converting.Progress }

func (e exited) Progress() <-chan converting.Progress { return e.progress }
func (exited) Pause() error                           { return nil }
func (exited) Resume() error                          { return nil }
func (exited) Kill()                                  {}
func (exited) Wait() converting.Exit                  { return converting.Exit{} }
func (exited) Tail() string                           { return "" }

// ProcessorOnly passes every test run on the processor, at real time, and
// finds no graphics card.
type ProcessorOnly struct{}

// Signature is the one every simulated Node has.
var Signature = converting.Signature(strings.Repeat("5", 64))

func (ProcessorOnly) Signature(context.Context) (converting.Signature, []converting.Device, error) {
	return Signature, nil, nil
}

func (ProcessorOnly) Test(context.Context, encode.Key) (converting.Proved, error) {
	return converting.Proved{
		Passed: true, Pipeline: encode.Pipeline{Path: encode.Processor}, Speed: 1, Pauses: true, Signature: Signature,
	}, nil
}

func (ProcessorOnly) Runs() (role.Fit, role.ProbeReason) { return role.Weak, role.NoGPU }

func (ProcessorOnly) Warm(context.Context, func()) error { return nil }
