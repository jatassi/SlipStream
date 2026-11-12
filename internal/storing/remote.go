package storing

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/jatassi/slipstream/internal/media"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1/storagev1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// Remote is StorageService as the Controller reaches it on a Node, in
// process for its own roots. It parses answers into domain types: a
// folder's error is *FolderError and a root that may not take a write now
// is *NotReady. Any other error is the call's, for rpc.NotSent and
// rpc.Lost to read.
type Remote func(role.NodeID) storagev1connect.StorageServiceHandler

// Folders is FolderService as the Controller reaches it on any Node, in
// process for its own folders, parsing answers as Remote does.
type Folders func(role.NodeID) storagev1connect.FolderServiceHandler

// ErrInvalid is a path or folder name the Node can't use: a path that isn't
// absolute, or a name that isn't one folder's.
var ErrInvalid = errors.New("the Node can't use that path or name")

func (f Folders) Browse(ctx context.Context, node role.NodeID, path string, showHidden bool) (Listing, error) {
	res, err := f(node).Browse(ctx, &storagev1.BrowseRequest{Path: path, ShowHidden: showHidden})
	if err != nil {
		return Listing{}, folderAnswer(err)
	}
	l := Listing{Path: res.GetPath(), Parent: res.GetParent(), Hidden: int(res.GetHidden()), Service: res.GetService()}
	if l.Volume, err = parseVolume(res.GetVolume()); err != nil {
		return Listing{}, err
	}
	for _, seg := range res.GetSegments() {
		l.Segments = append(l.Segments, Segment{Name: seg.GetName(), Path: seg.GetPath()})
	}
	for _, f := range res.GetFolders() {
		l.Folders = append(l.Folders, Folder{Name: f.GetName(), Path: f.GetPath(), Readable: f.GetReadable()})
	}
	for _, p := range res.GetPlaces() {
		place := Place{Name: p.GetName(), Path: p.GetPath(), Free: p.GetFree(), Total: p.GetTotal()}
		if p.GetKind() != storagev1.VolumeKind_VOLUME_KIND_UNSPECIFIED {
			if place.Kind, err = ParseKind(p.GetKind()); err != nil {
				return Listing{}, err
			}
		}
		if len(p.GetRootId()) > 0 {
			if place.Root, err = storage.UUIDFromBytes(p.GetRootId()); err != nil {
				return Listing{}, fmt.Errorf("a place's root id: %w", err)
			}
		}
		l.Places = append(l.Places, place)
	}
	return l, nil
}

func (f Folders) Suggest(ctx context.Context, node role.NodeID, text string) ([]string, error) {
	res, err := f(node).Suggest(ctx, &storagev1.SuggestRequest{Text: text})
	if err != nil {
		return nil, folderAnswer(err)
	}
	return res.GetPaths(), nil
}

func (f Folders) MakeFolder(ctx context.Context, node role.NodeID, parent, name string) (string, error) {
	res, err := f(node).MakeFolder(ctx, &storagev1.MakeFolderRequest{Parent: parent, Name: name})
	if err != nil {
		return "", folderAnswer(err)
	}
	return res.GetPath(), nil
}

// Check judges path on node for purpose; others are the roots a root may
// not overlap.
func (f Folders) Check(ctx context.Context, node role.NodeID, path string, purpose Purpose, others []Other) (CheckResult, error) {
	req := &storagev1.CheckRequest{Path: path, Purpose: purpose.Wire()}
	for _, o := range others {
		req.Others = append(req.Others, &storagev1.Other{Id: o.ID.Bytes(), Path: o.Path})
	}
	res, err := f(node).Check(ctx, req)
	if err != nil {
		return CheckResult{}, answer(err)
	}
	c := CheckResult{Path: res.GetPath()}
	if c.Volume, err = parseVolume(res.GetVolume()); err != nil {
		return CheckResult{}, err
	}
	if res.GetRefusal() != storagev1.Reason_REASON_UNSPECIFIED {
		reason, err := reasons.parse(res.GetRefusal())
		if err != nil {
			return CheckResult{}, err
		}
		c.Refusal = &Refusal{Reason: reason}
		if len(res.GetRootId()) > 0 {
			if c.Refusal.Root, err = storage.UUIDFromBytes(res.GetRootId()); err != nil {
				return CheckResult{}, fmt.Errorf("the overlapped root's id: %w", err)
			}
		}
	}
	if len(res.GetMarker()) > 0 {
		if c.Marker, err = storage.UUIDFromBytes(res.GetMarker()); err != nil {
			return CheckResult{}, fmt.Errorf("the marker's root id: %w", err)
		}
	}
	return c, nil
}

