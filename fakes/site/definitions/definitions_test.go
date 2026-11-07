package definitions_test

import (
	"io/fs"
	"testing"

	"github.com/jatassi/slipstream/fakes/site/definitions"
	"github.com/jatassi/slipstream/internal/cardigann"
)

func TestEachDefinitionLoadsUnlessItIsWrittenNotTo(t *testing.T) {
	refused := map[string]bool{"broken.yml": true, "larkspur-v3.yml": true}
	files, err := fs.Glob(definitions.FS, "*.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		src, err := fs.ReadFile(definitions.FS, f)
		if err != nil {
			t.Fatal(err)
		}
		_, err = cardigann.Load(src)
		if (err != nil) != refused[f] {
			t.Errorf("Load(%s) = %v, want refused %t", f, err, refused[f])
		}
	}
}
