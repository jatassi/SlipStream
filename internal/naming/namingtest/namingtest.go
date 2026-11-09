// Package namingtest checks a Content Type's naming declaration in its
// own tests.
package namingtest

import (
	"math"
	"testing"

	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/works"
)

// Conform fails t once for each rule n's declaration breaks: names and
// aliases unique by key, sample values of their tokens' kinds, sample
// variants the type declares, defaults that pass the save check and
// samples that name.
func Conform(t testing.TB, n naming.Facet) {
	t.Helper()
	for _, problem := range naming.Conformance(n.Naming()) {
		t.Error(problem)
	}
}

// Name is the root-relative path n's default templates give a file of
// sub as an .mkv, with level folders, where the templates say.
func Name(t testing.TB, n naming.Facet, sub naming.Subject, file naming.FileFacts) string {
	t.Helper()
	const typ works.TypeID = "namingtest"
	s, err := naming.New(naming.Config{Facets: map[works.TypeID]naming.Facet{typ: n}}).SchemeIn(t.Context(), nil, typ)
	if err != nil {
		t.Fatal(err)
	}
	named, err := s.Name(naming.Placing{Subject: sub, LevelFolders: true, Landing: naming.Rendered(), File: file, Ext: ".mkv", Budget: math.MaxInt})
	if err != nil {
		t.Fatal(err)
	}
	return named.Rel
}
