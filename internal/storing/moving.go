package storing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sync/atomic"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/media"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// MaxProbePage is the most paths one ProbeMove page names, its source and
// destination paths and payload files together, so a Controller bug
// can't ask for an answer over the link's limit.
const MaxProbePage = 10_000

// Library is a file in a library root this Node holds.
type Library struct {
	Root storage.UUID
	Rel  Rel
}

// Arriving is the file at Path, as the torrent's metainfo names it, of
// the torrent with InfoHash in the folder a Download Client keeps it in on
// this Node. Client names an external client, nil for the built-in one,
// and ContentType is "" for an adopted torrent.
type Arriving struct {
	Client      *[16]byte
	ContentType string
	InfoHash    bencode.InfoHash
	Path        Rel
}

func (Library) source()  {}
func (Arriving) source() {}

// ProbeMove is one page of a Move plan's reads on one Node. Videos say
// which files are a source file's companions.
type ProbeMove struct {
	Sources      []ProbeRoot
	Destinations []ProbeRoot
	Payloads     []Payload
	Videos       media.Videos
}

// ProbeRoot is paths under one root.
type ProbeRoot struct {
	Root storage.UUID
	Rels []Rel
}

// Probed is a page's answer. Files is each source file its root holds and
// Missing each it doesn't. Companions is each source file's companions, in
// its folder. Payloads is each payload file that lies on this Node; one
// that doesn't is left out. Occupants is the files in each destination
// path's folder whose names equal its name without case, which may be the
// path itself.
type Probed struct {
	Files      map[Library]ProbedFile
	Missing    []Library
	Companions map[Library][]ProbedFile
	Payloads   map[Payload]ProbedFile
	Occupants  map[Library][]ProbedFile
}

// ProbedFile is a file a probe read. Rel is its path in its root, zero for
// a payload file. Volume is the Volume.ID a payload file lies on. Ledger
// marks a companion whose bytes its root's Sidecar ledger holds.
type ProbedFile struct {
	Rel    Rel
	Size   int64
	Seen   Seen
	Volume string
	Ledger bool
}

// Removal is what RemoveIf found at a path.
type Removal uint8

const (
	Removed Removal = iota + 1
	// Gone is a path no file was at.
	Gone
	// Changed is a file whose size or modified time isn't the one asked,
	// so it stays.
	Changed
)

var removals = enum[Removal, storagev1.Removal]{
	{Removed, storagev1.Removal_REMOVAL_REMOVED, "removed"},
	{Gone, storagev1.Removal_REMOVAL_GONE, "gone"},
	{Changed, storagev1.Removal_REMOVAL_CHANGED, "changed"},
}

func (r Removal) String() string { return removals.of(r).word }

// ledgers is the local.db that holds the Sidecar ledger, once the Node's
// half opened it.
type ledgers = atomic.Pointer[storage.DB[storage.Local]]

// OpenLibrary is the path of rel in root for a Transfer's sender. It
// serves while the Node drains Storage, and answers *source.Absent for a
// file it can't serve.
func (s *Store) OpenLibrary(ctx context.Context, root storage.UUID, rel Rel) (string, error) {
	dir, err := s.resolve(ctx, root, false)
	if err != nil {
		return "", &source.Absent{Reason: source.NotReady, Detail: err.Error()}
	}
	_, err = s.stat(ctx, dir, rel)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", &source.Absent{Reason: source.Gone}
	case err != nil:
		return "", &source.Absent{Reason: source.NotReady, Detail: err.Error()}
	}
	return s.host.Style.Under(dir, rel), nil
}

func (s *Store) ProbeMove(ctx context.Context, req *storagev1.ProbeMoveRequest) (*storagev1.ProbeMoveResponse, error) {
	n := len(req.GetPayloads())
	for _, r := range append(req.GetSources(), req.GetDestinations()...) {
		n += len(r.GetRels())
	}
	if n > MaxProbePage {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%d paths is more than one page of %d", n, MaxProbePage)
	}
	res := &storagev1.ProbeMoveResponse{}
	if len(req.GetSources()) > 0 {
		videos, err := videosOf(req.GetVideoExtensions())
		if err != nil {
			return nil, err
		}
		for _, r := range req.GetSources() {
			probed, err := s.probeSource(ctx, r, videos)
			if err != nil {
				return nil, err
			}
			res.Sources = append(res.Sources, probed)
		}
	}
	for _, r := range req.GetDestinations() {
		probed, err := s.probeDestination(ctx, r)
		if err != nil {
			return nil, err
		}
		res.Destinations = append(res.Destinations, probed)
	}
	for _, p := range req.GetPayloads() {
		probed, err := s.probePayload(ctx, p)
		if err != nil {
			return nil, err
		}
		res.Payloads = append(res.Payloads, probed)
	}
	return res, nil
}

