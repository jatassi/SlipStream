package converting

import (
	"context"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
)

// Converter is node.Env.Converter: how this Node runs ffmpeg.
type Converter interface {
	Start(ctx context.Context, args []string) (Run, error)
}

// Run is one ffmpeg process tree.
type Run interface {
	// Progress is closed at exit.
	Progress() <-chan Progress
	// Pause writes "p".
	Pause() error
	// Resume writes "u".
	Resume() error
	// Kill ends the whole tree.
	Kill()
	Wait() Exit
	// Tail is the last 2 KiB of ffmpeg's error output so far.
	Tail() string
}

type Progress struct {
	OutTimeUs int64
	Speed     float64
}

// Exit is a run's exit code and the last 2 KiB of its error output.
type Exit struct {
	Code int
	Tail string
}

// Proofs is node.Env.Proofs: the Node's signature, its test runs and the
// ffmpeg-runs probe.
type Proofs interface {
	// Signature runs no encode.
	Signature(ctx context.Context) (Signature, []Device, error)
	// Test walks pipeline.For(GOOS, k.Shape).
	Test(ctx context.Context, k encode.Key) (Proved, error)
	// Runs is the ffmpeg-runs probe from its cache: Fits when the base test
	// passed on the processor and a hardware row opened an encoder, Weak
	// no_gpu when only the processor passed, Fails ffmpeg_fails when
	// neither, Weak ffmpeg_testing before the first answer.
	Runs() (role.Fit, role.ProbeReason)
	// Warm runs the base test now and whenever the signature changes, and
	// calls changed after each answer so the Node reports its probes again.
	Warm(ctx context.Context, changed func()) error
}

// Signature is SHA-256 over ffmpeg -version and each adapter's driver
// version, in hex.
type Signature string

// Device is a video adapter as the Node's drivers report it.
type Device struct {
	Name, Vendor, Driver string
}

type Proved struct {
	Passed   bool
	Pipeline encode.Pipeline
	Device   string
	// Speed is a real-time factor, scaled to 1080p.
	Speed     float64
	Pauses    bool
	Signature Signature
	// Tries are each row tried, with its error, for Details.
	Tries []Try
}

type Try struct {
	Pipeline encode.Pipeline
	Err      string
}
