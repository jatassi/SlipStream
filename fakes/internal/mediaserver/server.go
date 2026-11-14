// Package mediaserver is the state every Media Server fake answers from:
// libraries and their folders, items read from the files on disk, users'
// watch data, collections, scans, path reports and sessions. It knows no
// wire. Each product fake parses its routes into these calls and renders
// the answers in its product's bodies, and Behavior holds where the
// products differ, as the recorded runs measured them.
//
// Every wait is a timer on the clock the fake runs on, so inside a
// synctest bubble a 60 s monitor delay passes in fake time.
package mediaserver

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/storing"
)

// Behavior is how one product differs from the others.
type Behavior struct {
	// MonitorDelay is how long a path report waits before its folder is
	// scanned: Jellyfin 60 s, Emby 90 s.
	MonitorDelay time.Duration
	// RestartsOnRepeat is Jellyfin's timer per path, which a repeat report
	// restarts. Off, one timer covers every report until it fires, as
	// Emby's does.
	RestartsOnRepeat bool
	// FreshRootIgnores is Jellyfin ignoring a report into a library never
	// scanned with content in it.
	FreshRootIgnores bool
	// FreshSectionIgnores is Plex ignoring a partial scan into a section
	// never scanned with content in it (#101 P9).
	FreshSectionIgnores bool
	// ScanDropsPending is Jellyfin dropping every pending report when a full
	// scan starts.
	ScanDropsPending bool
	// StableKeyInFolder is Plex's item per title in a folder, so a rename
	// in the folder keeps the item's key. A new item whose title matches an
	// item that lost that file takes its watch data, and its collections
	// when both are in one library (recorded run 2026-11-14, cases B to
	// D). Off, every file is its own item.
	StableKeyInFolder bool
	// PurgesVacated is Plex's autoEmptyTrash: a partial scan of a folder
	// whose item lost every file deletes the item with its watch data and
	// collections (case E). Off, the item keeps its missing files.
	PurgesVacated bool
	// Editions is Plex Pass: a {edition-...} tag in a name makes its own
	// item.
	Editions bool
	// ScanTime is how long a full scan runs, and must be more than zero so
	// a scan ends after it was asked for: Plex takes 0.25 s on a tiny
	// section (case G).
	ScanTime time.Duration
}

// Disk is where the server's folders live on this machine: Root, as the
// server names it, is the local folder Media, so a folder such as
// /media/movies or D:\Plex\Movies under Root is Media/movies or
// Media/Movies. The fake reads it on each scan.
type Disk struct {
	Media string
	Root  link.ServerPath
}

func (d Disk) local(p link.ServerPath) (string, bool) {
	if p.Key() == d.Root.Key() {
		return d.Media, true
	}
	rel, ok := p.Rel(d.Root)
	if !ok {
		return "", false
	}
	return filepath.Join(d.Media, filepath.FromSlash(rel.String())), true
}

func (d Disk) server(local string) (link.ServerPath, error) {
	rel, err := filepath.Rel(d.Media, local)
	if err != nil {
		return link.ServerPath{}, fmt.Errorf("%s isn't under %s: %w", local, d.Media, err)
	}
	if rel == "." {
		return d.Root, nil
	}
	r, err := storing.ParseRel(filepath.ToSlash(rel))
	if err != nil {
		return link.ServerPath{}, fmt.Errorf("%s isn't under %s: %w", local, d.Media, err)
	}
	return d.Root.Join(r), nil
}

// Library is one Plex section or one Jellyfin or Emby library.
type Library struct {
	Key  string
	Name string
	// Type is the product's own word for what the library holds.
	Type    string
	Folders []link.ServerPath
	// Scanned is a library that was scanned with its content in it before
	// the fake started, as one an Admin made on a full folder was.
	Scanned bool
}

// LibraryState is a library and its scan.
type LibraryState struct {
	Library
	Scan ScanState
}

// ScanState is a library's full scan. Ended is the last scan's end, zero
// before the first.
type ScanState struct {
	Running bool
	Started time.Time
	Ended   time.Time
}

// Item is one item the server holds, a copy the caller may keep.
type Item struct {
	Key     string
	Library string
	GUID    string
	Title   string
	Year    int
	// Show, Season and Episode are an episode's, and ShowKey is its show's
	// item key. All are zero for a movie.
	Show            string
	ShowKey         string
	Season, Episode int
	Edition         string
	Folder          link.ServerPath
	Parts           []Part
	Added           time.Time
}

// Part is one file of an item and the Sidecars bound to it.
type Part struct {
	ID       int
	File     link.ServerPath
	Size     int64
	Sidecars []link.ServerPath
	// Missing is a file gone from disk that the item still lists, because
	// PurgesVacated is off.
	Missing bool
}

func (it *Item) clone() Item {
	c := *it
	c.Parts = make([]Part, len(it.Parts))
	for i, p := range it.Parts {
		c.Parts[i] = p
		c.Parts[i].Sidecars = slices.Clone(p.Sidecars)
	}
	return c
}

// Server is one Media Server's state, shared by every front that reaches
// it.
type Server struct {
	b    Behavior
	disk Disk

	mu          sync.Mutex
	primed      bool
	libs        []*library
	items       map[string]*Item
	shows       map[string]string
	next        int
	parts       int
	sessionKeys int
	watch       map[watchKey]link.UserData
	collections map[string][]string
	sessions    []*Session
	pending     map[string]*time.Timer
	batch       *batch
	subs        map[*Subscription]struct{}
}

type library struct {
	Library
	// scanned is whether a scan ran with content in the library.
	scanned bool
	scan    ScanState
	timer   *time.Timer
}

