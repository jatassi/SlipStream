// Package clientcases is the cases every Download Client conformance
// suite runs alike: a complete add over files already on disk, the last
// announce a Move's guard reads, a paused add that reads paused until
// it is listed and checked, and a resume and a pause the target lists.
//
// A client may read a run it asked for before the target lists it, so a
// case that resumes or pauses also polls a witness: a second client of
// the same target that never calls Act or Add, whose reads are the
// target's own listing.
package clientcases

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/torrentfile"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// Folder is a folder the target saves to, as the target names it and as
// this machine sees it.
type Folder struct {
	Client, Node string
}

// poll is how often a case reads the target while it waits.
const poll = time.Second

// pauseWait bounds each wait for a pause, a resume or an add to read
// settled, and pauseEvery is how often it reads: Transmission 4.1.4 read
// stopped up to 485 ms after the call, on a tick of about 0.5 s,
// qBittorrent 4.6.7 listed an add 4 ms after it answered, and qBittorrent
// 4.6.7 and 5.3.0 listed a start up to 1523 ms and a stop up to 1476 ms
// after the call, at their next refresh, which came every 1.5 s.
const (
	pauseWait  = 2 * time.Second
	pauseEvery = 100 * time.Millisecond
)

// CompleteAdd adds whole over its payload written whole in folder, and a
// second torrent over its payload cut short, both paused and complete,
// and asserts that once their checks end the client holds whole finished
// and the short one not. It removes the short one and leaves whole
// paused for LastAnnounce.
func CompleteAdd(t *testing.T, c clients.Client, label clients.Label, folder Folder, whole torrentfile.Seeded) {
	t.Helper()
	short := torrentfile.Seed(t, "http://tracker.invalid/announce")
	t.Cleanup(func() { _ = c.Remove(context.WithoutCancel(t.Context()), short.Hash, true) })
	for _, p := range []struct {
		s torrentfile.Seeded
		n int
	}{{whole, len(whole.Data)}, {short, len(short.Data) / 2}} {
		path := filepath.Join(folder.Node, p.s.Name)
		if err := os.MkdirAll(folder.Node, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, p.s.Data[:p.n], 0o600); err != nil {
			t.Fatal(err)
		}
		add := clients.Add{Metainfo: p.s.Metainfo, Hash: p.s.Hash, Folder: folder.Client, Label: label, Paused: true, Complete: true}
		if err := c.Add(t.Context(), add); err != nil {
			t.Fatalf("Add of a complete torrent: %v", err)
		}
	}
	got := checked(t, c, whole.Hash, short.Hash)
	if s := got[whole.Hash]; !s.Finished || s.Done != s.Size || s.Run != clients.Paused {
		t.Errorf("after a complete add over the whole file: %+v, want it finished and paused", s)
	}
	if s := got[short.Hash]; s.Finished || s.Done >= s.Size {
		t.Errorf("after a complete add over a short file: %+v, want it unfinished", s)
	}
	if err := c.Remove(t.Context(), short.Hash, true); err != nil {
		t.Errorf("Remove: %v", err)
	}
}

// checked polls until the client lists each of hashes with no check
// running, and answers their statuses.
func checked(t *testing.T, c clients.Client, hashes ...bencode.InfoHash) map[bencode.InfoHash]clients.Status {
	t.Helper()
	deadline := time.Now().Add(time.Minute)
	for {
		got, err := c.Poll(t.Context(), hashes)
		if err != nil {
			t.Fatalf("Poll: %v", err)
		}
		done := len(got) == len(hashes)
		for _, s := range got {
			done = done && s.Activity != clients.Checking
		}
		if done {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatalf("after a minute the client lists %+v, want each of %d torrents listed and checked", got, len(hashes))
		}
		time.Sleep(poll)
	}
}

