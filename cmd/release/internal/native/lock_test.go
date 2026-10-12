package native

import (
	"slices"
	"testing"
)

func TestWantListsTheLayersEachTargetShips(t *testing.T) {
	in, recipe := fixture(t)
	got, err := Want(in, recipe)
	if err != nil {
		t.Fatal(err)
	}
	want := []LayerID{
		{KindToolchain, "linux-amd64", "c871a31bd840d2719733f3d367edc4c1807ebcd6ba999b630294eb6bab409d39"},
		{KindLibs, "linux-amd64", "5198b18496aea3325f343b78e62f606383025afa4445da5ef2a971e1b13ae398"},
		{KindLibs, "darwin-arm64", "8c0bac6bc6c590541977390e9181d8c0cd634f805c78e61568ea806710f41548"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Want = %v, want %v", got, want)
	}
	if got, want := got[1].URL("https://example.test/dl"),
		"https://example.test/dl/layer-5198b18496aea3325f343b78e62f606383025afa4445da5ef2a971e1b13ae398/"+
			"libs-linux-amd64-5198b18496aea3325f343b78e62f606383025afa4445da5ef2a971e1b13ae398.tar.gz"; got != want {
		t.Fatalf("URL = %s, want %s", got, want)
	}
}

func TestStaleNamesEachDifference(t *testing.T) {
	pinned := func(kind Kind, target, digest string) Layer {
		return Layer{LayerID: LayerID{kind, target, digest}}
	}
	want := []LayerID{{KindLibs, "linux-amd64", "aaa"}, {KindLibs, "linux-arm64", "bbb"}, {KindLibs, "darwin-arm64", "ccc"}}
	lock := Lock{Layers: []Layer{
		pinned(KindLibs, "linux-amd64", "aaa"),
		pinned(KindLibs, "linux-arm64", "old"),
		pinned(KindLibs, "windows-amd64", "ddd"),
	}}
	got := lock.Stale(want)
	wantStale := []string{
		"pins libs layer ddd for windows-amd64, which native/inputs.json does not list",
		"pins libs layer old for linux-arm64, inputs and recipe build bbb",
		"pins no libs layer for darwin-arm64, inputs and recipe build ccc",
	}
	if !slices.Equal(got, wantStale) {
		t.Fatalf("Stale =\n%q\nwant\n%q", got, wantStale)
	}
	if got := (Lock{Layers: []Layer{pinned(KindLibs, "linux-amd64", "aaa")}}).Stale(want[:1]); len(got) != 0 {
		t.Fatalf("Stale of a current lock = %q", got)
	}
}