// Verify checks files at path on node in batches of 500, calling progress
// with the count checked after each, and answers the missing ones.
func (r Remote) Verify(ctx context.Context, node role.NodeID, path string, files []Recorded, progress func(checked int)) ([]MissingFile, error) {
	return r.verify(ctx, node, &storagev1.VerifyRequest{At: &storagev1.VerifyRequest_Path{Path: path}}, files, progress)
}

// VerifyIn is Verify under the root at, its marker read again first: a
// root unmounted since a walk is *NotReady, never every file missing.
func (r Remote) VerifyIn(ctx context.Context, at At, files []Recorded, progress func(checked int)) ([]MissingFile, error) {
	return r.verify(ctx, at.Node, &storagev1.VerifyRequest{At: &storagev1.VerifyRequest_RootId{RootId: at.Root.Bytes()}}, files, progress)
}

// verify sends files in batches to the folder or root where names.
func (r Remote) verify(ctx context.Context, node role.NodeID, where *storagev1.VerifyRequest, files []Recorded, progress func(checked int)) ([]MissingFile, error) {
	var missing []MissingFile
	for at := 0; at < len(files); at += verifyBatch {
		batch := files[at:min(at+verifyBatch, len(files))]
		req := &storagev1.VerifyRequest{At: where.GetAt()}
		for _, f := range batch {
			req.Files = append(req.Files, &storagev1.Recorded{Path: f.Sits.String(), Size: f.Size, Next: f.Next.String()})
		}
		res, err := r(node).Verify(ctx, req)
		if err != nil {
			return nil, answer(err)
		}
		for _, m := range res.GetMissing() {
			reason, err := missingReasons.parse(m.GetReason())
			if err != nil {
				return nil, err
			}
			missing = append(missing, MissingFile{Index: at + int(m.GetIndex()), Reason: reason})
		}
		progress(at + len(batch))
	}
	return missing, nil
}

// Placed is a file the Node put in a root, and what a scan compares of it.
type Placed struct {
	Size int64
	// Step is how the file came to be there: Rename for a Move.
	Step Step
	Seen Seen
	// Left are the companions a carrying Move found another file at the
	// new name of, where each now lies.
	Left []Rel
}

// Place puts payload p at to under the root by a link, a clone or, with
// allowCopy, a copy. Another file there is ErrOccupied. A place that
// needs a copy without allowCopy is *NeedsCopyError, and a clone or copy
// the volume has no room for *NeedsSpaceError.
func (r Remote) Place(ctx context.Context, at At, p Payload, to Rel, allowCopy bool) (Placed, error) {
	res, err := r(at.Node).Place(ctx, &storagev1.PlaceRequest{
		RootId: at.Root.Bytes(), InfoHash: p.InfoHash[:], FileIndex: int64(p.FileIndex), To: to.String(), AllowCopy: allowCopy,
	})
	if err != nil {
		return Placed{}, placeAnswer(err)
	}
	step, err := steps.parse(res.GetStep())
	if err != nil {
		return Placed{}, err
	}
	return Placed{Size: res.GetSize(), Step: step, Seen: seenOf(res.GetIdentity())}, nil
}

func placeAnswer(err error) error {
	if d, ok := rpc.DetailOf[*storagev1.NeedsCopy](err); ok {
		return &NeedsCopyError{Size: d.GetSize()}
	}
	if _, ok := rpc.DetailOf[*storagev1.NeedsSpace](err); ok {
		return &NeedsSpaceError{}
	}
	return occupied(err)
}

// Move renames from to to within the root, and with carry from's
// companions after it. Another file at to is ErrOccupied.
func (r Remote) Move(ctx context.Context, at At, from, to Rel, carry bool) (Placed, error) {
	res, err := r(at.Node).Move(ctx, &storagev1.MoveRequest{RootId: at.Root.Bytes(), From: from.String(), To: to.String(), Carry: carry})
	if err != nil {
		return Placed{}, occupied(err)
	}
	left, err := relsOf(res.GetLeft())
	if err != nil {
		return Placed{}, fmt.Errorf("a companion left: %w", err)
	}
	return Placed{Size: res.GetSize(), Step: Rename, Seen: seenOf(res.GetIdentity()), Left: left}, nil
}

