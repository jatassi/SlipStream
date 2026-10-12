package rules

import (
	"fmt"
	"go/ast"
	"go/token"
	"net/netip"
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "endpoint-literals",
		Doc:   "external hosts appear only in a package's DefaultEndpoints (spec #127, Architecture lints)",
		Check: checkEndpointLiterals,
	})
}

var (
	stunURI  = regexp.MustCompile(`(?i)\b(?:stuns?|turns?):([^\s/?#"'<>]+)`)
	urlRef   = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://([^\s/?#"'<>]*)`)
	hostPort = regexp.MustCompile(`(\[[0-9A-Fa-f:.]+\]|[A-Za-z0-9.-]+):\d{1,5}\b`)
	dnsName  = regexp.MustCompile(`^(?i:[a-z0-9]([a-z0-9-]*[a-z0-9])?)(\.(?i:[a-z0-9]([a-z0-9-]*[a-z0-9])?))*\.?$`)
)

var reservedSuffixes = []string{"localhost", "test", "example", "invalid", "example.com", "example.net", "example.org"}

type hostKind int

const (
	anyHostname hostKind = iota
	dottedHostname
)

func checkEndpointLiterals(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		for _, lit := range hostLiterals(f.Syntax) {
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return nil, fmt.Errorf("%s: unquote %s: %w", f.Path, lit.Value, err)
			}
			if host, ok := externalHost(s); ok {
				findings = append(findings, lint.Finding{
					Path:    f.Path,
					Line:    repo.Fset.Position(lit.Pos()).Line,
					Message: fmt.Sprintf("host %s outside DefaultEndpoints; declare it in the package's Endpoints section", host),
				})
			}
		}
	}
	return findings, nil
}

func hostLiterals(file *ast.File) []*ast.BasicLit {
	var lits []*ast.BasicLit
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			return !isDefaultEndpoints(n)
		case *ast.BasicLit:
			if n.Kind == token.STRING {
				lits = append(lits, n)
			}
		}
		return true
	})
	return lits
}

func isDefaultEndpoints(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || fn.Name.Name != "DefaultEndpoints" || fn.Type.Results == nil {
		return false
	}
	results := fn.Type.Results.List
	if len(results) != 1 || len(results[0].Names) > 1 {
		return false
	}
	ident, ok := results[0].Type.(*ast.Ident)
	return ok && ident.Name == "Endpoints"
}

func externalHost(s string) (string, bool) {
	for _, m := range stunURI.FindAllStringSubmatchIndex(s, -1) {
		if host := splitHost(s[m[2]:m[3]]); hostIsExternal(host, anyHostname) {
			return host, true
		}
	}
	s = stunURI.ReplaceAllString(s, " ")
	for _, m := range urlRef.FindAllStringSubmatchIndex(s, -1) {
		if host := splitHost(urlHost(s[m[2]:m[3]])); hostIsExternal(host, anyHostname) {
			return host, true
		}
	}
	s = urlRef.ReplaceAllString(s, " ")
	for _, m := range hostPort.FindAllStringSubmatchIndex(s, -1) {
		if host := strings.Trim(s[m[2]:m[3]], "[]"); hostIsExternal(host, dottedHostname) {
			return host, true
		}
	}
	return "", false
}

func urlHost(authority string) string {
	if i := strings.LastIndex(authority, "@"); i >= 0 {
		return authority[i+1:]
	}
	return authority
}

func splitHost(hostport string) string {
	if strings.HasPrefix(hostport, "[") {
		if end := strings.Index(hostport, "]"); end > 0 {
			return hostport[1:end]
		}
		return ""
	}
	host, _, _ := strings.Cut(hostport, ":")
	return host
}

func hostIsExternal(host string, kind hostKind) bool {
	if addr, err := netip.ParseAddr(host); err == nil {
		addr = addr.Unmap()
		return !addr.IsLoopback() && !addr.IsUnspecified()
	}
	if !dnsName.MatchString(host) {
		return false
	}
	name := strings.ToLower(strings.TrimSuffix(host, "."))
	if kind == dottedHostname && !isDottedName(name) {
		return false
	}
	for _, reserved := range reservedSuffixes {
		if name == reserved || strings.HasSuffix(name, "."+reserved) {
			return false
		}
	}
	return true
}

func isDottedName(name string) bool {
	labels := strings.Split(name, ".")
	return len(labels) > 1 && strings.ContainsFunc(labels[len(labels)-1], func(r rune) bool { return r < '0' || r > '9' })
}
