package metadataapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/imagecdn"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
)

const immutable = "private, max-age=31536000, immutable"

type got struct {
	status int
	header http.Header
	body   []byte
}

func (s *served) image(t *testing.T, r artwork.Ref, size artwork.Size) got {
	t.Helper()
	return s.get(t, minter.URL(r)+"?size="+string(size))
}

func (s *served) get(t *testing.T, path string) got {
	t.Helper()
	res := s.owner.Browse(t, path)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return got{status: res.StatusCode, header: res.Header, body: body}
}

// direct is the image fake-image-cdn answers at path, fetched past the
// Controller.
func (s *served) direct(t *testing.T, path string) []byte {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, s.cdn.Address.URL(path), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.cdn.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s past the Controller = %d, %v", path, res.StatusCode, err)
	}
	return body
}

func poster(path string) artwork.Ref {
	return artwork.Ref{Source: "tmdb-images", Shape: artwork.ShapePoster, Path: path}
}

func (g got) unavailable(t *testing.T, what string) {
	t.Helper()
	var p struct{ Code string }
	if g.status != http.StatusNotFound || g.header.Get("Cache-Control") != "no-store" || json.Unmarshal(g.body, &p) != nil || p.Code != "artwork_unavailable" {
		t.Fatalf("%s answered %d, Cache-Control %q, %s; want 404 artwork_unavailable that no cache keeps", what, g.status, g.header.Get("Cache-Control"), g.body)
	}
}

func (s *served) fetchedExactly(t *testing.T, want ...string) {
	t.Helper()
	if all := s.fetched.all(); !slices.Equal(all, want) {
		t.Fatalf("the Controller fetched %v from the image host, want %v", all, want)
	}
}

func TestEachSizeIsServedImmutable(t *testing.T) {
	s := serve(t)
	r := poster("/kqjL17yufvn9OVLyXYpvtyrFfak.jpg")
	tmdbSize := map[artwork.Size]string{artwork.SizeSmall: "w185", artwork.SizeMedium: "w342", artwork.SizeLarge: "w780", artwork.SizeOriginal: "original"}
	var fetched []string
	for _, size := range []artwork.Size{artwork.SizeSmall, artwork.SizeMedium, artwork.SizeLarge, artwork.SizeOriginal} {
		at := imagecdn.Root + "/" + tmdbSize[size] + r.Path
		fetched = append(fetched, at)
		for range 2 {
			g := s.image(t, r, size)
			if g.status != http.StatusOK || g.header.Get("Content-Type") != "image/jpeg" || g.header.Get("Cache-Control") != immutable ||
				!bytes.Equal(g.body, s.direct(t, at)) {
				t.Fatalf("the %s poster answered %d %s, Cache-Control %q, %d bytes; want 200 image/jpeg, immutable, and the image host's %s",
					size, g.status, g.header.Get("Content-Type"), g.header.Get("Cache-Control"), len(g.body), at)
			}
		}
	}
	s.fetchedExactly(t, fetched...)
}

func TestTheRouteRefusesATokenItDidntIssue(t *testing.T) {
	s := serve(t)
	r := poster("/kqjL17yufvn9OVLyXYpvtyrFfak.jpg")
	payload, mac, _ := strings.Cut(minter.Token(r), ".")
	_, otherMAC, _ := strings.Cut(minter.Token(poster("/8ZTVqvKDQ8emSGUEMjsS4yHAwrp.jpg")), ".")
	for name, token := range map[string]string{
		"another install's": artwork.NewMinter([]byte("elsewhere")).Token(r),
		"a spliced one":     payload + "." + otherMAC,
		"a tampered one":    payload + "A." + mac,
		"garbage":           "not-a-token",
	} {
		s.get(t, artwork.Route+token+"?size=medium").unavailable(t, name+" token")
	}
	s.fetchedExactly(t)
}

