package metadata_test

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/metadata/metadatatest"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

type cdn struct{}

func (cdn) ID() artwork.SourceID { return "cdn" }

func (cdn) URL(_ artwork.Shape, path string, size artwork.Size) (*url.URL, error) {
	return url.Parse("https://images.example/" + string(size) + path)
}

func (cdn) Rate() (float64, int) { return 100, 10 }

// pictures is a metadata source that only names its image host.
type pictures struct{ metadata.Source }

func (pictures) Sources() []artwork.Source { return []artwork.Source{cdn{}} }

// poster answers every request with the same JPEG, counting them.
type poster struct {
	body  []byte
	asked atomic.Int32
}

func newPoster(t *testing.T) *poster {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewGray(image.Rect(0, 0, 2, 3)), nil); err != nil {
		t.Fatal(err)
	}
	return &poster{body: b.Bytes()}
}

func (p *poster) RoundTrip(*http.Request) (*http.Response, error) {
	p.asked.Add(1)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"image/jpeg"}},
		Body:       io.NopCloser(bytes.NewReader(p.body)),
	}, nil
}

func TestASenderReadsAPosterByItsReferenceOrItsPublicURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	host := newPoster(t)
	lib := storagetest.Library(t, works.Schema(), metadata.Schema())
	m := metadatatest.Artwork(t, lib, map[works.TypeID]metadata.Source{"pictures": pictures{}}, &http.Client{Transport: host}, metadata.DefaultTimings())
	ref := artwork.Ref{Source: "cdn", Shape: artwork.ShapePoster, Path: "/heat.jpg"}

	for range 2 {
		img, err := m.Image(ctx, ref, artwork.SizeMedium)
		if err != nil || img.Type != "image/jpeg" || !bytes.Equal(img.Body, host.body) {
			t.Fatalf("the poster read %s of %d bytes, %v; want the JPEG", img.Type, len(img.Body), err)
		}
	}
	if got := host.asked.Load(); got != 1 {
		t.Fatalf("two reads asked the image host %d times, want once and then the store", got)
	}
	if got, err := m.PublicURL(ref, artwork.SizeMedium); err != nil || got != "https://images.example/medium/heat.jpg" {
		t.Fatalf("the public URL is %q, %v; want the source's own URL at medium", got, err)
	}
	if _, err := m.PublicURL(artwork.Ref{Source: "elsewhere", Path: "/heat.jpg"}, artwork.SizeMedium); err == nil {
		t.Fatal("a reference to a source the Controller doesn't know had a public URL")
	}
}
