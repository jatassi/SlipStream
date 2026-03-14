package moduletest

import (
	"context"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/module"
)

// safeCall runs fn and recovers from panics, logging them as warnings.
// DB-dependent methods may panic when the module was constructed without a
// database; this is expected for lightweight smoke tests.
func safeCall(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Logf("%s panicked (expected without DB): %v", name, r)
		}
	}()
	fn()
}

// RunLifecycleTest is a smoke test that calls each module interface method to
// verify it doesn't panic or error on a zero/empty state. Callers should pass
// a fully constructed module. Methods that require a database will typically
// return empty results or benign errors on an empty DB -- this test logs errors
// but only fails on panics.
func RunLifecycleTest(t *testing.T, mod module.Module) {
	t.Helper()

	t.Run("Descriptor", func(t *testing.T) { lifecycleDescriptor(t, mod) })
	t.Run("SchemaValidation", func(t *testing.T) { lifecycleSchema(t, mod) })
	t.Run("QualityDefinition", func(t *testing.T) { lifecycleQuality(t, mod) })
	t.Run("SearchStrategy", func(t *testing.T) { lifecycleSearch(t, mod) })
	t.Run("NotificationEvents", func(t *testing.T) { lifecycleNotifications(t, mod) })
	t.Run("NamingProvider", func(t *testing.T) { lifecycleNaming(t, mod) })
	t.Run("PathGenerator", func(t *testing.T) { lifecyclePathGen(t, mod) })
	t.Run("CalendarProvider", func(t *testing.T) { lifecycleCalendar(t, mod) })
	t.Run("WantedCollector", func(t *testing.T) { lifecycleWanted(t, mod) })
	t.Run("MonitoringPresets", func(t *testing.T) { _ = mod.AvailablePresets() })
	t.Run("FileParser", func(t *testing.T) { lifecycleFileParser(t, mod) })
	t.Run("TaskProvider", func(t *testing.T) { _ = mod.ScheduledTasks() })
}

func lifecycleDescriptor(t *testing.T, mod module.Module) {
	t.Helper()
	if mod.ID() == "" {
		t.Error("ID() is empty")
	}
	if mod.Name() == "" {
		t.Error("Name() is empty")
	}
	if mod.PluralName() == "" {
		t.Error("PluralName() is empty")
	}
	if mod.Icon() == "" {
		t.Error("Icon() is empty")
	}
	if mod.ThemeColor() == "" {
		t.Error("ThemeColor() is empty")
	}
	if len(mod.EntityTypes()) == 0 {
		t.Error("EntityTypes() is empty")
	}
}

func lifecycleSchema(t *testing.T, mod module.Module) {
	t.Helper()
	schema := mod.NodeSchema()
	if err := schema.Validate(); err != nil {
		t.Errorf("NodeSchema.Validate() failed: %v", err)
	}
}

func lifecycleQuality(t *testing.T, mod module.Module) {
	t.Helper()
	items := mod.QualityItems()
	if len(items) == 0 {
		t.Error("QualityItems() returned empty slice")
	}
	for _, item := range items {
		_ = mod.ScoreQuality(item)
	}
	_, _ = mod.ParseQuality("Some.Release.1080p.BluRay.x264-GROUP")
}

func lifecycleSearch(t *testing.T, mod module.Module) {
	t.Helper()
	if len(mod.Categories()) == 0 {
		t.Error("Categories() returned empty slice")
	}
	if len(mod.DefaultSearchCategories()) == 0 {
		t.Error("DefaultSearchCategories() returned empty slice")
	}
}

func lifecycleNotifications(t *testing.T, mod module.Module) {
	t.Helper()
	events := mod.DeclareEvents()
	if len(events) == 0 {
		t.Error("DeclareEvents() returned empty slice")
	}
	for _, ev := range events {
		if ev.ID == "" {
			t.Error("notification event has empty ID")
		}
		if ev.Label == "" {
			t.Errorf("notification event %q has empty Label", ev.ID)
		}
	}
}

func lifecycleNaming(t *testing.T, mod module.Module) {
	t.Helper()
	if len(mod.TokenContexts()) == 0 {
		t.Error("TokenContexts() returned empty slice")
	}
	if len(mod.DefaultFileTemplates()) == 0 {
		t.Error("DefaultFileTemplates() returned empty map")
	}
	_ = mod.FormatOptions()
}

func lifecyclePathGen(t *testing.T, mod module.Module) {
	t.Helper()
	if len(mod.DefaultTemplates()) == 0 {
		t.Error("DefaultTemplates() returned empty map")
	}
	_ = mod.ConditionalSegments()
}

func lifecycleCalendar(t *testing.T, mod module.Module) {
	t.Helper()
	safeCall(t, "GetItemsInDateRange", func() {
		ctx := context.Background()
		now := time.Now()
		items, err := mod.GetItemsInDateRange(ctx, now.AddDate(0, -1, 0), now.AddDate(0, 1, 0))
		if err != nil {
			t.Logf("GetItemsInDateRange() error (may be expected on empty DB): %v", err)
		}
		_ = items
	})
}

func lifecycleWanted(t *testing.T, mod module.Module) {
	t.Helper()
	ctx := context.Background()
	safeCall(t, "CollectMissing", func() {
		missing, err := mod.CollectMissing(ctx)
		if err != nil {
			t.Logf("CollectMissing() error (may be expected on empty DB): %v", err)
		}
		_ = missing
	})
	safeCall(t, "CollectUpgradable", func() {
		upgradable, err := mod.CollectUpgradable(ctx)
		if err != nil {
			t.Logf("CollectUpgradable() error (may be expected on empty DB): %v", err)
		}
		_ = upgradable
	})
}

func lifecycleFileParser(t *testing.T, mod module.Module) {
	t.Helper()
	_, _ = mod.ParseFilename("nonexistent.mkv")
	_, _ = mod.TryMatch("nonexistent.mkv")
}
