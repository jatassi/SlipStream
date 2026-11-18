package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SampleRate is the rate Decoder streams audio at.
const SampleRate = 16000

// Decoder streams one audio stream of a file as mono 16 kHz signed 16-bit
// little-endian PCM with the bundled ffmpeg, padded to the presentation
// start so audio that starts late reads at its true time. FFprobe picks
// the stream and reads where it starts.
type Decoder struct {
	FFmpeg  string
	FFprobe string
}

// Recipe names how Decoder turns a stream into PCM, for a Speech Map's
// header: a change makes every map stale.
const Recipe = "mono-downmix,async-pad"

// Decoding is one audio stream being decoded, read as PCM.
type Decoding struct {
	// Stream is the audio ordinal decoded.
	Stream int
	// Start is how late the stream starts after the file does, which the
	// PCM begins with as silence.
	Start  time.Duration
	out    io.Reader
	ended  bool
	cmd    *exec.Cmd
	stderr *bytes.Buffer
}

func (a *Decoding) Read(p []byte) (int, error) {
	n, err := a.out.Read(p)
	if errors.Is(err, io.EOF) {
		a.ended = true
	}
	return n, err //nolint:wrapcheck // io.Reader's io.EOF stays bare
}

var (
	// ErrStart is a bundled tool that can't run.
	ErrStart = errors.New("start the bundled ffmpeg")
	// ErrNoAudio is a file without the audio stream asked for.
	ErrNoAudio = errors.New("the file has no audio stream")
)

// DecodeError is a file ffmpeg or ffprobe can't read. First is the first
// line of its error output.
type DecodeError struct{ First string }

func (e *DecodeError) Error() string { return "decode the audio: " + e.First }

// Audio starts decoding stream, an audio ordinal, or -1 for the default
// track, else the first. A stream the file lacks is ErrNoAudio before
// ffmpeg starts.
func (d Decoder) Audio(ctx context.Context, path string, stream int) (*Decoding, error) {
	ordinal, start, err := d.pick(ctx, path, stream)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, d.FFmpeg, "-v", "error", "-nostdin", "-protocol_whitelist", "file,pipe", "-i", path, //nolint:gosec // the bundled ffmpeg on a media file
		"-map", "0:a:"+strconv.Itoa(ordinal), "-vn", "-sn", "-dn", "-ac", "1", "-ar", strconv.Itoa(SampleRate),
		"-af", "aresample=async=1:first_pts=0", "-f", "s16le", "-acodec", "pcm_s16le", "-")
	cmd.WaitDelay = waitDelay
	a := &Decoding{Stream: ordinal, Start: start, cmd: cmd, stderr: &bytes.Buffer{}}
	cmd.Stderr = a.stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStart, err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStart, err)
	}
	a.out = out
	return a, nil
}

// Close answers how ffmpeg ended once the whole stream was read: nil, or a
// *DecodeError when it failed. Before then it kills ffmpeg and answers nil.
func (a *Decoding) Close() error {
	if !a.ended {
		_ = a.cmd.Process.Kill()
		_ = a.cmd.Wait()
		return nil
	}
	err := a.cmd.Wait()
	if _, exited := errors.AsType[*exec.ExitError](err); exited {
		return &DecodeError{First: firstLine(a.stderr.String())}
	}
	if err != nil {
		return fmt.Errorf("run ffmpeg: %w", err)
	}
	return nil
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

type probedAudio struct {
	Streams []struct {
		Start       string `json:"start_time"`
		Disposition struct {
			Default int `json:"default"`
		} `json:"disposition"`
	} `json:"streams"`
	Format struct {
		Start string `json:"start_time"`
	} `json:"format"`
}

// pick is the ordinal stream names and how late it starts.
func (d Decoder) pick(ctx context.Context, path string, stream int) (int, time.Duration, error) {
	cmd := exec.CommandContext(ctx, d.FFprobe, "-v", "error", "-protocol_whitelist", "file,pipe", "-print_format", "json", //nolint:gosec // the bundled ffprobe on a media file
		"-select_streams", "a", "-show_entries", "stream=start_time:stream_disposition=default:format=start_time", path)
	cmd.WaitDelay = waitDelay
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	switch _, exited := errors.AsType[*exec.ExitError](err); {
	case ctx.Err() != nil:
		return 0, 0, fmt.Errorf("run ffprobe: %w", ctx.Err())
	case exited:
		return 0, 0, &DecodeError{First: firstLine(stderr.String())}
	case err != nil:
		return 0, 0, fmt.Errorf("%w: %w", ErrStart, err)
	}
	var p probedAudio
	if err := json.Unmarshal(out, &p); err != nil {
		return 0, 0, fmt.Errorf("read ffprobe's streams: %w", err)
	}
	if stream < 0 {
		stream = 0
		for i, s := range p.Streams {
			if s.Disposition.Default == 1 {
				stream = i
				break
			}
		}
	}
	if stream >= len(p.Streams) {
		return 0, 0, ErrNoAudio
	}
	at, err := seconds(p.Streams[stream].Start)
	if err != nil {
		return 0, 0, err
	}
	from, err := seconds(p.Format.Start)
	if err != nil {
		return 0, 0, err
	}
	return stream, max(0, at-from), nil
}
