package rules

import (
	"fmt"
	"go/parser"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

const releaseMain = "./cmd/slipstream"

var releaseTargets = []lint.Target{
	{GOOS: "linux", GOARCH: "amd64"},
	{GOOS: "linux", GOARCH: "arm64"},
	{GOOS: "windows", GOARCH: "amd64"},
	{GOOS: "darwin", GOARCH: "arm64"},
	{GOOS: "darwin", GOARCH: "amd64"},
}

var forbiddenTrees = []string{"fakes", "cmd/release"}

func init() {
	lint.Register(lint.Rule{
		Name:  "release-imports",
		Doc:   "cmd/slipstream depends on nothing under fakes/ or cmd/release for any target (ADR 0014)",
		Check: checkReleaseImports,
	})
}

type forbiddenImport struct {
	file    string
	line    int
	targets []string
}

func checkReleaseImports(repo *lint.Repo) ([]lint.Finding, error) {
	found := map[string]*forbiddenImport{}
	for _, t := range releaseTargets {
		pkgs, err := repo.Deps(releaseMain, t)
		if err != nil {
			return nil, err
		}
		root := pkgs[len(pkgs)-1]
		module := root.Module.Path
		parents := importParents(pkgs, root.ImportPath)
		for _, p := range pkgs {
			if !forbidden(module, p.ImportPath) {
				continue
			}
			chain := []string{p.ImportPath}
			for parents[chain[0]] != root.ImportPath {
				chain = slices.Insert(chain, 0, parents[chain[0]])
			}
			file, line, err := importSite(repo, root, chain[0])
			if err != nil {
				return nil, err
			}
			for i := range chain {
				chain[i] = strings.TrimPrefix(chain[i], module+"/")
			}
			key := strings.Join(chain, " -> ")
			if found[key] == nil {
				found[key] = &forbiddenImport{file: file, line: line}
			}
			found[key].targets = append(found[key].targets, t.String())
		}
	}
	findings := make([]lint.Finding, 0, len(found))
	for chain, f := range found {
		findings = append(findings, lint.Finding{
			Path:    f.file,
			Line:    f.line,
			Message: fmt.Sprintf("cmd/slipstream imports %s on %s", chain, strings.Join(f.targets, ", ")),
		})
	}
	return findings, nil
}

func forbidden(module, importPath string) bool {
	for _, tree := range forbiddenTrees {
		prefix := module + "/" + tree
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}

func importParents(pkgs []lint.Package, root string) map[string]string {
	byPath := make(map[string]lint.Package, len(pkgs))
	for _, p := range pkgs {
		byPath[p.ImportPath] = p
	}
	parents := map[string]string{root: ""}
	queue := []string{root}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, imp := range byPath[cur].Imports {
			if _, seen := parents[imp]; !seen {
				parents[imp] = cur
				queue = append(queue, imp)
			}
		}
	}
	return parents
}

func importSite(repo *lint.Repo, root lint.Package, importPath string) (string, int, error) {
	names := slices.Concat(root.GoFiles, root.CgoFiles)
	slices.Sort(names)
	for _, name := range names {
		file := filepath.Join(root.Dir, name)
		syntax, err := parser.ParseFile(repo.Fset, file, nil, parser.ImportsOnly)
		if err != nil {
			return "", 0, err
		}
		for _, imp := range syntax.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == importPath {
				rel, err := filepath.Rel(repo.Root, file)
				return filepath.ToSlash(rel), repo.Fset.Position(imp.Pos()).Line, err
			}
		}
	}
	return "", 0, fmt.Errorf("no file in %s imports %s", root.ImportPath, importPath)
}
