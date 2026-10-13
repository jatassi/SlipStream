package world_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/invopop/jsonschema"
	validator "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
)

var update = flag.Bool("update", false, "rewrite manifest.schema.json from the Manifest type")

const schemaFile = "manifest.schema.json"

func schema(t *testing.T) []byte {
	t.Helper()
	r := jsonschema.Reflector{
		Anonymous:      true,
		DoNotReference: true,
		Mapper: func(t reflect.Type) *jsonschema.Schema {
			if t == reflect.TypeFor[scenario.Setup]() {
				return &jsonschema.Schema{Type: "string", Enum: []any{scenario.SetupNone.String(), scenario.SetupController.String(), scenario.SetupJoin.String()}}
			}
			return nil
		},
	}
	out, err := json.MarshalIndent(r.Reflect(world.Manifest{}), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(out, '\n')
}

func TestManifestSchema(t *testing.T) {
	want := schema(t)
	if *update {
		if err := os.WriteFile(schemaFile, want, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(schemaFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("manifest.schema.json is stale; run go test ./fakes/world -run TestManifestSchema -update")
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
