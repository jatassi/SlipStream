package conversions

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/encode"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestAPinnedRetryMovesOnWhenItsNodeCantRunIt: a retry pinned to Attic
// waits for Attic's new test run of its key. A failed one, or Attic
// lapsing, moves it on to another Node with Attic excluded. A passing
// one, a failed one of another key or one under an old signature keeps
// it pinned.
func TestAPinnedRetryMovesOnWhenItsNodeCantRunIt(t *testing.T) {
	l := newLife(t)
	l.key = l.prove(t, l.attic)
	other := encode.Key{Shape: l.key.Shape, Class: encode.ClassH264x8}
	const sig = converting.Signature("current")
	tries := []converting.Try{{Pipeline: encode.Pipeline{Path: encode.Processor}, Err: "the encoder didn't open"}}
	tested := func(k encode.Key, p converting.Proved) func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return l.c.prove(ctx, tx, l.attic, sig, k, p)
		}
	}
	cases := []struct {
		name  string
		then  func(ctx context.Context, tx *storage.Tx[storage.Library]) error
		moved bool
	}{
		{"its test run fails", tested(l.key, converting.Proved{Signature: sig, Tries: tries}), true},
		{"its test run passes", tested(l.key, converting.Proved{Passed: true, Pipeline: encode.Pipeline{Path: encode.Processor}, Speed: 3, Signature: sig}), false},
		{"another key's test run fails", tested(other, converting.Proved{Signature: sig, Tries: tries}), false},
		{"a test run under an old signature fails", tested(l.key, converting.Proved{Signature: "old", Tries: tries}), false},
		{"its Node lapses", func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			if err := l.c.putBack(ctx, tx, l.attic); err != nil {
				return err
			}
			return l.c.unpin(ctx, tx, l.attic)
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := l.seed(t, seed{Stage: StageQueued, Pin: &l.attic, Attempt: 1, Failed: []role.NodeID{l.attic}})
			l.write(t, tc.then)
			a := l.read(t, r.ID)
			if moved := a.Pin == nil && slices.Equal(a.Exclude, []role.NodeID{l.attic}); moved != tc.moved || a.Stage != StageQueued {
				t.Errorf("%s pinned to %v excluding %v, want moved on %v", a.Stage, a.Pin, a.Exclude, tc.moved)
			}
		})
	}
}

// TestTestRunsAndFactsBeyondTheirCaps: a test run with a signature that
// isn't 64 hex characters, or a passing one on a pipeline its OS has no
// row for, is refused, and a passing speed, a device name and the tries
// are held to their caps. Facts past their caps lose the parts past them.
func TestTestRunsAndFactsBeyondTheirCaps(t *testing.T) {
	sig := strings.Repeat("ab", 32)
	qsv := &cv1.Pipeline{Path: "hardware", Api: "qsv"}
	tries := make([]*cv1.Try, 12)
	for i := range tries {
		tries[i] = &cv1.Try{Pipeline: qsv, Error: strings.Repeat("x", 3000)}
	}
	cases := []struct {
		name    string
		res     *cv1.TestResponse
		os      string
		refused bool
		speed   float64
	}{
		{"a short signature", &cv1.TestResponse{Signature: "abc", Tries: tries}, "windows", true, 0},
		{"a failed run with no tries", &cv1.TestResponse{Signature: sig}, "windows", true, 0},
		{"a pipeline the OS has no row for", &cv1.TestResponse{Passed: true, Pipeline: &cv1.Pipeline{Path: "hardware", Api: "videotoolbox"}, Signature: sig}, "windows", true, 0},
		{"a failed run on any pipeline", &cv1.TestResponse{Pipeline: &cv1.Pipeline{Path: "hardware", Api: "videotoolbox"}, Signature: sig, Tries: tries}, "windows", false, 0},
		{"a speed past 1,000", &cv1.TestResponse{Passed: true, Pipeline: qsv, Speed: 5000, Signature: sig, Device: strings.Repeat("d", 300)}, "windows", false, 1000},
		{"a speed under 0.01", &cv1.TestResponse{Passed: true, Pipeline: qsv, Speed: 0, Signature: sig}, "windows", false, 0.01},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := proofOf(tc.res, tc.os)
			if (err != nil) != tc.refused {
				t.Fatalf("refused %v, want %v", err, tc.refused)
			}
			if p.Speed != tc.speed || len(p.Device) > 128 || len(p.Tries) > 8 || slices.ContainsFunc(p.Tries, func(tr converting.Try) bool { return len(tr.Err) > 2<<10 }) {
				t.Errorf("speed %v, a device of %d bytes and %d tries, want %v within the caps", p.Speed, len(p.Device), len(p.Tries), tc.speed)
			}
		})
	}

	in, out := ID{UUID: storage.NewUUID()}, ID{UUID: storage.NewUUID()}
	progress := func(id ID, us int64) *cv1.Progress {
		return &cv1.Progress{ConversionId: id.Bytes(), Attempt: 1, OutTimeUs: us, Speed: 5000}
	}
	f := factsOf(&cv1.Facts{
		Signature: "short", Devices: make([]*cv1.Device, 10), Scratch: &cv1.Volume{Free: 10, Total: 5},
		Progress: []*cv1.Progress{progress(in, int64(3*time.Hour/time.Microsecond)), progress(out, 1)},
	}, 1, map[ID]time.Duration{in: time.Hour})
	if f.Signature != "" || len(f.Devices) != 8 || f.Scratch != nil {
		t.Errorf("signature %q, %d devices, scratch %v, want none, 8 and unknown", f.Signature, len(f.Devices), f.Scratch)
	}
	if p, ok := f.Progress[in]; len(f.Progress) != 1 || !ok || p.Out != 90*time.Minute || p.Speed != 1000 {
		t.Errorf("progress %+v, want only the intent's job, at 1.5 times its length and a speed of 1,000", f.Progress)
	}
}
