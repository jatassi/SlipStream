package imports

import (
	"maps"
	"regexp"
	"slices"
	"testing"
)

func TestEverySourceHasARule(t *testing.T) {
	var table string
	if err := newWorld(t).lib.QueryRowContext(t.Context(), "SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'imports_import'").Scan(&table); err != nil {
		t.Fatal(err)
	}
	check := regexp.MustCompile(`source IN \(([^)]*)\)`).FindStringSubmatch(table)
	if check == nil {
		t.Fatalf("imports_import has no source check: %s", table)
	}
	var stored []Source
	for _, m := range regexp.MustCompile(`'([^']*)'`).FindAllStringSubmatch(check[1], -1) {
		stored = append(stored, Source(m[1]))
	}
	slices.Sort(stored)
	if ruled := slices.Sorted(maps.Keys(sourceRules)); !slices.Equal(ruled, stored) {
		t.Errorf("sources with a rule %v, sources the store takes %v", ruled, stored)
	}
}
