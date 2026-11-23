package downloading

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/bencode"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

var unsafeInName = strings.NewReplacer(`\`, "_", "/", "_", ":", "_", "*", "_", "?", "_", `"`, "_", "<", "_", ">", "_", "|", "_")

// folderName is a Content Type's name as a folder name on every OS.
func folderName(name string) string { return unsafeInName.Replace(name) }

// builtinFolder is the folder the built-in client keeps torrent h of
// Content Type ct in, under its downloads folder dir.
func builtinFolder(dir, ct string, h bencode.InfoHash) string {
	return filepath.Join(dir, folderName(ct), h.String())
}

// errNoClient is a Download Client this Node runs no actor for, or whose
// download folder its actor doesn't know yet.
var errNoClient = errors.New("this Node hasn't read the Download Client's folder yet")

// clientBase is the download folder of a Download Client on this Node:
// the built-in client's downloads folder when client is nil, else the
// external client's as this Node sees it.
func (n *Node) clientBase(client *[16]byte) (string, error) {
	if client == nil {
		return n.builtin.Dir(), nil
	}
	id, err := storage.UUIDFromBytes(client[:])
	if err != nil {
		return "", err
	}
	a := n.actor(id)
	if a == nil {
		return "", errNoClient
	}
	f, ok := a.Folder()
	if !ok {
		return "", errNoClient
	}
	return f, nil
}

// StageDownload is the folder a Transfer stages in for the torrent's
// Download Client: a staging folder in the client's download folder, so
// the payload's place into the client's folder stays on one volume.
func (n *Node) StageDownload(_ context.Context, client *[16]byte, _ string, _ bencode.InfoHash, transfer storage.UUID) (string, error) {
	base, err := n.clientBase(client)
	if err != nil {
		return "", err
	}
	folder := filepath.Join(base, storing.StagingDir, transfer.String())
	if err := os.MkdirAll(folder, 0o750); err != nil {
		return "", fmt.Errorf("make the staging folder: %w", err)
	}
	return folder, nil
}

// Arriving is where a payload file a Move placed in a Download Client's
// folder lies on this Node: under the built-in client's folder for the
// torrent, or under the external client's Content Type subfolder.
func (n *Node) Arriving(a storing.Arriving) (string, error) {
	base, err := n.clientBase(a.Client)
	if err != nil {
		return "", err
	}
	folder := filepath.Join(base, folderName(a.ContentType))
	if a.Client == nil {
		folder = builtinFolder(base, a.ContentType, a.InfoHash)
	}
	return filepath.Join(folder, filepath.FromSlash(a.Path.String())), nil
}

// PlacePayload puts a Move's payload in the folder req's Download Client
// keeps the torrent in: each staged file moved there, or each file of the
// torrent another client on this Node holds linked, cloned or copied. A
// file already placed counts as placed, so placing again converges. It
// answers the folder as the client sees it.
func (n *Node) PlacePayload(ctx context.Context, req *dlv1.PlacePayloadRequest) (*dlv1.PlacePayloadResponse, error) {
	var h bencode.InfoHash
	if len(req.GetInfoHash()) != len(h) {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "an info hash is %d bytes, not %d", len(h), len(req.GetInfoHash()))
	}
	copy(h[:], req.GetInfoHash())
	base, under, folder, err := n.placeTarget(req.GetClientId(), req.GetContentType(), h)
	if err != nil {
		return nil, err
	}
	to := func(p string) (storing.Rel, error) {
		rel, err := storing.ParseRel(path.Join(under, p))
		if err != nil {
			return storing.Rel{}, rpc.Errorf(rpc.CodeInvalidArgument, "the file %q: %v", p, err)
		}
		return rel, nil
	}
	switch from := req.GetFrom().(type) {
	case *dlv1.PlacePayloadRequest_Staged:
		err = n.placeStaged(ctx, base, from.Staged, to)
	case *dlv1.PlacePayloadRequest_Held:
		err = n.placeHeld(ctx, base, h, from.Held, req.GetAllowCopy(), to)
	default:
		err = rpc.Errorf(rpc.CodeInvalidArgument, "a place names no payload")
	}
	if err != nil {
		return nil, err
	}
	return &dlv1.PlacePayloadResponse{Folder: folder}, nil
}