type watchKey struct{ user, item string }

// New is a server holding libs, reading its folders from disk. It does no
// IO: a library marked Scanned is read on the first call.
func New(b Behavior, disk Disk, libs []Library) *Server {
	s := &Server{
		b: b, disk: disk,
		items: map[string]*Item{}, shows: map[string]string{},
		watch: map[watchKey]link.UserData{}, collections: map[string][]string{},
		pending: map[string]*time.Timer{}, subs: map[*Subscription]struct{}{},
	}
	for _, l := range libs {
		s.libs = append(s.libs, &library{Library: l, scanned: l.Scanned})
	}
	return s
}

// lock takes the lock and, the first time, reads every library marked
// Scanned, so New stays free of IO.
func (s *Server) lock() {
	s.mu.Lock()
	if s.primed {
		return
	}
	s.primed = true
	for _, l := range s.libs {
		if l.Scanned {
			for _, f := range l.Folders {
				s.scanFolder(l, f)
			}
		}
	}
}

func (s *Server) key() string {
	s.next++
	return strconv.Itoa(s.next)
}

// Libraries are the server's libraries in the order New had them.
func (s *Server) Libraries() []LibraryState {
	s.lock()
	defer s.mu.Unlock()
	out := make([]LibraryState, len(s.libs))
	for i, l := range s.libs {
		out[i] = LibraryState{Library: l.Library, Scan: l.scan}
		out[i].Folders = slices.Clone(l.Folders)
	}
	return out
}

func (s *Server) library(key string) (*library, bool) {
	i := slices.IndexFunc(s.libs, func(l *library) bool { return l.Key == key })
	if i < 0 {
		return nil, false
	}
	return s.libs[i], true
}

// libraryOf is the library with a folder p is in.
func (s *Server) libraryOf(p link.ServerPath) (*library, link.ServerPath, bool) {
	for _, l := range s.libs {
		for _, f := range l.Folders {
			if p.Within(f) {
				return l, f, true
			}
		}
	}
	return nil, link.ServerPath{}, false
}

// Item is the item key, false when the server holds none.
func (s *Server) Item(key string) (Item, bool) {
	s.lock()
	defer s.mu.Unlock()
	it, ok := s.items[key]
	if !ok {
		return Item{}, false
	}
	return it.clone(), true
}

// ItemAt is the item with a file at p.
func (s *Server) ItemAt(p link.ServerPath) (Item, bool) {
	s.lock()
	defer s.mu.Unlock()
	if it, _, ok := s.partAt(p); ok {
		return it.clone(), true
	}
	return Item{}, false
}

func (s *Server) partAt(p link.ServerPath) (*Item, int, bool) {
	for _, it := range s.items {
		for i, part := range it.Parts {
			if part.File.Key() == p.Key() {
				return it, i, true
			}
		}
	}
	return nil, 0, false
}

// Items are a page of library's items in key order, and how many it
// holds.
func (s *Server) Items(library string, start, size int) (page []Item, total int) {
	s.lock()
	defer s.mu.Unlock()
	var all []*Item
	for _, it := range s.items {
		if it.Library == library {
			all = append(all, it)
		}
	}
	slices.SortFunc(all, func(a, b *Item) int { return cmp.Compare(keyNumber(a.Key), keyNumber(b.Key)) })
	total = len(all)
	start = min(start, total)
	end := min(start+size, total)
	out := make([]Item, 0, end-start)
	for _, it := range all[start:end] {
		out = append(out, it.clone())
	}
	return out, total
}

func keyNumber(k string) int {
	n, _ := strconv.Atoi(k)
	return n
}

// UserData is user's data on item.
func (s *Server) UserData(user, item string) link.UserData {
	s.lock()
	defer s.mu.Unlock()
	d := s.watch[watchKey{user, item}]
	d.User = user
	return d
}

// SetUserData replaces user's data on item.
func (s *Server) SetUserData(user, item string, d link.UserData) {
	s.lock()
	defer s.mu.Unlock()
	d.User = user
	s.watch[watchKey{user, item}] = d
}

// Played counts one more play of item by user, as a scrobble does.
func (s *Server) Played(user, item string, at time.Time) {
	s.lock()
	defer s.mu.Unlock()
	k := watchKey{user, item}
	d := s.watch[k]
	d.User, d.Played, d.PlayCount, d.LastPlayed, d.Position = user, true, d.PlayCount+1, at, 0
	s.watch[k] = d
}

// Collections are the names of the collections item is in.
func (s *Server) Collections(item string) []string {
	s.lock()
	defer s.mu.Unlock()
	return slices.Clone(s.collections[item])
}

// AddToCollection puts items in the collection name.
func (s *Server) AddToCollection(name string, items ...string) {
	s.lock()
	defer s.mu.Unlock()
	for _, it := range items {
		if !slices.Contains(s.collections[it], name) {
			s.collections[it] = append(s.collections[it], name)
		}
	}
}

// forget deletes item with its watch data and collections.
func (s *Server) forget(it *Item) {
	delete(s.items, it.Key)
	delete(s.collections, it.Key)
	for k := range s.watch {
		if k.item == it.Key {
			delete(s.watch, k)
		}
	}
	s.publish(Event{Kind: ItemRemoved, Library: it.Library, Item: it.clone()})
}

// carry moves from's watch data to to, and its collections when both are
// in one library.
func (s *Server) carry(from, to *Item) {
	for k, d := range s.watch {
		if k.item == from.Key {
			s.watch[watchKey{k.user, to.Key}] = d
		}
	}
	if from.Library == to.Library {
		s.collections[to.Key] = slices.Clone(s.collections[from.Key])
	}
}