// lister lists each folder under dir once.
type lister struct {
	s      *Store
	dir    string
	listed map[Rel][]Entry
}

func (l lister) list(ctx context.Context, folder Rel) ([]Entry, error) {
	if es, ok := l.listed[folder]; ok {
		return es, nil
	}
	es, err := l.s.entries(ctx, l.s.host.Style.Under(l.dir, folder))
	if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, errNotAFolder) {
		return nil, l.s.folderError(err)
	}
	l.listed[folder] = es
	return es, nil
}

func (s *Store) probeRoot(ctx context.Context, r *storagev1.ProbeRoot) (storage.UUID, lister, []Rel, error) {
	root, err := rootID(r.GetRootId())
	if err != nil {
		return root, lister{}, nil, err
	}
	rels := make([]Rel, len(r.GetRels()))
	for i, p := range r.GetRels() {
		if rels[i], err = relOf(p); err != nil {
			return root, lister{}, nil, err
		}
	}
	dir, err := s.resolve(ctx, root, false)
	if err != nil {
		return root, lister{}, nil, WireError(err)
	}
	return root, lister{s: s, dir: dir, listed: map[Rel][]Entry{}}, rels, nil
}

func (s *Store) probeSource(ctx context.Context, r *storagev1.ProbeRoot, videos media.Videos) (*storagev1.ProbedSource, error) {
	root, l, rels, err := s.probeRoot(ctx, r)
	if err != nil {
		return nil, err
	}
	out := &storagev1.ProbedSource{Files: make([]*storagev1.ProbedFile, len(rels))}
	var hashes [][]byte
	var marked []*storagev1.ProbedCompanion
	for i, rel := range rels {
		info, err := s.stat(ctx, l.dir, rel)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			out.Files[i] = &storagev1.ProbedFile{Missing: true}
			continue
		case err != nil:
			return nil, WireError(err)
		}
		f := &storagev1.ProbedFile{Size: info.Size(), Identity: identityOf(info)}
		folder, name := rel.split()
		es, err := l.list(ctx, folder)
		if err != nil {
			return nil, WireError(err)
		}
		for _, c := range companions(es, name, videos) {
			at := folder.join(c)
			cinfo, err := s.stat(ctx, l.dir, at)
			switch {
			case errors.Is(err, fs.ErrNotExist):
				continue
			case err != nil:
				return nil, WireError(err)
			}
			pc := &storagev1.ProbedCompanion{Name: c, Size: cinfo.Size(), Identity: identityOf(cinfo)}
			if subtitleName(c) {
				_, sha, err := s.hash(ctx, l.dir, at)
				if err != nil {
					return nil, WireError(err)
				}
				hashes, marked = append(hashes, sha[:]), append(marked, pc)
			}
			f.Companions = append(f.Companions, pc)
		}
		out.Files[i] = f
	}
	return out, s.markLedger(ctx, root, hashes, marked)
}

// markLedger marks each of cs whose hash, at the same index, root's
// Sidecar ledger holds.
func (s *Store) markLedger(ctx context.Context, root storage.UUID, hashes [][]byte, cs []*storagev1.ProbedCompanion) error {
	if len(cs) == 0 {
		return nil
	}
	local, err := s.ledgerDB()
	if err != nil {
		return err
	}
	held, err := ledgerHolds(ctx, local, root, hashes)
	if err != nil {
		return err
	}
	for i, c := range cs {
		c.Ledger = containsHash(held, hashes[i])
	}
	return nil
}

func (s *Store) probeDestination(ctx context.Context, r *storagev1.ProbeRoot) (*storagev1.ProbedDestination, error) {
	_, l, rels, err := s.probeRoot(ctx, r)
	if err != nil {
		return nil, err
	}
	out := &storagev1.ProbedDestination{Paths: make([]*storagev1.ProbedPath, len(rels))}
	for i, rel := range rels {
		folder, _ := rel.split()
		es, err := l.list(ctx, folder)
		if err != nil {
			return nil, WireError(err)
		}
		found, err := s.occupants(ctx, l.dir, rel, es)
		if err != nil {
			return nil, WireError(err)
		}
		p := &storagev1.ProbedPath{}
		for _, o := range found {
			_, name := o.path.split()
			p.Occupants = append(p.Occupants, &storagev1.ProbedOccupant{Name: name, Size: o.info.Size(), Identity: identityOf(o.info)})
		}
		out.Paths[i] = p
	}
	return out, nil
}

