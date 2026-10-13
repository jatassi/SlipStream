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
	want := []string{"linux-amd64:pr", "linux-arm64:native", "windows-amd64:native", "darwin-arm64:native"}
	if !slices.Equal(got, want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
}

func TestParseInputsRejectsInconsistentInputs(t *testing.T) {
	for _, tc := range []struct{ inputs, want string }{
		{`{"targets": [{"name": "linux-amd64", "gate": "nightly"}]}`,
			`native inputs: target linux-amd64: gate "nightly" is neither "pr" nor "native"`},
		{`{"targets": [{"name": "linux-amd64", "gate": "pr", "toolchain": "musl-cross"}]}`,
			`native inputs: target linux-amd64: toolchain "musl-cross" is not listed`},
		{`{"toolchains": [{"name": "musl-cross", "runtimes": [{"name": "libgcc", "source": "gcc"}]}]}`,
			`native inputs: toolchain musl-cross: lists either sources or an Xcode version, not both or neither`},
		{`{"toolchains": [{"name": "xcode", "xcode": "16.4", "sources": [{"name": "clang"}]}]}`,
			`native inputs: toolchain xcode: lists either sources or an Xcode version, not both or neither`},
		{`{"toolchains": [{"name": "musl-cross", "sources": [{"name": "musl", "sha256": "d585fd3b613c66151fc3249e8ed44f77020cb5e6c1e635a616d3f9f82460512a"}],
		   "runtimes": [{"name": "libgcc", "source": "gcc"}]}]}`,
			`native inputs: toolchain musl-cross: runtime libgcc names no source "gcc"`},
		{`{"targets": [{"name": "linux-amd64", "gate": "pr", "toolchain": "xcode"}], "toolchains": [{"name": "xcode", "xcode": "16.4"}],
		  "tools": [{"name": "ffmpeg", "archives": {}}]}`,
			`native inputs: tool ffmpeg: no archive for target linux-amd64`},
		{`{"tools": [{"name": "ffmpeg", "archives": {"plan9-amd64": "https://example.test/ffmpeg.zip"}}]}`,
			`native inputs: tool ffmpeg: archive for unknown target "plan9-amd64"`},
	} {
		if _, err := ParseInputs([]byte(tc.inputs)); err == nil || err.Error() != tc.want {
			t.Errorf("ParseInputs(%s) err = %v, want %s", tc.inputs, err, tc.want)
		}
	}
}
