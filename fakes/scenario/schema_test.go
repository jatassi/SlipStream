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

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
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
	var roles []any
	for _, name := range role.Names() {
		roles = append(roles, string(name))
	}
	var qualities []any
	for _, q := range release.Qualities() {
		qualities = append(qualities, q.String())
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
			case reflect.TypeFor[role.Name]():
				return &jsonschema.Schema{Type: "string", Enum: roles}
			case reflect.TypeFor[Bytes]():
				return &jsonschema.Schema{Type: "string", Pattern: bytesPattern}
			case reflect.TypeFor[QualityKey]():
				return &jsonschema.Schema{Type: "string", Enum: qualities}
			case reflect.TypeFor[fault.Fault]():
				return &jsonschema.Schema{Type: "string"}
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
	if want := []string{"cgnat-home", "first-run", "home", "music", "serving", "single-machine", "sizes", "targets", "torrents"}; !reflect.DeepEqual(got, want) {
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

// yamlValue reads data as an editor's YAML 1.2 schema does, where an
// unquoted 2021-04-16 is a string, not a timestamp.
func yamlValue(t *testing.T, data []byte) any {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	untime(&doc)
	var v any
	if err := doc.Decode(&v); err != nil {
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

func untime(n *yaml.Node) {
	if n.Tag == "!!timestamp" {
		n.Tag = "!!str"
	}
	for _, c := range n.Content {
		untime(c)
	}
}
