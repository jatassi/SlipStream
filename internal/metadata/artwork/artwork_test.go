package artwork_test

import (
	"net/url"
	"testing"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
)

func TestRefZeroAndShapesRoundTrip(t *testing.T) {
	if !(artwork.Ref{}).IsZero() {
		t.Error("the empty Ref is not zero")
	}
	for _, r := range []artwork.Ref{{Source: "tmdb"}, {Shape: artwork.ShapePoster}, {Path: "/a.jpg"}} {
		if r.IsZero() {
			t.Errorf("%+v reads as zero", r)
		}
	}
	for _, s := range []artwork.Shape{artwork.ShapePoster, artwork.ShapeBackdrop, artwork.ShapeLogo, artwork.ShapePhoto, artwork.ShapeCover} {
		got, err := artwork.ParseShape(string(s))
		if err != nil || got != s {
			t.Errorf("ParseShape(%q) = %q, %v", s, got, err)
		}
	}
	for _, s := range []artwork.Size{artwork.SizeSmall, artwork.SizeMedium, artwork.SizeLarge, artwork.SizeOriginal} {
		got, err := artwork.ParseSize(string(s))
		if err != nil || got != s {
			t.Errorf("ParseSize(%q) = %q, %v", s, got, err)
		}
	}
	for _, s := range []string{"", "Poster", "w185"} {
		if _, err := artwork.ParseShape(s); err == nil {
			t.Errorf("ParseShape(%q) accepted", s)
		}
		if _, err := artwork.ParseSize(s); err == nil {
			t.Errorf("ParseSize(%q) accepted", s)
		}
	}
}

type source struct{ id artwork.SourceID }

func (s *source) ID() artwork.SourceID { return s.id }

func (*source) URL(artwork.Shape, string, artwork.Size) (*url.URL, error) { return &url.URL{}, nil }

func (*source) Rate() (perSecond float64, burst int) { return 1, 1 }

func TestSourcesKeepOneSourcePerID(t *testing.T) {
	tmdb, coverart := &source{"tmdb"}, &source{"coverart"}
	s := artwork.NewSources([]artwork.Source{tmdb}, []artwork.Source{tmdb, coverart})
	if len(s) != 2 || s["tmdb"] != tmdb || s["coverart"] != coverart {
		t.Fatalf("sources = %v", s)
	}
	defer func() {
		if recover() == nil {
			t.Error("two sources with one id did not panic")
		}
	}()
	artwork.NewSources([]artwork.Source{tmdb, &source{"tmdb"}})
}
