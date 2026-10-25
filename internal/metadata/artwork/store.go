package artwork

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/metadata/artwork/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Local

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is the store's index, in the Controller's local.db, so a Backup
// carries no image.
func Schema() storage.Schema[file] { return storage.NewSchema[file]("artwork", schemaFiles) }

const megabyte = 1 << 20

var (
	// ErrNotHeld means the store holds no such image: fetch it.
	ErrNotHeld = errors.New("the artwork store holds no such image")
	// ErrMissing means a fetch of the image failed and its retry time
	// hasn't come.
	ErrMissing = errors.New("the image couldn't be fetched")
	// ErrUnfit is a body that isn't whole as the image type it claims.
	ErrUnfit = errors.New("not a whole PNG, JPEG, WebP or SVG image")
)

// Image is one image's bytes and media type.
type Image struct {
	Type string
	Body []byte
}

// Store is the Controller's byte cache of images, one file per distinct
// image named by its SHA-256, indexed in local.db and bounded by the
// bound it opens with, dropping the images served longest ago.
type Store struct {
	dir   string
	local *storage.DB[file]
	// bound is the most bytes the store holds.
	bound func(context.Context) (int64, error)
	// files guards the image files: Put and Trim change them, Open reads.
	files sync.RWMutex
}

// OpenStore opens the store in dir, removing what a crash left there: a
// partly written file, or one no index row names. It then trims to bound.
func OpenStore(ctx context.Context, dir string, local *storage.DB[file], boundMB func(context.Context) (int64, error)) (*Store, error) {
	s := &Store{dir: dir, local: local, bound: func(ctx context.Context) (int64, error) {
		mb, err := boundMB(ctx)
		return mb * megabyte, err
	}}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("artwork store: %w", err)
	}
	if err := s.sweep(ctx); err != nil {
		return nil, err
	}
	if err := s.Trim(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) sweep(ctx context.Context) error {
	held, err := db.New(s.local).ListBlobs(ctx)
	if err != nil {
		return fmt.Errorf("artwork store: %w", err)
	}
	indexed := map[string]bool{}
	for _, h := range held {
		indexed[h] = true
	}
	root, err := os.OpenRoot(s.dir)
	if err != nil {
		return fmt.Errorf("artwork store: %w", err)
	}
	defer func() { _ = root.Close() }()
	err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || indexed[d.Name()] && path == filepath.ToSlash(s.rel(d.Name())) {
			return err
		}
		return root.Remove(path)
	})
	if err != nil {
		return fmt.Errorf("artwork store: sweep: %w", err)
	}
	return nil
}

func (s *Store) rel(hash string) string { return filepath.Join(hash[:2], hash) }

func (s *Store) path(hash string) string { return filepath.Join(s.dir, s.rel(hash)) }

type key = db.GetEntryParams

func keyOf(w Want) key {
	return key{Source: string(w.Ref.Source), Shape: string(w.Ref.Shape), Path: w.Ref.Path, Size: string(w.Size)}
}

// Open is the image the store holds for w, marked served now. It is
// ErrNotHeld when the store holds none, and ErrMissing while a failed
// fetch's retry time hasn't come.
func (s *Store) Open(ctx context.Context, w Want) (Image, error) {
	s.files.RLock()
	defer s.files.RUnlock()
	k := keyOf(w)
	var e db.GetEntryRow
	err := s.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		var err error
		if e, err = q.GetEntry(ctx, k); err != nil || e.Hash == nil {
			return err
		}
		return q.MarkServed(ctx, db.MarkServedParams{ServedAt: storage.Now(), Source: k.Source, Shape: k.Shape, Path: k.Path, Size: k.Size})
	})
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return Image{}, ErrNotHeld
	case err != nil:
		return Image{}, fmt.Errorf("artwork store: %w", err)
	case e.Hash == nil && storage.Now().Compare(e.RetryAt) < 0:
		return Image{}, ErrMissing
	case e.Hash == nil:
		return Image{}, ErrNotHeld
	}
	body, err := os.ReadFile(s.path(*e.Hash))
	if errors.Is(err, fs.ErrNotExist) {
		return Image{}, ErrNotHeld
	}
	if err != nil {
		return Image{}, fmt.Errorf("artwork store: %w", err)
	}
	return Image{Type: *e.Type, Body: body}, nil
}

// Put stores img as w, replacing what the store held for it only once
// img is whole as the type it claims, then trims the store to its bound.
// It answers img as stored. An image that isn't whole is ErrUnfit and
// changes nothing.
func (s *Store) Put(ctx context.Context, w Want, img Image) (Image, error) {
	t, err := fit(img)
	if err != nil {
		return Image{}, err
	}
	sum := sha256.Sum256(img.Body)
	hash := hex.EncodeToString(sum[:])
	s.files.Lock()
	defer s.files.Unlock()
	if err := s.write(hash, img.Body); err != nil {
		return Image{}, err
	}
	k := keyOf(w)
	var replaced *string
	err = s.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		was, err := q.GetEntry(ctx, k)
		if err != nil && !errors.Is(err, storage.ErrNoRows) {
			return err
		}
		if err := q.PutBlob(ctx, db.PutBlobParams{Hash: hash, Bytes: int64(len(img.Body)), Type: t}); err != nil {
			return err
		}
		if err := q.PutEntry(ctx, db.PutEntryParams{Source: k.Source, Shape: k.Shape, Path: k.Path, Size: k.Size, Hash: &hash, ServedAt: storage.Now()}); err != nil {
			return err
		}
		if was.Hash == nil || *was.Hash == hash {
			return nil
		}
		n, err := q.DeleteUnheldBlob(ctx, *was.Hash)
		if n > 0 {
			replaced = was.Hash
		}
		return err
	})
	if err != nil {
		return Image{}, fmt.Errorf("artwork store: %w", err)
	}
	if replaced != nil {
		if err := s.remove(*replaced); err != nil {
			return Image{}, err
		}
	}
	return Image{Type: t, Body: img.Body}, s.trim(ctx)
}

