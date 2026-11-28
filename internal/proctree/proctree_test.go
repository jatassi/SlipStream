package proctree_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/proctree"
)

// asChild is the argument that makes this test binary play a process of
// the tree: a parent that starts a beating grandchild and waits.
const asChild = "proctree-test-child"

func TestMain(m *testing.M) {
	if len(os.Args) > 3 && os.Args[1] == asChild {
		// syscall.Exit skips the coverage hook of a -cover build.
		syscall.Exit(play(os.Args[2], os.Args[3]))
	}
	os.Exit(m.Run())
}

func play(role, beat string) int {
	switch role {
	case "parent":
		grandchild := exec.CommandContext(context.Background(), os.Args[0], asChild, "beat", beat)
		if grandchild.Start() != nil {
			return 9
		}
		time.Sleep(time.Hour)
	case "beat":
		f, err := os.OpenFile(beat, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return 9
		}
		for {
			_, _ = f.WriteString(".")
			time.Sleep(20 * time.Millisecond)
		}
	}
	return 0
}

func TestAKilledTreeLeavesNoChild(t *testing.T) {
	for _, c := range []struct {
		name string
		kill func(end func(), cancel context.CancelFunc)
	}{
		{"end", func(end func(), _ context.CancelFunc) { end() }},
		{"the context ending", func(_ func(), cancel context.CancelFunc) { cancel() }},
	} {
		t.Run(c.name, func(t *testing.T) {
			beat := filepath.Join(t.TempDir(), "beat")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], asChild, "parent", beat)
			end, err := proctree.Start(cmd)
			if err != nil {
				t.Fatal(err)
			}
			defer end()
			for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(20 * time.Millisecond) {
				if b, _ := os.ReadFile(beat); len(b) > 0 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("the grandchild never beat")
				}
			}
			c.kill(end, cancel)
			_ = cmd.Wait()
			last, _ := os.ReadFile(beat)
			time.Sleep(300 * time.Millisecond)
			if now, _ := os.ReadFile(beat); len(now) != len(last) {
				t.Fatalf("the grandchild still runs: %d beats, then %d", len(last), len(now))
			}
		})
	}
}
