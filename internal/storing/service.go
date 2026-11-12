package storing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/media"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

const (
	suggestions = 25
	verifyBatch = 500
	surveyBatch = 2000
)

func (s *Store) Browse(ctx context.Context, req *storagev1.BrowseRequest) (*storagev1.BrowseResponse, error) {
	l, err := s.browse(ctx, req.GetPath(), req.GetShowHidden())
	if err != nil {
		return nil, wireError(err)
	}
	res := &storagev1.BrowseResponse{
		Path: l.Path, Parent: l.Parent, Hidden: int64(l.Hidden), Service: l.Service,
	}
	if l.Volume != nil {
		res.Volume = volumeWire(*l.Volume)
	}
	for _, seg := range l.Segments {
		res.Segments = append(res.Segments, &storagev1.Segment{Name: seg.Name, Path: seg.Path})
	}
	for _, f := range l.Folders {
		res.Folders = append(res.Folders, &storagev1.Folder{Name: f.Name, Path: f.Path, Readable: f.Readable})
	}
	for _, p := range l.Places {
		place := &storagev1.Place{Name: p.Name, Path: p.Path, Kind: kindWire(p.Kind), Free: p.Free, Total: p.Total}
		if !p.Root.IsZero() {
			place.RootId = p.Root.Bytes()
		}
		res.Places = append(res.Places, place)
	}
	return res, nil
}

func (s *Store) Suggest(ctx context.Context, req *storagev1.SuggestRequest) (*storagev1.SuggestResponse, error) {
	paths, err := s.suggest(ctx, req.GetText())
	if err != nil {
		return nil, wireError(err)
	}
	return &storagev1.SuggestResponse{Paths: paths}, nil
}

func (s *Store) MakeFolder(ctx context.Context, req *storagev1.MakeFolderRequest) (*storagev1.MakeFolderResponse, error) {
	p, err := s.makeFolder(ctx, req.GetParent(), req.GetName())
	if err != nil {
		return nil, wireError(err)
	}
	return &storagev1.MakeFolderResponse{Path: p}, nil
}

func (s *Store) Check(ctx context.Context, req *storagev1.CheckRequest) (*storagev1.CheckResponse, error) {
	purpose, err := purposes.parse(req.GetPurpose())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	others := make([]Other, 0, len(req.GetOthers()))
	for _, o := range req.GetOthers() {
		id, err := storage.UUIDFromBytes(o.GetId())
		if err != nil {
			return nil, rpc.Errorf(rpc.CodeInvalidArgument, "another root's id: %v", err)
		}
		others = append(others, Other{ID: id, Path: o.GetPath()})
	}
	c, err := s.check(ctx, req.GetPath(), purpose, others, purpose != PurposeClientFolder)
	if err != nil {
		return nil, wireError(err)
	}
	res := &storagev1.CheckResponse{Path: c.Path}
	if c.Volume != nil {
		res.Volume = volumeWire(*c.Volume)
	}
	if c.Refusal != nil {
		res.Refusal = c.Refusal.Reason.Wire()
		if !c.Refusal.Root.IsZero() {
			res.RootId = c.Refusal.Root.Bytes()
		}
	}
	if !c.Marker.IsZero() {
		res.Marker = c.Marker.Bytes()
	}
	return res, nil
}

