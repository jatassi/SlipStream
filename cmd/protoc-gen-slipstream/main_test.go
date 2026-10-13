package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

var update = flag.Bool("update", false, "rewrite testdata/placed.golden")

// run compiles testdata/<name>.proto as fixture.proto beside the real
// options.proto with the buf tools/go.mod pins, then runs the plugin on it.
func run(t *testing.T, name string) *pluginpb.CodeGeneratorResponse {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	copyFile(t, filepath.Join("testdata", name+".proto"), filepath.Join(dir, "fixture.proto"))
	copyFile(t, filepath.Join(root, "proto", "slipstream", "node", "v1", "options.proto"), filepath.Join(dir, "slipstream", "node", "v1", "options.proto"))
	if err := os.WriteFile(filepath.Join(dir, "buf.yaml"), []byte("version: v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	image := filepath.Join(t.TempDir(), "image.binpb")
	cmd := exec.CommandContext(t.Context(), "go", "tool", "-modfile="+filepath.Join(root, "tools", "go.mod"), "buf", "build", dir, "--as-file-descriptor-set", "-o", image)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("buf build: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	var set descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(raw, &set); err != nil {
		t.Fatal(err)
	}
	p, err := options.New(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"fixture.proto"},
		Parameter:      proto.String("paths=source_relative,Mslipstream/node/v1/options.proto=github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"),
		ProtoFile:      set.GetFile(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := generate(p); err != nil {
		p.Error(err)
	}
	return p.Response()
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	b, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPlacedServicesGetTypedDescriptors(t *testing.T) {
	res := run(t, "placed")
	if res.GetError() != "" {
		t.Fatalf("error: %s", res.GetError())
	}
	if len(res.GetFile()) != 1 {
		t.Fatalf("got %d files, want 1", len(res.GetFile()))
	}
	f := res.GetFile()[0]
	if f.GetName() != "fixturev1connect/fixture.slipstream.go" {
		t.Errorf("name = %q", f.GetName())
	}
	golden := filepath.Join("testdata", "placed.golden")
	if *update {
		if err := os.WriteFile(golden, []byte(f.GetContent()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if f.GetContent() != string(want) {
		t.Errorf("content:\n%s\nwant:\n%s", f.GetContent(), want)
	}
}

func TestRefusesAnUndispatchableService(t *testing.T) {
	for name, want := range map[string]string{
		"unplaced":  "fixture.proto:7: service EchoService has no served_by option",
		"streaming": "fixture.proto:11: method WatchService.Watch streams; a dispatched service is unary",
		"badrole":   `fixture.proto:9: service SearchingService is served by role "Media_Server", which is not a Role name`,
	} {
		t.Run(name, func(t *testing.T) {
			res := run(t, name)
			if res.GetError() != want {
				t.Errorf("error = %q, want %q", res.GetError(), want)
			}
			if len(res.GetFile()) != 0 {
				t.Errorf("emitted %d files on error", len(res.GetFile()))
			}
		})
	}
}
