package rules

import (
	"go/ast"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "role-defs-wired",
		Doc:   "internal/node serves and holds every package-level role.Def and role.WideDef, and no two Defs share a Name (spec #132, Lints and CI checks)",
		Check: checkRoleDefsWired,
	})
}

const (
	roleWiring = "internal/node"
	roleTree   = "internal/role"
)

// defWiring is, per role Def type and method, the internal/node function
// that must call it, "" for any. Every Node holds a Node-wide channel, so
// its Hold goes among the Roles both halves hold, and only the Controller
// serves it.
var defWiring = map[string]map[string]string{
	"Def":     {"Serve": "", "Hold": ""},
	"WideDef": {"Serve": "startController", "Hold": "heldRoles"},
}

var wiringVerbs = map[string]string{"Serve": "served", "Hold": "held"}

type roleDef struct {
	key, label string
	// typ is the Def's type, a key of defWiring.
	typ string
	// constant is the role constant the Def names, "" for any other
	// expression.
	constant string
	at       lint.Finding
}

type roleDefsIndex struct {
	repo      *lint.Repo
	rolePkg   string
	constants map[string]string
	defs      []roleDef
	// wired is, per method, each Def's key and the internal/node functions
	// that call that method on it, "" for a call outside any.
	wired map[string]map[string][]string
}

func checkRoleDefsWired(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	idx := roleDefsIndex{
		repo:      repo,
		rolePkg:   path.Join(module, roleTree),
		constants: map[string]string{},
		wired:     map[string]map[string][]string{"Serve": {}, "Hold": {}},
	}
	for _, f := range files {
		idx.add(f, path.Join(module, path.Dir(f.Path)))
	}
	var findings []lint.Finding
	declared := map[string]bool{}
	for _, d := range idx.defs {
		for _, method := range []string{"Serve", "Hold"} {
			in, where := defWiring[d.typ][method], roleWiring
			if in != "" {
				where += "'s " + in
			}
			if !slices.ContainsFunc(idx.wired[method][d.key], func(fn string) bool { return in == "" || fn == in }) {
				findings = append(findings, d.finding(d.label+" is not "+wiringVerbs[method]+" in "+where))
			}
		}
		name, ok := idx.constants[d.constant]
		switch {
		case !ok:
			findings = append(findings, d.finding(d.label+" names its Role without a role constant"))
		case declared[name]:
			findings = append(findings, d.finding("role "+name+" is declared twice"))
		default:
			declared[name] = true
		}
	}
	return findings, nil
}

func (d roleDef) finding(message string) lint.Finding {
	f := d.at
	f.Message = message
	return f
}

func (idx *roleDefsIndex) add(f lint.GoFile, pkg string) {
	dir := path.Dir(f.Path)
	imports := importNames(f)
	if dir == roleWiring {
		idx.addWiring(imports, f.Syntax)
	}
	for _, decl := range f.Syntax.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			vs, _ := spec.(*ast.ValueSpec)
			switch {
			case gen.Tok == token.CONST && dir == roleTree:
				idx.addConstants(vs)
			case gen.Tok == token.VAR && !strings.HasSuffix(f.Syntax.Name.Name, "test"):
				idx.addDefs(f, pkg, dir, imports, vs)
			}
		}
	}
}

func (idx *roleDefsIndex) addWiring(imports map[string]string, file *ast.File) {
	for _, decl := range file.Decls {
		fn := ""
		if fd, ok := decl.(*ast.FuncDecl); ok {
			fn = fd.Name.Name
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || idx.wired[sel.Sel.Name] == nil {
				return true
			}
			if pkg, name, ok := qualifiedName(imports, sel.X); ok {
				key := pkg + "." + name
				idx.wired[sel.Sel.Name][key] = append(idx.wired[sel.Sel.Name][key], fn)
			}
			return true
		})
	}
}

func (idx *roleDefsIndex) addConstants(vs *ast.ValueSpec) {
	for i, name := range vs.Names[:min(len(vs.Names), len(vs.Values))] {
		if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			idx.constants[name.Name], _ = strconv.Unquote(lit.Value)
		}
	}
}

func (idx *roleDefsIndex) addDefs(f lint.GoFile, pkg, dir string, imports map[string]string, vs *ast.ValueSpec) {
	for i, name := range vs.Names[:min(len(vs.Names), len(vs.Values))] {
		lit, ok := vs.Values[i].(*ast.CompositeLit)
		if !ok {
			continue
		}
		typ, ok := idx.defType(imports, lit.Type)
		if !ok {
			continue
		}
		var constant string
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok || !isIdent(kv.Key, "Name") {
				continue
			}
			if pkg, name, ok := qualifiedName(imports, kv.Value); ok && pkg == idx.rolePkg {
				constant = name
			}
		}
		idx.defs = append(idx.defs, roleDef{
			key:      pkg + "." + name.Name,
			label:    dir + "." + name.Name,
			typ:      typ,
			constant: constant,
			at:       lint.Finding{Path: f.Path, Line: idx.repo.Fset.Position(name.Pos()).Line},
		})
	}
}

// defType is the role Def type typ instantiates, false for any other type.
func (idx *roleDefsIndex) defType(imports map[string]string, typ ast.Expr) (string, bool) {
	switch t := typ.(type) {
	case *ast.IndexExpr:
		typ = t.X
	case *ast.IndexListExpr:
		typ = t.X
	}
	pkg, name, ok := qualifiedName(imports, typ)
	return name, ok && pkg == idx.rolePkg && defWiring[name] != nil
}

func isIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}
