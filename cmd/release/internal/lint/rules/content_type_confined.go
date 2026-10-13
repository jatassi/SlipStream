package rules

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "content-type-confined",
		Doc:   "core names no Content Type, Content Types never import each other, and surface never depends on internal/contenttype (spec #133, Lints and CI checks; ADR 0001)",
		Check: checkContentTypeConfined,
	})
}

const surfacePkg = "./internal/node/surface"

// coreExempt are the trees outside core besides Content Type packages and
// testdata: the two mains with their process tests, the lint rules, which
// necessarily name ids, and the test-only trees.
var coreExempt = []string{"cmd/slipstream/", "cmd/slipstream-e2e/", "cmd/release/", "fakes/", "e2e/"}

var webScripts = []string{".ts", ".tsx", ".mts", ".cts", ".js", ".jsx"}

type contentTypeCheck struct {
	repo     *lint.Repo
	module   string
	ids      []string
	findings map[lint.Finding]bool
}

func checkContentTypeConfined(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	c := contentTypeCheck{repo: repo, module: module, findings: map[lint.Finding]bool{}}
	for _, f := range files {
		c.defines(f)
	}
	for _, f := range files {
		c.imports(f)
		if isCore(f.Path) {
			c.goNames(f)
		}
	}
	if err := c.web(); err != nil {
		return nil, err
	}
	if err := c.surface(); err != nil {
		return nil, err
	}
	findings := make([]lint.Finding, 0, len(c.findings))
	for f := range c.findings {
		findings = append(findings, f)
	}
	return findings, nil
}

func isCore(file string) bool {
	return !underAny(file, coreExempt) &&
		!slices.Contains(strings.Split(path.Dir(file), "/"), "testdata") &&
		!isContentTypePackage(path.Dir(file))
}

func (c *contentTypeCheck) report(file string, pos token.Pos, format string, args ...any) {
	c.reportLine(file, c.repo.Fset.Position(pos).Line, format, args...)
}

func (c *contentTypeCheck) reportLine(file string, line int, format string, args ...any) {
	c.findings[lint.Finding{Path: file, Line: line, Message: fmt.Sprintf(format, args...)}] = true
}

// defines collects each Content Type's id from its contenttype.Define call.
// internal/contenttype and its subtree may call Define too: its own tests
// and contenttypetest.Fixture, whose id is never a Content Type's.
func (c *contentTypeCheck) defines(f lint.GoFile) {
	dir := path.Dir(f.Path)
	isType := isContentTypePackage(dir)
	exempt := dir == contentTypePkg || strings.HasPrefix(dir, contentTypePkg+"/")
	imports := importNames(f)
	ast.Inspect(f.Syntax, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isDefineCall(c.module, imports, call) {
			return true
		}
		switch {
		case isType:
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				c.report(f.Path, call.Pos(), "contenttype.Define takes the Content Type id as a string literal")
				return true
			}
			id, _ := strconv.Unquote(lit.Value)
			c.ids = append(c.ids, id)
		case !exempt:
			c.report(f.Path, call.Pos(), "contenttype.Define outside a Content Type package")
		}
		return true
	})
}

func (c *contentTypeCheck) imports(f lint.GoFile) {
	importer, _ := contentTypeScope(path.Dir(f.Path))
	importerIsType := isContentTypePackage(path.Dir(f.Path))
	for _, imp := range f.Syntax.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		rel, ok := strings.CutPrefix(p, c.module+"/")
		if !ok || !isContentTypePackage(rel) {
			continue
		}
		imported, _ := contentTypeScope(rel)
		switch {
		case isCore(f.Path):
			c.report(f.Path, imp.Pos(), "core imports %s; reach a Content Type through its facet", rel)
		case importerIsType && importer != imported:
			c.report(f.Path, imp.Pos(), "Content Type %s imports Content Type %s; Content Types never import each other", importer, imported)
		}
	}
}

func (c *contentTypeCheck) goNames(f lint.GoFile) {
	ast.Inspect(f.Syntax, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BasicLit:
			if n.Kind != token.STRING {
				return true
			}
			if value, err := strconv.Unquote(n.Value); err == nil {
				c.literal(f.Path, c.repo.Fset.Position(n.Pos()).Line, value)
			}
		case *ast.Ident:
			for _, id := range c.ids {
				if slices.ContainsFunc(identWords(n.Name), func(w string) bool { return strings.EqualFold(w, id) }) {
					c.report(f.Path, n.Pos(), "identifier %s names Content Type %s", n.Name, id)
				}
			}
		}
		return true
	})
}

func (c *contentTypeCheck) literal(file string, line int, value string) {
	for _, id := range c.ids {
		if strings.EqualFold(value, id) {
			c.reportLine(file, line, "core names Content Type %s; ask its facet instead", id)
		}
	}
}

