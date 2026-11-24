package storing

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/speech"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

//go:embed schema/*.sql
var sidecarSchemaFiles embed.FS

// SidecarSchema is the local set "subtitles": this Node's Speech Map index
// and its Sidecar ledger.
func SidecarSchema() storage.Schema[storage.Local] {
	return storage.NewSchema[storage.Local]("subtitles", sidecarSchemaFiles)
}

// SidecarTimings bound SidecarService on this Node.
type SidecarTimings struct {
	// BuildWait bounds one BuildSpeechMap call; the build goes on and the
	// next call joins it.
	BuildWait time.Duration `yaml:"build_wait"`
	// MapCap bounds an encoded map's bytes, SidecarCap a Sidecar's.
	MapCap     int `yaml:"map_cap"`
	SidecarCap int `yaml:"sidecar_cap"`
	// Queue bounds the builds waiting on this Node.
	Queue int `yaml:"queue"`
	// SurveyBatch bounds the videos of one ListSidecars call.
	SurveyBatch int `yaml:"survey_batch"`
}

func DefaultSidecarTimings() SidecarTimings {
	return SidecarTimings{BuildWait: 30 * time.Second, MapCap: 1 << 20, SidecarCap: 8 << 20, Queue: 64, SurveyBatch: 200}
}

// Sidecars is this Node's half of Subtitles for the roots it holds: the
// Speech Maps it builds and keeps for their files, and the subtitle bytes
// beside their videos, with the ledger of every provider's hash it wrote. It
// trusts its caller, the Controller, and checks only what the bytes on
// this Node can tell: roots, names, identity and hashes.
type Sidecars struct {
	store  *Store
	local  *storage.DB[storage.Local]
	maps   string
	mapper speech.Mapper
	t      SidecarTimings
	builds *builds
}

// NewSidecars keeps its maps in dir, removing what no index row names and
// the rows whose map is gone. Builds run under ctx.
func NewSidecars(ctx context.Context, s *Store, local *storage.DB[storage.Local], dir string, m speech.Mapper, t SidecarTimings) (*Sidecars, error) {
	sc := &Sidecars{store: s, local: local, maps: dir, mapper: m, t: t}
	s.ledger.Store(local)
	sc.builds = newBuilds(ctx, sc)
	if err := sc.sweep(ctx); err != nil {
		return nil, fmt.Errorf("speech maps: %w", err)
	}
	return sc, nil
}

// Identity is a file as a map or a write names it: what a walk reads of
// it, and its size.
type Identity struct {
	Seen Seen
	Size int64
}

func (id Identity) Same(o Identity) bool { return id.Seen.Same(o.Seen) && id.Size == o.Size }

func (id Identity) encode() []byte {
	b := binary.AppendVarint(nil, id.Seen.Modified.UnixMilli())
	b = binary.AppendVarint(b, id.Size)
	b = append(binary.AppendUvarint(b, uint64(len(id.Seen.ID.Volume))), id.Seen.ID.Volume...)
	return append(binary.AppendUvarint(b, uint64(len(id.Seen.ID.File))), id.Seen.ID.File...)
}

func (id Identity) wire() *storagev1.Identity {
	file, _ := hex.DecodeString(id.Seen.ID.File)
	return &storagev1.Identity{ModifiedUnixMs: id.Seen.Modified.UnixMilli(), FileId: file, VolumeId: id.Seen.ID.Volume}
}

func identityOfInfo(info Info) Identity {
	return Identity{Seen: seenOf(identityOf(info)), Size: info.Size()}
}

// tailPattern is every tail WriteSidecar writes: an optional disambiguator,
// a language, an optional region, forced and SDH, and a text format.
var tailPattern = regexp.MustCompile(`^([0-9a-f]{8}\.)?[a-z]{2,3}(-[A-Z]{2})?(\.forced)?(\.sdh)?\.(srt|ass|ssa)$`)

// WritableTail is whether WriteSidecar writes a Sidecar under tail.
func WritableTail(tail string) bool { return tailPattern.MatchString(tail) }

// subtitleExts are the extensions ListSidecars lists and ReadSidecar and
// RemoveSidecar act on.
var subtitleExts = []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".idx", ".sup", ".smi"}

func subtitleName(name string) bool {
	return slices.Contains(subtitleExts, strings.ToLower(path.Ext(name)))
}

func (s *Sidecars) ListSidecars(ctx context.Context, req *storagev1.ListSidecarsRequest) (*storagev1.ListSidecarsResponse, error) {
	if len(req.GetVideos()) > s.t.SurveyBatch {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%d videos is more than one batch of %d", len(req.GetVideos()), s.t.SurveyBatch)
	}
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	videos, err := videosOf(req.GetVideoExtensions())
	if err != nil {
		return nil, err
	}
	listings, err := listingsOf(req.GetVideos())
	if err != nil {
		return nil, err
	}
	dir, err := s.store.resolve(ctx, root, false)
	if err != nil {
		return nil, WireError(err)
	}
	res := &storagev1.ListSidecarsResponse{}
	folders := map[Rel][]Entry{}
	for _, l := range listings {
		listed, err := s.listed(ctx, dir, l, videos, folders)
		if err != nil {
			return nil, err
		}
		if err := s.markLedger(ctx, root, listed.Companions); err != nil {
			return nil, err
		}
		res.Listed = append(res.Listed, listed)
	}
	return res, nil
}

