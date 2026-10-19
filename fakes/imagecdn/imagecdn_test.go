package imagecdn_test

import (
	"errors"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/imagecdn"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/imagecdn"
	"github.com/jatassi/slipstream/fakes/internal/images"
	"github.com/jatassi/slipstream/fakes/scenario"
)

type reply struct {
	status, contentType string
	declared            int64
	body                []byte
	bodyErr             error
	after               time.Duration
}

func get(t *testing.T, f faketest.Served, path string) reply {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.Address.URL(imagecdn.Root+path), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	return reply{status: res.Status, contentType: res.Header.Get("Content-Type"), declared: res.ContentLength, body: body, bodyErr: err, after: time.Since(start)}
}

func inBothTiers(t *testing.T, test func(*testing.T, faketest.Served), opts ...faketest.Option) {
	t.Run("loopback", func(t *testing.T) { test(t, faketest.Loopback(t, imagecdn.Fake, scenario.Scenario{}, opts...)) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, faketest.InMemory(t, imagecdn.Fake, scenario.Scenario{}, opts...)) })
	})
}

// near is whether a decoded color is c, within what JPEG's rounding moves
// a flat field.
func near(a, c color.RGBA) bool {
	d := func(x, y uint8) int { return max(int(x)-int(y), int(y)-int(x)) }
	return d(a.R, c.R) <= 4 && d(a.G, c.G) <= 4 && d(a.B, c.B) <= 4
}

func TestEachPathAndSizeHasItsOwnDominantColour(t *testing.T) {
	type image struct{ path, size, contentType string }
	var all []image
	for _, f := range []struct{ path, contentType string }{
		{"/kqjL17yufvn9OVLyXYpvtyrFfak.jpg", "image/jpeg"},
		{"/8ZTVqvKDQ8emSGUEMjsS4yHAwrp.jpg", "image/jpeg"},
		{"/fiVW06jE7z9YnO4trhaMEdclSiC.png", "image/png"},
		{"/wwemzKWzjKYJFfCeiB57q3r4Bcm.svg", "image/svg+xml"},
		{"/qJ2tW6WMUDux911r6m7haRef0WH.webp", "image/webp"},
	} {
		for _, size := range []string{"w92", "w185", "w500", "h632"} {
			all = append(all, image{f.path, size, f.contentType})
		}
	}
	inBothTiers(t, func(t *testing.T, f faketest.Served) {
		for _, img := range all {
			r := get(t, f, "/"+img.size+img.path)
			got, err := images.Dominant(r.body)
			if r.status != "200 OK" || r.contentType != img.contentType || err != nil {
				t.Errorf("GET %s%s = %s %s, %v; want 200 OK %s", img.size, img.path, r.status, r.contentType, err, img.contentType)
				continue
			}
			for _, other := range all {
				if is := other == img; near(got, images.Color(other.path, other.size)) != is {
					t.Errorf("GET %s%s is mostly %v; %s%s's color is %v", img.size, img.path, got, other.size, other.path, images.Color(other.path, other.size))
				}
			}
		}
	})
}

func TestTheFakeAnswersAsTMDBsImageHostDoes(t *testing.T) {
	inBothTiers(t, func(t *testing.T, f faketest.Served) {
		for path, want := range map[string]string{
			"/w93/a.jpg":      "400 Bad Request text/html <h1>Image size not supported</h1>",
			"/w92/a.gif":      "404 Not Found text/html <h1>File Not Found</h1>",
			"/w92/a":          "404 Not Found text/html <h1>File Not Found</h1>",
			"/w92/a/b.jpg":    "404 Not Found text/html <h1>File Not Found</h1>",
			"/../3/movie/550": "404 Not Found text/html <h1>File Not Found</h1>",
		} {
			if r := get(t, f, path); fmt.Sprintf("%s %s %s", r.status, r.contentType, r.body) != want {
				t.Errorf("GET %s = %s %s %q, want %s", path, r.status, r.contentType, r.body, want)
			}
		}
	})
}

func rule(id, file string, f fault.Fault) fault.Rule {
	return fault.Rule{ID: id, Target: fault.Target(imagecdn.Fake.Name), Route: "image", Params: map[string]string{"file": file}, Fault: f}
}

func TestTruncatedAndWrongTypeFaults(t *testing.T) {
	inBothTiers(t, func(t *testing.T, f faketest.Served) {
		cut := get(t, f, "/w500/truncated.jpg")
		if cut.status != "200 OK" || cut.declared <= int64(len(cut.body)) || !errors.Is(cut.bodyErr, io.ErrUnexpectedEOF) {
			t.Errorf("a truncated image answered %s, declared %d, sent %d and ended with %v; want 200 OK, half sent, then an unexpected EOF", cut.status, cut.declared, len(cut.body), cut.bodyErr)
		}
		wrong := get(t, f, "/w500/wrong.png")
		got, err := images.Dominant(wrong.body)
		if wrong.status != "200 OK" || wrong.contentType != "text/html" || err != nil || got != images.Color("/wrong.png", "w500") {
			t.Errorf("a wrongly typed image answered %s %s mostly %v, %v; want 200 OK text/html and its own PNG", wrong.status, wrong.contentType, got, err)
		}
		if whole := get(t, f, "/w500/whole.jpg"); whole.status != "200 OK" || whole.contentType != "image/jpeg" || whole.bodyErr != nil {
			t.Errorf("an image no rule names answered %s %s, %v", whole.status, whole.contentType, whole.bodyErr)
		}
	}, faketest.WithFaults(rule("truncated", "truncated.jpg", fault.Truncate{}), rule("wrong", "wrong.png", fault.ContentType("text/html"))))
}

func TestMissingAndSlowFaults(t *testing.T) {
	inBothTiers(t, func(t *testing.T, f faketest.Served) {
		if r := get(t, f, "/w500/missing.jpg"); r.status != "404 Not Found" {
			t.Errorf("a missing image answered %s, want 404 Not Found", r.status)
		}
		slow := get(t, f, "/w92/slow.png")
		if want := time.Duration(len(slow.body)) * time.Second / 1024; slow.status != "200 OK" || slow.bodyErr != nil || slow.after < want*9/10 {
			t.Errorf("a slow image answered %s with %d bytes after %v, %v; want them at 1KiB/s, about %v", slow.status, len(slow.body), slow.after, slow.bodyErr, want)
		}
	}, faketest.WithFaults(rule("missing", "missing.jpg", fault.Status{Code: http.StatusNotFound}), rule("slow", "slow.png", fault.SlowBody(1<<10))))
}

func TestTheFakeConforms(t *testing.T) {
	inBothTiers(t, func(t *testing.T, f faketest.Served) {
		conformance.Run(t, conformance.Target{
			Images: f.Address.URL(imagecdn.Root),
			Client: f.Client,
			Known: []conformance.Known{
				{Path: "/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg", Type: "image/jpeg"},
				{Path: "/wwemzKWzjKYJFfCeiB57q3r4Bcm.png", Type: "image/png"},
				{Path: "/wwemzKWzjKYJFfCeiB57q3r4Bcm.svg", Type: "image/svg+xml"},
			},
		})
	})
}
