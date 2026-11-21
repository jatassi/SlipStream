package storage_test

import (
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/storage"
)

func TestSumTellsSchemasThatMigrateDifferentlyApart(t *testing.T) {
	schema := func(set, second string) storage.Schema[storage.Library] {
		return storage.NewSchema[storage.Library](set, fstest.MapFS{
			"schema/00001_pin.sql": {Data: []byte("-- +goose Up\nCREATE TABLE pin (id INTEGER PRIMARY KEY) STRICT;\n")},
			"schema/00002_pin.sql": {Data: []byte(second)},
		})
	}
	base := schema("pin", "-- +goose Up\nCREATE TABLE pin_two (id INTEGER PRIMARY KEY) STRICT;\n")
	if base.Sum() != schema("pin", "-- +goose Up\nCREATE TABLE pin_two (id INTEGER PRIMARY KEY) STRICT;\n").Sum() {
		t.Error("two schemas with the same set and files have different Sums")
	}
	for name, other := range map[string]storage.Schema[storage.Library]{
		"another set":    schema("pins", "-- +goose Up\nCREATE TABLE pin_two (id INTEGER PRIMARY KEY) STRICT;\n"),
		"another file":   schema("pin", "-- +goose Up\nCREATE TABLE pin_three (id INTEGER PRIMARY KEY) STRICT;\n"),
		"an earlier cut": base.Through(1),
	} {
		if other.Sum() == base.Sum() {
			t.Errorf("%s has the same Sum", name)
		}
	}
}