// sidecarListing is a video and the stems whose companions it lists, its own
// first.
type sidecarListing struct {
	video Rel
	stems []Rel
}

func listingsOf(ls []*storagev1.SidecarListing) ([]sidecarListing, error) {
	out := make([]sidecarListing, 0, len(ls))
	for _, v := range ls {
		video, err := relOf(v.GetVideo())
		if err != nil {
			return nil, err
		}
		l := sidecarListing{video: video, stems: []Rel{video}}
		for _, a := range v.GetAlso() {
			also, err := relOf(a)
			if err != nil {
				return nil, err
			}
			l.stems = append(l.stems, also)
		}
		out = append(out, l)
	}
	return out, nil
}

func (s *Sidecars) listed(ctx context.Context, dir string, l sidecarListing, videos media.Videos, folders map[Rel][]Entry) (*storagev1.SidecarListed, error) {
	listed := &storagev1.SidecarListed{Video: l.video.String()}
	if _, err := s.store.stat(ctx, dir, l.video); errors.Is(err, fs.ErrNotExist) {
		listed.VideoMissing = true
	} else if err != nil {
		return nil, WireError(err)
	}
	seen := map[Rel]bool{}
	for _, stem := range l.stems {
		found, err := s.companionsOf(ctx, dir, stem, videos, folders)
		if err != nil {
			return nil, WireError(err)
		}
		for _, c := range found {
			if !seen[c.path] {
				seen[c.path] = true
				listed.Companions = append(listed.Companions, &storagev1.SidecarCompanion{Path: c.path.String(), Tail: c.tail, Size: c.size, Sha256: c.sha[:]})
			}
		}
	}
	return listed, nil
}

type companion struct {
	path Rel
	tail string
	size int64
	sha  [32]byte
}

// companionsOf are the subtitle companions of the video at stem, there or
// not, with their sizes and hashes. folders caches each folder's entries.
func (s *Sidecars) companionsOf(ctx context.Context, dir string, stem Rel, videos media.Videos, folders map[Rel][]Entry) ([]companion, error) {
	folder, name := stem.split()
	entries, ok := folders[folder]
	if !ok {
		var err error
		entries, err = s.store.entries(ctx, s.store.host.Style.Under(dir, folder))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, s.store.folderError(err)
		}
		folders[folder] = entries
	}
	var out []companion
	for _, c := range companions(entries, name, videos) {
		if !subtitleName(c) {
			continue
		}
		p := folder.join(c)
		size, sha, err := s.store.hash(ctx, dir, p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, companion{path: p, tail: strings.TrimPrefix(c, stemOf(name)+"."), size: size, sha: sha})
	}
	return out, nil
}

