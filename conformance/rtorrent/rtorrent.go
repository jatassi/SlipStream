// Package rtorrent is the conformance suite for rTorrent: it drives
// SlipStream's rTorrent client against fake-rtorrent on every PR and
// against a real rTorrent behind nginx in the online job, so a fake that
// drifts from the real one fails.
package rtorrent

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/torrentfile"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/clients/rtorrent"
)

type Target struct {
	// Address is the XML-RPC address, as an Admin types it.
	Address            string
	Username, Password string
	HTTP               *http.Client
	// Folder is a folder the target saves to, as the target names it.
	Folder string
	Log    *slog.Logger
}

// Run drives the rTorrent client through every call SlipStream makes and
// asserts what holds on both sides: Test reads a supported version with
// no warnings, a paused add shows on a later poll in the folder with the
// label, a second add of it changes nothing, its one file is listed, it
// starts and stops, and a remove drops it. A target with a password also
// gets a wrong one refused.
func Run(t *testing.T, tgt Target) {
	t.Helper()
	label, err := clients.ParseLabel(clients.DefaultLabel, clients.RTorrent)
	if err != nil {
		t.Fatal(err)
	}
	r := run{t: t, tgt: tgt, label: label}
	r.c = r.open(tgt.Password)
	r.signIn()
	meta, hash, name := torrentfile.New(t)
	r.hash = hash
	t.Cleanup(func() { _ = r.open(tgt.Password).Remove(context.WithoutCancel(t.Context()), hash, true) })
	r.add(meta, name)
	r.act()
	r.remove()
	if tgt.Password != "" {
		_, err := r.open(tgt.Password+"-wrong").Test(t.Context(), label)
		if p, ok := errors.AsType[*clients.Problem](err); !ok || p.Cause != clients.WrongCredentials {
			t.Errorf("Test with a wrong password = %v, want WrongCredentials", err)
		}
	}
}

type run struct {
	t     *testing.T
	tgt   Target
	label clients.Label
	c     clients.Client
	hash  bencode.InfoHash
}

func (r run) open(password string) clients.Client {
	a, err := clients.ParseAddress(r.tgt.Address, clients.RTorrent)
	if err != nil {
		r.t.Fatal(err)
	}
	return rtorrent.Open(clients.Session{
		HTTP: r.tgt.HTTP, Address: a, Username: r.tgt.Username, Secrets: clients.Secrets{Password: password},
		Call: time.Minute, Log: r.tgt.Log,
	})
}

func (r run) signIn() {
	ctx := r.t.Context()
	tested, err := r.c.Test(ctx, r.label)
	if err != nil {
		r.t.Fatalf("Test: %v", err)
	}
	if tested.Version.Less(clients.RTorrent.Facts().Floor) || len(tested.Warnings) != 0 {
		r.t.Errorf("Test = %+v, want a supported version and no warnings", tested)
	}
	if v, err := r.c.Version(ctx); err != nil || v != tested.Version {
		r.t.Errorf("Version = %s, %v, want %s", v, err, tested.Version)
	}
	r.t.Logf("%s runs rTorrent %s", r.tgt.Address, tested.Version)
	if folder, err := r.c.DefaultFolder(ctx, r.label); err != nil || folder == "" {
		r.t.Errorf("DefaultFolder = %q, %v, want directory.default", folder, err)
	}
}

func (r run) poll() (clients.Status, bool) {
	got, err := r.c.Poll(r.t.Context(), []bencode.InfoHash{r.hash})
	if err != nil {
		r.t.Fatalf("Poll: %v", err)
	}
	s, ok := got[r.hash]
	return s, ok
}

// listed polls until the target lists the torrent, since rTorrent loads
// an add after it responds.
func (r run) listed() (clients.Status, bool) {
	for range 40 {
		if s, ok := r.poll(); ok {
			return s, true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return clients.Status{}, false
}

func (r run) add(meta []byte, name string) {
	ctx := r.t.Context()
	if err := r.c.Add(ctx, clients.Add{Metainfo: meta, Hash: r.hash, Folder: r.tgt.Folder, Label: r.label, Paused: true}); err != nil {
		r.t.Fatalf("Add: %v", err)
	}
	s, ok := r.listed()
	if !ok {
		r.t.Fatal("the target never listed the added torrent")
	}
	if err := r.c.Add(ctx, clients.Add{Metainfo: meta, Hash: r.hash, Label: r.label}); err != nil {
		r.t.Errorf("adding the torrent again: %v, want the duplicate ignored", err)
	}
	if s.Name != name || s.Label != r.label.String() || s.Folder != r.tgt.Folder || s.Run != clients.Paused || s.Finished || s.Size != torrentfile.Size || s.Done != 0 {
		r.t.Errorf("after a paused add: %+v, want %s paused in %s labeled %s with nothing done", s, name, r.tgt.Folder, r.label)
	}
	if again, _ := r.poll(); again.Folder != r.tgt.Folder || again.Run != clients.Paused {
		r.t.Errorf("after adding it again: %+v, want it unchanged", again)
	}
	files, err := r.c.Files(ctx, r.hash)
	if err != nil || !slices.Equal(files, []clients.File{{Index: 0, Path: name, Size: torrentfile.Size}}) {
		r.t.Errorf("Files = %+v, %v, want the one file", files, err)
	}
}

func (r run) act() {
	for _, a := range []clients.Act{clients.Resume, clients.Recheck, clients.Announce} {
		if err := r.c.Act(r.t.Context(), a, []bencode.InfoHash{r.hash}); err != nil {
			r.t.Errorf("Act(%d): %v", a, err)
		}
	}
	if s, _ := r.poll(); s.Run != clients.Started || s.Finished {
		r.t.Errorf("after Resume: %+v, want started and not finished", s)
	}
	if err := r.c.Act(r.t.Context(), clients.Pause, []bencode.InfoHash{r.hash}); err != nil {
		r.t.Errorf("Act(Pause): %v", err)
	}
	if s, _ := r.poll(); s.Run != clients.Paused {
		r.t.Errorf("after Pause: %+v, want paused", s)
	}
}

func (r run) remove() {
	if err := r.c.Remove(r.t.Context(), r.hash, true); err != nil {
		r.t.Errorf("Remove: %v", err)
	}
	if s, ok := r.poll(); ok {
		r.t.Errorf("after Remove the target lists %+v", s)
	}
	if err := r.c.Remove(r.t.Context(), r.hash, true); err != nil {
		r.t.Errorf("removing a torrent the target no longer holds: %v", err)
	}
}
