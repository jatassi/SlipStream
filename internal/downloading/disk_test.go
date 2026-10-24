package downloading_test

import (
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/downloading"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

const gb = 1 << 30

func sampledAt(f *dlv1.Facts) time.Time { return time.UnixMilli(f.GetDownloads().GetSampledAtUnixMs()) }

func TestTheClientSamplesItsDownloadsVolumeAtStoragesCadence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		n.free.Store(500 * gb)
		c := n.start()
		in := live()
		in.Intent.SampleEverySeconds = 300
		c.send(in)
		first := c.h.lastFacts()
		if v := first.GetDownloads(); v.GetMount() != "/data" || v.GetFree() != 500*gb || v.GetTotal() != volumeTotal {
			t.Fatalf("the facts carry %v, want /data with 500 GB free sampled as the intent came", v)
		}

		n.free.Store(400 * gb)
		time.Sleep(5 * time.Minute)
		synctest.Wait()
		next := c.h.lastFacts()
		if next.GetDownloads().GetFree() != 400*gb || sampledAt(next).Sub(sampledAt(first)) != 5*time.Minute {
			t.Fatalf("five minutes on the facts carry %v, want a fresh sample of 400 GB", next.GetDownloads())
		}
		if downloading.Def.Notable(first, next) {
			t.Error("a fresh volume sample marks the Node changed, want only the client itself to")
		}

		faster := live()
		faster.Intent.SampleEverySeconds = 60
		n.free.Store(300 * gb)
		c.send(faster)
		if got := c.h.lastFacts().GetDownloads().GetFree(); got != 300*gb {
			t.Errorf("a new cadence sampled %d bytes free, want a sample at once", got)
		}
	})
}

func TestAClientWhoseDownloadsFolderIsMissingStillSamplesItsVolume(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		n.dir = filepath.Join(n.dir, "downloads")
		n.free.Store(500 * gb)
		c := n.start()
		in := live()
		in.Intent.SampleEverySeconds = 300
		c.send(in)
		if v := c.h.lastFacts().GetDownloads(); v.GetFree() != 500*gb {
			t.Fatalf("the facts carry %v before any torrent made the downloads folder, want its volume with 500 GB free", v)
		}
	})
}

func TestADiskFullDownloadResumesOnceAFreshSampleLiftsTheGuard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		n.free.Store(500 * gb)
		f := newFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		n.world.Transfer(f.hash, 1000, 0)
		id := n.hold(f)
		c := n.start()
		intent := func(hold bool) {
			in := live(f.started(id, 1))
			in.Intent.SampleEverySeconds, in.Intent.HoldDownloads = 300, hold
			c.send(in)
		}
		intent(false)
		time.Sleep(10 * time.Second)
		synctest.Wait()

		n.free.Store(0)
		n.world.DiskFull(true)
		synctest.Wait()
		if r := c.h.latest(t, id); r.GetHold() != dlv1.Hold_HOLD_DISK_FULL {
			t.Fatalf("a download that ran out of space reports %v, want paused for disk", r)
		}
		if got := c.h.lastFacts().GetDownloads().GetFree(); got != 0 {
			t.Fatalf("the facts after the disk filled carry %d bytes free, want a sample taken at once", got)
		}
		intent(false)
		if r := c.h.latest(t, id); r.GetHold() != dlv1.Hold_HOLD_DISK_FULL {
			t.Fatalf("with no more space than when it filled the download reports %v, want still paused for disk", r)
		}

		intent(true)
		n.world.DiskFull(false)
		n.free.Store(30 * gb)
		time.Sleep(5 * time.Minute)
		synctest.Wait()
		if r := c.h.latest(t, id); r.GetHold() != dlv1.Hold_HOLD_DISK_FULL {
			t.Fatalf("while the guard holds the download reports %v, want still paused for disk", r)
		}

		intent(false)
		if r := c.h.latest(t, id); r.GetHold() != dlv1.Hold_HOLD_UNSPECIFIED || r.GetEngine() != dlv1.EngineState_ENGINE_STATE_DOWNLOADING {
			t.Fatalf("once the guard lifted on a fresh sample the download reports %v, want downloading", r)
		}
		time.Sleep(time.Minute)
		synctest.Wait()
		if !c.h.latest(t, id).GetComplete() {
			t.Error("the download didn't finish after it resumed")
		}
	})
}
