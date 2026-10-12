package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	register("migrations-frozen", "fail when a change edits, deletes or renames a migration that already merged", runMigrationsFrozen)
}

func runMigrationsFrozen(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("migrations-frozen", flag.ContinueOnError)
	base := fs.String("base", "", "revision the change merges into, such as origin/main, or HEAD^1 on main")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if *base == "" {
		return errors.New("-base is required")
	}
	return migrationsFrozen(ctx, os.Stdout, ".", *base)
}

// Goose records only a version, so an edited migration never reruns on a
// file that already holds it.
func migrationsFrozen(ctx context.Context, w io.Writer, dir, base string) error {
	changes, err := gitChanges(ctx, dir, base, "HEAD")
	if err != nil {
		return err
	}
	var findings []lint.Finding
	for _, c := range changes {
		if !lint.IsOwnerSQL(c.Old.Path, "schema") {
			continue
		}
		change := "edited"
		switch {
		case c.New.Path == "":
			change = "deleted"
		case c.New.Path != c.Old.Path:
			change = "renamed"
		}
		findings = append(findings, lint.Finding{
			Path:    c.Old.Path,
			Line:    1,
			Message: fmt.Sprintf("%s after it merged into %s; add a new migration instead", change, base),
		})
	}
	return lint.Run(w, lint.Open(ctx, dir), []lint.Rule{{
		Name:  "migrations-frozen",
		Check: func(*lint.Repo) ([]lint.Finding, error) { return findings, nil },
	}})
}