func (s *Store) probePayload(ctx context.Context, p *storagev1.Payload) (*storagev1.ProbedPayload, error) {
	h, err := infoHashOf(p.GetInfoHash())
	if err != nil {
		return nil, err
	}
	path, err := s.cfg.Payloads.Path(h, int(p.GetFileIndex()))
	if err != nil {
		return &storagev1.ProbedPayload{Missing: true}, nil //nolint:nilerr // a payload the Node hasn't finished is missing
	}
	info, err := timed(ctx, s, path, func(ctx context.Context) (Info, error) { return s.cfg.Disk.Stat(ctx, path) })
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return &storagev1.ProbedPayload{Missing: true}, nil
	case err != nil:
		return nil, WireError(s.folderError(err))
	}
	vol, err := s.Volume(ctx, path)
	if err != nil {
		return nil, WireError(s.folderError(err))
	}
	return &storagev1.ProbedPayload{Size: info.Size(), Identity: identityOf(info), Volume: vol.ID}, nil
}

func infoHashOf(b []byte) (bencode.InfoHash, error) {
	var h bencode.InfoHash
	if len(b) != len(h) {
		return h, rpc.Errorf(rpc.CodeInvalidArgument, "an info hash is %d bytes, not %d", len(b), len(h))
	}
	return bencode.InfoHash(b), nil
}

func (s *Store) RemoveIf(ctx context.Context, req *storagev1.RemoveIfRequest) (*storagev1.RemoveIfResponse, error) {
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	rel, err := relOf(req.GetPath())
	if err != nil {
		return nil, err
	}
	removal, err := s.removeIf(ctx, root, rel, req.GetSize(), time.UnixMilli(req.GetModifiedUnixMs()))
	if err != nil {
		return nil, WireError(err)
	}
	return &storagev1.RemoveIfResponse{Removal: removals.of(removal).wire}, nil
}

// removeIf removes rel under root only while its size and modified time,
// to the millisecond, are the ones asked.
func (s *Store) removeIf(ctx context.Context, root storage.UUID, rel Rel, size int64, modified time.Time) (Removal, error) {
	dir, err := s.resolve(ctx, root, false)
	if err != nil {
		return 0, err
	}
	info, err := timed(ctx, s, dir, func(ctx context.Context) (Info, error) { return s.cfg.Disk.Stat(ctx, s.host.Style.Under(dir, rel)) })
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return Gone, nil
	case err != nil:
		return 0, s.folderError(err)
	case info.Dir() || info.Size() != size || info.ModTime().UnixMilli() != modified.UnixMilli():
		return Changed, nil
	}
	err = s.removeHeld(ctx, s.host.Style.Under(dir, rel))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return Gone, nil
	case err != nil:
		return 0, s.folderError(err)
	}
	return Removed, nil
}

// errInUse is a remove another program's open handle refused.
var errInUse = errors.New("another program has the file open")

// removeHeld deletes path, trying again while another program holds it
// open, up to RemoveBound after the first try.
func (s *Store) removeHeld(ctx context.Context, path string) error {
	deadline := time.Now().Add(s.cfg.RemoveBound)
	wait := s.cfg.RemoveRetry
	for {
		err := s.cfg.Disk.Remove(ctx, path)
		left := time.Until(deadline)
		if !errors.Is(err, errInUse) || left <= 0 {
			return err
		}
		t := time.NewTimer(min(wait, left))
		select {
		case <-t.C:
		case <-ctx.Done():
			t.Stop()
			return fmt.Errorf("remove %s: %w", path, ctx.Err())
		}
		wait *= 2
	}
}

// recordLedger records the file at rel under root in root's Sidecar
// ledger.
func (s *Store) recordLedger(ctx context.Context, root storage.UUID, rel Rel) error {
	local, err := s.ledgerDB()
	if err != nil {
		return err
	}
	dir, err := s.resolve(ctx, root, false)
	if err != nil {
		return err
	}
	_, sha, err := s.hash(ctx, dir, rel)
	if err != nil {
		return err
	}
	return putLedger(ctx, local, root, sha, rel)
}

func (s *Store) ledgerDB() (*storage.DB[storage.Local], error) {
	local := s.ledger.Load()
	if local == nil {
		return nil, rpc.Errorf(rpc.CodeUnavailable, "this Node keeps no Sidecar ledger until it is claimed")
	}
	return local, nil
}

