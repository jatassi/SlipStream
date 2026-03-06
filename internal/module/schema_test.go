package module

import (
	"testing"
)

func TestNodeSchema_Validate_MovieFlat(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "movie", PluralName: "movies", IsRoot: true, IsLeaf: true},
		},
	}
	if err := schema.Validate(); err != nil {
		t.Fatalf("expected valid movie schema, got error: %v", err)
	}
}

func TestNodeSchema_Validate_TV3Level(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "series", PluralName: "series", IsRoot: true},
			{Name: "season", PluralName: "seasons"},
			{Name: "episode", PluralName: "episodes", IsLeaf: true},
		},
	}
	if err := schema.Validate(); err != nil {
		t.Fatalf("expected valid TV schema, got error: %v", err)
	}
}

func TestNodeSchema_Validate_NoLevels(t *testing.T) {
	schema := NodeSchema{}
	if err := schema.Validate(); err == nil {
		t.Fatal("expected error for empty schema, got nil")
	}
}

func TestNodeSchema_Validate_NoRoot(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "episode", PluralName: "episodes", IsLeaf: true},
		},
	}
	err := schema.Validate()
	if err == nil {
		t.Fatal("expected error for schema with no root, got nil")
	}
}

func TestNodeSchema_Validate_NoLeaf(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "series", PluralName: "series", IsRoot: true},
		},
	}
	err := schema.Validate()
	if err == nil {
		t.Fatal("expected error for schema with no leaf, got nil")
	}
}

func TestNodeSchema_Validate_RootNotFirst(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "season", PluralName: "seasons"},
			{Name: "series", PluralName: "series", IsRoot: true},
			{Name: "episode", PluralName: "episodes", IsLeaf: true},
		},
	}
	err := schema.Validate()
	if err == nil {
		t.Fatal("expected error for root not first, got nil")
	}
}

func TestNodeSchema_Validate_LeafNotLast(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "series", PluralName: "series", IsRoot: true},
			{Name: "episode", PluralName: "episodes", IsLeaf: true},
			{Name: "season", PluralName: "seasons"},
		},
	}
	err := schema.Validate()
	if err == nil {
		t.Fatal("expected error for leaf not last, got nil")
	}
}

func TestNodeSchema_Validate_DuplicateNames(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "item", PluralName: "items", IsRoot: true},
			{Name: "item", PluralName: "children", IsLeaf: true},
		},
	}
	err := schema.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate names, got nil")
	}
}

func TestNodeSchema_Validate_DuplicatePluralNames(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "parent", PluralName: "items", IsRoot: true},
			{Name: "child", PluralName: "items", IsLeaf: true},
		},
	}
	err := schema.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate plural names, got nil")
	}
}

func TestNodeSchema_Validate_SingleLevelRootNotLeaf(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "thing", PluralName: "things", IsRoot: true, IsLeaf: false},
		},
	}
	err := schema.Validate()
	if err == nil {
		t.Fatal("expected error for single level that is root but not leaf, got nil")
	}
}

func TestNodeSchema_Root(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "series", PluralName: "series", IsRoot: true},
			{Name: "season", PluralName: "seasons"},
			{Name: "episode", PluralName: "episodes", IsLeaf: true},
		},
	}
	root, err := schema.Root()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root.Name != "series" {
		t.Fatalf("expected root name 'series', got %q", root.Name)
	}
}

func TestNodeSchema_Root_Empty(t *testing.T) {
	schema := NodeSchema{}
	_, err := schema.Root()
	if err == nil {
		t.Fatal("expected error for empty schema Root(), got nil")
	}
}

func TestNodeSchema_Leaf(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "series", PluralName: "series", IsRoot: true},
			{Name: "season", PluralName: "seasons"},
			{Name: "episode", PluralName: "episodes", IsLeaf: true},
		},
	}
	leaf, err := schema.Leaf()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if leaf.Name != "episode" {
		t.Fatalf("expected leaf name 'episode', got %q", leaf.Name)
	}
}

func TestNodeSchema_Leaf_Empty(t *testing.T) {
	schema := NodeSchema{}
	_, err := schema.Leaf()
	if err == nil {
		t.Fatal("expected error for empty schema Leaf(), got nil")
	}
}

func TestNodeSchema_Depth(t *testing.T) {
	tests := []struct {
		name     string
		levels   int
		expected int
	}{
		{"single level", 1, 1},
		{"three levels", 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levels := make([]NodeLevel, tt.levels)
			schema := NodeSchema{Levels: levels}
			if got := schema.Depth(); got != tt.expected {
				t.Fatalf("expected depth %d, got %d", tt.expected, got)
			}
		})
	}
}

func TestNodeSchema_FlatMovieRootAndLeafSame(t *testing.T) {
	schema := NodeSchema{
		Levels: []NodeLevel{
			{Name: "movie", PluralName: "movies", IsRoot: true, IsLeaf: true},
		},
	}
	root, err := schema.Root()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	leaf, err := schema.Leaf()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root.Name != leaf.Name {
		t.Fatalf("expected root and leaf to be the same for flat schema, got root=%q leaf=%q", root.Name, leaf.Name)
	}
}