func (s *Store) Verify(ctx context.Context, req *storagev1.VerifyRequest) (*storagev1.VerifyResponse, error) {
	if len(req.GetFiles()) > verifyBatch {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%d files is more than one batch of %d", len(req.GetFiles()), verifyBatch)
	}
	files := make([]Recorded, 0, len(req.GetFiles()))
	for _, f := range req.GetFiles() {
		sits, err := relOf(f.GetPath())
		if err != nil {
			return nil, err
		}
		var next Rel
		if f.GetNext() != "" {
			if next, err = relOf(f.GetNext()); err != nil {
				return nil, err
			}
		}
		files = append(files, Recorded{Sits: sits, Next: next, Size: f.GetSize()})
	}
	var dir string
	switch at := req.GetAt().(type) {
	case *storagev1.VerifyRequest_Path:
		var err error
		if dir, err = s.clean(at.Path); err != nil {
			return nil, err
		}
	case *storagev1.VerifyRequest_RootId:
		root, err := rootID(at.RootId)
		if err != nil {
			return nil, err
		}
		if dir, err = s.resolve(ctx, root, false); err != nil {
			return nil, wireError(err)
		}
	default:
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "verify names neither a folder nor a root")
	}
	missing, err := s.verify(ctx, dir, files)
	if err != nil {
		return nil, wireError(err)
	}
	res := &storagev1.VerifyResponse{}
	for _, m := range missing {
		res.Missing = append(res.Missing, &storagev1.Missing{Index: int64(m.Index), Reason: m.Reason.Wire()})
	}
	return res, nil
}

func (s *Store) Place(ctx context.Context, req *storagev1.PlaceRequest) (*storagev1.PlaceResponse, error) {
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	var h bencode.InfoHash
	if len(req.GetInfoHash()) != len(h) {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "an info hash is %d bytes, not %d", len(h), len(req.GetInfoHash()))
	}
	copy(h[:], req.GetInfoHash())
	to, err := relOf(req.GetTo())
	if err != nil {
		return nil, err
	}
	info, step, err := s.place(ctx, root, Payload{InfoHash: h, FileIndex: int(req.GetFileIndex())}, to, req.GetAllowCopy())
	if err != nil {
		return nil, wireError(err)
	}
	return &storagev1.PlaceResponse{Size: info.Size(), Step: step.Wire(), Identity: identityOf(info)}, nil
}

func (s *Store) Move(ctx context.Context, req *storagev1.MoveRequest) (*storagev1.MoveResponse, error) {
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	from, err := relOf(req.GetFrom())
	if err != nil {
		return nil, err
	}
	to, err := relOf(req.GetTo())
	if err != nil {
		return nil, err
	}
	info, left, err := s.move(ctx, root, from, to, req.GetCarry())
	if err != nil {
		return nil, wireError(err)
	}
	res := &storagev1.MoveResponse{Size: info.Size(), Identity: identityOf(info)}
	for _, l := range left {
		res.Left = append(res.Left, l.String())
	}
	return res, nil
}

func (s *Store) Survey(ctx context.Context, req *storagev1.SurveyRequest) (*storagev1.SurveyResponse, error) {
	if len(req.GetPlanned()) > surveyBatch {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%d renames is more than one batch of %d", len(req.GetPlanned()), surveyBatch)
	}
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	planned := make([]Planned, 0, len(req.GetPlanned()))
	for _, p := range req.GetPlanned() {
		from, err := relOf(p.GetFrom())
		if err != nil {
			return nil, err
		}
		to, err := relOf(p.GetTo())
		if err != nil {
			return nil, err
		}
		planned = append(planned, Planned{From: from, To: to})
	}
	found, err := s.survey(ctx, root, planned, req.GetCompanions())
	if err != nil {
		return nil, wireError(err)
	}
	res := &storagev1.SurveyResponse{}
	for _, f := range found {
		w := &storagev1.Surveyed{}
		for _, o := range f.occupants {
			w.Occupants = append(w.Occupants, &storagev1.Occupant{Path: o.path.String(), Identity: identityOf(o.info)})
		}
		for _, c := range f.companions {
			w.Companions = append(w.Companions, c.String())
		}
		res.Surveyed = append(res.Surveyed, w)
	}
	return res, nil
}

func (s *Store) Prune(ctx context.Context, req *storagev1.PruneRequest) (*storagev1.PruneResponse, error) {
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	folder, err := relOf(req.GetFolder())
	if err != nil {
		return nil, err
	}
	keep := make([]Rel, 0, len(req.GetKeep()))
	for _, k := range req.GetKeep() {
		rel, err := relOf(k)
		if err != nil {
			return nil, err
		}
		keep = append(keep, rel)
	}
	removed, err := s.prune(ctx, root, folder, keep)
	if err != nil {
		return nil, wireError(err)
	}
	res := &storagev1.PruneResponse{}
	for _, r := range removed {
		res.Removed = append(res.Removed, r.String())
	}
	return res, nil
}

