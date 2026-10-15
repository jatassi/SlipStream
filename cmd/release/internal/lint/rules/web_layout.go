package rules

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "web-layout",
		Doc:   "signed-in pages live under web/src/routes/_app/ or web/src/routes/portal/, component tests under web/src/, browser tests under web/e2e/, and web/harness/ is a library the app never imports (specs #131, #135 and #136)",
		Check: checkWebLayout,
	})
}

const (
	webRoutes  = "web/src/routes/"
	webApp     = "web/src/"
	webHarness = "web/harness/"
)

// publicRoutes render without the _app guard.
var publicRoutes = []string{"__root.tsx", "setup.tsx", "sign-in.tsx", "worker.tsx"}

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
		if !isTypeScript(f) || !strings.HasPrefix(f, webApp) && !strings.HasPrefix(f, webHarness) {
			continue
		}
		src, err := repo.Read(f)
		if err != nil {
			return nil, err
		}
		findings = append(findings, harnessImports(f, src)...)
	}
	return findings, nil
}

var importSpecifier = regexp.MustCompile(`(?:\bfrom|\bimport)\s*\(?\s*["']([^"']+)["']`)

// harnessImports keeps web/harness/ apart from the app and the browser
// tier, which it serves and must not depend on.
func harnessImports(f string, src []byte) []lint.Finding {
	var findings []lint.Finding
	for _, m := range importSpecifier.FindAllSubmatchIndex(src, -1) {
		spec := string(src[m[2]:m[3]])
		if !strings.HasPrefix(spec, ".") {
			continue
		}
		target := path.Join(path.Dir(f), spec) + "/"
		var msg string
		switch {
		case strings.HasPrefix(f, webHarness) && (strings.HasPrefix(target, webApp) || strings.HasPrefix(target, "web/e2e/")):
			msg = "web/harness/ imports nothing from web/src/ or web/e2e/"
		case strings.HasPrefix(f, webApp) && strings.HasPrefix(target, webHarness):
			msg = "the app never imports web/harness/"
		default:
			continue
		}
		findings = append(findings, lint.Finding{Path: f, Line: 1 + strings.Count(string(src[:m[0]]), "\n"), Message: msg})
	}
	return findings
}

func isTypeScript(f string) bool {
	return strings.HasSuffix(f, ".ts") || strings.HasSuffix(f, ".tsx")
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
	case strings.HasPrefix(f, webHarness) && slices.Contains(strings.Split(f, "/"), "routes"):
		return "web/harness/ is a library and holds no routes"
	case strings.HasPrefix(f, webRoutes) && !guardedOrPublic(strings.TrimPrefix(f, webRoutes)):
		return "signed-in pages live under web/src/routes/_app/ or web/src/routes/portal/; a public route is listed in web-layout"
	}
	return ""
}

// guardedLayouts are the layouts whose guard every page under them passes:
// the Admin app and the Member portal.
var guardedLayouts = []string{"_app", "portal"}

// guardedOrPublic reports whether a file under routes/ is a guarded layout
// or one of its pages, a listed public route, a test, or a file the router
// ignores because a path element starts with "-".
func guardedOrPublic(route string) bool {
	return slices.ContainsFunc(guardedLayouts, func(layout string) bool {
		return route == layout+".tsx" || strings.HasPrefix(route, layout+"/")
	}) ||
		slices.Contains(publicRoutes, route) ||
		strings.HasSuffix(route, ".test.tsx") ||
		slices.ContainsFunc(strings.Split(route, "/"), func(elem string) bool { return strings.HasPrefix(elem, "-") })
}
