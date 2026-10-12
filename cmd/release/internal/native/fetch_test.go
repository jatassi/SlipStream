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

// publish serves one packed fake libs layer the way a GitHub release does,
// redirecting to a signed URL that accepts only GET, and counts the
// requests it gets.
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
		switch {
		case r.URL.Path == "/"+id.Tag()+"/"+id.AssetName():
			http.Redirect(w, r, "/signed/"+id.AssetName(), http.StatusFound)
		case r.URL.Path == "/signed/"+id.AssetName() && r.Method == http.MethodGet:
			_, _ = w.Write(asset)
		case r.URL.Path == "/signed/"+id.AssetName():
			w.WriteHeader(http.StatusUnauthorized)
		default:
			http.NotFound(w, r)
		}
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
	if n := requests.Load(); n != 2 {
		t.Fatalf("server got %d requests over two fetches, want the redirect and the download", n)
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

func TestFetchedBundleBuildsWithTheToolchainAndLibsLayers(t *testing.T) {
	root, cache := t.TempDir(), t.TempDir()
	in, recipe := fixture(t)
	if err := os.CopyFS(filepath.Join(root, RecipeDir), recipe); err != nil {
		t.Fatal(err)
	}
	want, err := Want(in, recipe)
	if err != nil {
		t.Fatal(err)
	}
	lock := Lock{}
	for i, id := range want {
		lock.Layers = append(lock.Layers, Layer{id, id.AssetName(), id.URL("https://example.test"), strings.Repeat(string(rune('a'+i)), 64)})
	}
	if err := lock.Write(filepath.Join(root, LockPath)); err != nil {
		t.Fatal(err)
	}
	target, err := in.Target("linux-amd64")
	if err != nil {
		t.Fatal(err)
	}

	_, err = FetchedBundle(root, cache, in, target)
	if wantErr := "the toolchain layer for linux-amd64 is not fetched: run scripts/native-deps fetch -target linux-amd64"; err == nil || err.Error() != wantErr {
		t.Fatalf("FetchedBundle err = %v, want %s", err, wantErr)
	}

	toolchain, libs := LayerDir(cache, lock.Layers[0]), LayerDir(cache, lock.Layers[1])
	for path, content := range map[string]string{
		filepath.Join(toolchain, MetadataFile): `{"cc": "bin/x86_64-linux-musl-gcc", "cxx": "bin/x86_64-linux-musl-g++"}`,
		filepath.Join(libs, MetadataFile):      `{"include_dirs": ["include"], "static_libs": ["lib/libtorrent-rasterbar.a"], "system_libs": ["-static", "-lstdc++"]}`,
	} {
		if err := writeFile(path, strings.NewReader(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := FetchedBundle(root, cache, in, target)
	if err != nil {
		t.Fatal(err)
	}
	if b.Libs != lock.Layers[1] {
		t.Errorf("Libs = %+v, want %+v", b.Libs, lock.Layers[1])
	}
	cflags := "-O2 -g -DSLIPSTREAM_NATIVE_BUNDLE -I" + filepath.Join(libs, "include")
	wantEnv := []EnvVar{
		{"CGO_ENABLED", "1"},
		{"GOFLAGS", "-tags=netgo"},
		{"CC", filepath.Join(toolchain, "bin", "x86_64-linux-musl-gcc")},
		{"CXX", filepath.Join(toolchain, "bin", "x86_64-linux-musl-g++")},
		{"CGO_CFLAGS", cflags},
		{"CGO_CXXFLAGS", cflags},
		{"CGO_LDFLAGS", filepath.Join(libs, "lib", "libtorrent-rasterbar.a") + " -static -lstdc++"},
	}
	if !slices.Equal(b.Env, wantEnv) {
		t.Errorf("Env =\n%q\nwant\n%q", b.Env, wantEnv)
	}
}

func TestFetchedBundleBuildsWithXcodeAndTheLibsLayer(t *testing.T) {
	root, cache := t.TempDir(), t.TempDir()
	in, recipe := fixture(t)
	if err := os.CopyFS(filepath.Join(root, RecipeDir), recipe); err != nil {
		t.Fatal(err)
	}
	target, err := in.Target("darwin-arm64")
	if err != nil {
		t.Fatal(err)
	}
	ids, err := in.Layers(target, recipe)
	if err != nil {
		t.Fatal(err)
	}
	libs := Layer{ids[0], ids[0].AssetName(), ids[0].URL("https://example.test"), strings.Repeat("c", 64)}
	if err := (Lock{Layers: []Layer{libs}}).Write(filepath.Join(root, LockPath)); err != nil {
		t.Fatal(err)
	}
	dir := LayerDir(cache, libs)
	metadata := `{"include_dirs": ["include"], "static_libs": ["lib/libssl.a"], "system_libs": ["-lc++"]}`
	if err := writeFile(filepath.Join(dir, MetadataFile), strings.NewReader(metadata), 0o644); err != nil {
		t.Fatal(err)
	}

	b, err := FetchedBundle(root, cache, in, target)
	if err != nil {
		t.Fatal(err)
	}
	cflags := "-O2 -g -DSLIPSTREAM_NATIVE_BUNDLE -I" + filepath.Join(dir, "include")
	wantEnv := []EnvVar{
		{"CGO_ENABLED", "1"},
		{"GOFLAGS", "-tags="},
		{"CC", "clang"},
		{"CXX", "clang++"},
		{"CGO_CFLAGS", cflags},
		{"CGO_CXXFLAGS", cflags},
		{"CGO_LDFLAGS", filepath.Join(dir, "lib", "libssl.a") + " -lc++"},
		{"MACOSX_DEPLOYMENT_TARGET", "13.0"},
	}
	if b.Libs != libs || !slices.Equal(b.Env, wantEnv) {
		t.Errorf("FetchedBundle = %+v, %q\nwant %+v, %q", b.Libs, b.Env, libs, wantEnv)
	}
}

func TestCacheRootFollowsTheEnvironment(t *testing.T) {
	t.Setenv(CacheRootEnv, filepath.Join("ci", "native"))
	got, err := CacheRoot()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("ci", "native"); got != want {
		t.Errorf("CacheRoot() = %q, want %q", got, want)
	}
}