// Survey answers each planned rename within the root, in batches of 2000:
// the files at its target, compared without case, and with companions its
// source's companions. A root that isn't ready is *NotReady.
func (r Remote) Survey(ctx context.Context, at At, planned []Planned, companions bool) ([]Surveyed, error) {
	out := make([]Surveyed, 0, len(planned))
	for i := 0; i < len(planned); i += surveyBatch {
		req := &storagev1.SurveyRequest{RootId: at.Root.Bytes(), Companions: companions}
		for _, p := range planned[i:min(i+surveyBatch, len(planned))] {
			req.Planned = append(req.Planned, &storagev1.Planned{From: p.From.String(), To: p.To.String()})
		}
		res, err := r(at.Node).Survey(ctx, req)
		if err != nil {
			return nil, answer(err)
		}
		for _, w := range res.GetSurveyed() {
			var s Surveyed
			for _, o := range w.GetOccupants() {
				path, err := ParseRel(o.GetPath())
				if err != nil {
					return nil, fmt.Errorf("a file at a target: %w", err)
				}
				s.Occupants = append(s.Occupants, Occupant{Path: path, Seen: seenOf(o.GetIdentity())})
			}
			if s.Companions, err = relsOf(w.GetCompanions()); err != nil {
				return nil, fmt.Errorf("a companion: %w", err)
			}
			out = append(out, s)
		}
	}
	return out, nil
}

// Prune removes folder and each parent below the root while it is empty,
// never a held folder, a kept folder or a parent of one, and answers the
// folders removed, innermost first.
func (r Remote) Prune(ctx context.Context, at At, folder Rel, keep []Rel) ([]Rel, error) {
	req := &storagev1.PruneRequest{RootId: at.Root.Bytes(), Folder: folder.String()}
	for _, k := range keep {
		req.Keep = append(req.Keep, k.String())
	}
	res, err := r(at.Node).Prune(ctx, req)
	if err != nil {
		return nil, answer(err)
	}
	removed, err := relsOf(res.GetRemoved())
	if err != nil {
		return nil, fmt.Errorf("a folder removed: %w", err)
	}
	return removed, nil
}

func relsOf(ss []string) ([]Rel, error) {
	var out []Rel
	for _, s := range ss {
		rel, err := ParseRel(s)
		if err != nil {
			return nil, err
		}
		out = append(out, rel)
	}
	return out, nil
}

// Walk lists one page of the root's video files after the cursor, up to
// limit of them, 1 to 1000. A root that isn't ready is *NotReady, and one
// that stops answering mid-page a *FolderError.
func (r Remote) Walk(ctx context.Context, at At, after Cursor, limit int) (Page, error) {
	res, err := r(at.Node).Walk(ctx, &storagev1.WalkRequest{RootId: at.Root.Bytes(), After: []byte(after.b), Limit: int32(limit)}) //nolint:gosec // the Node refuses a limit past 1000
	if err != nil {
		return Page{}, answer(err)
	}
	p := Page{Next: Cursor{string(res.GetNext())}, Done: len(res.GetNext()) == 0, CaseFold: res.GetCaseInsensitive()}
	for _, e := range res.GetEntries() {
		path, err := ParseRel(e.GetPath())
		if err != nil {
			return Page{}, fmt.Errorf("a walked file: %w", err)
		}
		p.Entries = append(p.Entries, Walked{Path: path, Size: e.GetSize(), Seen: seenOf(e.GetIdentity())})
	}
	for _, u := range res.GetUnread() {
		reason, err := unreadReasons.parse(u.GetReason())
		if err != nil {
			return Page{}, err
		}
		p.Unread = append(p.Unread, Unread{Path: u.GetPath(), Reason: reason})
	}
	return p, nil
}

// Probe reads path's media record under the root. A file ffprobe can't
// read is *media.UnreadableError, one gone since the walk a *FolderError
// NotFound, and a root that isn't ready *NotReady.
func (r Remote) Probe(ctx context.Context, at At, path Rel) (media.Record, error) {
	res, err := r(at.Node).Probe(ctx, &storagev1.ProbeRequest{RootId: at.Root.Bytes(), Path: path.String()})
	if d, ok := rpc.DetailOf[*storagev1.Unprobed](err); ok {
		return media.Record{}, &media.UnreadableError{Line: d.GetLine()}
	}
	if err != nil {
		return media.Record{}, answer(err)
	}
	var rec media.Record
	if err := json.Unmarshal(res.GetRecord(), &rec); err != nil {
		return media.Record{}, fmt.Errorf("read the media record: %w", err)
	}
	return rec, nil
}

