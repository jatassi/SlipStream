package seeding

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestImportsOnlyTheStandardLibrary(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), f.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); strings.Contains(strings.Split(p, "/")[0], ".") {
				t.Errorf("%s imports %s; seeding imports only the standard library", f.Name(), p)
			}
		}
	}
}
