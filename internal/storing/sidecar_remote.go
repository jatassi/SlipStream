package storing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jatassi/slipstream/internal/media"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1/storagev1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// SidecarRemote is SidecarService as the Controller reaches it on each
// Node, in process for its own roots.
type SidecarRemote func(role.NodeID) SidecarCalls

// SidecarCalls is SidecarService on one Node in Go types. A root that may
// not take a write now is *NotReady and a folder's error is *FolderError;
// any other error is the call's, for rpc.NotSent and rpc.Lost to read.
// Every byte slice a write takes is bytes the Controller already parsed.
type SidecarCalls interface {
	Build(ctx context.Context, b MapBuild) (MapAnswer, error)
	ReadMap(ctx context.Context, file, root storage.UUID, path Rel, id Identity) ([]byte, error)
	PruneMaps(ctx context.Context, after, before storage.UUID, keep []storage.UUID) (int, error)
	List(ctx context.Context, root storage.UUID, videos media.Videos, ls []SidecarListing) ([]SidecarListed, error)
	Read(ctx context.Context, root storage.UUID, path Rel, sha [32]byte) ([]byte, error)
	Write(ctx context.Context, w SidecarWrite) (Rel, error)
	Remove(ctx context.Context, root storage.UUID, path Rel, sha [32]byte) error
}

// SidecarCallsOf is SidecarCalls over the service h.
func SidecarCallsOf(h storagev1connect.SidecarServiceHandler) SidecarCalls { return sidecarCalls{h} }

type sidecarCalls struct {
	h storagev1connect.SidecarServiceHandler
}

var (
	// ErrStale is a file whose identity or size isn't the one asked.
	ErrStale = errors.New("the file changed since it was last read")
	// ErrChanged is other bytes at a path than the hash asked.
	ErrChanged = errors.New("other bytes are at the path")
)

// ChangedError is other bytes at a path: those with SHA256, or none.
type ChangedError struct{ SHA256 [32]byte }

func (e *ChangedError) Error() string { return fmt.Sprintf("%v: sha256 %x", ErrChanged, e.SHA256) }
func (e *ChangedError) Unwrap() error { return ErrChanged }

// OccupiedError is a Sidecar's name holding other bytes than the write
// may replace, those with SHA256.
type OccupiedError struct{ SHA256 [32]byte }

func (e *OccupiedError) Error() string { return fmt.Sprintf("%v: sha256 %x", ErrOccupied, e.SHA256) }
func (e *OccupiedError) Unwrap() error { return ErrOccupied }

type MapState uint8

const (
	MapBuilt MapState = iota + 1
	MapBuilding
	MapQueued
	MapFailed
)

type MapFault uint8

const (
	NoAudio MapFault = iota + 1
	DecodeFailed
	NodeFault
)

// MapBuild asks a Node for the map of the file at Path under Root.
type MapBuild struct {
	File, Root storage.UUID
	Path       Rel
	// Stream is the audio ordinal, -1 for the default track, else the
	// first.
	Stream int
	// Slots is how many builds the Node may run at once now; Rank orders
	// the builds that wait, 0 first.
	Slots, Rank int
}

type MapAnswer struct {
	State    MapState
	Identity Identity
	Frames   int
	Density  float64
	Stream   int
	Fault    MapFault
	// Words is the decoder's first error line on a fault.
	Words string
}

// SidecarListing asks for the companions of Video, and of each stem in
// Also, the places its Sidecars may sit: a rename's From, a parked row's
// path.
type SidecarListing struct {
	Video Rel
	Also  []Rel
}

type SidecarListed struct {
	Video        Rel
	VideoMissing bool
	Companions   []SidecarCompanion
}

type SidecarCompanion struct {
	Path   Rel
	Tail   string
	Size   int64
	SHA256 [32]byte
	// Ledger is whether the Node's ledger holds the hash for the root.
	Ledger bool
}

// SidecarWrite writes Bytes beside Video as <stem>.<Tail>. Replaces is
// the hash of the bytes it may overwrite, nil for none.
type SidecarWrite struct {
	Root     storage.UUID
	Video    Rel
	Identity Identity
	Tail     string
	Bytes    []byte
	SHA256   [32]byte
	Replaces *[32]byte
}

var mapStates = map[storagev1.SpeechMapState]MapState{
	storagev1.SpeechMapState_SPEECH_MAP_STATE_BUILT:    MapBuilt,
	storagev1.SpeechMapState_SPEECH_MAP_STATE_BUILDING: MapBuilding,
	storagev1.SpeechMapState_SPEECH_MAP_STATE_QUEUED:   MapQueued,
	storagev1.SpeechMapState_SPEECH_MAP_STATE_FAILED:   MapFailed,
}