func (s *Store) Remove(ctx context.Context, req *storagev1.RemoveRequest) (*storagev1.RemoveResponse, error) {
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	rel, err := relOf(req.GetPath())
	if err != nil {
		return nil, err
	}
	if err := s.remove(ctx, root, rel, req.GetRecursive()); err != nil {
		return nil, wireError(err)
	}
	return &storagev1.RemoveResponse{}, nil
}

func (s *Store) List(ctx context.Context, req *storagev1.ListRequest) (*storagev1.ListResponse, error) {
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	folder, err := heldFolders.parse(req.GetFolder())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	names, err := s.list(ctx, root, folder)
	if err != nil {
		return nil, wireError(err)
	}
	return &storagev1.ListResponse{Names: names}, nil
}

func (s *Store) Walk(ctx context.Context, req *storagev1.WalkRequest) (*storagev1.WalkResponse, error) {
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	limit := int(req.GetLimit())
	if limit < 1 || limit > walkLimit {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "a page lists 1 to %d entries, not %d", walkLimit, limit)
	}
	var after []string
	if len(req.GetAfter()) > 0 {
		after = strings.Split(string(req.GetAfter()), "\x00")
	}
	p, err := s.walk(ctx, root, after, limit)
	if err != nil {
		return nil, wireError(err)
	}
	res := &storagev1.WalkResponse{Next: []byte(strings.Join(p.next, "\x00")), CaseInsensitive: p.caseFold}
	for _, f := range p.files {
		res.Entries = append(res.Entries, &storagev1.Walked{Path: f.rel.String(), Size: f.info.Size(), Identity: identityOf(f.info)})
	}
	for _, u := range p.unread {
		res.Unread = append(res.Unread, &storagev1.Unread{Path: u.Path, Reason: u.Reason.Wire()})
	}
	return res, nil
}

func (s *Store) Probe(ctx context.Context, req *storagev1.ProbeRequest) (*storagev1.ProbeResponse, error) {
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	rel, err := relOf(req.GetPath())
	if err != nil {
		return nil, err
	}
	r, err := s.probe(ctx, root, rel)
	if u, ok := errors.AsType[*media.UnreadableError](err); ok {
		return nil, rpc.ErrorWith(rpc.CodeFailedPrecondition, &storagev1.Unprobed{Line: u.Line}, u.Error())
	}
	if err != nil {
		return nil, wireError(err)
	}
	record, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("encode the media record: %w", err)
	}
	return &storagev1.ProbeResponse{Record: record}, nil
}

// identityOf is what a scan compares of info besides its size.
func identityOf(info Info) *storagev1.Identity {
	return &storagev1.Identity{ModifiedUnixMs: info.ModTime().UnixMilli(), FileId: info.FileID(), VolumeId: info.Volume()}
}

func rootID(b []byte) (storage.UUID, error) {
	id, err := storage.UUIDFromBytes(b)
	if err != nil {
		return id, rpc.Errorf(rpc.CodeInvalidArgument, "root id: %v", err)
	}
	return id, nil
}

