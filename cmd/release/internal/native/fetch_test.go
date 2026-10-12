package native

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
)

const fakeDigest = "d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1"

// publish serves one packed fake libs layer the way a GitHub release does
// and counts the requests it gets.
func publish(t *testing.T) (base string, asset []byte, requests *atomic.Int32) {
	t.Helper()
	layer := t.TempDir()
	for path, content := range map[string]string{
		MetadataFile:                `{"include_dirs": ["include"], "static_libs": ["lib/libfake.a"]}`,
		"include/fake.h":            "#define FAKE 1\n",
		"lib/libfake.a":             "!<arch>\n",
		"licenses/fake-license.txt": "MIT\n",
	} {
		if err := writeFile(filepath.Join(layer, filepath.FromSlash(path)), strings.NewReader(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	archive := filepath.Join(t.TempDir(), "layer.tar.gz")
	if err := pack(layer, archive); err != nil {
		t.Fatal(err)
	}
	asset, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	requests = &atomic.Int32{}
	id := LayerID{KindLibs, "linux-amd64", fakeDigest}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/"+id.Tag()+"/"+id.AssetName() {
			http.NotFound(w, r)
			return
		}
		w.Write(asset)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, asset, requests
}

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestFetchVerifiesExtractsAndThenMakesNoRequest(t *testing.T) {
	base, asset, requests := publish(t)
	id := LayerID{KindLibs, "linux-amd64", fakeDigest}
	layer := Layer{id, id.AssetName(), id.URL(base), sha(asset)}
	cache := t.TempDir()

	dir, err := Fetch(context.Background(), cache, layer)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(cache, "libs", "linux-amd64", sha(asset)); dir != want {
		t.Fatalf("Fetch dir = %s, want %s", dir, want)
	}
	header, err := os.ReadFile(filepath.Join(dir, "include", "fake.h"))
	if err != nil || string(header) != "#define FAKE 1\n" {
		t.Fatalf("include/fake.h = %q, %v", header, err)
	}
	md, err := ReadMetadata(dir)
	if err != nil || !slices.Equal(md.StaticLibs, []string{"lib/libfake.a"}) {
		t.Fatalf("metadata = %+v, %v", md, err)
	}
	if entries, _ := os.ReadDir(filepath.Dir(dir)); len(entries) != 1 {
		t.Fatalf("%s holds %d entries, want only the layer", filepath.Dir(dir), len(entries))
	}

	again, err := Fetch(context.Background(), cache, layer)
	if err != nil || again != dir {
		t.Fatalf("second Fetch = %s, %v", again, err)
	}
	if n := requests.Load(); n != 1 {
		t.Fatalf("server got %d requests over two fetches, want 1", n)
	}
}

func TestFetchWithWrongSHALeavesNothing(t *testing.T) {
	base, asset, _ := publish(t)
	id := LayerID{KindLibs, "linux-amd64", fakeDigest}
	wrong := strings.Repeat("0", 64)
	layer := Layer{id, id.AssetName(), id.URL(base), wrong}
	cache := t.TempDir()

	_, err := Fetch(context.Background(), cache, layer)
	want := layer.URL + " has SHA-256 " + sha(asset) + ", native/lock.json pins " + wrong
	if err == nil || err.Error() != want {
		t.Fatalf("Fetch err = %v, want %s", err, want)
	}
	entries, err := os.ReadDir(filepath.Join(cache, "libs", "linux-amd64"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("cache holds %v (%v) after a bad fetch, want nothing", entries, err)
	}
}

func TestPinHashesPublishedAssets(t *testing.T) {
	base, asset, _ := publish(t)
	id := LayerID{KindLibs, "linux-amd64", fakeDigest}

	lock, err := Pin(context.Background(), base, []LayerID{id})
	if err != nil {
		t.Fatal(err)
	}
	want := Lock{Layers: []Layer{{id, "libs-linux-amd64-" + fakeDigest + ".tar.gz", base + "/layer-" + fakeDigest + "/libs-linux-amd64-" + fakeDigest + ".tar.gz", sha(asset)}}}
	if !slices.Equal(lock.Layers, want.Layers) {
		t.Fatalf("Pin = %+v, want %+v", lock, want)
	}

	unpublished := LayerID{KindLibs, "linux-arm64", fakeDigest}
	_, err = Pin(context.Background(), base, []LayerID{id, unpublished})
	if wantErr := unpublished.URL(base) + " is not published; " + PublishHint; err == nil || err.Error() != wantErr {
		t.Fatalf("Pin err = %v, want %s", err, wantErr)
	}
}

func TestPublishedChecksTheAsset(t *testing.T) {
	base, _, _ := publish(t)
	for _, tc := range []struct {
		id   LayerID
		want bool
	}{
		{LayerID{KindLibs, "linux-amd64", fakeDigest}, true},
		{LayerID{KindLibs, "linux-arm64", fakeDigest}, false},
	} {
		got, err := Published(context.Background(), base, tc.id)
		if err != nil || got != tc.want {
			t.Errorf("Published(%s) = %v, %v; want %v", tc.id.AssetName(), got, err, tc.want)
		}
	}
}

func TestFetchedLibsNamesTheFetchCommand(t *testing.T) {
	root := t.TempDir()
	in, recipe := fixture(t)
	if err := os.CopyFS(filepath.Join(root, RecipeDir), recipe); err != nil {
		t.Fatal(err)
	}
	want, err := Want(in, recipe)
	if err != nil {
		t.Fatal(err)
	}
	lock := Lock{Layers: []Layer{{want[0], want[0].AssetName(), want[0].URL("https://example.test"), strings.Repeat("a", 64)}}}
	if err := lock.Write(filepath.Join(root, LockPath)); err != nil {
		t.Fatal(err)
	}
	target, err := in.Target("linux-amd64")
	if err != nil {
		t.Fatal(err)
	}

	_, err = FetchedLibs(root, t.TempDir(), in, target)
	if wantErr := "the libs layer for linux-amd64 is not fetched: run scripts/native-deps fetch -target linux-amd64"; err == nil || err.Error() != wantErr {
		t.Fatalf("FetchedLibs err = %v, want %s", err, wantErr)
	}
}
