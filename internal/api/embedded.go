package api

import (
	"fmt"
	"reflect"
)

// mustDocumentEmbedded panics when t reaches a struct that embeds an
// unexported struct type: encoding/json promotes its fields, but Huma
// leaves them out of the document.
func mustDocumentEmbedded(t reflect.Type, seen map[reflect.Type]bool) {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || seen[t] {
		return
	}
	seen[t] = true
	for f := range t.Fields() {
		if f.Anonymous && !f.IsExported() && f.Type.Kind() == reflect.Struct {
			panic(fmt.Sprintf("api: %s embeds unexported %s, whose fields the document would leave out; export it", t, f.Type))
		}
		mustDocumentEmbedded(f.Type, seen)
	}
}
