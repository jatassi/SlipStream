package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProtoDriftNamesEveryStaleFile(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	pb := t.TempDir()
	if err := os.CopyFS(pb, os.DirFS(filepath.Join(root, "internal", "pb"))); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := protoDrift(t.Context(), root, pb, &out); err != nil {
		t.Fatalf("fresh copy: %v\n%s", err, out.String())
	}

	edited := filepath.Join(pb, "slipstream", "link", "v1", "link.pb.go")
	if err := os.WriteFile(edited, []byte("package linkv1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pb, "slipstream", "extra.go"), []byte("package slipstream\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	err = protoDrift(t.Context(), root, pb, &out)
	want := "internal/pb is stale; run go run ./cmd/release proto\n" +
		"  internal/pb/slipstream/extra.go\n" +
		"  internal/pb/slipstream/link/v1/link.pb.go\n"
	if out.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", out.String(), want)
	}
	if err == nil || err.Error() != "internal/pb is stale" {
		t.Errorf("err = %v, want internal/pb is stale", err)
	}
}

func TestFrozenImportsRefusesAnythingButGoogleProtobuf(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "proto", "slipstream", "link", "v1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := "syntax = \"proto3\";\n\npackage slipstream.link.v1;\n\nimport \"google/protobuf/any.proto\";\nimport public \"slipstream/node/v1/options.proto\";\n"
	if err := os.WriteFile(filepath.Join(dir, "link.proto"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	err := frozenImports(root)
	want := "proto/slipstream/link/v1/link.proto:6: imports slipstream/node/v1/options.proto; slipstream.link.v1 imports nothing outside google/protobuf"
	if err == nil || err.Error() != want {
		t.Errorf("err = %v, want %s", err, want)
	}
}

const frozenFixture = `syntax = "proto3";

package slipstream.link.v1;

message PairRequest {
  bytes secret = 1;
  string name = 2;
}
`

func frozenRepo(t *testing.T, tags ...string) string {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q", "-b", "main")
	write("proto/buf.yaml", "version: v2\nbreaking:\n  use:\n    - FILE\n")
	write("proto/slipstream/link/v1/link.proto", frozenFixture)
	git("add", ".")
	git("commit", "-q", "-m", "release")
	for _, tag := range tags {
		git("tag", tag)
	}
	git("checkout", "-q", "-b", "change")
	write("proto/slipstream/link/v1/link.proto", strings.Replace(frozenFixture, "bytes secret", "string secret", 1))
	git("commit", "-q", "-am", "retype the secret")
	return dir
}

func TestProtoFrozenRunsBufBreakingAgainstTheLatestV1Tag(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := frozenRepo(t, "v0.9.0", "v1.0.0")
	var out strings.Builder
	err = protoFrozen(t.Context(), &out, root, dir, "main")
	want := "slipstream/link/v1/link.proto:6:3:Field \"1\" with name \"secret\" on message \"PairRequest\" changed type from \"bytes\" to \"string\".\n"
	if out.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", out.String(), want)
	}
	if err == nil || !strings.HasPrefix(err.Error(), "slipstream.link.v1 breaks against v1.0.0: ") {
		t.Errorf("err = %v, want a break against v1.0.0", err)
	}
}

func TestProtoFrozenWithoutAV1TagIsNotFrozenYet(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := frozenRepo(t, "v0.9.0")
	var out strings.Builder
	if err := protoFrozen(t.Context(), &out, root, dir, "main"); err != nil {
		t.Fatal(err)
	}
	if out.String() != "no v1+ release tag; slipstream.link.v1 is not frozen yet\n" {
		t.Errorf("output = %q", out.String())
	}
}
