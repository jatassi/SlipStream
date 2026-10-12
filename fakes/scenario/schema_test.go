package scenario

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/invopop/jsonschema"
	validator "github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

var update = flag.Bool("update", false, "rewrite scenario.schema.json from the Scenario type")

const (
	schemaFile = "scenario.schema.json"
	modeline   = "# yaml-language-server: $schema=../scenario.schema.json\n"
)

const durationPattern = `^[-+]?(0|(([0-9]+(\.[0-9]*)?|\.[0-9]+)(ns|us|µs|μs|ms|s|m|h))+)$`

func schema(t *testing.T) []byte {
	t.Helper()
	setups := make([]any, len(setupNames))
	for i, name := range setupNames {
		setups[i] = name
	}
	r := jsonschema.Reflector{
		Anonymous:                  true,
		DoNotReference:             true,
		FieldNameTag:               "yaml",
		RequiredFromJSONSchemaTags: true,
		Mapper: func(t reflect.Type) *jsonschema.Schema {
			switch t {
			case reflect.TypeFor[time.Duration]():
				return &jsonschema.Schema{Type: "string", Pattern: durationPattern}
			case reflect.TypeFor[Setup]():
				return &jsonschema.Schema{Type: "string", Enum: setups}
			}
			return nil
		},
	}
	out, err := json.MarshalIndent(r.Reflect(Scenario{}), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(out, '\n')
}

func TestSchema(t *testing.T) {
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
		t.Fatal("scenario.schema.json is stale; run go test ./fakes/scenario -run TestSchema -update")
	}

	compiled := compileSchema(t, got)
	for _, name := range Catalog() {
		data, err := catalog.ReadFile("catalog/" + name + ".yaml")
		if err != nil {
			t.Fatal(err)
		}
		if err := compiled.Validate(yamlValue(t, data)); err != nil {
			t.Errorf("catalog %s: %v", name, err)
		}
	}
	err = compiled.Validate(yamlValue(t, []byte("seed: x\nnodes: [{name: home, setup: none}]\nfavourite: 1\n"))) //nolint:misspell // an unknown key is the point
	if err == nil || !strings.Contains(err.Error(), "additional properties 'favourite' not allowed") {
		t.Fatalf("unknown key validated: %v", err)
	}
}

func TestCatalog(t *testing.T) {
	entries, err := os.ReadDir("catalog")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".yaml")
		got = append(got, name)
		data, err := os.ReadFile("catalog/" + e.Name())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(data, []byte(modeline)) {
			t.Errorf("catalog/%s does not start with %q", e.Name(), modeline)
		}
		if _, err := Load(name); err != nil {
			t.Error(err)
		}
	}
	if want := []string{"first-run", "single-machine"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog = %v, want %v", got, want)
	}
}

func compileSchema(t *testing.T, data []byte) *validator.Schema {
	t.Helper()
	doc, err := validator.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	c := validator.NewCompiler()
	if err := c.AddResource(schemaFile, doc); err != nil {
		t.Fatal(err)
	}
	s, err := c.Compile(schemaFile)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func yamlValue(t *testing.T, data []byte) any {
	t.Helper()
	var v any
	if err := yaml.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	j, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	out, err := validator.UnmarshalJSON(bytes.NewReader(j))
	if err != nil {
		t.Fatal(err)
	}
	return out
}
