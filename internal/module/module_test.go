package module

import (
	"context"
	"testing"
	"time"

	"github.com/google/wire"
)

// mockModule satisfies the full Module interface with stub returns.
type mockModule struct {
	id          Type
	name        string
	entityTypes []EntityType
	schema      NodeSchema
}

func (m *mockModule) ID() Type                  { return m.id }
func (m *mockModule) Name() string              { return m.name }
func (m *mockModule) PluralName() string        { return m.name + "s" }
func (m *mockModule) Icon() string              { return "" }
func (m *mockModule) ThemeColor() string        { return "" }
func (m *mockModule) NodeSchema() NodeSchema    { return m.schema }
func (m *mockModule) EntityTypes() []EntityType { return m.entityTypes }
func (m *mockModule) Wire() wire.ProviderSet    { return wire.NewSet() }

func (m *mockModule) Search(_ context.Context, _ string, _ SearchOptions) ([]SearchResult, error) {
	return nil, nil //nolint:nilnil // test stub
}

func (m *mockModule) GetByID(_ context.Context, _ string) (*MediaMetadata, error) {
	return nil, nil //nolint:nilnil // test stub
}

func (m *mockModule) GetExtendedInfo(_ context.Context, _ string) (*ExtendedMetadata, error) {
	return nil, nil //nolint:nilnil // test stub
}

func (m *mockModule) RefreshMetadata(_ context.Context, _ int64) (*RefreshResult, error) {
	return nil, nil //nolint:nilnil // test stub
}

func (m *mockModule) Categories() []int              { return nil }
func (m *mockModule) DefaultSearchCategories() []int { return nil }

func (m *mockModule) FilterRelease(_ context.Context, _ *ReleaseForFilter, _ SearchableItem) (reject bool, reason string) {
	return false, ""
}

func (m *mockModule) TitlesMatch(_, _ string) bool                        { return false }
func (m *mockModule) BuildSearchCriteria(_ SearchableItem) SearchCriteria { return SearchCriteria{} }
func (m *mockModule) IsGroupSearchEligible(_ context.Context, _ EntityType, _ int64, _ int, _ bool) bool {
	return false
}
func (m *mockModule) SuppressChildSearches(_ EntityType, _ int64, _ int) []int64 {
	return nil
}

func (m *mockModule) MatchDownload(_ context.Context, _ *CompletedDownload) ([]MatchedEntity, error) {
	return nil, nil
}
func (m *mockModule) ImportFile(_ context.Context, _ string, _ *MatchedEntity, _ *QualityInfo) (*ImportResult, error) {
	return nil, nil //nolint:nilnil // test stub
}
func (m *mockModule) SupportsMultiFileDownload() bool { return false }
func (m *mockModule) MatchIndividualFile(_ context.Context, _ string, _ *MatchedEntity) (*MatchedEntity, error) {
	return nil, nil //nolint:nilnil // test stub
}
func (m *mockModule) IsGroupImportReady(_ context.Context, _ *MatchedEntity, _ []MatchedEntity) bool {
	return false
}
func (m *mockModule) MediaInfoFields() []MediaInfoFieldDecl { return nil }

func (m *mockModule) DefaultTemplates() map[string]string            { return nil }
func (m *mockModule) AvailableVariables(_ string) []TemplateVariable { return nil }
func (m *mockModule) ResolveTemplate(_ string, _ map[string]any) (string, error) {
	return "", nil
}
func (m *mockModule) ConditionalSegments() []ConditionalSegment { return nil }
func (m *mockModule) IsSpecialNode(_ context.Context, _ EntityType, _ int64) (bool, error) {
	return false, nil
}

func (m *mockModule) TokenContexts() []TokenContext           { return nil }
func (m *mockModule) DefaultFileTemplates() map[string]string { return nil }
func (m *mockModule) FormatOptions() []FormatOption           { return nil }

func (m *mockModule) GetItemsInDateRange(_ context.Context, _, _ time.Time) ([]CalendarItem, error) {
	return nil, nil
}

func (m *mockModule) QualityItems() []QualityItem                       { return nil }
func (m *mockModule) ParseQuality(_ string) (*QualityResult, error)     { return nil, nil } //nolint:nilnil // test stub
func (m *mockModule) ScoreQuality(_ QualityItem) int                    { return 0 }
func (m *mockModule) IsUpgrade(_, _ QualityItem, _ int64) (bool, error) { return false, nil }

func (m *mockModule) CollectMissing(_ context.Context) ([]SearchableItem, error)    { return nil, nil }
func (m *mockModule) CollectUpgradable(_ context.Context) ([]SearchableItem, error) { return nil, nil }

func (m *mockModule) AvailablePresets() []MonitoringPreset { return nil }
func (m *mockModule) ApplyPreset(_ context.Context, _ int64, _ string, _ map[string]any) error {
	return nil
}

func (m *mockModule) ParseFilename(_ string) (*ParseResult, error) { return nil, nil } //nolint:nilnil // test stub
func (m *mockModule) MatchToEntity(_ context.Context, _ *ParseResult) (*MatchedEntity, error) {
	return nil, nil //nolint:nilnil // test stub
}
func (m *mockModule) TryMatch(_ string) (float64, *ParseResult) { return 0, nil }

func (m *mockModule) CreateMockMetadataProvider() MetadataProvider                    { return nil }
func (m *mockModule) CreateSampleLibraryData(_ context.Context, _ *MockContext) error { return nil }
func (m *mockModule) CreateTestRootFolders(_ context.Context, _ *MockContext) error   { return nil }

func (m *mockModule) DeclareEvents() []NotificationEvent { return nil }

