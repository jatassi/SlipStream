package contenttype_test

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/works"
)

func assertPanics(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if got := recover(); got != want {
			t.Errorf("panic = %v, want %q", got, want)
		}
	}()
	fn()
}

func open(contenttype.Deps, struct{}) contenttype.Type { return contenttypetest.New() }

func TestDefineRefusesABadID(t *testing.T) {
	assertPanics(t, `contenttype: id "Fixture" is not ^[a-z][a-z0-9]{1,15}$`, func() {
		contenttype.Define[struct{}]("Fixture", nil, nil, open)
	})
	assertPanics(t, `contenttype: id "abcdefghijklmnopq" is not ^[a-z][a-z0-9]{1,15}$`, func() {
		contenttype.Define[struct{}]("abcdefghijklmnopq", nil, nil, open)
	})
	assertPanics(t, "contenttype: id goose names a set storage reserves", func() {
		contenttype.Define[struct{}]("goose", nil, nil, open)
	})
}

func TestDefineNamesTheSetByTheID(t *testing.T) {
	files := fstest.MapFS{"schema/00001_shelf.sql": {Data: []byte("-- +goose Up\nCREATE TABLE shelf_x (id INTEGER PRIMARY KEY) STRICT;\n")}}
	d := contenttype.Define[struct{}]("shelf", files, nil, open)
	if s, ok := d.Schema(); !ok || s.Set() != "shelf" {
		t.Fatalf("schema = %v, %t; want set shelf", s.Set(), ok)
	}
	if _, ok := contenttype.Define[struct{}]("bare", nil, nil, open).Schema(); ok {
		t.Fatal("a type with no migrations has a schema")
	}
}

func TestEndpointsAreFreshPerCall(t *testing.T) {
	d := contenttypetest.New().Def()
	a, b := d.Endpoints().(*contenttypetest.Endpoints), d.Endpoints().(*contenttypetest.Endpoints)
	if a == b || a.API.String() != "https://fixture.invalid/v1" {
		t.Fatalf("sections %p %p hold %s; want two sections holding https://fixture.invalid/v1", a, b, a.API)
	}
	if got := contenttype.Define[struct{}]("bare", nil, nil, open).Endpoints(); got != nil {
		t.Fatalf("a type with no endpoints has section %v", got)
	}
}

func TestSetIsSortedAndRefusesATwiceRegisteredID(t *testing.T) {
	a := contenttype.Define[struct{}]("alpha", nil, nil, open)
	z := contenttype.Define[struct{}]("zulu", nil, nil, open)
	if got := contenttype.NewSet(z, a).IDs(); !slices.Equal(got, []works.TypeID{"alpha", "zulu"}) {
		t.Fatalf("ids = %v, want [alpha zulu]", got)
	}
	assertPanics(t, "contenttype: fixture is registered twice", func() {
		contenttype.NewSet(contenttypetest.New().Def(), contenttypetest.New().Def())
	})
}

func TestProjectHandsEachConsumerItsFacet(t *testing.T) {
	f := contenttypetest.New()
	got := contenttype.Project(map[works.TypeID]contenttype.Type{"fixture": f}, func(t contenttype.Type) works.Catalog { return t })
	if len(got) != 1 || got["fixture"] != works.Catalog(f) {
		t.Fatalf("projected = %v, want the fixture's Catalog under fixture", got)
	}
}

func TestATypeLackingAFacetDoesNotCompile(t *testing.T) {
	out, err := exec.CommandContext(t.Context(), "go", "build", "-o", os.DevNull, "./testdata/gate").CombinedOutput()
	if err == nil {
		t.Fatal("testdata/gate compiled")
	}
	want := "cannot use contenttypetest.New() (value of type *contenttypetest.Fixture) as Type value in variable declaration: *contenttypetest.Fixture does not implement Type (missing method Name)"
	if !strings.Contains(string(out), want) {
		t.Fatalf("build output lacks %q:\n%s", want, out)
	}
}
