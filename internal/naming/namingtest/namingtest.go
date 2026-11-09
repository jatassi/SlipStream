// Package namingtest checks a Content Type's naming declaration in its
// own tests.
package namingtest

import (
	"testing"

	"github.com/jatassi/slipstream/internal/naming"
)

// Conform fails t once for each rule n's declaration breaks: names and
// aliases unique by key, sample values of their tokens' kinds, sample
// variants the type declares, defaults that pass the save check and
// samples that name.
func Conform(t testing.TB, n naming.Namer) {
	t.Helper()
	for _, problem := range naming.Conformance(n.Naming()) {
		t.Error(problem)
	}
}
