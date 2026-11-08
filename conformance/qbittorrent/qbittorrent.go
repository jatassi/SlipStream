// Package qbittorrent is the conformance suite for qBittorrent: it drives
// SlipStream's qBittorrent client against fake-qbittorrent on every PR
// and against a real qBittorrent in the online job, so a fake that drifts
// from the real WebUI fails.
package qbittorrent

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
	"github.com/jatassi/slipstream/internal/torrent/clients/qbittorrent"
)

type Target struct {
	// Address is the WebUI address, as an Admin types it.
	Address            string
	Username, Password string
	// APIKey signs in in place of the username and password, "" for none.
	APIKey string
	HTTP   *http.Client
	// Folder is a folder the target saves to, as the target names it.
	Folder string
	Log    *slog.Logger
}

// Run drives the qBittorrent client through every call SlipStream makes
// and asserts what holds on both sides: Test reads a supported version
// with no warnings, a paused add lands in the folder in the category, a
// second add of it changes nothing, its one file is listed, it starts and
// stops, and a remove with data drops it. A wrong password, or a wrong
// API key for a target signed in with one, is refused. One wrong
// password never reaches qBittorrent's default ban of five.
func Run(t *testing.T, tgt Target) {
	t.Helper()
	label, err := clients.ParseLabel(clients.DefaultLabel, clients.QBittorrent)
	if err != nil {
		t.Fatal(err)
	}
	r := run{t: t, tgt: tgt, label: label}
	r.c = r.open(tgt.Password, tgt.APIKey)
	r.signIn()
	meta, hash, name := torrentfile.New(t)
	r.hash = hash
	t.Cleanup(func() { _ = r.open(tgt.Password, tgt.APIKey).Remove(context.WithoutCancel(t.Context()), hash, true) })
	r.add(meta, name)
	r.act()
	r.remove()
	r.refused()
	if err := r.c.Close(t.Context()); err != nil {
		t.Errorf("Close: %v", err)
	}
}

type run struct {
	t     *testing.T
	tgt   Target
	label clients.Label
	c     clients.Client
	hash  bencode.InfoHash
}

func (r run) open(password, apiKey string) clients.Client {
	a, err := clients.ParseAddress(r.tgt.Address, clients.QBittorrent)
	if err != nil {
		r.t.Fatal(err)
	}
	return qbittorrent.Open(clients.Session{
		HTTP: r.tgt.HTTP, Address: a, Username: r.tgt.Username, Secrets: clients.Secrets{Password: password, APIKey: apiKey},
		Call: time.Minute, Log: r.tgt.Log,
	})
}

func (r run) signIn() {
	ctx := r.t.Context()
	tested, err := r.c.Test(ctx, r.label)
	if err != nil {
		r.t.Fatalf("Test: %v", err)
	}
	if tested.Version.Less(clients.QBittorrent.Facts().Floor) || len(tested.Warnings) != 0 {
		r.t.Errorf("Test = %+v, want a supported version and no warnings", tested)
	}
	if v, err := r.c.Version(ctx); err != nil || v != tested.Version {
		r.t.Errorf("Version = %s, %v, want %s", v, err, tested.Version)
	}
	r.t.Logf("%s runs qBittorrent %s", r.tgt.Address, tested.Version)
	if folder, err := r.c.DefaultFolder(ctx, r.label); err != nil || folder == "" {
		r.t.Errorf("DefaultFolder = %q, %v, want the save path", folder, err)
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

func (r run) add(meta []byte, name string) {
	ctx := r.t.Context()
	if err := r.c.Add(ctx, clients.Add{Metainfo: meta, Hash: r.hash, Folder: r.tgt.Folder, Label: r.label, Paused: true}); err != nil {
		r.t.Fatalf("Add: %v", err)
	}
	if err := r.c.Add(ctx, clients.Add{Metainfo: meta, Hash: r.hash, Label: r.label}); err != nil {
		r.t.Errorf("adding the torrent again: %v, want the duplicate ignored", err)
	}
	s, ok := r.poll()
	if !ok || s.Name != name || s.Label != r.label.String() || s.Folder != r.tgt.Folder || s.Run != clients.Paused || s.Finished || s.Size != torrentfile.Size || s.Done != 0 {
		r.t.Errorf("after a paused add: %+v, %v, want %s paused in %s in category %s with nothing done", s, ok, name, r.tgt.Folder, r.label)
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
}

func (r run) refused() {
	if r.tgt.APIKey != "" {
		_, err := r.open("", "qbt_"+r.tgt.APIKey).Test(r.t.Context(), r.label)
		if p, ok := errors.AsType[*clients.Problem](err); !ok || p.Cause != clients.WrongAPIKey {
			r.t.Errorf("Test with a wrong API key = %v, want WrongAPIKey", err)
		}
	}
	if r.tgt.Password != "" {
		_, err := r.open(r.tgt.Password+"-wrong", "").Test(r.t.Context(), r.label)
		if p, ok := errors.AsType[*clients.Problem](err); !ok || p.Cause != clients.WrongCredentials {
			r.t.Errorf("Test with a wrong password = %v, want WrongCredentials", err)
		}
	}
}
