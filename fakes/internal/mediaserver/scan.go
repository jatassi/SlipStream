package mediaserver

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

var (
	videoExt   = []string{".mkv", ".mp4", ".m4v", ".avi", ".mov", ".ts", ".wmv"}
	sidecarExt = []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".idx", ".sup", ".smi"}
	yearRe     = regexp.MustCompile(`^(.*?)\s*\((\d{4})\)`)
	episodeRe  = regexp.MustCompile(`(?i)\bS(\d{1,3})E(\d{1,4})\b`)
	editionRe  = regexp.MustCompile(`\{edition-([^}]*)\}`)
	providerRe = regexp.MustCompile(`(?i)\[(tmdb|imdb|tvdb)id-([^\]]+)\]|\{(tmdb|imdb|tvdb)-([^}]+)\}`)
)

// ErrNoLibrary is a call naming a library the server doesn't hold.
var ErrNoLibrary = errors.New("no such library")

// base is p's last segment.
func base(p link.ServerPath) string {
	s := p.String()
	return s[strings.LastIndexAny(s, `\/`)+1:]
}

func stem(name string) string { return strings.TrimSuffix(name, filepath.Ext(name)) }

// named is what a file's name says it holds.
type named struct {
	guid, title     string
	year            int
	show            string
	season, episode int
	edition         string
	provider        string
}

func describe(file link.ServerPath) named {
	name := stem(base(file))
	var n named
	if m := editionRe.FindStringSubmatch(name); m != nil {
		n.edition = m[1]
	}
	if m := providerRe.FindStringSubmatch(name); m != nil {
		n.provider = strings.ToLower(m[1]+m[3]) + "-" + m[2] + m[4]
		name = strings.TrimSpace(providerRe.ReplaceAllString(name, ""))
	}
	if m := episodeRe.FindStringSubmatchIndex(name); m != nil {
		n.season, _ = strconv.Atoi(name[m[2]:m[3]])
		n.episode, _ = strconv.Atoi(name[m[4]:m[5]])
		n.show = strings.TrimRight(name[:m[0]], " -._")
		if n.show == "" {
			n.show = base(file.Dir().Dir())
		}
		n.title = fmt.Sprintf("%s S%02dE%02d", n.show, n.season, n.episode)
		n.guid = fmt.Sprintf("episode/%s/s%02de%02d", strings.ToLower(n.show), n.season, n.episode)
		return n
	}
	n.title = strings.TrimSpace(editionRe.ReplaceAllString(name, ""))
	if m := yearRe.FindStringSubmatch(name); m != nil {
		n.title = m[1]
		n.year, _ = strconv.Atoi(m[2])
	}
	n.guid = fmt.Sprintf("movie/%s/%d", strings.ToLower(n.title), n.year)
	return n
}

// onDisk is a folder's files as a scan reads them.
type onDisk struct {
	videos   map[string]disked
	sidecars []link.ServerPath
}

type disked struct {
	path link.ServerPath
	size int64
}

// read walks folder on disk. A folder that is gone holds nothing.
func (s *Server) read(folder link.ServerPath) onDisk {
	d := onDisk{videos: map[string]disked{}}
	root, ok := s.disk.local(folder)
	if !ok {
		return d
	}
	_ = filepath.WalkDir(root, func(local string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return nil //nolint:nilerr // a folder that vanished mid-walk holds nothing
		}
		ext := strings.ToLower(filepath.Ext(local))
		p, perr := s.disk.server(local)
		if perr != nil {
			return nil //nolint:nilerr // a file outside the root isn't the server's
		}
		switch {
		case slices.Contains(videoExt, ext):
			var size int64
			if info, ierr := e.Info(); ierr == nil {
				size = info.Size()
			}
			d.videos[p.Key()] = disked{path: p, size: size}
		case slices.Contains(sidecarExt, ext):
			d.sidecars = append(d.sidecars, p)
		}
		return nil
	})
	return d
}

func (s *Server) exists(p link.ServerPath) bool {
	local, ok := s.disk.local(p)
	if !ok {
		return false
	}
	_, err := os.Stat(local)
	return err == nil
}

// RefreshFolder is a partial scan of folder in library, which Plex runs at
// once. A folder outside the library's folders scans nothing.
func (s *Server) RefreshFolder(library string, folder link.ServerPath) error {
	s.lock()
	defer s.mu.Unlock()
	l, ok := s.library(library)
	if !ok {
		return fmt.Errorf("library %s: %w", library, ErrNoLibrary)
	}
	if s.b.FreshSectionIgnores && l.fresh() {
		return nil
	}
	if !slices.ContainsFunc(l.Folders, folder.Within) {
		return nil
	}
	s.scanFolder(l, folder)
	return nil
}

