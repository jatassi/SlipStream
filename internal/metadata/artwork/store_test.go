package artwork_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type opened struct {
	*artwork.Store
	dir   string
	local *storage.DB[storage.Local]
}

func openStore(t *testing.T, mb int64) opened {
	t.Helper()
	o := opened{dir: t.TempDir(), local: storagetest.Local(t, artwork.Schema())}
	o.Store = reopen(t, o, mb)
	return o
}

func reopen(t *testing.T, o opened, mb int64) *artwork.Store {
	t.Helper()
	s, err := artwork.OpenStore(t.Context(), o.dir, o.local, func(context.Context) (int64, error) { return mb, nil })
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func want(path string) artwork.Want {
	return artwork.Want{Ref: artwork.Ref{Source: "tmdb-images", Shape: artwork.ShapePoster, Path: path}, Size: artwork.SizeMedium}
}

func pngOf(t *testing.T, side int) artwork.Image {
	t.Helper()
	m := image.NewNRGBA(image.Rect(0, 0, side, side))
	noise := rand.NewChaCha8(sha256.Sum256([]byte(strconv.Itoa(side))))
	_, _ = noise.Read(m.Pix)
	var b bytes.Buffer
	if err := png.Encode(&b, m); err != nil {
		t.Fatal(err)
	}
	return artwork.Image{Type: "image/png", Body: b.Bytes()}
}

func jpegOf(t *testing.T) artwork.Image {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewGray(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	return artwork.Image{Type: "image/jpeg", Body: b.Bytes()}
}

func webpOf() artwork.Image {
	return artwork.Image{Type: "image/webp", Body: []byte("RIFF\x12\x00\x00\x00WEBPVP8L\x05\x00\x00\x00\x2f\x00\x00\x00\x00\x00")}
}

func svgOf() artwork.Image {
	return artwork.Image{Type: "image/svg+xml", Body: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="2" height="3"><rect width="2" height="3" fill="#123456"/></svg>` + "\n")}
}

func held(t *testing.T, s *artwork.Store, w artwork.Want, img artwork.Image) {
	t.Helper()
	got, err := s.Open(t.Context(), w)
	if err != nil || got.Type != img.Type || !bytes.Equal(got.Body, img.Body) {
		t.Fatalf("Open(%s) = %s of %d bytes, %v; want the %s of %d bytes put there", w.Ref.Path, got.Type, len(got.Body), err, img.Type, len(img.Body))
	}
}

func notHeld(t *testing.T, s *artwork.Store, w artwork.Want) {
	t.Helper()
	if got, err := s.Open(t.Context(), w); !errors.Is(err, artwork.ErrNotHeld) {
		t.Fatalf("Open(%s) = %d bytes, %v; want ErrNotHeld", w.Ref.Path, len(got.Body), err)
	}
}

func files(t *testing.T, dir string) (names []string, size int64) {
	t.Helper()
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", d.Name(), err)
		}
		names, size = append(names, d.Name()), size+info.Size()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return names, size
}

func TestEachTypeIsStoredAndOpened(t *testing.T) {
	s := openStore(t, 1)
	for i, img := range []artwork.Image{pngOf(t, 4), jpegOf(t), webpOf(), svgOf()} {
		w := want(fmt.Sprintf("/%d", i))
		notHeld(t, s.Store, w)
		got, err := s.Put(t.Context(), w, artwork.Image{Type: img.Type + "; charset=binary", Body: img.Body})
		if err != nil || got.Type != img.Type {
			t.Fatalf("Put(%s) = %s, %v; want it stored as %s", img.Type, got.Type, err, img.Type)
		}
		held(t, s.Store, w, img)
	}
}

func TestTheStoreDropsTheLeastRecentlyServed(t *testing.T) {
	s := openStore(t, 1)
	img := func(side int) artwork.Image {
		i := pngOf(t, side)
		if len(i.Body) < 300<<10 || len(i.Body) > 500<<10 {
			t.Fatalf("a %dpx test image is %d bytes; the test wants each about 400KB", side, len(i.Body))
		}
		return i
	}
	a, b, c := want("/a.png"), want("/b.png"), want("/c.png")
	ai, bi, ci := img(330), img(331), img(332)
	for _, put := range []struct {
		w   artwork.Want
		img artwork.Image
	}{{a, ai}, {b, bi}} {
		if _, err := s.Put(t.Context(), put.w, put.img); err != nil {
			t.Fatal(err)
		}
	}
	held(t, s.Store, a, ai)
	if _, err := s.Put(t.Context(), c, ci); err != nil {
		t.Fatal(err)
	}
	notHeld(t, s.Store, b)
	held(t, s.Store, a, ai)
	held(t, s.Store, c, ci)
	if names, size := files(t, s.dir); len(names) != 2 || size > 1<<20 {
		t.Errorf("the store keeps %d files of %d bytes on disk, want 2 within its 1 MB", len(names), size)
	}

	shared := want("/shared.png")
	if _, err := s.Put(t.Context(), shared, ai); err != nil {
		t.Fatal(err)
	}
	if names, _ := files(t, s.dir); len(names) != 2 {
		t.Errorf("the same image under a second reference makes %d files, want the 2 already there", len(names))
	}
}

func TestATruncatedBodyNeverReplacesAGoodImage(t *testing.T) {
	s := openStore(t, 1)
	for _, img := range []artwork.Image{pngOf(t, 4), jpegOf(t), webpOf(), svgOf()} {
		w := want("/" + img.Type)
		if _, err := s.Put(t.Context(), w, img); err != nil {
			t.Fatal(err)
		}
		for name, bad := range map[string]artwork.Image{
			"truncated":         {Type: img.Type, Body: img.Body[:len(img.Body)*2/3]},
			"empty":             {Type: img.Type},
			"sent as text/html": {Type: "text/html", Body: img.Body},
			"sent untyped":      {Body: img.Body},
		} {
			if _, err := s.Put(t.Context(), w, bad); !errors.Is(err, artwork.ErrUnfit) {
				t.Errorf("Put(%s %s) = %v, want ErrUnfit", name, img.Type, err)
			}
		}
		if err := s.MarkMissing(t.Context(), w, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		held(t, s.Store, w, img)
	}
}

func TestAMissIsAnsweredUntilItsRetry(t *testing.T) {
	s := openStore(t, 1)
	soon, spent := want("/soon.jpg"), want("/spent.jpg")
	if err := s.MarkMissing(t.Context(), soon, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkMissing(t.Context(), spent, time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(t.Context(), soon); !errors.Is(err, artwork.ErrMissing) {
		t.Errorf("Open before the retry = %v, want ErrMissing", err)
	}
	notHeld(t, s.Store, spent)
	img := jpegOf(t)
	if _, err := s.Put(t.Context(), soon, img); err != nil {
		t.Fatal(err)
	}
	held(t, s.Store, soon, img)
}

func TestAReopenedStoreClearsWhatACrashLeft(t *testing.T) {
	s := openStore(t, 1)
	w, img := want("/kept.png"), pngOf(t, 4)
	if _, err := s.Put(t.Context(), w, img); err != nil {
		t.Fatal(err)
	}
	stray := filepath.Join(s.dir, "ab", "ab"+string(bytes.Repeat([]byte("0"), 62)))
	if err := os.MkdirAll(filepath.Dir(stray), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{stray, filepath.Join(s.dir, ".put-123")} {
		if err := os.WriteFile(f, []byte("half"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s.Store = reopen(t, s, 1)
	if names, _ := files(t, s.dir); len(names) != 1 {
		t.Errorf("after reopening the store keeps %v, want only the image it indexed", names)
	}
	held(t, s.Store, w, img)
}

func TestAShrunkBoundTrimsOnOpen(t *testing.T) {
	s := openStore(t, 2)
	for i, side := range []int{330, 331, 332} {
		if _, err := s.Put(t.Context(), want(strconv.Itoa(i)), pngOf(t, side)); err != nil {
			t.Fatal(err)
		}
	}
	s.Store = reopen(t, s, 1)
	if names, size := files(t, s.dir); len(names) != 2 || size > 1<<20 {
		t.Errorf("reopened at 1 MB the store keeps %d files of %d bytes, want 2 within 1 MB", len(names), size)
	}
	notHeld(t, s.Store, want("0"))
}