func (s *Store) remove(hash string) error {
	if err := os.Remove(s.path(hash)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("artwork store: %w", err)
	}
	return nil
}

// write puts body in its file through a temporary file renamed into
// place, so the file is never partly written.
func (s *Store) write(hash string, body []byte) error {
	dst := s.path(hash)
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return fmt.Errorf("artwork store: %w", err)
	}
	tmp, err := os.CreateTemp(s.dir, ".put-*")
	if err != nil {
		return fmt.Errorf("artwork store: %w", err)
	}
	_, err = tmp.Write(body)
	if err == nil {
		err = tmp.Sync()
	}
	err = errors.Join(err, tmp.Close())
	if err == nil {
		err = os.Rename(tmp.Name(), dst)
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("artwork store: %w", err)
	}
	return nil
}

// MarkMissing records that fetching w failed, so Open answers ErrMissing
// until retry. It never replaces an image the store holds.
func (s *Store) MarkMissing(ctx context.Context, w Want, retry time.Time) error {
	k := keyOf(w)
	err := s.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).MarkMissing(ctx, db.MarkMissingParams{Source: k.Source, Shape: k.Shape, Path: k.Path, Size: k.Size, RetryAt: storage.FromTime(retry), ServedAt: storage.Now()})
	})
	if err != nil {
		return fmt.Errorf("artwork store: %w", err)
	}
	return nil
}

// Trim drops the images served longest ago until the store fits its
// bound, and forgets failed fetches whose retry time has come.
func (s *Store) Trim(ctx context.Context) error {
	s.files.Lock()
	defer s.files.Unlock()
	return s.trim(ctx)
}

func (s *Store) trim(ctx context.Context) error {
	bound, err := s.bound(ctx)
	if err != nil {
		return fmt.Errorf("artwork store: read its size: %w", err)
	}
	var dropped []string
	err = s.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if err := q.DeleteSpentMisses(ctx, storage.Now()); err != nil {
			return err
		}
		dropped, err = shrink(ctx, q, bound)
		return err
	})
	if err != nil {
		return fmt.Errorf("artwork store: trim: %w", err)
	}
	for _, h := range dropped {
		if err := s.remove(h); err != nil {
			return err
		}
	}
	return nil
}

// shrink forgets the images served longest ago until the rest fit in
// bound, answering the files no reference holds any more.
func shrink(ctx context.Context, q *db.Queries, bound int64) ([]string, error) {
	held, err := q.HeldBytes(ctx)
	if err != nil {
		return nil, err
	}
	var dropped []string
	for held > bound {
		e, err := q.LeastRecentlyServed(ctx)
		if err != nil {
			return nil, err
		}
		if err := q.DeleteEntry(ctx, db.DeleteEntryParams{Source: e.Source, Shape: e.Shape, Path: e.Path, Size: e.Size}); err != nil {
			return nil, err
		}
		n, err := q.DeleteUnheldBlob(ctx, *e.Hash)
		if err != nil {
			return nil, err
		}
		if n > 0 {
			held -= e.Bytes
			dropped = append(dropped, *e.Hash)
		}
	}
	return dropped, nil
}

// whole says whether a body is a complete image of each type the store
// keeps, reading the end each format marks.
var whole = map[string]func([]byte) bool{
	"image/png": func(b []byte) bool {
		return bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")) && bytes.HasSuffix(b, []byte("IEND\xae\x42\x60\x82"))
	},
	"image/jpeg": func(b []byte) bool {
		return bytes.HasPrefix(b, []byte{0xff, 0xd8, 0xff}) && bytes.HasSuffix(bytes.TrimRight(b, "\x00"), []byte{0xff, 0xd9})
	},
	"image/webp": func(b []byte) bool {
		return len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP" && int(binary.LittleEndian.Uint32(b[4:8]))+8 == len(b)
	},
	"image/svg+xml": func(b []byte) bool {
		t := bytes.TrimSpace(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")))
		return bytes.HasPrefix(t, []byte("<")) && bytes.Contains(t, []byte("<svg")) && bytes.HasSuffix(t, []byte("</svg>"))
	},
}

// fit is img's media type when img is whole as that type.
func fit(img Image) (string, error) {
	t, _, err := mime.ParseMediaType(img.Type)
	t = strings.ToLower(t)
	if check, ok := whole[t]; err == nil && ok && check(img.Body) {
		return t, nil
	}
	return "", fmt.Errorf("%w: %d bytes sent as %q", ErrUnfit, len(img.Body), img.Type)
}