func relOf(s string) (Rel, error) {
	rel, err := ParseRel(s)
	if err != nil {
		return rel, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	return rel, nil
}

// wireError is err as a handler answers it: a *FolderError, *NotReady or
// ErrOccupied keeps its meaning across the wire, and any other error is
// the call's as it is.
func wireError(err error) error {
	if fe, ok := errors.AsType[*FolderError](err); ok {
		return rpc.ErrorWith(folderCodes[fe.Kind], &storagev1.FolderError{Kind: fe.Kind.Wire(), OsText: fe.Text, Account: fe.Account}, fe.Error())
	}
	if nr, ok := errors.AsType[*NotReady](err); ok {
		detail := &storagev1.NotReady{Draining: nr.Draining, Unknown: nr.Unknown}
		if nr.State != 0 {
			detail.State = nr.State.Wire()
		}
		return rpc.ErrorWith(rpc.CodeFailedPrecondition, detail, nr.Error())
	}
	if errors.Is(err, ErrOccupied) {
		return rpc.Errorf(rpc.CodeAlreadyExists, "%v", err)
	}
	if nc, ok := errors.AsType[*NeedsCopyError](err); ok {
		return rpc.ErrorWith(rpc.CodeFailedPrecondition, &storagev1.NeedsCopy{Size: nc.Size}, nc.Error())
	}
	if ns, ok := errors.AsType[*NeedsSpaceError](err); ok {
		return rpc.ErrorWith(rpc.CodeResourceExhausted, &storagev1.NeedsSpace{}, ns.Error())
	}
	return err
}

var folderCodes = map[FolderErrorKind]rpc.Code{
	FolderNotFound:     rpc.CodeNotFound,
	FolderNotAFolder:   rpc.CodeFailedPrecondition,
	FolderDenied:       rpc.CodePermissionDenied,
	FolderUnresponsive: rpc.CodeDeadlineExceeded,
	FolderIO:           rpc.CodeInternal,
}

func (s *Store) folderError(err error) *FolderError {
	fe := &FolderError{Kind: folderErrorOf(err), Text: err.Error()}
	if fe.Kind == FolderDenied {
		fe.Account = s.host.Account
	}
	return fe
}

func (s *Store) clean(p string) (string, error) {
	c, err := s.host.Style.Clean(p)
	if err != nil {
		return "", rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	return c, nil
}

// entries lists the clean folder dir under the stat deadline.
func (s *Store) entries(ctx context.Context, dir string) ([]Entry, error) {
	return timed(ctx, s, dir, func(ctx context.Context) ([]Entry, error) {
		info, err := s.cfg.Disk.Stat(ctx, dir)
		if err != nil {
			return nil, err
		}
		if !info.Dir() {
			return nil, fmt.Errorf("%s: %w", dir, errNotAFolder)
		}
		return s.cfg.Disk.ReadDir(ctx, dir)
	})
}

// browse lists a folder's subfolders, or the Node's places for an empty
// path: its roots first, then its drives or mounts.
func (s *Store) browse(ctx context.Context, path string, showHidden bool) (Listing, error) {
	if path == "" {
		return s.places(ctx)
	}
	dir, err := s.clean(path)
	if err != nil {
		return Listing{}, err
	}
	es, err := s.entries(ctx, dir)
	if err != nil {
		return Listing{}, s.folderError(err)
	}
	l := Listing{Path: dir, Segments: s.host.Style.Segments(dir)}
	l.Parent, _ = s.host.Style.Parent(dir)
	for _, e := range es {
		switch {
		case !e.Dir:
		case e.Hidden && !showHidden:
			l.Hidden++
		default:
			l.Folders = append(l.Folders, Folder{Name: e.Name, Path: s.host.Style.Join(dir, e.Name), Readable: e.Readable})
		}
	}
	slices.SortFunc(l.Folders, func(a, b Folder) int { return naturalOrder(a.Name, b.Name) })
	if v, err := timed(ctx, s, dir, func(ctx context.Context) (Volume, error) { return s.cfg.Disk.Volume(ctx, dir) }); err == nil {
		l.Volume = &v
	}
	return l, nil
}

func naturalOrder(a, b string) int {
	switch {
	case naturalLess(a, b):
		return -1
	case naturalLess(b, a):
		return 1
	}
	return 0
}

func (s *Store) places(ctx context.Context) (Listing, error) {
	s.mu.Lock()
	roots, sampled := s.roots, maps.Clone(s.sampled)
	s.mu.Unlock()
	l := Listing{Service: s.host.Service}
	for _, r := range roots {
		v := sampled[r.id]
		l.Places = append(l.Places, Place{Name: r.path, Path: r.path, Kind: v.Kind, Free: v.Free, Total: v.Total, Root: r.id})
	}
	disks, err := timed(ctx, s, "places", s.cfg.Disk.Places)
	if err != nil {
		return Listing{}, s.folderError(err)
	}
	l.Places = append(l.Places, disks...)
	return l, nil
}

// suggest lists up to 25 folders completing typed, in the separators it
// was typed with. Text that names no folder yet, or a folder that isn't
// there, has none.
func (s *Store) suggest(ctx context.Context, typed string) ([]string, error) {
	dir, prefix, _, ok := s.host.Style.Split(typed)
	if !ok {
		return nil, nil
	}
	es, err := s.entries(ctx, dir)
	if err != nil {
		if k := folderErrorOf(err); k == FolderNotFound || k == FolderNotAFolder {
			return nil, nil
		}
		return nil, s.folderError(err)
	}
	want := strings.ToLower(prefix)
	var names []string
	for _, e := range es {
		if e.Dir && (!e.Hidden || strings.HasPrefix(prefix, ".")) && strings.HasPrefix(strings.ToLower(e.Name), want) {
			names = append(names, e.Name)
		}
	}
	slices.SortFunc(names, naturalOrder)
	typedDir := typed[:len(typed)-len(prefix)]
	paths := make([]string, 0, min(len(names), suggestions))
	for _, name := range names[:min(len(names), suggestions)] {
		paths = append(paths, typedDir+name)
	}
	return paths, nil
}

func (s *Store) makeFolder(ctx context.Context, parent, name string) (string, error) {
	dir, err := s.clean(parent)
	if err != nil {
		return "", err
	}
	if !s.host.Style.name(name) {
		return "", rpc.Errorf(rpc.CodeInvalidArgument, "%q is not a folder name", name)
	}
	p := s.host.Style.Join(dir, name)
	_, err = timed(ctx, s, dir, func(ctx context.Context) (struct{}, error) {
		info, err := s.cfg.Disk.Stat(ctx, dir)
		if err != nil {
			return struct{}{}, err
		}
		if !info.Dir() {
			return struct{}{}, fmt.Errorf("%s: %w", dir, errNotAFolder)
		}
		return struct{}{}, s.cfg.Disk.Mkdir(ctx, p)
	})
	switch {
	case errors.Is(err, fs.ErrExist):
		return "", rpc.Errorf(rpc.CodeAlreadyExists, "%v", err)
	case err != nil:
		return "", s.folderError(err)
	}
	return p, nil
}

var reasonOf = map[FolderErrorKind]Reason{
	FolderNotFound:     ReasonNotFound,
	FolderNotAFolder:   ReasonNotAFolder,
	FolderDenied:       ReasonUnwritable,
	FolderUnresponsive: ReasonUnresponsive,
}

// Judge is the FolderService's Check on this Node, apart from no roots
// but the ones it holds.
func (s *Store) Judge(ctx context.Context, path string, purpose Purpose) (CheckResult, error) {
	return s.check(ctx, path, purpose, nil, purpose != PurposeClientFolder)
}

// JudgeClientFolder is Judge for an external client's download folder,
// writable too when write is set: the Node deletes the files of a kind
// whose remove leaves them.
func (s *Store) JudgeClientFolder(ctx context.Context, path string, write bool) (CheckResult, error) {
	return s.check(ctx, path, PurposeClientFolder, nil, write)
}

// check judges path as a folder for purpose: absolute, there, a folder,
// apart from the folders purpose reserves, on no network share for
// downloads, and writable by a temp file when write is set.
func (s *Store) check(ctx context.Context, path string, purpose Purpose, others []Other, write bool) (CheckResult, error) {
	dir, err := s.host.Style.Clean(path)
	switch {
	case errors.Is(err, ErrNotAbsolute):
		return CheckResult{Path: path, Refusal: &Refusal{Reason: ReasonNotAbsolute}}, nil
	case err != nil:
		return CheckResult{}, err
	}
	c := CheckResult{Path: dir}
	m, err := s.read(ctx, dir)
	if err != nil {
		reason, ok := reasonOf[folderErrorOf(err)]
		if !ok {
			return CheckResult{}, s.folderError(err)
		}
		c.Refusal = &Refusal{Reason: reason}
		return c, nil
	}
	c.Marker = m.id
	if v, err := timed(ctx, s, dir, func(ctx context.Context) (Volume, error) { return s.cfg.Disk.Volume(ctx, dir) }); err == nil {
		c.Volume = &v
	}
	if r := judge(s.host.Style, dir, s.reserved(purpose, others), s.identity(ctx)); r != nil {
		c.Refusal = r
		return c, nil
	}
	if purpose == PurposeDownloads && c.Volume != nil && c.Volume.Kind == Network {
		c.Refusal = &Refusal{Reason: ReasonNetworkShare}
		return c, nil
	}
	if !write {
		return c, nil
	}
	if _, err := timed(ctx, s, dir, func(ctx context.Context) (struct{}, error) { return struct{}{}, s.tryWrite(ctx, dir) }); err != nil {
		reason := ReasonUnwritable
		if errors.Is(err, errUnresponsive) {
			reason = ReasonUnresponsive
		}
		c.Refusal = &Refusal{Reason: reason}
	}
	return c, nil
}

// reserved is every folder a folder for purpose may not overlap: for a
// root, others and the Node's data and downloads folders; for downloads
// and a client folder, every root the Node holds.
func (s *Store) reserved(purpose Purpose, others []Other) []reserved {
	switch purpose {
	case PurposeRoot:
		return append(apartFrom(others), forRoot(s.host.Data, s.cfg.Downloads())...)
	case PurposeDownloads, PurposeClientFolder:
		s.mu.Lock()
		held := make([]Other, len(s.roots))
		for i, r := range s.roots {
			held[i] = Other{ID: r.id, Path: r.path}
		}
		s.mu.Unlock()
		return apartFrom(held)
	}
	panic(fmt.Sprintf("storing: purpose %d has no rules", purpose))
}

// identity compares two folders by file identity, each stat once under
// the stat deadline. A folder that can't be read is the same as none.
func (s *Store) identity(ctx context.Context) func(a, b string) bool {
	seen := map[string]Info{}
	stat := func(p string) Info {
		if info, ok := seen[p]; ok {
			return info
		}
		info, err := timed(ctx, s, p, func(ctx context.Context) (Info, error) { return s.cfg.Disk.Stat(ctx, p) })
		if err != nil {
			info = nil
		}
		seen[p] = info
		return info
	}
	return func(a, b string) bool {
		ia, ib := stat(a), stat(b)
		return ia != nil && ib != nil && ia.Same(ib)
	}
}

func (s *Store) tryWrite(ctx context.Context, dir string) error {
	var b [8]byte
	_, _ = rand.Read(b[:])
	p := s.host.Style.Join(dir, ".slipstream-check-"+hex.EncodeToString(b[:]))
	f, err := s.cfg.Disk.Create(ctx, p)
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, "slipstream")
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if removeErr := s.cfg.Disk.Remove(ctx, p); err == nil {
		err = removeErr
	}
	return err
}

// verify checks each recorded file in the folder dir, answering the ones
// missing or of another size.
func (s *Store) verify(ctx context.Context, dir string, files []Recorded) ([]MissingFile, error) {
	var missing []MissingFile
	for i, f := range files {
		reason, err := s.sized(ctx, dir, f.Sits, f.Size)
		if err == nil && reason == FileMissing && f.Next != (Rel{}) {
			reason, err = s.sized(ctx, dir, f.Next, f.Size)
		}
		if err != nil {
			return nil, err
		}
		if reason != 0 {
			missing = append(missing, MissingFile{Index: i, Reason: reason})
		}
	}
	return missing, nil
}

func (s *Store) sized(ctx context.Context, dir string, rel Rel, size int64) (MissingReason, error) {
	info, err := timed(ctx, s, dir, func(ctx context.Context) (Info, error) {
		return s.cfg.Disk.Stat(ctx, s.host.Style.Under(dir, rel))
	})
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return FileMissing, nil
	case err != nil:
		return 0, s.folderError(err)
	case info.Dir():
		return FileMissing, nil
	case info.Size() != size:
		return SizeDiffers, nil
	}
	return 0, nil
}