func seenOf(id *storagev1.Identity) Seen {
	return Seen{
		Modified: time.UnixMilli(id.GetModifiedUnixMs()),
		ID:       FileID{Volume: id.GetVolumeId(), File: hex.EncodeToString(id.GetFileId())},
	}
}

// Remove removes rel from the root; a file already gone wraps
// fs.ErrNotExist.
func (r Remote) Remove(ctx context.Context, at At, rel Rel) error {
	_, err := r(at.Node).Remove(ctx, &storagev1.RemoveRequest{RootId: at.Root.Bytes(), Path: rel.String()})
	if err == nil {
		return nil
	}
	err = answer(err)
	if fe, ok := errors.AsType[*FolderError](err); ok && fe.Kind == FolderNotFound {
		return fmt.Errorf("%w: %w", fs.ErrNotExist, fe)
	}
	return err
}

// RemoveAll removes the folder rel from the root and everything in it; a
// folder already gone is removed.
func (r Remote) RemoveAll(ctx context.Context, at At, rel Rel) error {
	if _, err := r(at.Node).Remove(ctx, &storagev1.RemoveRequest{RootId: at.Root.Bytes(), Path: rel.String(), Recursive: true}); err != nil {
		return answer(err)
	}
	return nil
}

func (r Remote) List(ctx context.Context, at At, folder HeldFolder) ([]string, error) {
	res, err := r(at.Node).List(ctx, &storagev1.ListRequest{RootId: at.Root.Bytes(), Folder: folder.Wire()})
	if err != nil {
		return nil, answer(err)
	}
	return res.GetNames(), nil
}

// Sample has node judge every root and sample its volume now, and answers
// when that pass began on node's clock, as its facts name it.
func (r Remote) Sample(ctx context.Context, node role.NodeID) (time.Time, error) {
	res, err := r(node).Sample(ctx, &storagev1.SampleRequest{})
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(res.GetPassedAtUnixMs()), nil
}

// answer is a call's error as the domain reads it: the *FolderError or
// *NotReady its detail carries, else the call's own.
func answer(err error) error {
	if d, ok := rpc.DetailOf[*storagev1.FolderError](err); ok {
		kind, perr := folderErrorKinds.parse(d.GetKind())
		if perr != nil {
			return fmt.Errorf("%w (%w)", err, perr)
		}
		return &FolderError{Kind: kind, Text: d.GetOsText(), Account: d.GetAccount()}
	}
	if d, ok := rpc.DetailOf[*storagev1.NotReady](err); ok {
		nr := &NotReady{Draining: d.GetDraining(), Unknown: d.GetUnknown()}
		if d.GetState() != storagev1.RootState_ROOT_STATE_UNSPECIFIED {
			state, perr := ParseRootState(d.GetState())
			if perr != nil {
				return fmt.Errorf("%w (%w)", err, perr)
			}
			nr.State = state
		}
		return nr
	}
	return err
}

// folderAnswer is a folder op's error as the domain reads it: a path or
// name refused is ErrInvalid, and a folder already there fs.ErrExist.
func folderAnswer(err error) error {
	if code := rpc.CodeOf(err); code == rpc.CodeInvalidArgument {
		return fmt.Errorf("%w: %w", ErrInvalid, err)
	} else if code == rpc.CodeAlreadyExists {
		return fmt.Errorf("%w: %w", fs.ErrExist, err)
	}
	return answer(err)
}

func occupied(err error) error {
	if rpc.CodeOf(err) == rpc.CodeAlreadyExists {
		return fmt.Errorf("%w: %w", ErrOccupied, err)
	}
	return answer(err)
}

func parseVolume(w *storagev1.Volume) (*Volume, error) {
	if w == nil {
		return nil, nil
	}
	v := &Volume{ID: w.GetId(), Name: w.GetName(), Mount: w.GetMount(), Total: w.GetTotal(), Free: w.GetFree(), SampledAt: time.UnixMilli(w.GetSampledAtUnixMs())}
	kind, err := ParseKind(w.GetKind())
	if err != nil {
		return nil, err
	}
	v.Kind = kind
	return v, nil
}