func (m *mockModule) ComputeAvailabilityDate(_ context.Context, _ int64) (*time.Time, error) {
	return nil, nil //nolint:nilnil // test stub
}
func (m *mockModule) CheckReleaseDateTransitions(_ context.Context) (int, error) { return 0, nil }

func (m *mockModule) RegisterRoutes(_ RouteGroup) {}

func (m *mockModule) ScheduledTasks() []ScheduledTask { return nil }

func newMockMovieModule() *mockModule {
	return &mockModule{
		id:          TypeMovie,
		name:        "Movie",
		entityTypes: []EntityType{EntityMovie},
		schema: NodeSchema{
			Levels: []NodeLevel{
				{Name: "movie", PluralName: "movies", IsRoot: true, IsLeaf: true},
			},
		},
	}
}

func newMockTVModule() *mockModule {
	return &mockModule{
		id:          TypeTV,
		name:        "TV",
		entityTypes: []EntityType{EntitySeries, EntitySeason, EntityEpisode},
		schema: NodeSchema{
			Levels: []NodeLevel{
				{Name: "series", PluralName: "series", IsRoot: true},
				{Name: "season", PluralName: "seasons"},
				{Name: "episode", PluralName: "episodes", IsLeaf: true},
			},
		},
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	movie := newMockMovieModule()
	tv := newMockTVModule()

	reg.Register(movie)
	reg.Register(tv)

	got := reg.Get(TypeMovie)
	if got == nil {
		t.Fatal("expected to get movie module, got nil")
	}
	if got.ID() != TypeMovie {
		t.Fatalf("expected module ID %q, got %q", TypeMovie, got.ID())
	}

	got = reg.Get(TypeTV)
	if got == nil {
		t.Fatal("expected to get TV module, got nil")
	}
	if got.ID() != TypeTV {
		t.Fatalf("expected module ID %q, got %q", TypeTV, got.ID())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	reg := NewRegistry()
	got := reg.Get(Type("nonexistent"))
	if got != nil {
		t.Fatalf("expected nil for unknown module type, got %v", got)
	}
}

func TestRegistry_All_Order(t *testing.T) {
	reg := NewRegistry()
	movie := newMockMovieModule()
	tv := newMockTVModule()

	reg.Register(tv)
	reg.Register(movie)

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(all))
	}
	if all[0].ID() != TypeTV {
		t.Fatalf("expected first module to be TV, got %q", all[0].ID())
	}
	if all[1].ID() != TypeMovie {
		t.Fatalf("expected second module to be Movie, got %q", all[1].ID())
	}
}

func TestRegistry_Types_Order(t *testing.T) {
	reg := NewRegistry()
	movie := newMockMovieModule()
	tv := newMockTVModule()

	reg.Register(movie)
	reg.Register(tv)

	types := reg.Types()
	if len(types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(types))
	}
	if types[0] != TypeMovie {
		t.Fatalf("expected first type to be movie, got %q", types[0])
	}
	if types[1] != TypeTV {
		t.Fatalf("expected second type to be tv, got %q", types[1])
	}
}

func TestRegistry_DuplicateID_Panics(t *testing.T) {
	reg := NewRegistry()
	movie1 := newMockMovieModule()
	movie2 := newMockMovieModule()

	reg.Register(movie1)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate registration, got none")
		}
	}()
	reg.Register(movie2)
}

func TestRegistry_EntityTypeConflict_Panics(t *testing.T) {
	reg := NewRegistry()
	movie := newMockMovieModule()

	conflicting := &mockModule{
		id:          Type("other"),
		name:        "Other",
		entityTypes: []EntityType{EntityMovie},
		schema: NodeSchema{
			Levels: []NodeLevel{
				{Name: "other", PluralName: "others", IsRoot: true, IsLeaf: true},
			},
		},
	}

	reg.Register(movie)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on entity type conflict, got none")
		}
	}()
	reg.Register(conflicting)
}

func TestRegistry_ModuleForEntityType(t *testing.T) {
	reg := NewRegistry()
	movie := newMockMovieModule()
	tv := newMockTVModule()

	reg.Register(movie)
	reg.Register(tv)

	got := reg.ModuleForEntityType(EntityMovie)
	if got == nil {
		t.Fatal("expected module for EntityMovie, got nil")
	}
	if got.ID() != TypeMovie {
		t.Fatalf("expected movie module, got %q", got.ID())
	}

	got = reg.ModuleForEntityType(EntityEpisode)
	if got == nil {
		t.Fatal("expected module for EntityEpisode, got nil")
	}
	if got.ID() != TypeTV {
		t.Fatalf("expected TV module, got %q", got.ID())
	}

	got = reg.ModuleForEntityType(EntitySeason)
	if got == nil {
		t.Fatal("expected module for EntitySeason, got nil")
	}
	if got.ID() != TypeTV {
		t.Fatalf("expected TV module for season, got %q", got.ID())
	}
}

func TestRegistry_ModuleForEntityType_NotFound(t *testing.T) {
	reg := NewRegistry()
	got := reg.ModuleForEntityType(EntityType("nonexistent"))
	if got != nil {
		t.Fatalf("expected nil for unknown entity type, got %v", got)
	}
}

func TestRegistry_InvalidSchema_Panics(t *testing.T) {
	reg := NewRegistry()
	invalid := &mockModule{
		id:          Type("bad"),
		name:        "Bad",
		entityTypes: []EntityType{EntityType("bad")},
		schema:      NodeSchema{},
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on invalid schema, got none")
		}
	}()
	reg.Register(invalid)
}
