package imagecdn_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
	"testing/fstest"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/imagecdn"
	"github.com/jatassi/slipstream/fakes/internal/artwork"
	"github.com/jatassi/slipstream/fakes/internal/images"
	"github.com/jatassi/slipstream/fakes/scenario"
)

func picture(t *testing.T, width, height int, encode func(*bytes.Buffer, image.Image) error) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 0x80, A: 0xff})
		}
	}
	var b bytes.Buffer
	if err := encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func size(t *testing.T, body []byte) (width, height int, format string) {
	t.Helper()
	c, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("decode %d bytes: %v", len(body), err)
	}
	return c.Width, c.Height, format
}

type stored struct {
	poster, logo, svg []byte
}

func storeOf(t *testing.T) (artwork.Store, stored) {
	t.Helper()
	files := stored{
		poster: picture(t, 370, 555, func(b *bytes.Buffer, m image.Image) error { return jpeg.Encode(b, m, nil) }),
		logo:   picture(t, 400, 160, func(b *bytes.Buffer, m image.Image) error { return png.Encode(b, m) }),
		svg:    []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="4"><rect width="10" height="4" fill="#123456"/></svg>`),
	}
	return artwork.New(fstest.MapFS{
		"st0redP0ster.jpg":                      {Data: files.poster},
		"st0redL0go.png":                        {Data: files.logo},
		"st0redL0go.svg":                        {Data: files.svg},
		"tvdb/v4/series/7/posters/st0red.jpg":   {Data: files.poster},
		"tvdb/v4/series/7/clearlogo/st0red.png": {Data: files.logo},
	}), files
}

func inBothTiersWith(t *testing.T, store artwork.Store, test func(*testing.T, faketest.Served)) {
	t.Run("loopback", func(t *testing.T) { test(t, faketest.Loopback(t, imagecdn.WithStore(store), scenario.Scenario{})) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, faketest.InMemory(t, imagecdn.WithStore(store), scenario.Scenario{})) })
	})
}

func TestStoredImagesAnswerTheirBytesAtOriginal(t *testing.T) {
	store, files := storeOf(t)
	inBothTiersWith(t, store, func(t *testing.T, f faketest.Served) {
		for path, want := range map[string][]byte{
			"/original/st0redP0ster.jpg": files.poster,
			"/original/st0redL0go.png":   files.logo,
		} {
			if r := get(t, f, path); r.status != "200 OK" || !bytes.Equal(r.body, want) {
				t.Errorf("GET %s = %s with %d bytes; want the %d stored bytes", path, r.status, len(r.body), len(want))
			}
		}
		for _, path := range []string{"/w92/st0redL0go.svg", "/original/st0redL0go.svg"} {
			if r := get(t, f, path); r.status != "200 OK" || r.contentType != "image/svg+xml" || !bytes.Equal(r.body, files.svg) {
				t.Errorf("GET %s = %s %s %q; want the stored SVG", path, r.status, r.contentType, r.body)
			}
		}
		r := get(t, f, "/w185/n0tSt0red.jpg")
		if got, err := images.Dominant(r.body); err != nil || !near(got, images.Color("/n0tSt0red.jpg", "w185")) {
			t.Errorf("GET a file the store lacks is mostly %v, %v; want its generated color", got, err)
		}
	})
}

func TestStoredImagesScaleToTheSizeAsked(t *testing.T) {
	store, _ := storeOf(t)
	inBothTiersWith(t, store, func(t *testing.T, f faketest.Served) {
		for path, want := range map[string]struct {
			width, height       int
			format, contentType string
		}{
			"/w185/st0redP0ster.jpg": {185, 277, "jpeg", "image/jpeg"},
			"/w92/st0redP0ster.jpg":  {92, 138, "jpeg", "image/jpeg"},
			"/w185/st0redL0go.png":   {185, 74, "png", "image/png"},
			"/w500/st0redP0ster.jpg": {500, 750, "jpeg", "image/jpeg"},
			"/h632/st0redL0go.png":   {1580, 632, "png", "image/png"},
		} {
			r := get(t, f, path)
			width, height, format := size(t, r.body)
			if r.status != "200 OK" || r.contentType != want.contentType || width != want.width || height != want.height || format != want.format {
				t.Errorf("GET %s = %s %s, a %dx%d %s; want a %dx%d %s", path, r.status, r.contentType, width, height, format, want.width, want.height, want.format)
			}
			if again := get(t, f, path); !bytes.Equal(again.body, r.body) {
				t.Errorf("GET %s twice answered different bytes", path)
			}
		}
	})
}

func TestStoredTVDBArtworkAnswersItsBytesAndAThumbnailAtHalfItsWidth(t *testing.T) {
	store, files := storeOf(t)
	inBothTiersWith(t, store, func(t *testing.T, f faketest.Served) {
		for path, want := range map[string][]byte{
			"/banners/v4/series/7/posters/st0red.jpg":   files.poster,
			"/banners/v4/series/7/clearlogo/st0red.png": files.logo,
		} {
			if r := fetch(t, f, path); r.status != "200 OK" || !bytes.Equal(r.body, want) {
				t.Errorf("GET %s = %s with %d bytes; want the %d stored bytes", path, r.status, len(r.body), len(want))
			}
		}
		for path, want := range map[string]struct {
			width, height       int
			format, contentType string
		}{
			"/banners/v4/series/7/posters/st0red_t.jpg":   {185, 277, "jpeg", "image/jpeg"},
			"/banners/v4/series/7/clearlogo/st0red_t.png": {200, 80, "png", "image/png"},
		} {
			r := fetch(t, f, path)
			width, height, format := size(t, r.body)
			if r.status != "200 OK" || r.contentType != want.contentType || width != want.width || height != want.height || format != want.format {
				t.Errorf("GET %s = %s %s, a %dx%d %s; want a %dx%d %s", path, r.status, r.contentType, width, height, format, want.width, want.height, want.format)
			}
		}
		unknown := "/banners/v4/series/7/posters/n0tSt0red.jpg"
		if got, err := images.Dominant(fetch(t, f, unknown).body); err != nil || !near(got, images.Color(unknown, "original")) {
			t.Errorf("GET %s is mostly %v, %v; want its generated color", unknown, got, err)
		}
	})
}