var mapFaults = map[storagev1.SpeechMapFault]MapFault{
	storagev1.SpeechMapFault_SPEECH_MAP_FAULT_NO_AUDIO:      NoAudio,
	storagev1.SpeechMapFault_SPEECH_MAP_FAULT_DECODE_FAILED: DecodeFailed,
	storagev1.SpeechMapFault_SPEECH_MAP_FAULT_NODE_FAULT:    NodeFault,
}

// Build asks the Node for b's map and answers within the Node's build wait:
// built, still building, queued, or failed with its fault.
func (c sidecarCalls) Build(ctx context.Context, b MapBuild) (MapAnswer, error) {
	res, err := c.h.BuildSpeechMap(ctx, &storagev1.BuildSpeechMapRequest{
		FileId: b.File.Bytes(), RootId: b.Root.Bytes(), Path: b.Path.String(), Stream: int32(b.Stream), Slots: int32(b.Slots), Rank: int32(b.Rank), //nolint:gosec // an ordinal and counts far under 2³¹
	})
	if err != nil {
		return MapAnswer{}, answer(err)
	}
	state, ok := mapStates[res.GetState()]
	if !ok {
		return MapAnswer{}, fmt.Errorf("a speech map state the Controller doesn't know: %v", res.GetState())
	}
	a := MapAnswer{State: state, Frames: int(res.GetFrames()), Density: res.GetDensity(), Stream: int(res.GetStream()), Words: res.GetWords()}
	switch state {
	case MapBuilt:
		a.Identity = Identity{Seen: seenOf(res.GetIdentity()), Size: res.GetSize()}
	case MapFailed:
		if a.Fault, ok = mapFaults[res.GetFault()]; !ok {
			return MapAnswer{}, fmt.Errorf("a speech map fault the Controller doesn't know: %v", res.GetFault())
		}
	case MapBuilding, MapQueued:
	}
	return a, nil
}

// ReadMap is the encoded map of file at path under root, ErrStale when the
// Node holds none of the file as id names it.
func (c sidecarCalls) ReadMap(ctx context.Context, file, root storage.UUID, path Rel, id Identity) ([]byte, error) {
	res, err := c.h.ReadSpeechMap(ctx, &storagev1.ReadSpeechMapRequest{
		FileId: file.Bytes(), RootId: root.Bytes(), Path: path.String(), Identity: id.wire(), Size: id.Size,
	})
	if err != nil {
		return nil, answer(err)
	}
	if res.GetStale() {
		return nil, ErrStale
	}
	return res.GetMap(), nil
}

// PruneMaps deletes the Node's maps of the files in [after, before) not in
// keep; a zero before has no end. It answers how many went.
func (c sidecarCalls) PruneMaps(ctx context.Context, after, before storage.UUID, keep []storage.UUID) (int, error) {
	req := &storagev1.PruneSpeechMapsRequest{After: after.Bytes()}
	if !before.IsZero() {
		req.Before = before.Bytes()
	}
	for _, k := range keep {
		req.Keep = append(req.Keep, k.Bytes())
	}
	res, err := c.h.PruneSpeechMaps(ctx, req)
	if err != nil {
		return 0, answer(err)
	}
	return int(res.GetRemoved()), nil
}

// List lists the subtitle companions of each video under root, one
// answer per listing in order, at most the Node's survey batch at once.
func (c sidecarCalls) List(ctx context.Context, root storage.UUID, videos media.Videos, ls []SidecarListing) ([]SidecarListed, error) {
	req := &storagev1.ListSidecarsRequest{RootId: root.Bytes(), VideoExtensions: videos.Extensions()}
	for _, l := range ls {
		w := &storagev1.SidecarListing{Video: l.Video.String()}
		for _, a := range l.Also {
			w.Also = append(w.Also, a.String())
		}
		req.Videos = append(req.Videos, w)
	}
	res, err := c.h.ListSidecars(ctx, req)
	if err != nil {
		return nil, answer(err)
	}
	if len(res.GetListed()) != len(ls) {
		return nil, fmt.Errorf("the Node listed %d videos of %d", len(res.GetListed()), len(ls))
	}
	out := make([]SidecarListed, len(ls))
	for i, l := range res.GetListed() {
		out[i] = SidecarListed{Video: ls[i].Video, VideoMissing: l.GetVideoMissing()}
		for _, c := range l.GetCompanions() {
			p, err := ParseRel(c.GetPath())
			if err != nil {
				return nil, fmt.Errorf("a listed companion: %w", err)
			}
			sha, err := hashOf(c.GetSha256())
			if err != nil {
				return nil, fmt.Errorf("a listed companion's hash: %w", err)
			}
			out[i].Companions = append(out[i].Companions, SidecarCompanion{Path: p, Tail: c.GetTail(), Size: c.GetSize(), SHA256: sha, Ledger: c.GetLedger()})
		}
	}
	return out, nil
}

