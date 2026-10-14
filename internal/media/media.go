// Package media probes video files with the bundled ffprobe, where they
// lie: each Node serves MediaService over its own downloads, so the
// Controller never names a path on another machine.
package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	mediav1 "github.com/jatassi/slipstream/internal/pb/slipstream/media/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/media/v1/mediav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/targets"
)

// Probe is what probing found: the container and the first video stream.
type Probe struct {
	Duration  time.Duration `json:"duration"`
	Container string        `json:"container"`
	Codec     string        `json:"codec"`
	Width     int           `json:"width"`
	Height    int           `json:"height"`
}

// Resolution is the probe's resolution class.
func (p Probe) Resolution() targets.Resolution { return Class(p.Width, p.Height) }

type Prober interface {
	Probe(ctx context.Context, path string) (Probe, error)
}

type Timings struct {
	// Probe bounds one ffprobe run.
	Probe time.Duration `yaml:"probe"`
}

func DefaultTimings() Timings { return Timings{Probe: 30 * time.Second} }

// FFprobe is the bundled ffprobe at Path.
type FFprobe struct{ Path string }

func (f FFprobe) Probe(ctx context.Context, file string) (Probe, error) {
	cmd := exec.CommandContext(ctx, f.Path, "-v", "error", "-protocol_whitelist", "file,pipe", "-print_format", "json", "-show_format", "-show_streams", file) //nolint:gosec // the bundled ffprobe on a downloaded file
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return Probe{}, fmt.Errorf("ffprobe: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	return ParseFFprobe(out)
}

type ffprobeOutput struct {
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"streams"`
}

// ParseFFprobe reads ffprobe's JSON: the format's duration and name, and
// the first video stream's codec and size.
func ParseFFprobe(out []byte) (Probe, error) {
	var doc ffprobeOutput
	if err := json.Unmarshal(out, &doc); err != nil {
		return Probe{}, fmt.Errorf("read ffprobe's answer: %w", err)
	}
	seconds, err := strconv.ParseFloat(doc.Format.Duration, 64)
	if err != nil {
		return Probe{}, fmt.Errorf("read ffprobe's answer: duration %q: %w", doc.Format.Duration, err)
	}
	p := Probe{Duration: time.Duration(seconds * float64(time.Second)).Round(time.Millisecond), Container: doc.Format.FormatName}
	for _, s := range doc.Streams {
		if s.CodecType == "video" {
			p.Codec, p.Width, p.Height = s.CodecName, s.Width, s.Height
			return p, nil
		}
	}
	return Probe{}, errors.New("the file has no video stream")
}

// Class is the resolution class of a width by height video. The width
// counts too, so a cropped video keeps its class.
func Class(width, height int) targets.Resolution {
	switch {
	case width >= 3200 || height >= 2000:
		return targets.R2160p
	case width >= 1800 || height >= 1000:
		return targets.R1080p
	case width >= 1200 || height >= 700:
		return targets.R720p
	case width >= 1000 || height >= 560:
		return targets.R576p
	}
	return targets.R480p
}

// Payloads is where this Node's downloaded torrent files lie.
type Payloads interface {
	Path(h bencode.InfoHash, index int) (string, error)
}

// Service is MediaService on this Node.
type Service struct {
	prober   Prober
	payloads Payloads
	timings  Timings
}

func NewService(p Prober, payloads Payloads, t Timings) *Service {
	return &Service{prober: p, payloads: payloads, timings: t}
}

func (s *Service) Probe(ctx context.Context, req *mediav1.ProbeRequest) (*mediav1.ProbeResponse, error) {
	var h bencode.InfoHash
	if len(req.GetInfoHash()) != len(h) {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "an info hash is %d bytes, not %d", len(h), len(req.GetInfoHash()))
	}
	copy(h[:], req.GetInfoHash())
	file, err := s.payloads.Path(h, int(req.GetFileIndex()))
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeUnavailable, "%v", err)
	}
	ctx, cancel := context.WithTimeout(ctx, s.timings.Probe)
	defer cancel()
	p, err := s.prober.Probe(ctx, file)
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeFailedPrecondition, "%v", err)
	}
	return &mediav1.ProbeResponse{
		DurationMs: p.Duration.Milliseconds(),
		Container:  p.Container,
		Codec:      p.Codec,
		Width:      int32(p.Width),  //nolint:gosec // a video's width fits
		Height:     int32(p.Height), //nolint:gosec // a video's height fits
	}, nil
}

// Remote is MediaService as the Controller reaches it on a Node.
type Remote func(role.NodeID) mediav1connect.MediaServiceHandler

// Probe probes the index'th file of torrent h on node. Its errors are the
// call's, for rpc.NotSent and rpc.Lost to read.
func (r Remote) Probe(ctx context.Context, node role.NodeID, h bencode.InfoHash, index int) (Probe, error) {
	res, err := r(node).Probe(ctx, &mediav1.ProbeRequest{InfoHash: h[:], FileIndex: int32(index)}) //nolint:gosec // a torrent's file index fits
	if err != nil {
		return Probe{}, err
	}
	return Probe{
		Duration:  time.Duration(res.GetDurationMs()) * time.Millisecond,
		Container: res.GetContainer(),
		Codec:     res.GetCodec(),
		Width:     int(res.GetWidth()),
		Height:    int(res.GetHeight()),
	}, nil
}

// videoExtensions are the extensions a probed file may have.
var videoExtensions = []string{"mkv", "mp4", "m4v", "avi", "ts"}

// IsVideo reports whether name, '/' separated, names a video file by its
// extension.
func IsVideo(name string) bool {
	ext := strings.TrimPrefix(path.Ext(name), ".")
	return slices.Contains(videoExtensions, strings.ToLower(ext))
}
