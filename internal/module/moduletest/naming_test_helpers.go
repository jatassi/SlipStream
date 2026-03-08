package moduletest

import (
	"testing"

	"github.com/slipstream/slipstream/internal/module"
)

// NamingFixture defines a test case for naming/path template resolution.
type NamingFixture struct {
	TemplateName   string         // key in DefaultTemplates, e.g. "movie-folder"
	TokenData      map[string]any // template variables
	ExpectedOutput string         // expected resolved string
}

// RunNamingTest validates that a module's NamingProvider and PathGenerator produce
// the expected output for each fixture.
func RunNamingTest(t *testing.T, provider module.NamingProvider, pathGen module.PathGenerator, fixtures []NamingFixture) {
	t.Helper()

	t.Run("NamingProvider", func(t *testing.T) {
		assertTokenContexts(t, provider)
		assertDefaultFileTemplates(t, provider)
	})

	t.Run("PathGenerator", func(t *testing.T) {
		templates := pathGen.DefaultTemplates()
		if len(templates) == 0 {
			t.Error("DefaultTemplates() returned empty map")
		}
	})

	for _, fx := range fixtures {
		t.Run("Resolve/"+fx.TemplateName, func(t *testing.T) {
			assertTemplateResolves(t, pathGen, fx)
		})
	}
}

func assertTokenContexts(t *testing.T, provider module.NamingProvider) {
	t.Helper()
	contexts := provider.TokenContexts()
	if len(contexts) == 0 {
		t.Error("TokenContexts() returned empty slice")
	}
	for _, ctx := range contexts {
		if ctx.Name == "" {
			t.Error("TokenContext has empty Name")
		}
		if ctx.Label == "" {
			t.Errorf("TokenContext %q has empty Label", ctx.Name)
		}
	}
}

func assertDefaultFileTemplates(t *testing.T, provider module.NamingProvider) {
	t.Helper()
	templates := provider.DefaultFileTemplates()
	if len(templates) == 0 {
		t.Error("DefaultFileTemplates() returned empty map")
	}
}

func assertTemplateResolves(t *testing.T, pathGen module.PathGenerator, fx NamingFixture) {
	t.Helper()
	templates := pathGen.DefaultTemplates()
	tmpl, ok := templates[fx.TemplateName]
	if !ok {
		t.Fatalf("template %q not found in DefaultTemplates()", fx.TemplateName)
	}
	result, err := pathGen.ResolveTemplate(tmpl, fx.TokenData)
	if err != nil {
		t.Fatalf("ResolveTemplate() error: %v", err)
	}
	if fx.ExpectedOutput != "" && result != fx.ExpectedOutput {
		t.Errorf("got %q, want %q", result, fx.ExpectedOutput)
	}
}