// identWords splits an identifier into its snake_case and camelCase words.
// A run of capitals is one word, ending before a capital that starts a
// lowercase word, so HTTPMusic is HTTP and Music.
func identWords(name string) []string {
	var words []string
	for _, part := range strings.Split(name, "_") {
		runes := []rune(part)
		start := 0
		for i := 1; i < len(runes); i++ {
			lowerBefore := unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1])
			acronymEnds := unicode.IsUpper(runes[i-1]) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsUpper(runes[i]) && (lowerBefore || acronymEnds) {
				words = append(words, string(runes[start:i]))
				start = i
			}
		}
		if start < len(runes) {
			words = append(words, string(runes[start:]))
		}
	}
	return words
}

func (c *contentTypeCheck) web() error {
	if len(c.ids) == 0 {
		return nil
	}
	files, err := c.repo.Files()
	if err != nil {
		return err
	}
	for _, file := range files {
		if !strings.HasPrefix(file, "web/src/") || !slices.Contains(webScripts, path.Ext(file)) || !isCore(file) {
			continue
		}
		src, err := c.repo.Read(file)
		if err != nil {
			return err
		}
		for _, lit := range scriptStrings(string(src)) {
			c.literal(file, lit.line, lit.value)
		}
	}
	return nil
}

type scriptString struct {
	line  int
	value string
}

// scriptStrings lexes the string literals of a TypeScript or JavaScript
// source: quoted strings and template literals without interpolation.
// Comments are skipped. A quote that does not close on its line (an
// apostrophe in JSX text) yields nothing, so the lexer resynchronises at the
// next line.
func scriptStrings(src string) []scriptString {
	var lits []scriptString
	line := 1
	depth := 0
	var templates []int
	for i := 0; i < len(src); {
		switch c := src[i]; {
		case c == '\n':
			line++
			i++
		case strings.HasPrefix(src[i:], "//"):
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				return lits
			}
			i += end
		case strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return lits
			}
			line += strings.Count(src[i:i+2+end], "\n")
			i += end + 4
		case c == '\'' || c == '"':
			start := line
			end := quoteEnd(src, i+1, c, &line)
			if end < len(src) && src[end] == c {
				lits = append(lits, scriptString{start, src[i+1 : end]})
				end++
			}
			i = end
		case c == '`':
			start := line
			end, interpolated := templateEnd(src, i+1, &line)
			if !interpolated && end < len(src) {
				lits = append(lits, scriptString{start, src[i+1 : end]})
			}
			if interpolated {
				templates = append(templates, depth)
			}
			i = end + 1
		case c == '{':
			depth++
			i++
		case c == '}' && len(templates) > 0 && templates[len(templates)-1] == depth:
			templates = templates[:len(templates)-1]
			end, interpolated := templateEnd(src, i+1, &line)
			if interpolated {
				templates = append(templates, depth)
			}
			i = end + 1
		case c == '}':
			depth--
			i++
		default:
			i++
		}
	}
	return lits
}

// quoteEnd is the index of the quote closing a string begun before i, or of
// the newline or end of source where the string breaks off.
func quoteEnd(src string, i int, quote byte, line *int) int {
	for ; i < len(src); i++ {
		switch src[i] {
		case quote, '\n':
			return i
		case '\\':
			i = escaped(src, i, line)
		}
	}
	return i
}

// escaped steps over the character a backslash at i escapes, counting a
// line continuation.
func escaped(src string, i int, line *int) int {
	if i+1 < len(src) && src[i+1] == '\n' {
		*line++
	}
	return i + 1
}

// templateEnd scans a template literal's text from i to its closing backtick,
// or to the ${ that opens an interpolation, returning the index of that
// backtick or of the interpolation's brace.
func templateEnd(src string, i int, line *int) (end int, interpolated bool) {
	for i < len(src) {
		switch {
		case src[i] == '`':
			return i, false
		case strings.HasPrefix(src[i:], "${"):
			return i + 1, true
		case src[i] == '\\':
			i = escaped(src, i, line)
		case src[i] == '\n':
			*line++
		}
		i++
	}
	return len(src), false
}

// surface reports the import in internal/node/surface that brings
// internal/contenttype into its build on any release target.
func (c *contentTypeCheck) surface() error {
	for _, t := range releaseTargets {
		pkgs, err := c.repo.Deps(surfacePkg, t)
		if err != nil {
			return err
		}
		root := pkgs[len(pkgs)-1]
		contentType := path.Join(root.Module.Path, contentTypePkg)
		parents := importParents(pkgs, root.ImportPath)
		for _, p := range pkgs {
			if p.ImportPath != contentType && !strings.HasPrefix(p.ImportPath, contentType+"/") {
				continue
			}
			chain := importChain(parents, root.ImportPath, p.ImportPath)
			file, line, err := importSite(c.repo, root, chain[0])
			if err != nil {
				return err
			}
			c.reportLine(file, line, "surface depends on internal/contenttype; the API document must not vary with registered Content Types")
		}
	}
	return nil
}
