package moduletest

import (
	"testing"

	"github.com/slipstream/slipstream/internal/module"
)

// RunSchemaTest validates that a module's NodeSchema is structurally correct.
// It checks that Validate() passes, root and leaf levels have HasMonitored=true,
// and the entity types count matches the schema levels count.
func RunSchemaTest(t *testing.T, mod module.Module) {
	t.Helper()
	t.Run("SchemaValidation", func(t *testing.T) {
		schema := mod.NodeSchema()

		if err := schema.Validate(); err != nil {
			t.Fatalf("NodeSchema.Validate() failed: %v", err)
		}

		root, err := schema.Root()
		if err != nil {
			t.Fatalf("NodeSchema.Root() error: %v", err)
		}
		if !root.HasMonitored {
			t.Errorf("root level %q: HasMonitored should be true", root.Name)
		}

		leaf, err := schema.Leaf()
		if err != nil {
			t.Fatalf("NodeSchema.Leaf() error: %v", err)
		}
		if !leaf.HasMonitored {
			t.Errorf("leaf level %q: HasMonitored should be true", leaf.Name)
		}

		entityTypes := mod.EntityTypes()
		if len(entityTypes) != len(schema.Levels) {
			t.Errorf("EntityTypes() count %d != NodeSchema.Levels count %d", len(entityTypes), len(schema.Levels))
		}
	})
}
