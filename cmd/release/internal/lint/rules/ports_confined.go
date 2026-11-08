package rules

import (
	"go/ast"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "ports-confined",
		Doc:   "a ported site reaches the network and the Node only through internal/ports/port, and only the registry wiring names a port (S27 (Ported site Indexers), Architecture)",
		Check: checkPortsConfined,
	})
}

const (
	portsTree = "internal/ports"
	portKit   = "internal/ports/port"
)

var (
	// portsPlumbing are the ports packages that aren't a site or a family.
	portsPlumbing = []string{portKit + "/...", "internal/ports/all/...", "internal/ports/provenance/..."}
	// siteImports are the ports packages a site or a family may import.
	siteImports = []string{portKit + "/...", "internal/ports/internal/..."}
	siteForbids = []string{"internal/searching/...", "internal/cardigann/..."}
	// sendsOnItsOwn are the net/http and net names that would send a
	// request around the Node's Site.
	sendsOnItsOwn = map[string][]string{
		"net/http": {"Client", "Transport", "NewRequest", "NewRequestWithContext", "DefaultClient", "DefaultTransport", "Get", "Post", "Head", "PostForm"},
		"net":      {"Dial", "DialTimeout", "DialTCP", "DialUDP", "DialIP", "DialUnix", "Dialer"},
	}
)

func checkPortsConfined(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		dir := path.Dir(f.Path)
		site := inTree(dir, portsTree) && dir != portsTree && !inDirs(dir, portsPlumbing)
		for _, imp := range f.Syntax.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			pkg, ok := strings.CutPrefix(p, module+"/")
			if !ok {
				continue
			}
			if message := portImport(dir, pkg, site); message != "" {
				findings = append(findings, lint.Finding{Path: f.Path, Line: repo.Fset.Position(imp.Pos()).Line, Message: message})
			}
		}
		if site {
			findings = append(findings, ownSends(repo, f)...)
		}
	}
	return findings, nil
}

// portImport is why dir may not import pkg, "" when it may.
func portImport(dir, pkg string, site bool) string {
	switch {
	case site && inDirs(pkg, siteForbids):
		return pkg + " imported by a port; a port reaches the Node through internal/ports/port"
	case site && inTree(pkg, portsTree) && !inDirs(pkg, siteImports) && !inTree(pkg, dir):
		return pkg + " imported by a port, which imports only internal/ports/port and the families under internal/ports/internal"
	case inTree(dir, portKit) && inTree(pkg, portsTree) && !inTree(pkg, portKit):
		return pkg + " imported by internal/ports/port, which no port may depend on"
	case inTree(dir, searchingSide) && inTree(pkg, portsTree):
		return pkg + " imported by the Node's searching; it runs ports through Config.Native"
	case inTree(dir, "internal/indexers") && inTree(pkg, portsTree) && !inTree(pkg, portKit):
		return pkg + " imported by internal/indexers, which reads ports only through port.Catalog"
	}
	return ""
}

func ownSends(repo *lint.Repo, f lint.GoFile) []lint.Finding {
	imports := importNames(f)
	var findings []lint.Finding
	ast.Inspect(f.Syntax, func(n ast.Node) bool {
		sel, isSel := n.(*ast.SelectorExpr)
		if !isSel {
			return true
		}
		pkg, name, ok := qualifiedName(imports, sel)
		if ok && slices.Contains(sendsOnItsOwn[pkg], name) {
			findings = append(findings, lint.Finding{
				Path:    f.Path,
				Line:    repo.Fset.Position(n.Pos()).Line,
				Message: pkg + "." + name + " used by a port; send through the port.Indexer the framework passes",
			})
		}
		return true
	})
	return findings
}
