// Package mediabrowser is what fake-jellyfin and fake-emby share: their
// one API family over the shared Media Server core, with a Product per
// fake holding how each signs a call in, which routes it serves and how
// the core behaves under it. Bodies copy recorded run 2026-11-14 for
// Jellyfin and #71's probes for Emby.
package mediabrowser

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// Flavor is which product a fake plays.
type Flavor int

const (
	Jellyfin Flavor = iota
	Emby
)

// Product is one fake's product.
type Product struct {
	// Name is the fake's Spec name, the prefix of its cues and captures.
	Name   string
	Flavor Flavor
	// ProductName and TaskName are the product's own words in
	// /System/Info/Public and its scan task.
	ProductName string
	TaskName    string
	TaskID      string
	Behavior    mediaserver.Behavior
}

// Server is the one Jellyfin or Emby a test or the world plays.
type Server struct {
	Name    string
	Version string
	// ID is the server Id, 32 hex characters.
	ID string
	// Keys are the API keys it accepts.
	Keys  []string
	Users []User
	// MonitorDelay is the configured delay, the product's default when
	// zero.
	MonitorDelay time.Duration
	// Root is the folder the media dir is, as the server names it, such
	// as /media or D:\Media.
	Root      string
	Libraries []Library
}

// User is one server user. The first administrator owns the watch data a
// key's calls read when they name no user.
type User struct {
	Name     string
	Password string
	Admin    bool
}

// Library is one library. Type is its CollectionType, such as movies or
// tvshows, and Folders are under Root.
type Library struct {
	Name    string
	Type    string
	Folders []string
	// Scanned is a library scanned before the fake started.
	Scanned bool
}

// Captures are a Product's capture specs.
func Captures(name string) (reports, scans, items fake.CaptureSpec) {
	return fake.CaptureSpec{Name: name + "-reports", Kind: name + ".report"},
		fake.CaptureSpec{Name: name + "-scans", Kind: name + ".scan"},
		fake.CaptureSpec{Name: name + "-items", Kind: name + ".item"}
}

// Cues are a Product's cue specs: a playback, its stop, a restart and the
// server's own scheduled scan.
func Cues(name string) []fake.CueSpec {
	return []fake.CueSpec{{Name: name + "-play"}, {Name: name + "-stop"}, {Name: name + "-restart"}, {Name: name + "-scan"}}
}

// Report is a reports record's data: one POST /Library/Media/Updated.
type Report struct {
	Updates []ReportedPath `json:"updates"`
}

type ReportedPath struct {
	Path       string `json:"path"`
	UpdateType string `json:"updateType"`
}

// Scan is a scans record's data: the library refreshed, empty for every
// library.
type Scan struct {
	Library string `json:"library,omitempty"`
}

// ItemRecord is an items record's data.
type ItemRecord struct {
	Event   string   `json:"event"`
	ID      string   `json:"id"`
	Library string   `json:"library"`
	Files   []string `json:"files"`
}

type server struct {
	env     fake.Env
	p       Product
	srv     *Server
	core    *mediaserver.Server
	http    fake.Server
	reports *fake.Capture
	scans   *fake.Capture
	itemLog *fake.Capture

	mu      sync.Mutex
	keys    []string
	keyApps []keyApp
	tokens  map[string]session
	minted  int
}

// session is a user's access token and the DeviceId it signed in with.
type session struct {
	user   string
	device string
}

// Serve is the fake for p playing srv, nil for a world that names none.
func Serve(env fake.Env, p Product, srv *Server) (fake.Server, error) {
	s := &server{env: env, p: p, srv: srv, tokens: map[string]session{}}
	reports, scans, items := Captures(p.Name)
	var err error
	if s.reports, err = env.Capture(reports.Name); err != nil {
		return nil, err
	}
	if s.scans, err = env.Capture(scans.Name); err != nil {
		return nil, err
	}
	if srv != nil {
		if s.core, err = newCore(env, p, srv); err != nil {
			return nil, err
		}
		s.keys = append(s.keys, srv.Keys...)
		if s.itemLog, err = env.Capture(items.Name); err != nil {
			return nil, err
		}
		if err := s.cues(); err != nil {
			return nil, err
		}
	}
	s.http = fake.HTTP(env, s.routes())
	return s, nil
}

// id is a 32 hex id for seed, as both products show ids.
func id(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:16])
}

func libraryKey(name string) string { return id("library " + name) }

func userID(name string) string { return id("user " + name) }

func collectionID(name string) string { return id("collection " + name) }

func newCore(env fake.Env, p Product, srv *Server) (*mediaserver.Server, error) {
	root, err := link.ParsePath(srv.Root)
	if err != nil {
		return nil, err
	}
	libs := make([]mediaserver.Library, len(srv.Libraries))
	for i, l := range srv.Libraries {
		libs[i] = mediaserver.Library{Key: libraryKey(l.Name), Name: l.Name, Type: l.Type, Scanned: l.Scanned}
		for _, f := range l.Folders {
			fp, err := link.ParsePath(f)
			if err != nil {
				return nil, err
			}
			libs[i].Folders = append(libs[i].Folders, fp)
		}
	}
	b := p.Behavior
	if srv.MonitorDelay > 0 {
		b.MonitorDelay = srv.MonitorDelay
	}
	return mediaserver.New(b, mediaserver.Disk{Media: env.Media, Root: root}, libs), nil
}

// mint is a new access token.
func (s *server) mint() string {
	s.minted++
	return id("token " + s.srv.ID + " " + strconv.Itoa(s.minted))
}

// owner is the first administrator, whose data a key's calls read.
func (s *server) owner() string {
	for _, u := range s.srv.Users {
		if u.Admin {
			return u.Name
		}
	}
	return ""
}

func (s *server) userByID(uid string) (User, bool) {
	for _, u := range s.srv.Users {
		if userID(u.Name) == uid {
			return u, true
		}
	}
	return User{}, false
}

func (s *server) Serve(ctx context.Context, l net.Listener) error {
	if s.core != nil {
		go s.record(ctx)
	}
	return s.http.Serve(ctx, l)
}

// record writes each item event to the items capture until ctx ends.
func (s *server) record(ctx context.Context) {
	events := map[mediaserver.EventKind]string{mediaserver.ItemAdded: "added", mediaserver.ItemChanged: "changed", mediaserver.ItemRemoved: "removed"}
	for {
		sub := s.core.Subscribe()
		for done := false; !done; {
			select {
			case <-ctx.Done():
				sub.Cancel()
				return
			case <-sub.Done:
				done = true
			case e := <-sub.C:
				event, ok := events[e.Kind]
				if !ok {
					continue
				}
				rec := ItemRecord{Event: event, ID: e.Item.Key, Library: e.Library, Files: []string{}}
				for _, p := range e.Item.Parts {
					rec.Files = append(rec.Files, p.File.String())
				}
				s.itemLog.Record(ctx, rec)
			}
		}
	}
}
