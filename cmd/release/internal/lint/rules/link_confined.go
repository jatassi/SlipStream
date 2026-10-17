package rules

import (
	"go/ast"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "link-confined",
		Doc:   "connect, yamux, http2, internal/link and generated service constructors stay inside the Node link (spec #132, Lints and CI checks)",
		Check: checkLinkConfined,
	})
}

const linkAdvice = "; call the service through its rpc descriptor"

// linkMechanism is the Node link itself: the transport, call and sync
// dispatch, and the generated code.
var linkMechanism = []string{"internal/link/...", "internal/rpc", "internal/role/...", "internal/pb/..."}

// linkOwners may hold a *link.Link: the Node runtime and the packages it
// hands the link to.
var linkOwners = []string{"internal/link/...", "internal/node", "internal/rpc", "internal/role", "internal/nodes", "internal/pairing", "internal/transfers", "internal/serving", "fakes/..."}

var transportModules = []string{"connectrpc.com/connect", "golang.org/x/net/http2"}

var generatedConstructor = regexp.MustCompile(`^(New\w+ServiceClient|Register\w+ServiceHandler)$`)

type linkScan struct {
	repo            *lint.Repo
	linkPkg, pbTree string
	findings        []lint.Finding
}

func checkLinkConfined(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	scan := linkScan{repo: repo, linkPkg: path.Join(module, "internal/link"), pbTree: path.Join(module, "internal/pb")}
	for _, f := range files {
		dir := path.Dir(f.Path)
		test := strings.HasSuffix(f.Path, "_test.go")
		mechanism := inDirs(dir, linkMechanism)
		scan.imports(f, mechanism, test || inDirs(dir, linkOwners))
		if !mechanism && !test {
			scan.constructors(f)
		}
	}
	return scan.findings, nil
}

func (s *linkScan) add(f lint.GoFile, at ast.Node, message string) {
	s.findings = append(s.findings, lint.Finding{Path: f.Path, Line: s.repo.Fset.Position(at.Pos()).Line, Message: message})
}

func (s *linkScan) imports(f lint.GoFile, mechanism, owner bool) {
	for _, imp := range f.Syntax.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		if label := transportModule(p); label != "" && !mechanism {
			s.add(f, imp, label+" imported outside the Node link"+linkAdvice)
		}
		if inTree(p, s.linkPkg) && !owner {
			s.add(f, imp, "internal/link imported outside the Node runtime"+linkAdvice)
		}
	}
}

func (s *linkScan) constructors(f lint.GoFile) {
	imports := importNames(f)
	ast.Inspect(f.Syntax, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if pkg, name, ok := qualifiedName(imports, call.Fun); ok && inTree(pkg, s.pbTree) && generatedConstructor.MatchString(name) {
			s.add(f, call, path.Base(pkg)+"."+name+" called outside the Node link"+linkAdvice)
		}
		return true
	})
}

func transportModule(p string) string {
	for _, m := range transportModules {
		if inTree(p, m) {
			return m
		}
	}
	if strings.Contains(p, "yamux") {
		return p
	}
	return ""
}

func inTree(p, tree string) bool { return p == tree || strings.HasPrefix(p, tree+"/") }

// inDirs reports whether dir is one of dirs, where "x/..." also matches
// every directory below x.
func inDirs(dir string, dirs []string) bool {
	return slices.ContainsFunc(dirs, func(d string) bool {
		if tree, ok := strings.CutSuffix(d, "/..."); ok {
			return inTree(dir, tree)
		}
		return dir == d
	})
}