func (s *Store) place(ctx context.Context, root storage.UUID, p Payload, to Rel, allowCopy bool) (Info, Step, error) {
	dir, err := s.resolve(ctx, root, true)
	if err != nil {
		return nil, 0, err
	}
	from, err := s.cfg.Payloads.Path(p.InfoHash, p.FileIndex)
	if err != nil {
		return nil, 0, rpc.Errorf(rpc.CodeUnavailable, "%v", err)
	}
	placed, step, err := placer{s.cfg.Disk, s.host.Style, dir, s.walksOf(root)}.place(ctx, from, to, allowCopy)
	if err != nil {
		return nil, 0, s.refused(ctx, dir, err)
	}
	s.resample(root)
	return placed, step, nil
}

func (s *Store) move(ctx context.Context, root storage.UUID, from, to Rel, carry bool) (Info, []Rel, error) {
	dir, err := s.resolve(ctx, root, false)
	if err != nil {
		return nil, nil, err
	}
	moved, left, err := placer{s.cfg.Disk, s.host.Style, dir, s.walksOf(root)}.move(ctx, from, to, carry)
	if err != nil {
		return nil, nil, s.refused(ctx, dir, err)
	}
	return moved, left, nil
}

// refused is a failed write into a root's folder dir as its caller reads
// it: a folder that stopped taking writes, or answering, is *NotReady, so
// the write waits for the root. The facts follow at once.
func (s *Store) refused(ctx context.Context, dir string, err error) error {
	if _, needsCopy := errors.AsType[*NeedsCopyError](err); needsCopy || errors.Is(err, ErrOccupied) {
		return err
	}
	s.wake()
	ok, werr := timed(ctx, s, dir, func(ctx context.Context) (bool, error) { return s.cfg.Disk.Writable(ctx, dir) })
	switch {
	case errors.Is(werr, errUnresponsive):
		return &NotReady{State: Unresponsive}
	case werr == nil && !ok:
		return &NotReady{State: Unwritable}
	}
	return err
}

