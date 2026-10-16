package logs

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func readLines(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

func TestApplySetsTheLevel(t *testing.T) {
	dir := t.TempDir()
	l, err := OpenFile(dir, io.Discard, DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	l.Debug("hidden")
	debug := DefaultSettings()
	debug.Level = Debug
	if err := l.Apply(debug); err != nil {
		t.Fatal(err)
	}
	l.Debug("shown")
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	var msgs []string
	for _, line := range readLines(t, filepath.Join(dir, File)) {
		var r struct{ Msg string }
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatal(err)
		}
		msgs = append(msgs, r.Msg)
	}
	if want := []string{"shown"}; !slices.Equal(msgs, want) {
		t.Fatalf("records = %v, want %v", msgs, want)
	}
}

// TestARotationChangeLosesNoRecord swaps the rotating file under writers;
// run it with -race.
func TestARotationChangeLosesNoRecord(t *testing.T) {
	dir := t.TempDir()
	s := DefaultSettings()
	s.Compress = false
	l, err := Open(dir, io.Discard, s)
	if err != nil {
		t.Fatal(err)
	}
	const writers, each = 8, 500
	var wg sync.WaitGroup
	for w := range writers {
		wg.Go(func() {
			for i := range each {
				l.Info("record", "w", w, "i", i)
			}
		})
	}
	stop := make(chan struct{})
	swapped := make(chan int)
	go func() {
		n := 0
		for {
			select {
			case <-stop:
				swapped <- n
				return
			default:
			}
			s.MaxFiles = 1 + n%2
			if err := l.Apply(s); err != nil {
				t.Error(err)
			}
			n++
		}
	}()
	wg.Wait()
	close(stop)
	swaps := <-swapped
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	seen := map[[2]int]bool{}
	for _, line := range readLines(t, filepath.Join(dir, File)) {
		var r struct{ W, I int }
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		seen[[2]int{r.W, r.I}] = true
	}
	if len(seen) != writers*each || swaps == 0 {
		t.Fatalf("%d distinct records after %d swaps, want %d", len(seen), swaps, writers*each)
	}
}

func backups(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if e.Name() != File {
			out = append(out, e.Name())
		}
	}
	return out
}

func TestANewRotationAppliesFromTheNextRotation(t *testing.T) {
	dir := t.TempDir()
	s := DefaultSettings()
	s.MaxSizeMB, s.MaxFiles, s.Compress = 1, 5, false
	l, err := Open(dir, io.Discard, s)
	if err != nil {
		t.Fatal(err)
	}
	pad := strings.Repeat("x", 10<<10)
	fill := func(records int) {
		for range records {
			l.Info("fill", "pad", pad)
		}
	}
	fill(270)
	waitForBackups(t, dir, 2)
	s.MaxFiles = 1
	if err := l.Apply(s); err != nil {
		t.Fatal(err)
	}
	fill(180)
	waitForBackups(t, dir, 1)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
}

// waitForBackups waits for lumberjack's goroutine, which prunes backups
// after a rotation returns.
func waitForBackups(t *testing.T, dir string, n int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for len(backups(t, dir)) != n {
		if time.Now().After(deadline) {
			t.Fatalf("backups = %v, want %d", backups(t, dir), n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
