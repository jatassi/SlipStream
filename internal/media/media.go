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
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	mediav1 "github.com/jatassi/slipstream/internal/pb/slipstream/media/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/media/v1/mediav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
)

type Prober interface {
	Probe(ctx context.Context, path string) (Record, error)
}

type Timings struct {
	// Probe bounds one file's probe, both ffprobe runs.
	Probe time.Duration `yaml:"probe"`
}

func DefaultTimings() Timings { return Timings{Probe: 30 * time.Second} }

// waitDelay bounds how long an ffprobe stuck in kernel IO outlives its
// context, so a hung share can't hold a probe past its timeout or a
// Cancel.
const waitDelay = 5 * time.Second

// FFprobe is the bundled ffprobe at Path. It runs twice per file, as
// Parse reads.
type FFprobe struct{ Path string }

func (f FFprobe) Probe(ctx context.Context, file string) (Record, error) {
	streams, err := f.run(ctx, "-show_format", "-show_streams", file)
	if err != nil {
		return Record{}, err
	}
	frame, err := f.run(ctx, "-select_streams", "v:0", "-show_frames", "-read_intervals", "%+#1", file)
	if err != nil {
		return Record{}, err
	}
	return Parse(streams, frame)
}

func (f FFprobe) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, f.Path, append([]string{"-v", "error", "-protocol_whitelist", "file,pipe", "-print_format", "json"}, args...)...) //nolint:gosec // the bundled ffprobe on a media file
	cmd.WaitDelay = waitDelay
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("run ffprobe: %w", ctx.Err())
	}
	if _, exited := errors.AsType[*exec.ExitError](err); exited {
		line, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n")
		return nil, &UnreadableError{Line: line}
	}
	if err != nil {
		return nil, fmt.Errorf("run ffprobe: %w", err)
	}
	return out, nil
}

// UnreadableError is a file ffprobe can't read. Line is the first line of
// ffprobe's error output.
type UnreadableError struct{ Line string }

func (e *UnreadableError) Error() string { return "ffprobe can't read the file: " + e.Line }

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
	r, err := s.prober.Probe(ctx, file)
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeFailedPrecondition, "%v", err)
	}
	record, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("encode the media record: %w", err)
	}
	return &mediav1.ProbeResponse{Record: record}, nil
}

// Remote is MediaService as the Controller reaches it on a Node.
type Remote func(role.NodeID) mediav1connect.MediaServiceHandler

// Probe probes the index'th file of torrent h on node. Its errors are the
// call's, for rpc.NotSent and rpc.Lost to read.
func (r Remote) Probe(ctx context.Context, node role.NodeID, h bencode.InfoHash, index int) (Record, error) {
	res, err := r(node).Probe(ctx, &mediav1.ProbeRequest{InfoHash: h[:], FileIndex: int32(index)}) //nolint:gosec // a torrent's file index fits
	if err != nil {
		return Record{}, err
	}
	var rec Record
	if err := json.Unmarshal(res.GetRecord(), &rec); err != nil {
		return Record{}, fmt.Errorf("read the media record: %w", err)
	}
	return rec, nil
}