func TestAMissAnswers404NoStoreAndRetriesLater(t *testing.T) {
	missing := fault.Rule{ID: "missing", Target: fault.Target(imagecdn.Fake.Name), Route: "image", Params: map[string]string{"file": "gone.jpg"}, Fault: fault.Status{Code: http.StatusNotFound}, Times: 1}
	r := poster("/gone.jpg")
	at := imagecdn.Root + "/w342/gone.jpg"

	t.Run("inside the retry time", func(t *testing.T) {
		s := serveWorld(t, world{retry: 3600e9, faults: []fault.Rule{missing}})
		s.image(t, r, artwork.SizeMedium).unavailable(t, "an image the host hasn't")
		s.image(t, r, artwork.SizeMedium).unavailable(t, "the same image asked again")
		s.fetchedExactly(t, at)
	})
	t.Run("once the retry time has passed", func(t *testing.T) {
		s := serveWorld(t, world{faults: []fault.Rule{missing}})
		s.image(t, r, artwork.SizeMedium).unavailable(t, "an image the host hasn't")
		if g := s.image(t, r, artwork.SizeMedium); g.status != http.StatusOK || !bytes.Equal(g.body, s.direct(t, at)) {
			t.Fatalf("asked again after its retry time the image answered %d with %d bytes, want 200 and the image", g.status, len(g.body))
		}
		s.fetchedExactly(t, at, at)
	})
}

func TestABrokenImageIsNeverServed(t *testing.T) {
	rule := func(id, file string, f fault.Fault, times int) fault.Rule {
		return fault.Rule{ID: id, Target: fault.Target(imagecdn.Fake.Name), Route: "image", Params: map[string]string{"file": file}, Fault: f, Times: times}
	}
	s := serveWorld(t, world{faults: []fault.Rule{
		rule("cut", "cut.jpg", fault.Truncate{}, quick.Attempts),
		rule("html", "html.png", fault.ContentType("text/html"), 1),
	}})
	for _, file := range []string{"/cut.jpg", "/html.png"} {
		s.image(t, poster(file), artwork.SizeLarge).unavailable(t, file+" arriving broken")
		g := s.image(t, poster(file), artwork.SizeLarge)
		if want := s.direct(t, imagecdn.Root+"/w780"+file); g.status != http.StatusOK || !bytes.Equal(g.body, want) {
			t.Errorf("%s arriving whole answered %d with %d bytes, want 200 and the image's %d", file, g.status, len(g.body), len(want))
		}
	}
}

func TestSVGCarriesItsCSP(t *testing.T) {
	s := serve(t)
	logo := artwork.Ref{Source: "tmdb-images", Shape: artwork.ShapeLogo, Path: "/wwemzKWzjKYJFfCeiB57q3r4Bcm.svg"}
	g := s.image(t, logo, artwork.SizeSmall)
	if g.status != http.StatusOK || g.header.Get("Content-Type") != "image/svg+xml" || g.header.Get("Cache-Control") != immutable ||
		g.header.Get("Content-Security-Policy") != "default-src 'none'; style-src 'unsafe-inline'; sandbox" {
		t.Fatalf("an SVG logo answered %d %s, Cache-Control %q, CSP %q; want 200 image/svg+xml, immutable, under a policy that runs nothing",
			g.status, g.header.Get("Content-Type"), g.header.Get("Cache-Control"), g.header.Get("Content-Security-Policy"))
	}
}

func TestAnImageFailureNeverMovesAProvidersStatus(t *testing.T) {
	s := serveWorld(t, world{faults: []fault.Rule{
		{ID: "refused", Target: fault.Target(imagecdn.Fake.Name), Route: "image", Fault: fault.Status{Code: http.StatusUnauthorized}},
	}})
	s.image(t, poster("/kqjL17yufvn9OVLyXYpvtyrFfak.jpg"), artwork.SizeMedium).unavailable(t, "an image the host refused")
	s.owner.Expect(t, http.MethodGet, "/api/v1/metadata/providers", "", http.StatusOK, `{"providers":[`+unusedTMDB+`,`+unusedOMDb+`]}`)
}
