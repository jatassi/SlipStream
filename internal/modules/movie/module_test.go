package movie_test

import (
	"testing"

	"github.com/slipstream/slipstream/internal/module/moduletest"
	"github.com/slipstream/slipstream/internal/modules/movie"
	"github.com/slipstream/slipstream/internal/testutil"
)

func newTestModule(t *testing.T) *movie.Module {
	t.Helper()
	logger := testutil.NopLogger()
	return movie.NewModule(nil, nil, nil, nil, nil, &logger)
}

func TestMovieModuleSchema(t *testing.T) {
	mod := newTestModule(t)
	moduletest.RunSchemaTest(t, mod)
}

func TestMovieModuleParsing(t *testing.T) {
	mod := newTestModule(t)
	moduletest.RunParsingTest(t, mod, []moduletest.ParsingFixture{
		{Filename: "The Matrix (1999).mkv", ExpectedTitle: "The Matrix", ExpectedYear: 1999, ExpectMatch: true, MinConfidence: 0.5},
		{Filename: "Inception.2010.1080p.BluRay.x264-GROUP.mkv", ExpectedTitle: "Inception", ExpectedYear: 2010, ExpectMatch: true, MinConfidence: 0.5},
		{Filename: "S01E02.mkv", ExpectMatch: false},
		{Filename: "random_document.pdf", ExpectMatch: false},
	})
}

func TestMovieModuleNaming(t *testing.T) {
	mod := newTestModule(t)
	moduletest.RunNamingTest(t, mod, mod, []moduletest.NamingFixture{
		{
			TemplateName: "movie-folder",
			TokenData: map[string]any{
				"Movie Title": "Inception",
				"Year":        "2010",
			},
			ExpectedOutput: "Inception (2010)",
		},
	})
}

func TestMovieModuleLifecycle(t *testing.T) {
	mod := newTestModule(t)
	moduletest.RunLifecycleTest(t, mod)
}
