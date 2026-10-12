package rules

import (
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "web-layout",
		Doc:   "signed-in pages live under web/src/routes/_app/, component tests under web/src/ and browser tests under web/e2e/ (spec #131, Architecture checks)",
		Check: checkWebLayout,
	})
}

const webRoutes = "web/src/routes/"

// publicRoutes render without the _app guard.
var publicRoutes = []string{"__root.tsx", "setup.tsx", "sign-in.tsx"}

func checkWebLayout(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.Files()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		if msg := webPlacement(f); msg != "" {
			findings = append(findings, lint.Finding{Path: f, Line: 1, Message: msg})
		}
	}
	return findings, nil
}

func webPlacement(f string) string {
	if !strings.HasPrefix(f, "web/") {
		return ""
	}
	switch {
	case (strings.HasSuffix(f, ".test.ts") || strings.HasSuffix(f, ".test.tsx")) && !strings.HasPrefix(f, "web/src/"):
		return "component tests live under web/src/; browser tests are web/e2e/*.spec.ts"
	case strings.HasSuffix(f, ".spec.ts") && !strings.HasPrefix(f, "web/e2e/"):
		return "browser tests live under web/e2e/"
	case strings.HasPrefix(f, webRoutes) && !guardedOrPublic(strings.TrimPrefix(f, webRoutes)):
		return "signed-in pages live under web/src/routes/_app/; a public route is listed in web-layout"
	}
	return ""
}

// guardedOrPublic reports whether a file under routes/ is the _app layout
// or one of its pages, a listed public route, a test, or a file the router
// ignores because a path element starts with "-".
func guardedOrPublic(route string) bool {
	return route == "_app.tsx" ||
		strings.HasPrefix(route, "_app/") ||
		slices.Contains(publicRoutes, route) ||
		strings.HasSuffix(route, ".test.tsx") ||
		slices.ContainsFunc(strings.Split(route, "/"), func(elem string) bool { return strings.HasPrefix(elem, "-") })
}
