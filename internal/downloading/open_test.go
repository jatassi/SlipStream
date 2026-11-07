package downloading_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

func (f file) ref() source.Torrent { return source.Torrent{Hash: f.hash, File: 0} }

// absent is the reason Open gives for f, failing t when Open answers a
// path or an error that isn't *source.Absent.
func (c *client) absent(t *testing.T, f file) *source.Absent {
	t.Helper()
	path, err := c.Open(f.ref())
	a, ok := errors.AsType[*source.Absent](err)
	if !ok {
		t.Fatalf("Open of %s = %q, %v; want *source.Absent", f.name, path, err)
	}
	return a
}

func TestOpenAnswersTheFileOfACompleteTorrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		c := n.start()
		if got := c.absent(t, f); got.Reason != source.NotReady {
			t.Fatalf("Open before the client's first intent = %v, want not ready", got)
		}
		c.send(live(f.started(id, 1)))

		path, err := c.Open(f.ref())
		if want := filepath.Join(n.dir, "Movies", f.hash.String(), f.name); err != nil || path != want {
			t.Fatalf("Open = %q, %v; want %q", path, err, want)
		}
	})
}

func TestOpenOfATorrentTheClientDoesntHoldIsGone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		kept, other := newFile(t, "Orchard.mkv"), newFile(t, "Meadow.mkv")
		n.world.Payload(kept.hash, kept.payload)
		id := n.hold(kept)
		c := n.start()
		c.send(live(kept.started(id, 1)))
		if got := c.absent(t, other); got.Reason != source.Gone {
			t.Errorf("Open of a torrent the intent never listed = %v, want gone", got)
		}

		var atRemove error
		c.engine.onRemove(func() { _, atRemove = c.Open(kept.ref()) })
		discarding := live()
		discarding.Intent.Discards = []*dlv1.Discard{{Id: id.Bytes(), InfoHash: kept.hash[:]}}
		c.send(discarding)
		if a, ok := errors.AsType[*source.Absent](atRemove); !ok || a.Reason != source.Gone {
			t.Errorf("Open of a discarded torrent as the engine removes it = %v, want gone", atRemove)
		}
		if got := c.absent(t, kept); got.Reason != source.Gone {
			t.Errorf("Open of a discarded torrent = %v, want gone", got)
		}
	})
}

func TestOpenOfATorrentThatFailedAfterCompletingIsFailed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		c := n.start()
		c.send(live(f.started(id, 1)))
		if _, err := c.Open(f.ref()); err != nil {
			t.Fatalf("Open of the complete torrent = %v", err)
		}

		n.world.Fail(f.hash, &engine.Error{Category: engine.CategorySystem, Code: 5, Text: "input/output error"})
		synctest.Wait()
		if got, want := c.absent(t, f), (source.Absent{Reason: source.Failed, Detail: "input/output error"}); *got != want {
			t.Errorf("Open once the engine failed the torrent = %+v, want %+v", *got, want)
		}
	})
}

func TestOpenWhileTheTorrentIsCheckedIsNotReady(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		c := n.start()
		c.send(live(f.started(id, 1)))

		c.engine.holdChecks()
		if r := c.act(t, dlv1.Act_ACT_RECHECK, id); r.GetRefused() != dlv1.Refusal_REFUSAL_UNSPECIFIED {
			t.Fatalf("Recheck = %v, want it done", r)
		}
		if got := c.absent(t, f); got.Reason != source.NotReady {
			t.Fatalf("Open while the check runs = %v, want not ready", got)
		}

		if err := c.engine.finishChecks(); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if _, err := c.Open(f.ref()); err != nil {
			t.Errorf("Open once the check found every piece = %v, want the file", err)
		}
	})
}

func TestOpenOfATorrentACheckFoundIncompleteIsNotReady(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		c := n.start()
		c.send(live(f.started(id, 1)))
		path, err := c.Open(f.ref())
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
		n.world.Transfer(f.hash, 0, 0)

		if r := c.act(t, dlv1.Act_ACT_RECHECK, id); r.GetRefused() != dlv1.Refusal_REFUSAL_UNSPECIFIED {
			t.Fatalf("Recheck = %v, want it done", r)
		}
		if got := c.absent(t, f); got.Reason != source.NotReady {
			t.Errorf("Open of a torrent the check found missing pieces = %v, want not ready", got)
		}
	})
}

func TestOpenOfAFileThatWontOpenIsNotReadyWithTheOpenError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		c := n.start()
		c.send(live(f.started(id, 1)))
		path, err := c.Open(f.ref())
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}

		got := c.absent(t, f)
		if _, statErr := os.Open(path); got.Reason != source.NotReady || !strings.Contains(got.Detail, statErr.Error()) {
			t.Errorf("Open of a file removed by hand = %+v, want not ready with %q", *got, statErr)
		}
	})
}
