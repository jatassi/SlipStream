package world_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/invopop/jsonschema"
	validator "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
)

var update = flag.Bool("update", false, "rewrite manifest.schema.json and web/harness/manifest.gen.ts from the Manifest type")

const (
	schemaFile = "manifest.schema.json"
	zodFile    = "../../web/harness/manifest.gen.ts"
)

func schema(t *testing.T) (jsonSchema, zod []byte) {
	t.Helper()
	r := jsonschema.Reflector{
		Anonymous:      true,
		DoNotReference: true,
		Mapper: func(t reflect.Type) *jsonschema.Schema {
			switch t {
			case reflect.TypeFor[scenario.Setup]():
				return &jsonschema.Schema{Type: "string", Enum: []any{scenario.SetupNone.String(), scenario.SetupController.String(), scenario.SetupJoin.String()}}
			case reflect.TypeFor[world.NodeState]():
				return &jsonschema.Schema{Type: "string", Enum: []any{string(world.NodeRunning), string(world.NodeStopped), string(world.NodeKilled), string(world.NodeExited)}}
			case reflect.TypeFor[world.Claim]():
				return &jsonschema.Schema{Type: "string", Enum: []any{string(world.ClaimUnclaimed), string(world.ClaimController), string(world.ClaimWorker)}}
			}
			return nil
		},
	}
	s := r.Reflect(world.Manifest{})
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	ts, err := zodModule("manifestSchema", s)
	if err != nil {
		t.Fatal(err)
	}
	return append(out, '\n'), []byte(ts)
}

func TestManifestSchema(t *testing.T) {
	jsonSchema, zod := schema(t)
	for _, f := range []struct {
		path string
		want []byte
	}{{schemaFile, jsonSchema}, {zodFile, zod}} {
		if *update {
			if err := os.WriteFile(f.path, f.want, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		got, err := os.ReadFile(f.path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, f.want) {
			t.Errorf("%s is stale; run go test ./fakes/world -run TestManifestSchema -update", filepath.Base(f.path))
		}
	}
}

func validateManifest(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	schemaDoc, err := os.Open(schemaFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = schemaDoc.Close() }()
	doc, err := validator.UnmarshalJSON(schemaDoc)
	if err != nil {
		t.Fatal(err)
	}
	c := validator.NewCompiler()
	if err := c.AddResource(schemaFile, doc); err != nil {
		t.Fatal(err)
	}
	compiled, err := c.Compile(schemaFile)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := validator.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.Validate(manifest); err != nil {
		t.Fatalf("%s does not match %s: %v", path, schemaFile, err)
	}
}