// placeTarget is where a payload for h of ct goes in the client clientID
// names, the built-in one when it names none: the folder files are placed
// under, their path under it, and the save folder as the client sees it.
func (n *Node) placeTarget(clientID []byte, ct string, h bencode.InfoHash) (base, under, folder string, err error) {
	if len(clientID) == 0 {
		return n.builtin.Dir(), path.Join(folderName(ct), h.String()), builtinFolder(n.builtin.Dir(), ct, h), nil
	}
	id, err := storage.UUIDFromBytes(clientID)
	if err != nil {
		return "", "", "", rpc.Errorf(rpc.CodeInvalidArgument, "client id: %v", err)
	}
	a := n.actor(id)
	var known, added bool
	if a != nil {
		base, known = a.Folder()
		folder, added = a.AddFolder(ct)
	}
	if !known || !added {
		return "", "", "", rpc.Errorf(rpc.CodeUnavailable, "%v", errNoClient)
	}
	return base, folderName(ct), folder, nil
}

func (n *Node) placeStaged(ctx context.Context, base string, staged *dlv1.StagedPayload, to func(string) (storing.Rel, error)) error {
	transfer, err := storage.UUIDFromBytes(staged.GetTransferId())
	if err != nil {
		return rpc.Errorf(rpc.CodeInvalidArgument, "transfer id: %v", err)
	}
	for _, f := range staged.GetFiles() {
		dst, err := to(f.GetPath())
		if err != nil {
			return err
		}
		src, err := storing.ParseRel(path.Join(storing.StagingDir, transfer.String(), strconv.Itoa(int(f.GetPosition()))+".part"))
		if err != nil {
			return rpc.Errorf(rpc.CodeInvalidArgument, "position %d: %v", f.GetPosition(), err)
		}
		if _, err := storing.MoveUnder(ctx, n.cfg.Disk, base, src, dst); err != nil {
			return storing.WireError(err)
		}
	}
	return nil
}

func (n *Node) placeHeld(ctx context.Context, base string, h bencode.InfoHash, held *dlv1.HeldPayload, allowCopy bool, to func(string) (storing.Rel, error)) error {
	for _, f := range held.GetFiles() {
		dst, err := to(f.GetPath())
		if err != nil {
			return err
		}
		src, err := n.Path(h, int(f.GetIndex()))
		if err != nil {
			return rpc.Errorf(rpc.CodeFailedPrecondition, "file %d of the torrent: %v", f.GetIndex(), err)
		}
		if _, _, err := storing.PlaceUnder(ctx, n.cfg.Disk, base, src, dst, allowCopy); err != nil {
			return storing.WireError(err)
		}
	}
	return nil
}

// DiscardPayload removes what a Move's place may have put in req's
// Download Client's folder before an arriving Torrent recorded it: each
// file at its place whose size is the torrent's, and the folders that
// leaves empty under the Content Type's, or under the client's folder for
// an adopted torrent.
func (n *Node) DiscardPayload(ctx context.Context, req *dlv1.DiscardPayloadRequest) (*dlv1.DiscardPayloadResponse, error) {
	var h bencode.InfoHash
	if len(req.GetInfoHash()) != len(h) {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "an info hash is %d bytes, not %d", len(h), len(req.GetInfoHash()))
	}
	copy(h[:], req.GetInfoHash())
	base, under, _, err := n.placeTarget(req.GetClientId(), req.GetContentType(), h)
	if err != nil {
		return nil, err
	}
	var keep storing.Rel
	if ct := req.GetContentType(); ct != "" {
		if keep, err = storing.ParseRel(folderName(ct)); err != nil {
			return nil, rpc.Errorf(rpc.CodeInvalidArgument, "the Content Type %q: %v", ct, err)
		}
	}
	for _, f := range req.GetFiles() {
		rel, err := storing.ParseRel(path.Join(under, f.GetPath()))
		if err != nil {
			return nil, rpc.Errorf(rpc.CodeInvalidArgument, "the file %q: %v", f.GetPath(), err)
		}
		if req.GetHeld() && n.isHeld(ctx, h, int(f.GetIndex()), n.cfg.Disk.Host().Style.Under(base, rel)) {
			continue
		}
		if err := storing.DiscardUnder(ctx, n.cfg.Disk, base, keep, rel, f.GetSize()); err != nil {
			return nil, storing.WireError(err)
		}
	}
	return &dlv1.DiscardPayloadResponse{}, nil
}

// isHeld is whether the file at at is file index of torrent h as the
// client on this Node that holds it has it.
func (n *Node) isHeld(ctx context.Context, h bencode.InfoHash, index int, at string) bool {
	held, err := n.Path(h, index)
	if err != nil {
		return false
	}
	a, errA := n.cfg.Disk.Resolve(ctx, at)
	b, errB := n.cfg.Disk.Resolve(ctx, held)
	return errA == nil && errB == nil && a == b
}
