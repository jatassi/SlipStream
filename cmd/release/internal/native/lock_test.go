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
	want := []LayerID{{KindLibs, "linux-amd64", "6c4509948055f209c3ec2d01b7f0d552d687946d1852efc1d8569178a41efe9c"}}
	if !slices.Equal(got, want) {
		t.Fatalf("Want = %v, want %v", got, want)
	}
	if got, want := got[0].URL("https://example.test/dl"),
		"https://example.test/dl/layer-6c4509948055f209c3ec2d01b7f0d552d687946d1852efc1d8569178a41efe9c/"+
			"libs-linux-amd64-6c4509948055f209c3ec2d01b7f0d552d687946d1852efc1d8569178a41efe9c.tar.gz"; got != want {
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