// RefreshLibrary starts a full scan of library and answers when it
// started. A scan already running is that scan.
func (s *Server) RefreshLibrary(library string) (time.Time, error) {
	s.lock()
	defer s.mu.Unlock()
	l, ok := s.library(library)
	if !ok {
		return time.Time{}, fmt.Errorf("library %s: %w", library, ErrNoLibrary)
	}
	return s.startScan(l), nil
}

// RefreshAll starts a full scan of every library as one task, as
// Jellyfin's and Emby's RefreshLibrary task does, and answers when it
// started. A task already running is that task.
func (s *Server) RefreshAll() time.Time {
	s.lock()
	defer s.mu.Unlock()
	if s.task.Running {
		return s.task.Started
	}
	s.task.Running, s.task.Started = true, time.Now()
	for _, l := range s.libs {
		s.startScan(l)
	}
	return s.task.Started
}

// TaskState is the task RefreshAll starts.
func (s *Server) TaskState() ScanState {
	s.lock()
	defer s.mu.Unlock()
	return s.task
}

func (s *Server) startScan(l *library) time.Time {
	if l.scan.Running {
		return l.scan.Started
	}
	if s.b.ScanDropsPending {
		s.dropPending()
	}
	l.scan.Running, l.scan.Started = true, time.Now()
	s.publish(Event{Kind: ScanStarted, Library: l.Key})
	l.timer = time.AfterFunc(s.b.ScanTime, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		l.timer = nil
		for _, f := range l.Folders {
			s.scanFolder(l, f)
		}
		l.scan.Running, l.scan.Ended = false, time.Now()
		s.publish(Event{Kind: ScanEnded, Library: l.Key})
		if s.task.Running && !slices.ContainsFunc(s.libs, func(o *library) bool { return o.scan.Running }) {
			s.task.Running, s.task.Ended = false, l.scan.Ended
		}
	})
	return l.scan.Started
}

// AddFolder adds folder to library, as Jellyfin's
// /Library/VirtualFolders/Paths does. The folder starts fresh.
func (s *Server) AddFolder(library string, folder link.ServerPath) error {
	s.lock()
	defer s.mu.Unlock()
	l, ok := s.library(library)
	if !ok {
		return fmt.Errorf("library %s: %w", library, ErrNoLibrary)
	}
	if s.b.ScanDropsPending {
		s.dropPending()
	}
	if !slices.ContainsFunc(l.Folders, func(f link.ServerPath) bool { return f.Key() == folder.Key() }) {
		l.Folders = append(l.Folders, folder)
	}
	return nil
}

// ScanState is library's full scan.
func (s *Server) ScanState(library string) (ScanState, error) {
	s.lock()
	defer s.mu.Unlock()
	l, ok := s.library(library)
	if !ok {
		return ScanState{}, fmt.Errorf("library %s: %w", library, ErrNoLibrary)
	}
	return l.scan, nil
}

// RefreshItem reads the item's Sidecars again, as Plex's item refresh
// does.
func (s *Server) RefreshItem(key string) bool {
	s.lock()
	defer s.mu.Unlock()
	it, ok := s.items[key]
	if !ok {
		return false
	}
	s.bind(it.Folder, s.read(it.Folder).sidecars)
	s.publish(Event{Kind: ItemChanged, Library: it.Library, Item: it.clone()})
	return true
}

// scanFolder makes the server's items under folder match the disk. Files
// on disk are placed first, so a rename in one folder joins the item
// before its old file is dropped, and a move finds its vacated item before
// a scan of the source purges it.
func (s *Server) scanFolder(l *library, folder link.ServerPath) {
	d := s.read(folder)
	changed := s.placeAll(l, d)
	s.dropVanished(folder, d, changed)
	folders := map[string]link.ServerPath{}
	for _, it := range s.itemsUnder(folder) {
		folders[it.Folder.Key()] = it.Folder
	}
	for _, f := range folders {
		s.bind(f, d.sidecars)
	}
	for _, f := range l.Folders {
		if len(s.itemsUnder(f)) > 0 {
			l.scanned[f.Key()] = true
		}
	}
	for it, kind := range changed {
		s.publish(Event{Kind: kind, Library: it.Library, Item: it.clone()})
	}
}

// placeAll places every video on disk the server holds no part for, and
// answers the items it added or changed.
func (s *Server) placeAll(l *library, d onDisk) map[*Item]EventKind {
	changed := map[*Item]EventKind{}
	keys := make([]string, 0, len(d.videos))
	for k := range d.videos {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		v := d.videos[k]
		if it, i, ok := s.partAt(v.path); ok {
			it.Parts[i].Size, it.Parts[i].Missing = v.size, false
			continue
		}
		it, added := s.place(l, v)
		switch {
		case added:
			changed[it] = ItemAdded
		case changed[it] == "":
			changed[it] = ItemChanged
		}
	}
	return changed
}