func (s *Sidecars) ReadSidecar(ctx context.Context, req *storagev1.ReadSidecarRequest) (*storagev1.ReadSidecarResponse, error) {
	root, rel, sha, err := s.sidecarAt(req.GetRootId(), req.GetPath(), req.GetSha256())
	if err != nil {
		return nil, err
	}
	dir, err := s.store.resolve(ctx, root, false)
	if err != nil {
		return nil, WireError(err)
	}
	f, err := s.store.cfg.Disk.Open(ctx, s.store.host.Style.Under(dir, rel))
	if errors.Is(err, fs.ErrNotExist) {
		return &storagev1.ReadSidecarResponse{State: storagev1.ReadSidecarState_READ_SIDECAR_STATE_GONE}, nil
	}
	if err != nil {
		return nil, WireError(s.store.folderError(err))
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	data, err := io.ReadAll(io.TeeReader(io.LimitReader(f, int64(s.t.SidecarCap)+1), h))
	if err == nil && len(data) > s.t.SidecarCap {
		_, err = io.Copy(h, f)
	}
	if err != nil {
		return nil, WireError(s.store.folderError(err))
	}
	switch got := [32]byte(h.Sum(nil)); {
	case got != sha:
		return &storagev1.ReadSidecarResponse{State: storagev1.ReadSidecarState_READ_SIDECAR_STATE_CHANGED, Sha256: got[:]}, nil
	case len(data) > s.t.SidecarCap:
		return nil, rpc.Errorf(rpc.CodeFailedPrecondition, "%s is larger than the %d bytes a Sidecar may hold", rel, s.t.SidecarCap)
	}
	return &storagev1.ReadSidecarResponse{State: storagev1.ReadSidecarState_READ_SIDECAR_STATE_READ, Data: data}, nil
}

func (s *Sidecars) WriteSidecar(ctx context.Context, req *storagev1.WriteSidecarRequest) (*storagev1.WriteSidecarResponse, error) {
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	video, err := relOf(req.GetVideo())
	if err != nil {
		return nil, err
	}
	tail := req.GetTail()
	if !tailPattern.MatchString(tail) {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%q is not a Sidecar's tail", tail)
	}
	sha, err := hashOf(req.GetSha256())
	if err != nil {
		return nil, err
	}
	data := req.GetData()
	if len(data) > s.t.SidecarCap {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%d bytes is more than the %d a Sidecar may hold", len(data), s.t.SidecarCap)
	}
	if sha256.Sum256(data) != sha {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "the bytes don't hash to the sha256 sent")
	}
	var replaces *[32]byte
	if len(req.GetReplaces()) > 0 {
		r, err := hashOf(req.GetReplaces())
		if err != nil {
			return nil, err
		}
		replaces = &r
	}
	dir, err := s.store.resolve(ctx, root, true)
	if err != nil {
		return nil, WireError(err)
	}
	stale := &storagev1.WriteSidecarResponse{State: storagev1.WriteSidecarState_WRITE_SIDECAR_STATE_STALE}
	if video.segments()[0] == HoldingDir {
		return stale, nil
	}
	info, err := s.store.stat(ctx, dir, video)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return stale, nil
	case err != nil:
		return nil, WireError(err)
	case !identityOfInfo(info).Same(Identity{Seen: seenOf(req.GetIdentity()), Size: req.GetSize()}):
		return stale, nil
	}
	folder, name := video.split()
	target := folder.join(stemOf(name) + "." + tail)
	_, there, err := s.store.hash(ctx, dir, target)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, WireError(err)
	case there == sha:
		if err := s.putLedger(ctx, root, req.GetLedger(), sha, target); err != nil {
			return nil, err
		}
		return &storagev1.WriteSidecarResponse{State: storagev1.WriteSidecarState_WRITE_SIDECAR_STATE_WRITTEN, Path: target.String()}, nil
	case replaces == nil || there != *replaces:
		return &storagev1.WriteSidecarResponse{State: storagev1.WriteSidecarState_WRITE_SIDECAR_STATE_OCCUPIED, Occupant: there[:], Path: target.String()}, nil
	}
	if err := s.putLedger(ctx, root, req.GetLedger(), sha, target); err != nil {
		return nil, err
	}
	if err := s.write(ctx, s.store.host.Style.Under(dir, target), data); err != nil {
		return nil, WireError(s.store.refused(ctx, dir, err))
	}
	return &storagev1.WriteSidecarResponse{State: storagev1.WriteSidecarState_WRITE_SIDECAR_STATE_WRITTEN, Path: target.String()}, nil
}

// write lands data at p through p.part, synced before the rename.
func (s *Sidecars) write(ctx context.Context, p string, data []byte) error {
	d := s.store.cfg.Disk
	part := p + ".part"
	f, err := d.Create(ctx, part)
	if err != nil {
		return fmt.Errorf("write the Sidecar: %w", err)
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = d.Rename(ctx, part, p)
	}
	if err != nil {
		_ = d.Remove(ctx, part)
		return fmt.Errorf("write the Sidecar: %w", err)
	}
	return nil
}

func (s *Sidecars) RemoveSidecar(ctx context.Context, req *storagev1.RemoveSidecarRequest) (*storagev1.RemoveSidecarResponse, error) {
	root, rel, sha, err := s.sidecarAt(req.GetRootId(), req.GetPath(), req.GetSha256())
	if err != nil {
		return nil, err
	}
	dir, err := s.store.resolve(ctx, root, false)
	if err != nil {
		return nil, WireError(err)
	}
	removed := &storagev1.RemoveSidecarResponse{State: storagev1.RemoveSidecarState_REMOVE_SIDECAR_STATE_REMOVED}
	_, there, err := s.store.hash(ctx, dir, rel)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return removed, nil
	case err != nil:
		return nil, WireError(err)
	case there != sha:
		return &storagev1.RemoveSidecarResponse{State: storagev1.RemoveSidecarState_REMOVE_SIDECAR_STATE_CHANGED, Occupant: there[:]}, nil
	}
	if err := s.store.cfg.Disk.Remove(ctx, s.store.host.Style.Under(dir, rel)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, WireError(s.store.folderError(err))
	}
	return removed, nil
}

// sidecarAt parses a request that names a subtitle file by its root, path
// and hash.
func (s *Sidecars) sidecarAt(rootBytes []byte, p string, shaBytes []byte) (root storage.UUID, rel Rel, sha [32]byte, err error) {
	if root, err = rootID(rootBytes); err != nil {
		return root, Rel{}, sha, err
	}
	if rel, err = relOf(p); err != nil {
		return root, rel, sha, err
	}
	if !subtitleName(p) {
		return root, rel, sha, rpc.Errorf(rpc.CodeInvalidArgument, "%q is not a subtitle file", p)
	}
	sha, err = hashOf(shaBytes)
	return root, rel, sha, err
}

func hashOf(b []byte) ([32]byte, error) {
	if len(b) != sha256.Size {
		return [32]byte{}, rpc.Errorf(rpc.CodeInvalidArgument, "a sha256 is %d bytes, not %d", sha256.Size, len(b))
	}
	return [32]byte(b), nil
}