// ProbeMove reads one page of a Move plan on node. The Node answers the
// page whole or fails it: a root it can't read now is *NotReady.
func (r Remote) ProbeMove(ctx context.Context, node role.NodeID, p ProbeMove) (Probed, error) {
	req := &storagev1.ProbeMoveRequest{VideoExtensions: p.Videos.Extensions()}
	for _, at := range p.Sources {
		req.Sources = append(req.Sources, probeRootWire(at))
	}
	for _, at := range p.Destinations {
		req.Destinations = append(req.Destinations, probeRootWire(at))
	}
	for _, pl := range p.Payloads {
		req.Payloads = append(req.Payloads, &storagev1.Payload{InfoHash: pl.InfoHash[:], FileIndex: int64(pl.FileIndex)})
	}
	res, err := r(node).ProbeMove(ctx, req)
	if err != nil {
		return Probed{}, answer(err)
	}
	if len(res.GetSources()) != len(p.Sources) || len(res.GetDestinations()) != len(p.Destinations) || len(res.GetPayloads()) != len(p.Payloads) {
		return Probed{}, errors.New("the Node's probe answers another page than asked")
	}
	out := Probed{Files: map[Library]ProbedFile{}, Companions: map[Library][]ProbedFile{}, Payloads: map[Payload]ProbedFile{}, Occupants: map[Library][]ProbedFile{}}
	for i, at := range p.Sources {
		if err := out.sources(at, res.GetSources()[i].GetFiles()); err != nil {
			return Probed{}, err
		}
	}
	for i, at := range p.Destinations {
		if err := out.destinations(at, res.GetDestinations()[i].GetPaths()); err != nil {
			return Probed{}, err
		}
	}
	for i, pl := range p.Payloads {
		if w := res.GetPayloads()[i]; !w.GetMissing() {
			out.Payloads[pl] = ProbedFile{Size: w.GetSize(), Seen: seenOf(w.GetIdentity()), Volume: w.GetVolume()}
		}
	}
	return out, nil
}

func probeRootWire(at ProbeRoot) *storagev1.ProbeRoot {
	w := &storagev1.ProbeRoot{RootId: at.Root.Bytes()}
	for _, rel := range at.Rels {
		w.Rels = append(w.Rels, rel.String())
	}
	return w
}

func (p *Probed) sources(at ProbeRoot, files []*storagev1.ProbedFile) error {
	if len(files) != len(at.Rels) {
		return errors.New("the Node's probe answers another page than asked")
	}
	for i, rel := range at.Rels {
		key, w := Library{Root: at.Root, Rel: rel}, files[i]
		if w.GetMissing() {
			p.Missing = append(p.Missing, key)
			continue
		}
		p.Files[key] = ProbedFile{Rel: rel, Size: w.GetSize(), Seen: seenOf(w.GetIdentity())}
		folder, _ := rel.split()
		for _, c := range w.GetCompanions() {
			at, err := nameIn(folder, c.GetName())
			if err != nil {
				return fmt.Errorf("a companion: %w", err)
			}
			p.Companions[key] = append(p.Companions[key], ProbedFile{Rel: at, Size: c.GetSize(), Seen: seenOf(c.GetIdentity()), Ledger: c.GetLedger()})
		}
	}
	return nil
}

func (p *Probed) destinations(at ProbeRoot, paths []*storagev1.ProbedPath) error {
	if len(paths) != len(at.Rels) {
		return errors.New("the Node's probe answers another page than asked")
	}
	for i, rel := range at.Rels {
		key := Library{Root: at.Root, Rel: rel}
		folder, _ := rel.split()
		for _, o := range paths[i].GetOccupants() {
			path, err := nameIn(folder, o.GetName())
			if err != nil {
				return fmt.Errorf("a file at a destination: %w", err)
			}
			p.Occupants[key] = append(p.Occupants[key], ProbedFile{Rel: path, Size: o.GetSize(), Seen: seenOf(o.GetIdentity())})
		}
	}
	return nil
}

// nameIn is the file name in folder, which a Worker's answer can't take
// out of it.
func nameIn(folder Rel, name string) (Rel, error) {
	at, err := ParseRel(name)
	if err != nil {
		return Rel{}, err
	}
	if len(at.segments()) != 1 {
		return Rel{}, fmt.Errorf("%w: %q isn't one name", ErrBadRel, name)
	}
	return folder.join(name), nil
}

// RemoveIf removes rel from the root only while its size and modified
// time, to the millisecond, are the ones asked. A root its Node can't read
// now is *NotReady.
func (r Remote) RemoveIf(ctx context.Context, at At, rel Rel, size int64, modified time.Time) (Removal, error) {
	res, err := r(at.Node).RemoveIf(ctx, &storagev1.RemoveIfRequest{RootId: at.Root.Bytes(), Path: rel.String(), Size: size, ModifiedUnixMs: modified.UnixMilli()})
	if err != nil {
		return 0, answer(err)
	}
	return removals.parse(res.GetRemoval())
}

func arrivingOf(w *storagev1.ArrivingSource) (Arriving, error) {
	h, err := infoHashOf(w.GetInfoHash())
	if err != nil {
		return Arriving{}, err
	}
	a := Arriving{ContentType: w.GetContentType(), InfoHash: h}
	if c := w.GetClientId(); len(c) > 0 {
		if len(c) != 16 {
			return Arriving{}, rpc.Errorf(rpc.CodeInvalidArgument, "a client id is %d bytes, not 16", len(c))
		}
		a.Client = (*[16]byte)(c)
	}
	if a.Path, err = relOf(w.GetPath()); err != nil {
		return Arriving{}, err
	}
	return a, nil
}
