// Package main implements a scaffolding tool for new SlipStream modules.
//
// Usage:
//
//	go run ./scripts/new-module <module_id>
//
// Example:
//
//	go run ./scripts/new-module music
package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

//go:embed templates/*
var templateFS embed.FS

type templateData struct {
	ID          string // e.g. "music"
	PluralID    string // e.g. "musics" (simple plural)
	PackageName string // e.g. "music"
	Title       string // e.g. "Music"
	PluralTitle string // e.g. "Musics"
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run ./scripts/new-module <module_id>")
		fmt.Fprintln(os.Stderr, "Example: go run ./scripts/new-module music")
		os.Exit(1)
	}

	moduleID := strings.ToLower(strings.TrimSpace(os.Args[1]))
	if moduleID == "" {
		fmt.Fprintln(os.Stderr, "Error: module ID cannot be empty")
		os.Exit(1)
	}

	if moduleID == "movie" || moduleID == "tv" {
		fmt.Fprintf(os.Stderr, "Error: module %q already exists\n", moduleID)
		os.Exit(1)
	}

	data := templateData{
		ID:          moduleID,
		PluralID:    pluralize(moduleID),
		PackageName: moduleID,
		Title:       titleCase(moduleID),
		PluralTitle: titleCase(pluralize(moduleID)),
	}

	backendDir := filepath.Join("internal", "modules", moduleID)
	frontendDir := filepath.Join("web", "src", "modules", moduleID)
	queriesDir := filepath.Join("internal", "database", "queries")
	moduleMigrationsDir := filepath.Join(backendDir, "migrations")

	type fileSpec struct {
		templateName string
		outputPath   string
	}

	files := []fileSpec{
		// Backend Go files
		{"templates/module.go.tmpl", filepath.Join(backendDir, "module.go")},
		{"templates/migrate.go.tmpl", filepath.Join(backendDir, "migrate.go")},
		{"templates/metadata.go.tmpl", filepath.Join(backendDir, "metadata.go")},
		{"templates/search.go.tmpl", filepath.Join(backendDir, "search.go")},
		{"templates/file_parser.go.tmpl", filepath.Join(backendDir, "fileparser.go")},
		{"templates/import_handler.go.tmpl", filepath.Join(backendDir, "importhandler.go")},
		{"templates/monitoring.go.tmpl", filepath.Join(backendDir, "monitoring.go")},
		{"templates/notifications.go.tmpl", filepath.Join(backendDir, "notifications.go")},
		{"templates/calendar.go.tmpl", filepath.Join(backendDir, "calendar.go")},
		{"templates/wanted.go.tmpl", filepath.Join(backendDir, "wanted.go")},
		{"templates/quality.go.tmpl", filepath.Join(backendDir, "quality.go")},
		{"templates/path_naming.go.tmpl", filepath.Join(backendDir, "path_naming.go")},
		{"templates/release_dates.go.tmpl", filepath.Join(backendDir, "release_dates.go")},
		{"templates/mock.go.tmpl", filepath.Join(backendDir, "mock_factory.go")},
		{"templates/module_test.go.tmpl", filepath.Join(backendDir, "module_test.go")},

		// Module-scoped migration (initial schema)
		{"templates/initial_migration.sql.tmpl", filepath.Join(moduleMigrationsDir, "00001_initial.sql")},

		// SQL files (shared query layer)
		{"templates/queries.sql.tmpl", filepath.Join(queriesDir, data.PluralID+".sql")},

		// Frontend TypeScript files
		{"templates/config.ts.tmpl", filepath.Join(frontendDir, "index.ts")},
	}

	fmt.Printf("Scaffolding new module: %s\n\n", moduleID)

	for _, f := range files {
		if err := generateFile(f.templateName, f.outputPath, &data); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating %s: %v\n", f.outputPath, err)
			os.Exit(1)
		}
		fmt.Printf("  created %s\n", f.outputPath)
	}

	printNextSteps(&data)
}

func generateFile(templateName, outputPath string, data *templateData) error {
	content, err := templateFS.ReadFile(templateName)
	if err != nil {
		return fmt.Errorf("read template %s: %w", templateName, err)
	}

	funcMap := template.FuncMap{
		"lb": func() string { return "{" },
		"rb": func() string { return "}" },
	}

	tmpl, err := template.New(filepath.Base(templateName)).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse template %s: %w", templateName, err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("file already exists: %s", outputPath)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func pluralize(s string) string {
	if s == "" {
		return s
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "sh") || strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		r := []rune(s)
		prev := r[len(r)-2]
		if !isVowel(prev) {
			return string(r[:len(r)-1]) + "ies"
		}
	}
	return s + "s"
}

func isVowel(r rune) bool {
	switch unicode.ToLower(r) {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

func printNextSteps(data *templateData) {
	fmt.Printf(`
Scaffolding complete!

Next steps:

  1. Register the module type and entity type constants in:
       internal/module/types.go
     Add:
       Type%s Type = "%s"
       Entity%s EntityType = "%s"

  2. Module-scoped migrations are in:
       internal/modules/%s/migrations/00001_initial.sql
     These run automatically via the per-module migration track
     (goose table: goose_db_version_%s). Add future migrations
     as 00002_*.sql, 00003_*.sql, etc.

  3. Update the SQL queries file for sqlc and run:
       go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate

  4. Wire the module in internal/api/wire.go:
       %sModule = %s.NewModule(db, logger)
     And register with the module registry.

  5. Register the frontend module in web/src/modules/setup.ts:
       import { %sModuleConfig } from './%s'
       registerModule(%sModuleConfig)

  6. Implement the TODO stubs in each generated file.

  7. Run linters to verify:
       make lint
       cd web && bun run lint

`, data.Title, data.ID,
		data.Title, data.ID,
		data.ID, data.ID,
		data.ID, data.PackageName,
		data.ID, data.ID, data.ID,
	)
}
