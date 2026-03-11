// Package moduletest provides test helpers for validating module implementations.
// Contributors use these helpers in standard _test.go files:
//
//	func TestMyModule(t *testing.T) {
//	    mod := mymodule.New(...)
//	    moduletest.RunSchemaTest(t, mod)
//	    moduletest.RunParsingTest(t, mod, fixtures)
//	}
package moduletest
