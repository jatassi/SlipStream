package coverart_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/contenttypes/music/coverart"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/images"
	"github.com/jatassi/slipstream/fakes/scenario"
)

type reply struct {
	status, contentType string
	body                []byte
}

func get(t *testing.T, f faketest.Served, path string) reply {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.Address.URL(path), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return reply{status: res.Status, contentType: res.Header.Get("Content-Type"), body: body}
}

func inBothTiers(t *testing.T, test func(*testing.T, faketest.Served)) {
	t.Run("loopback", func(t *testing.T) { test(t, faketest.Loopback(t, coverart.Fake, scenario.Scenario{})) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, faketest.InMemory(t, coverart.Fake, scenario.Scenario{})) })
	})
}

func near(a, c color.RGBA) bool {
	d := func(x, y uint8) int { return max(int(x)-int(y), int(y)-int(x)) }
	return d(a.R, c.R) <= 4 && d(a.G, c.G) <= 4 && d(a.B, c.B) <= 4
}

func TestEachReleaseGroupAndSizeHasItsOwnFrontCover(t *testing.T) {
	type cover struct {
		mbid, image string
		edge        int
	}
	var all []cover
	for _, mbid := range []string{"b2000000-0000-4000-8000-000000000001", "b2000000-0000-4000-8000-000000000002"} {
		all = append(all, cover{mbid, "front", 1600})
		for _, edge := range coverart.Thumbnails {
			all = append(all, cover{mbid, fmt.Sprintf("front-%d", edge), edge})
		}
	}
	inBothTiers(t, func(t *testing.T, f faketest.Served) {
		for _, c := range all {
			path := "/release-group/" + c.mbid + "/" + c.image
			r := get(t, f, path)
			cfg, format, err := image.DecodeConfig(bytes.NewReader(r.body))
			if r.status != "200 OK" || r.contentType != "image/jpeg" || err != nil || format != "jpeg" || cfg.Width != c.edge || cfg.Height != c.edge {
				t.Errorf("GET %s = %s %s, %s %dx%d, %v; want a %dx%d JPEG", path, r.status, r.contentType, format, cfg.Width, cfg.Height, err, c.edge, c.edge)
				continue
			}
			got, err := images.Dominant(r.body)
			if err != nil {
				t.Fatal(err)
			}
			for _, other := range all {
				if is := other == c; near(got, images.Color(other.mbid, other.image)) != is {
					t.Errorf("GET %s is mostly %v; %s %s's color is %v", path, got, other.mbid, other.image, images.Color(other.mbid, other.image))
				}
			}
		}
	})
}

func TestAnythingElseIsNotFound(t *testing.T) {
	inBothTiers(t, func(t *testing.T, f faketest.Served) {
		for _, path := range []string{
			"/release-group/b2000000-0000-4000-8000-000000000001/front-300",
			"/release-group/b2000000-0000-4000-8000-000000000001/back",
			"/release-group/b2000000-0000-4000-8000-000000000001/front-",
			"/release/c2000000-0000-4000-8000-000000000001/front",
			"/",
		} {
			if r := get(t, f, path); r.status != "404 Not Found" {
				t.Errorf("GET %s = %s, want 404 Not Found", path, r.status)
			}
		}
	})
}
