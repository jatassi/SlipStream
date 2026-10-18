package storing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

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

func (r Remote) Browse(ctx context.Context, node role.NodeID, path string, showHidden bool) (Listing, error) {
	res, err := r(node).Browse(ctx, &storagev1.BrowseRequest{Path: path, ShowHidden: showHidden})
	if err != nil {
		return Listing{}, answer(err)
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

func (r Remote) Suggest(ctx context.Context, node role.NodeID, text string) ([]string, error) {
	res, err := r(node).Suggest(ctx, &storagev1.SuggestRequest{Text: text})
	if err != nil {
		return nil, answer(err)
	}
	return res.GetPaths(), nil
}

func (r Remote) MakeFolder(ctx context.Context, node role.NodeID, parent, name string) (string, error) {
	res, err := r(node).MakeFolder(ctx, &storagev1.MakeFolderRequest{Parent: parent, Name: name})
	if err != nil {
		return "", answer(err)
	}
	return res.GetPath(), nil
}

func (r Remote) Check(ctx context.Context, node role.NodeID, path string, others []Other) (CheckResult, error) {
	req := &storagev1.CheckRequest{Path: path}
	for _, o := range others {
		req.Others = append(req.Others, &storagev1.Other{Id: o.ID.Bytes(), Path: o.Path})
	}
	res, err := r(node).Check(ctx, req)
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
	var missing []MissingFile
	for at := 0; at < len(files); at += verifyBatch {
		batch := files[at:min(at+verifyBatch, len(files))]
		req := &storagev1.VerifyRequest{Path: path}
		for _, f := range batch {
			req.Files = append(req.Files, &storagev1.Recorded{Path: f.Path.String(), Size: f.Size})
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

// Place puts payload p at to under the root, answering the file's size.
// Another file there is ErrOccupied.
func (r Remote) Place(ctx context.Context, at At, p Payload, to Rel) (int64, error) {
	res, err := r(at.Node).Place(ctx, &storagev1.PlaceRequest{
		RootId: at.Root.Bytes(), InfoHash: p.InfoHash[:], FileIndex: int32(p.FileIndex), To: to.String(), //nolint:gosec // a torrent's file index fits
	})
	if err != nil {
		return 0, occupied(err)
	}
	return res.GetSize(), nil
}

// Move renames from to to within the root, answering the file's size.
// Another file at to is ErrOccupied.
func (r Remote) Move(ctx context.Context, at At, from, to Rel) (int64, error) {
	res, err := r(at.Node).Move(ctx, &storagev1.MoveRequest{RootId: at.Root.Bytes(), From: from.String(), To: to.String()})
	if err != nil {
		return 0, occupied(err)
	}
	return res.GetSize(), nil
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

func (r Remote) List(ctx context.Context, at At, folder HeldFolder) ([]string, error) {
	res, err := r(at.Node).List(ctx, &storagev1.ListRequest{RootId: at.Root.Bytes(), Folder: folder.Wire()})
	if err != nil {
		return nil, answer(err)
	}
	return res.GetNames(), nil
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