func (s *Store) remove(ctx context.Context, root storage.UUID, rel Rel, recursive bool) error {
	dir, err := s.resolve(ctx, root, false)
	if err != nil {
		return err
	}
	rm := s.cfg.Disk.Remove
	if recursive {
		rm = s.cfg.Disk.RemoveAll
	}
	if err := rm(ctx, s.host.Style.Under(dir, rel)); err != nil {
		return s.folderError(err)
	}
	return nil
}

// list names what root's held folder holds; a folder not made yet holds
// nothing.
func (s *Store) list(ctx context.Context, root storage.UUID, folder HeldFolder) ([]string, error) {
	dir, err := s.resolve(ctx, root, false)
	if err != nil {
		return nil, err
	}
	es, err := s.entries(ctx, s.host.Style.Join(dir, folder.String()))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, s.folderError(err)
	}
	names := make([]string, 0, len(es))
	for _, e := range es {
		names = append(names, e.Name)
	}
	return names, nil
}

func volumeWire(v Volume) *storagev1.Volume {
	return &storagev1.Volume{
		Id: v.ID, Name: v.Name, Mount: v.Mount, Kind: kindWire(v.Kind), Total: v.Total, Free: v.Free, SampledAtUnixMs: v.SampledAt.UnixMilli(),
	}
}

// kindWire is k on the wire, unspecified for a root not sampled yet.
func kindWire(k Kind) storagev1.VolumeKind {
	if k == 0 {
		return storagev1.VolumeKind_VOLUME_KIND_UNSPECIFIED
	}
	return k.Wire()
}

func (s PathStyle) name(n string) bool {
	seps := "/\x00"
	if s == Windows {
		seps = `/\` + "\x00"
	}
	return n != "" && n != "." && n != ".." && !strings.ContainsAny(n, seps)
}
