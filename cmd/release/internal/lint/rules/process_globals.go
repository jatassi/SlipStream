package rules

import (
	"fmt"
	"go/ast"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "process-globals",
		Doc:   "no subsystem reaches process-wide randomness, environment, logging or listeners past its Node (spec #127, Architecture lints)",
		Check: checkProcessGlobals,
	})
}

const runtimeTree = "internal/node/"

type processGlobal struct {
	pkg         string
	names       []string
	replacement string
}

const (
	useStream = "draw from the subsystem's seeded stream from its Node"
	useLogger = "log through the *slog.Logger the Node hands the subsystem"
	useConfig = "read the value from the Node's Config"
	useListen = "bind through the Node so readiness reports the address"
)

var processGlobals = []processGlobal{
	{pkg: "math/rand", replacement: useStream},
	{pkg: "math/rand/v2", replacement: useStream, names: []string{
		"ExpFloat64", "Float32", "Float64", "Int", "Int32", "Int32N", "Int64", "Int64N", "IntN", "N",
		"NormFloat64", "Perm", "Shuffle", "Uint", "Uint32", "Uint32N", "Uint64", "Uint64N", "UintN",
		"NewChaCha8", "NewPCG",
	}},
	{pkg: "os", replacement: useConfig, names: []string{"Getenv", "LookupEnv", "Environ"}},
	{pkg: "log/slog", replacement: useLogger, names: []string{
		"Debug", "DebugContext", "Info", "InfoContext", "Warn", "WarnContext", "Error", "ErrorContext",
		"Log", "LogAttrs", "With", "Default", "SetDefault", "SetLogLoggerLevel",
	}},
	{pkg: "log", replacement: useLogger},
	{pkg: "net", replacement: useListen, names: []string{
		"Listen", "ListenConfig", "ListenIP", "ListenMulticastUDP", "ListenPacket", "ListenTCP",
		"ListenUDP", "ListenUnix", "ListenUnixgram",
	}},
	{pkg: "net/http", replacement: "build an *http.Client from the Node's config", names: []string{
		"DefaultClient", "Get", "Head", "Post", "PostForm",
	}},
	{pkg: "net/http", replacement: "serve on a listener the Node binds", names: []string{
		"ListenAndServe", "ListenAndServeTLS",
	}},
	{pkg: "net/http", replacement: "register an operation with api.Register", names: []string{
		"Handle", "HandleFunc", "DefaultServeMux",
	}},
}

func checkProcessGlobals(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	report := func(f lint.GoFile, n ast.Node, what string, g processGlobal) {
		findings = append(findings, lint.Finding{
			Path:    f.Path,
			Line:    repo.Fset.Position(n.Pos()).Line,
			Message: fmt.Sprintf("%s is process-global; %s", what, g.replacement),
		})
	}
	for _, f := range files {
		if strings.HasPrefix(f.Path, runtimeTree) {
			continue
		}
		for _, imp := range f.Syntax.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if g, ok := bannedGlobal(p, ""); ok {
				report(f, imp, "package "+p, g)
			}
		}
		imports := importNames(f)
		ast.Inspect(f.Syntax, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Obj != nil || imports[pkg.Name] == "" {
				return true
			}
			if g, ok := bannedGlobal(imports[pkg.Name], sel.Sel.Name); ok {
				report(f, sel, imports[pkg.Name]+"."+sel.Sel.Name, g)
			}
			return true
		})
	}
	return findings, nil
}

func bannedGlobal(pkg, name string) (processGlobal, bool) {
	for _, g := range processGlobals {
		wholePackage := g.names == nil
		if g.pkg == pkg && wholePackage == (name == "") && (wholePackage || slices.Contains(g.names, name)) {
			return g, true
		}
	}
	return processGlobal{}, false
}