// LastAnnounce starts h, which announces to a tracker that answers it
// every interval, and asserts that Announces reads an answer at or after
// the start with the next announce after it within two intervals. It
// pauses h again as Pause does.
func LastAnnounce(t *testing.T, c, witness clients.Client, h bencode.InfoHash, interval time.Duration) {
	t.Helper()
	start := time.Now().Truncate(time.Second)
	if err := c.Act(t.Context(), clients.Resume, []bencode.InfoHash{h}); err != nil {
		t.Fatalf("Act(Resume): %v", err)
	}
	deadline := start.Add(2*interval + 5*poll)
	for {
		got, err := c.Announces(t.Context(), []bencode.InfoHash{h})
		if err != nil {
			t.Fatalf("Announces: %v", err)
		}
		if announcedSince(got[h], start) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Announces = %+v two intervals after the start, want an answer since %s and the next after it", got[h], start)
		}
		time.Sleep(poll)
	}
	Pause(t, c, witness, h)
}

// Pause pauses h and polls until the client reads it paused, which a
// client may do on its next tick after it answered, and until the witness
// does with no check running.
func Pause(t *testing.T, c, witness clients.Client, h bencode.InfoHash) {
	t.Helper()
	if err := c.Act(t.Context(), clients.Pause, []bencode.InfoHash{h}); err != nil {
		t.Errorf("Act(Pause): %v", err)
		return
	}
	deadline := time.Now().Add(pauseWait)
	for {
		got, err := c.Poll(t.Context(), []bencode.InfoHash{h})
		if err != nil {
			t.Fatalf("Poll: %v", err)
		}
		if got[h].Run == clients.Paused {
			break
		}
		if time.Now().After(deadline) {
			t.Errorf("%s after Pause: %+v, want paused", pauseWait, got[h])
			return
		}
		time.Sleep(pauseEvery)
	}
	witnessed(t, witness, h, clients.Paused)
}

// Resume resumes h and polls until the client reads it started with no
// check running, which a client may list a tick after it answered, and
// until the witness does, and answers the client's status. Each of the
// client's reads must say started, checking or not.
func Resume(t *testing.T, c, witness clients.Client, h bencode.InfoHash) clients.Status {
	t.Helper()
	if err := c.Act(t.Context(), clients.Resume, []bencode.InfoHash{h}); err != nil {
		t.Errorf("Act(Resume): %v", err)
		return clients.Status{}
	}
	deadline := time.Now().Add(pauseWait)
	for {
		got, err := c.Poll(t.Context(), []bencode.InfoHash{h})
		if err != nil {
			t.Fatalf("Poll: %v", err)
		}
		s := got[h]
		if s.Run != clients.Started {
			t.Errorf("after Resume: %+v, want it started", s)
			return s
		}
		if s.Activity != clients.Checking || time.Now().After(deadline) {
			witnessed(t, witness, h, clients.Started)
			return s
		}
		time.Sleep(pauseEvery)
	}
}

// witnessed polls until witness reads h at run with no check running.
func witnessed(t *testing.T, witness clients.Client, h bencode.InfoHash, run clients.Run) {
	t.Helper()
	deadline := time.Now().Add(pauseWait)
	for {
		got, err := witness.Poll(t.Context(), []bencode.InfoHash{h})
		if err != nil {
			t.Fatalf("the witness's Poll: %v", err)
		}
		s, ok := got[h]
		if ok && s.Run == run && s.Activity != clients.Checking {
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("after %s the witness reads %+v, %v, want it listed with run %d and no check running", pauseWait, s, ok, run)
			return
		}
		time.Sleep(pauseEvery)
	}
}

// PausedAdd polls until the client lists h, added paused, with no check
// running, which a client may do on its next tick after it answered the
// add, and answers its status. Each read must say paused, checking or
// not.
func PausedAdd(t *testing.T, c clients.Client, h bencode.InfoHash) (clients.Status, bool) {
	t.Helper()
	deadline := time.Now().Add(pauseWait)
	for {
		got, err := c.Poll(t.Context(), []bencode.InfoHash{h})
		if err != nil {
			t.Fatalf("Poll: %v", err)
		}
		s, ok := got[h]
		if ok && s.Run != clients.Paused {
			t.Errorf("after a paused add: %+v, want it paused", s)
			return s, true
		}
		if ok && s.Activity != clients.Checking {
			return s, true
		}
		if time.Now().After(deadline) {
			return s, ok
		}
		time.Sleep(pauseEvery)
	}
}

func announcedSince(lines []clients.TrackerLine, start time.Time) bool {
	for _, ln := range lines {
		if !ln.Last.Before(start) && ln.Next.After(ln.Last) {
			return true
		}
	}
	return false
}
