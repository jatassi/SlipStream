package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/cmd/release/internal/pack"
)

func TestTagOutputs(t *testing.T) {
	for _, tc := range []struct {
		tag  string
		want [][2]string
	}{
		{"v1.2.0", [][2]string{
			{"image", "ghcr.io/jatassi/slipstream"}, {"version", "1.2.0"}, {"prerelease", "false"},
			{"image-tags", "ghcr.io/jatassi/slipstream:1.2.0 ghcr.io/jatassi/slipstream:1.2 ghcr.io/jatassi/slipstream:1 ghcr.io/jatassi/slipstream:latest"},
		}},
		{"v1.3.0-beta.2", [][2]string{
			{"image", "ghcr.io/jatassi/slipstream"}, {"version", "1.3.0-beta.2"}, {"prerelease", "true"}, {"image-tags", "ghcr.io/jatassi/slipstream:beta"},
		}},
	} {
		v, err := pack.ParseTag(tc.tag)
		if err != nil {
			t.Fatal(err)
		}
		if got := tagOutputs(v); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("tagOutputs(%s) = %v, want %v", tc.tag, got, tc.want)
		}
	}
}

func TestReleaseNotesStateTheFloors(t *testing.T) {
	in, err := native.LoadInputs(filepath.Join("..", "..", native.InputsPath))
	if err != nil {
		t.Fatal(err)
	}
	got := releaseNotes("v1.2.0", "1.2.0", in)
	const want = "SlipStream 1.2.0\n\n## Supported systems\n\n" +
		"- Windows 10 or later (windows-amd64)\n" +
		"- macOS 13.0 or later (darwin-arm64)\n" +
		"- Linux with glibc 2.28 and kernel 4.18 or later (linux-amd64, linux-arm64)\n" +
		"\nInstall steps for each system: https://github.com/jatassi/SlipStream/blob/v1.2.0/docs/install.md\n"
	if got != want {
		t.Fatalf("release notes:\n%s\nwant:\n%s", got, want)
	}
}