// dropVanished drops the parts under folder that are gone from disk, or
// marks them missing when PurgesVacated is off. An item left with no part
// is deleted.
func (s *Server) dropVanished(folder link.ServerPath, d onDisk, changed map[*Item]EventKind) {
	for _, it := range s.itemsUnder(folder) {
		kept := it.Parts[:0]
		for _, p := range it.Parts {
			_, present := d.videos[p.File.Key()]
			if !present && !s.b.PurgesVacated {
				p.Missing = true
			}
			if present || !s.b.PurgesVacated {
				kept = append(kept, p)
			}
		}
		if len(kept) != len(it.Parts) && changed[it] == "" {
			changed[it] = ItemChanged
		}
		it.Parts = kept
		if len(it.Parts) == 0 {
			delete(changed, it)
			s.forget(it)
		}
	}
}

// place puts the file v in library l: in its title's item in the folder,
// or in a new item that takes over a vacated one's watch data. It is true
// for a new item.
func (s *Server) place(l *library, v disked) (*Item, bool) {
	n := describe(v.path)
	part := Part{ID: s.partID(), File: v.path, Size: v.size}
	folder := v.path.Dir()
	if it, ok := s.versionOf(l, folder, n); ok {
		it.Parts = append(it.Parts, part)
		return it, false
	}
	it := &Item{
		Key: s.key("item " + v.path.Key()), Library: l.Key, GUID: n.guid, Title: n.title, Year: n.year,
		Show: n.show, Season: n.season, Episode: n.episode, Edition: s.edition(n.edition), Provider: n.provider,
		Folder: folder, Parts: []Part{part}, Added: time.Now(),
	}
	if n.show != "" {
		show := l.Key + "/" + strings.ToLower(n.show)
		if _, ok := s.shows[show]; !ok {
			s.shows[show] = s.key("show " + show)
		}
		it.ShowKey = s.shows[show]
	}
	s.items[it.Key] = it
	if s.b.StableKeyInFolder {
		s.takeOver(it)
	}
	s.carryByProvider(it)
	return it, true
}

// versionOf is the item in folder that a file named n is another version
// of. With NewIDOnPathChange only an item with a file still on disk takes
// it, so a renamed file is a new item.
func (s *Server) versionOf(l *library, folder link.ServerPath, n named) (*Item, bool) {
	if s.b.ItemPerFile {
		return nil, false
	}
	for _, it := range s.items {
		if it.Library != l.Key || it.Folder.Key() != folder.Key() || it.GUID != n.guid || s.edition(it.Edition) != s.edition(n.edition) {
			continue
		}
		if s.b.NewIDOnPathChange && !slices.ContainsFunc(it.Parts, func(p Part) bool { return s.exists(p.File) }) {
			continue
		}
		return it, true
	}
	return nil, false
}

func (s *Server) edition(e string) string {
	if s.b.Editions {
		return e
	}
	return ""
}

// takeOver finds an item of it's title that lost its file, and moves that
// file's watch data to it. The vacated item with no file left is gone at
// once (case C).
func (s *Server) takeOver(it *Item) {
	for _, old := range s.items {
		if old == it || old.GUID != it.GUID {
			continue
		}
		i := slices.IndexFunc(old.Parts, func(p Part) bool { return p.Missing || !s.exists(p.File) })
		if i < 0 {
			continue
		}
		s.carry(old, it)
		old.Parts = slices.Delete(old.Parts, i, i+1)
		if len(old.Parts) == 0 {
			s.forget(old)
		}
		return
	}
}

func (s *Server) partID() int {
	s.parts++
	return s.parts
}

func (s *Server) itemsUnder(folder link.ServerPath) []*Item {
	var out []*Item
	for _, it := range s.items {
		if slices.ContainsFunc(it.Parts, func(p Part) bool { return p.File.Within(folder) }) {
			out = append(out, it)
		}
	}
	return out
}

// bind gives each part in folder the Sidecars beside it whose names start
// with the part's name, and every Sidecar there when the folder holds one
// item.
func (s *Server) bind(folder link.ServerPath, sidecars []link.ServerPath) {
	var in []*Item
	for _, it := range s.items {
		if it.Folder.Key() == folder.Key() {
			in = append(in, it)
		}
	}
	var beside []link.ServerPath
	for _, sc := range sidecars {
		if sc.Dir().Key() == folder.Key() {
			beside = append(beside, sc)
		}
	}
	for _, it := range in {
		for i := range it.Parts {
			p := &it.Parts[i]
			p.Sidecars = nil
			prefix := strings.ToLower(stem(base(p.File)) + ".")
			for _, sc := range beside {
				if len(in) == 1 || strings.HasPrefix(strings.ToLower(base(sc)), prefix) {
					p.Sidecars = append(p.Sidecars, sc)
				}
			}
		}
	}
}
