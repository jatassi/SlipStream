package logs

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/system"
)

func ringLog(t *testing.T, lines int) *Log {
	t.Helper()
	s := DefaultSettings()
	s.ViewerLines = lines
	l, err := OpenFile(t.TempDir(), io.Discard, s)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l
}

func read(t *testing.T, l *Log, after uint64) system.LogBatch {
	t.Helper()
	b, err := l.Read(t.Context(), after, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// shown is a batch as one line per record, with seqs relative to base.
func shown(l *Log, b system.LogBatch) []string {
	var out []string
	if b.Reset != nil {
		out = append(out, fmt.Sprintf("reset %s after %d", b.Reset.Reason, b.Reset.After-l.ring.base))
	}
	for _, r := range b.Records {
		out = append(out, fmt.Sprintf("%d %s %s %s %s", r.Seq-l.ring.base, r.Level, r.Subsystem, r.Message, r.Attrs))
	}
	return out
}

func TestReadGivesTheRingThenWhatComesNext(t *testing.T) {
	l := ringLog(t, 100)
	grabs := l.With("subsystem", "grabs")
	grabs.Debug("hidden at info")
	grabs.Info("grabbed", "work", 7)
	l.WithGroup("job").Warn("late", "subsystem", "nested")
	l.Error("failed", "subsystem", "http", "err", "boom")

	first := read(t, l, 0)
	want := []string{
		`1 info grabs grabbed {"work":7}`,
		`2 warn  late {"job":{"subsystem":"nested"}}`,
		`3 error http failed {"err":"boom"}`,
	}
	if got := shown(l, first); !slices.Equal(got, want) {
		t.Fatalf("first read = %q, want %q", got, want)
	}
	last := first.Records[2].Seq
	if got := shown(l, read(t, l, last)); got != nil {
		t.Fatalf("a read with nothing new = %q, want an empty batch", got)
	}
	done := make(chan system.LogBatch)
	go func() {
		b, _ := l.Read(t.Context(), last, time.Minute)
		done <- b
	}()
	time.Sleep(10 * time.Millisecond)
	l.Info("later")
	if got, want := shown(l, <-done), []string{"4 info  later {}"}; !slices.Equal(got, want) {
		t.Fatalf("a waiting read = %q, want %q", got, want)
	}
}

func TestALappedReaderStartsOverAtTheOldestRecord(t *testing.T) {
	l := ringLog(t, 100)
	for i := range 150 {
		l.Info("record", "i", i)
	}
	got := shown(l, read(t, l, l.ring.base+1))
	if len(got) != 101 || got[0] != "reset lapped after 50" || got[1] != `51 info  record {"i":50}` || got[100] != `150 info  record {"i":149}` {
		t.Fatalf("lapped read = %d lines from %q to %q", len(got), got[:2], got[len(got)-1])
	}
}

func TestASeqFromAnotherProcessStartsOver(t *testing.T) {
	l := ringLog(t, 100)
	l.Info("first")
	for _, after := range []uint64{l.ring.base - 5_000_000, l.ring.base + 9} {
		b := read(t, l, after)
		if got, want := shown(l, b), []string{"reset restarted after 0", "1 info  first {}"}; !slices.Equal(got, want) || !b.Reset.StartedAt.Equal(l.ring.opened) {
			t.Fatalf("read after %d = %q started %s, want %q started %s", after, got, b.Reset.StartedAt, want, l.ring.opened)
		}
	}
}

func TestOneReadReturnsAtMostAThousandRecords(t *testing.T) {
	l := ringLog(t, 2000)
	for i := range 1500 {
		l.Info("record", "i", i)
	}
	first := read(t, l, 0)
	second := read(t, l, first.Records[len(first.Records)-1].Seq)
	if len(first.Records) != 1000 || len(second.Records) != 500 || second.Records[0].Seq != l.ring.base+1001 {
		t.Fatalf("reads of %d and %d records, the second from %d", len(first.Records), len(second.Records), second.Records[0].Seq-l.ring.base)
	}
}

func TestApplyResizesTheRing(t *testing.T) {
	l := ringLog(t, 100)
	for i := range 20 {
		l.Info("record", "i", i)
	}
	s := DefaultSettings()
	s.ViewerLines = 5
	if err := l.Apply(s); err != nil {
		t.Fatal(err)
	}
	l.Info("after")
	got := shown(l, read(t, l, 0))
	if want := []string{`17 info  record {"i":16}`, `18 info  record {"i":17}`, `19 info  record {"i":18}`, `20 info  record {"i":19}`, `21 info  after {}`}; !slices.Equal(got, want) {
		t.Fatalf("after a shrink = %q, want %q", got, want)
	}
}

func TestEndReadsEndsEveryRead(t *testing.T) {
	l := ringLog(t, 100)
	waiting := make(chan error)
	go func() {
		_, err := l.Read(t.Context(), l.ring.base, time.Minute)
		waiting <- err
	}()
	time.Sleep(10 * time.Millisecond)
	l.EndReads()
	l.Info("still logged")
	_, later := l.Read(t.Context(), 0, time.Minute)
	if err := <-waiting; !errors.Is(err, ErrClosed) || !errors.Is(later, ErrClosed) {
		t.Fatalf("reads = %v and %v, want %v", err, later, ErrClosed)
	}
}

func TestArchiveZipsEveryFileOfTheFolder(t *testing.T) {
	dir := t.TempDir()
	s := DefaultSettings()
	s.MaxSizeMB, s.MaxAgeDays, s.Compress = 1, 0, false
	l, err := Open(dir, io.Discard, s)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	pad := strings.Repeat("x", 10<<10)
	for range 120 {
		l.Info("fill", "pad", pad)
	}
	waitForBackups(t, dir, 1)
	gz := []byte{0x1f, 0x8b, 1, 2, 3}
	if err := os.WriteFile(filepath.Join(dir, "slipstream-2000-01-01T00-00-00.000.log.gz"), gz, 0o600); err != nil {
		t.Fatal(err)
	}
	f, size, err := l.Archive(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	zr, err := zip.NewReader(f, size)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]string{File, "slipstream-2000-01-01T00-00-00.000.log.gz"}, backups(t, dir)[1])
	var names []string
	for _, e := range zr.File {
		names = append(names, e.Name)
	}
	if !slices.Equal(names, want) {
		t.Fatalf("zip entries = %q, want %q", names, want)
	}
	current := entry(t, zr.File[0])
	onDisk, err := os.ReadFile(filepath.Join(dir, File))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current, onDisk) || zr.File[1].Method != zip.Store || !bytes.Equal(entry(t, zr.File[1]), gz) {
		t.Fatalf("slipstream.log entry of %d bytes, file of %d; the .gz entry method %d", len(current), len(onDisk), zr.File[1].Method)
	}
}

func entry(t *testing.T, f *zip.File) []byte {
	t.Helper()
	r, err := f.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