// Read is the bytes at path under root if they hash to sha: a
// *ChangedError for other bytes, fs.ErrNotExist for none.
func (c sidecarCalls) Read(ctx context.Context, root storage.UUID, path Rel, sha [32]byte) ([]byte, error) {
	res, err := c.h.ReadSidecar(ctx, &storagev1.ReadSidecarRequest{RootId: root.Bytes(), Path: path.String(), Sha256: sha[:]})
	if err != nil {
		return nil, answer(err)
	}
	switch res.GetState() {
	case storagev1.ReadSidecarState_READ_SIDECAR_STATE_READ:
		return res.GetData(), nil
	case storagev1.ReadSidecarState_READ_SIDECAR_STATE_CHANGED:
		there, err := hashOf(res.GetSha256())
		if err != nil {
			return nil, fmt.Errorf("the changed bytes' hash: %w", err)
		}
		return nil, &ChangedError{SHA256: there}
	case storagev1.ReadSidecarState_READ_SIDECAR_STATE_GONE:
		return nil, fmt.Errorf("%s: %w", path, fs.ErrNotExist)
	case storagev1.ReadSidecarState_READ_SIDECAR_STATE_UNSPECIFIED:
	}
	return nil, fmt.Errorf("a read state the Controller doesn't know: %v", res.GetState())
}

// Write writes w and answers where it landed: ErrStale when
// the video isn't there as w.Identity names it, *OccupiedError when its
// name holds other bytes.
func (c sidecarCalls) Write(ctx context.Context, w SidecarWrite) (Rel, error) {
	req := &storagev1.WriteSidecarRequest{
		RootId: w.Root.Bytes(), Video: w.Video.String(), Identity: w.Identity.wire(), Size: w.Identity.Size, Tail: w.Tail, Data: w.Bytes, Sha256: w.SHA256[:],
	}
	if w.Replaces != nil {
		req.Replaces = w.Replaces[:]
	}
	res, err := c.h.WriteSidecar(ctx, req)
	if err != nil {
		return Rel{}, answer(err)
	}
	switch res.GetState() {
	case storagev1.WriteSidecarState_WRITE_SIDECAR_STATE_WRITTEN:
		p, err := ParseRel(res.GetPath())
		if err != nil {
			return Rel{}, fmt.Errorf("the written Sidecar's path: %w", err)
		}
		return p, nil
	case storagev1.WriteSidecarState_WRITE_SIDECAR_STATE_OCCUPIED:
		there, err := hashOf(res.GetOccupant())
		if err != nil {
			return Rel{}, fmt.Errorf("the occupant's hash: %w", err)
		}
		return Rel{}, &OccupiedError{SHA256: there}
	case storagev1.WriteSidecarState_WRITE_SIDECAR_STATE_STALE:
		return Rel{}, ErrStale
	case storagev1.WriteSidecarState_WRITE_SIDECAR_STATE_UNSPECIFIED:
	}
	return Rel{}, fmt.Errorf("a write state the Controller doesn't know: %v", res.GetState())
}

// Remove removes the bytes at path under root if they hash to sha,
// and answers nil when none are there; other bytes are a *ChangedError.
func (c sidecarCalls) Remove(ctx context.Context, root storage.UUID, path Rel, sha [32]byte) error {
	res, err := c.h.RemoveSidecar(ctx, &storagev1.RemoveSidecarRequest{RootId: root.Bytes(), Path: path.String(), Sha256: sha[:]})
	if err != nil {
		return answer(err)
	}
	switch res.GetState() {
	case storagev1.RemoveSidecarState_REMOVE_SIDECAR_STATE_REMOVED:
		return nil
	case storagev1.RemoveSidecarState_REMOVE_SIDECAR_STATE_CHANGED:
		there, err := hashOf(res.GetOccupant())
		if err != nil {
			return fmt.Errorf("the changed bytes' hash: %w", err)
		}
		return &ChangedError{SHA256: there}
	case storagev1.RemoveSidecarState_REMOVE_SIDECAR_STATE_UNSPECIFIED:
	}
	return fmt.Errorf("a remove state the Controller doesn't know: %v", res.GetState())
}
