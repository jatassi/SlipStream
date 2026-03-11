package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/slipstream/slipstream/internal/module"
)

// Result holds the validation outcome for a single module.
type Result struct {
	ModuleID string
	Errors   []string
	Warnings []string
}

func (r *Result) AddError(format string, args ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

func (r *Result) AddWarning(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

func (r *Result) OK() bool { return len(r.Errors) == 0 }

// ModuleInfo carries the data needed for validation. This is separate from
// module.Module because full Module instances require runtime dependencies
// (database, services, etc.) that are not available in a CLI context.
type ModuleInfo struct {
	Descriptor   module.Descriptor
	QualityItems []module.QualityItem
	Events       []module.NotificationEvent
	Categories   []int
}

// ValidateModule checks a module for correctness using its ModuleInfo.
// allModules is the full set of registered modules (for cross-module checks).
func ValidateModule(info *ModuleInfo, allModules []*ModuleInfo) *Result {
	moduleID := string(info.Descriptor.ID())
	r := &Result{ModuleID: moduleID}

	validateSchema(r, info.Descriptor)
	validateEntityTypeUniqueness(r, info, allModules)
	validateQualityItems(r, info.QualityItems)
	validateNotificationEvents(r, moduleID, info.Events)
	validateSearchCategories(r, info.Categories)
	validateMigrationDirectory(r, moduleID)
	validateFrontendDirectory(r, moduleID)

	r.AddWarning("interface compliance (Module interface) is enforced at compile time via var _ assertions")

	return r
}

func validateSchema(r *Result, desc module.Descriptor) {
	schema := desc.NodeSchema()
	if err := schema.Validate(); err != nil {
		r.AddError("node schema invalid: %v", err)
		return
	}

	levels := schema.Levels
	entityTypes := desc.EntityTypes()
	if len(levels) != len(entityTypes) {
		r.AddWarning("schema has %d levels but module declares %d entity types", len(levels), len(entityTypes))
	}
}

func validateEntityTypeUniqueness(r *Result, info *ModuleInfo, allModules []*ModuleInfo) {
	myID := info.Descriptor.ID()
	for _, other := range allModules {
		if other.Descriptor.ID() == myID {
			continue
		}
		for _, et := range info.Descriptor.EntityTypes() {
			for _, otherET := range other.Descriptor.EntityTypes() {
				if et == otherET {
					r.AddError("entity type %q conflicts with module %q", et, other.Descriptor.ID())
				}
			}
		}
	}
}

func validateQualityItems(r *Result, items []module.QualityItem) {
	if len(items) == 0 {
		r.AddWarning("module declares no quality items")
		return
	}

	seenIDs := make(map[int]string)
	for _, item := range items {
		if prev, exists := seenIDs[item.ID]; exists {
			r.AddError("duplicate quality item ID %d: %q and %q", item.ID, prev, item.Name)
		}
		seenIDs[item.ID] = item.Name

		if item.Name == "" {
			r.AddError("quality item ID %d has empty name", item.ID)
		}
	}
}

func validateNotificationEvents(r *Result, moduleID string, events []module.NotificationEvent) {
	if len(events) == 0 {
		r.AddWarning("module declares no notification events")
		return
	}

	prefix := moduleID + ":"
	seenIDs := make(map[string]bool)
	for _, e := range events {
		if !strings.HasPrefix(e.ID, prefix) {
			r.AddError("notification event ID %q does not follow %s<event> convention", e.ID, prefix)
		}
		if seenIDs[e.ID] {
			r.AddError("duplicate notification event ID: %q", e.ID)
		}
		seenIDs[e.ID] = true

		if e.Label == "" {
			r.AddError("notification event %q has empty label", e.ID)
		}
	}
}

func validateSearchCategories(r *Result, cats []int) {
	if len(cats) == 0 {
		r.AddWarning("module declares no search categories")
	}
}

var migrationFilePattern = regexp.MustCompile(`^(\d+)_.+\.(sql|go)$`)

func validateMigrationDirectory(r *Result, moduleID string) {
	migrationsDir := filepath.Join("internal", "modules", moduleID, "migrations")
	info, err := os.Stat(migrationsDir)
	if os.IsNotExist(err) {
		r.AddWarning("no migrations directory at %s", migrationsDir)
		return
	}
	if err != nil {
		r.AddWarning("error accessing migrations directory: %v", err)
		return
	}
	if !info.IsDir() {
		r.AddError("migrations path %s exists but is not a directory", migrationsDir)
		return
	}

	validateMigrationFiles(r, migrationsDir)
}

func validateMigrationFiles(r *Result, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		r.AddWarning("error reading migrations directory: %v", err)
		return
	}

	numbers := extractMigrationNumbers(entries)
	if len(numbers) == 0 {
		r.AddWarning("migrations directory exists but contains no migration files")
		return
	}

	sort.Ints(numbers)
	for i := 1; i < len(numbers); i++ {
		if numbers[i] != numbers[i-1]+1 {
			r.AddWarning("migration files not sequentially numbered: gap between %d and %d", numbers[i-1], numbers[i])
		}
	}
}

func extractMigrationNumbers(entries []os.DirEntry) []int {
	var numbers []int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(matches[1], "%d", &n); err == nil {
			numbers = append(numbers, n)
		}
	}
	return numbers
}

func validateFrontendDirectory(r *Result, moduleID string) {
	frontendDir := filepath.Join("web", "src", "modules", moduleID)
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		r.AddWarning("no frontend directory at %s", frontendDir)
	}
}
