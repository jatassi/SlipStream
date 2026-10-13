package main

import (
	"errors"
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

// releaseIn runs the release command args in dir, as go run ./cmd/release
// does there, with the repo's tools module beside it so buf resolves.
func releaseIn(t *testing.T, dir, args string) (code int, stdout, stderr string) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tools/go.mod", "tools/go.sum"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "tools"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), self)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), mainArgs+"="+args)
	var out, errOut strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err = cmd.Run()
	if _, exit := errors.AsType[*exec.ExitError](err); err != nil && !exit {
		t.Fatal(err)
	}
	return cmd.ProcessState.ExitCode(), out.String(), errOut.String()
}

func TestProtoFrozenExitCodes(t *testing.T) {
	code, stdout, stderr := releaseIn(t, frozenRepo(t, "v0.9.0", "v1.0.0"), "proto-frozen -base main")
	finding := "slipstream/link/v1/link.proto:6:3:Field \"1\" with name \"secret\" on message \"PairRequest\" changed type from \"bytes\" to \"string\".\n"
	if want := "release proto-frozen: slipstream.link.v1 breaks against v1.0.0: exit status 100\n"; code != 1 || stdout != finding || stderr != want {
		t.Errorf("a break: exit %d, stdout %q, stderr %q; want exit 1, %q, %q", code, stdout, stderr, finding, want)
	}
	code, stdout, stderr = releaseIn(t, frozenRepo(t, "v0.9.0"), "proto-frozen -base main")
	if want := "no v1+ release tag; slipstream.link.v1 is not frozen yet\n"; code != 0 || stdout != want || stderr != "" {
		t.Errorf("no v1 tag: exit %d, stdout %q, stderr %q; want exit 0, %q", code, stdout, stderr, want)
	}
}
