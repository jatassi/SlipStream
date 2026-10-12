package native

import (
	"slices"
	"testing"
)

func TestRepoInputsListEveryTarget(t *testing.T) {
	in, err := LoadInputs("../../../../native/inputs.json")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tg := range in.Targets {
		got = append(got, tg.Name+":"+string(tg.Gate))
	}
	want := []string{"linux-amd64:pr", "linux-arm64:native", "windows-amd64:native", "darwin-arm64:native", "darwin-amd64:native"}
	if !slices.Equal(got, want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
}

func TestParseInputsRejectsInconsistentInputs(t *testing.T) {
	for _, tc := range []struct{ inputs, want string }{
		{`{"targets": [{"name": "linux-amd64", "gate": "nightly"}]}`,
			`native inputs: target linux-amd64: gate "nightly" is neither "pr" nor "native"`},
		{`{"targets": [{"name": "linux-amd64", "gate": "pr", "layers": ["libs", "ffmpeg"]}]}`,
			`native inputs: target linux-amd64: unknown layer kind "ffmpeg"`},
		{`{"targets": [{"name": "linux-amd64", "gate": "pr", "layers": ["libs"]}]}`,
			`native inputs: target linux-amd64: the libs layer is built with the toolchain layer, which it does not list`},
		{`{"targets": [{"name": "linux-amd64", "gate": "pr", "toolchain": "musl-cross", "layers": ["toolchain"]}]}`,
			`native inputs: target linux-amd64: toolchain "musl-cross" is not listed`},
		{`{"toolchains": [{"name": "musl-cross", "runtimes": [{"name": "libgcc", "source": "gcc"}]}]}`,
			`native inputs: toolchain musl-cross: runtime libgcc names no source "gcc"`},
	} {
		if _, err := ParseInputs([]byte(tc.inputs)); err == nil || err.Error() != tc.want {
			t.Errorf("ParseInputs(%s) err = %v, want %s", tc.inputs, err, tc.want)
		}
	}
}
