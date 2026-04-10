package tv_test

import (
	"testing"

	"github.com/slipstream/slipstream/internal/module/moduletest"
	"github.com/slipstream/slipstream/internal/modules/tv"
	"github.com/slipstream/slipstream/internal/testutil"
)

func newTestModule(t *testing.T) *tv.Module {
	t.Helper()
	logger := testutil.NopLogger()
	return tv.NewModule(nil, nil, nil, nil, nil, nil, &logger)
}

func TestTVModuleSchema(t *testing.T) {
	mod := newTestModule(t)
	moduletest.RunSchemaTest(t, mod)
}

func TestTVModuleParsing(t *testing.T) {
	mod := newTestModule(t)
	moduletest.RunParsingTest(t, mod, []moduletest.ParsingFixture{
		{Filename: "Breaking Bad S01E01 720p.mkv", ExpectedTitle: "Breaking Bad", ExpectMatch: true, MinConfidence: 0.8},
		{Filename: "The.Office.S03E05.1080p.BluRay.x264.mkv", ExpectedTitle: "The Office", ExpectMatch: true, MinConfidence: 0.8},
		{Filename: "Show.Name.S02.1080p.BluRay.mkv", ExpectMatch: true, MinConfidence: 0.5},
		{Filename: "The Boys (2019) - S05E01.mkv", ExpectedTitle: "The Boys", ExpectedYear: 2019, ExpectMatch: true, MinConfidence: 0.8},
		{Filename: "The.Boys.2019.S05E01.1080p.WEB-DL.mkv", ExpectedTitle: "The Boys", ExpectedYear: 2019, ExpectMatch: true, MinConfidence: 0.8},
		{Filename: "1923.S01E01.mkv", ExpectedTitle: "1923", ExpectMatch: true, MinConfidence: 0.8},
		{Filename: "The Matrix (1999).mkv", ExpectMatch: false},
		{Filename: "random_document.pdf", ExpectMatch: false},
	})
}

func TestTVModuleNaming(t *testing.T) {
	mod := newTestModule(t)
	moduletest.RunNamingTest(t, mod, mod, []moduletest.NamingFixture{})
}

func TestTVModuleLifecycle(t *testing.T) {
	mod := newTestModule(t)
	moduletest.RunLifecycleTest(t, mod)
}
