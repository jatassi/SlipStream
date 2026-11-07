// Command prowlarr-sync keeps SlipStream's ports in step with Prowlarr.
// It is a maintainer tool and never ships in the release binary. Each
// command reads a local clone of Prowlarr; a blobless one is enough
// (git clone --filter=blob:none --no-checkout https://github.com/Prowlarr/Prowlarr).
//
//	prowlarr-sync report --prowlarr DIR [--ref develop] [--json]
//	prowlarr-sync check  [--prowlarr DIR]
//	prowlarr-sync bump   --prowlarr DIR --unit port:hdbits --to SHA
//	prowlarr-sync new    --prowlarr DIR [--ref develop] --site HDBits [--id hdbits] [--family gazelle]
//
// report writes what changed upstream since each unit's recorded commit
// as Markdown, or as JSON for the weekly routine. check is the manifest
// check, against Prowlarr's history when --prowlarr names a clone. bump
// moves one unit, or the hidden list (--unit hidden-list), to an
// upstream commit. new writes a port's boilerplate. Every command runs
// from the repository root, or from the one --repo names.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jatassi/slipstream/internal/ports/provenance"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "prowlarr-sync:", err)
		os.Exit(1)
	}
}

const usage = "usage: prowlarr-sync report|check|bump|new [flags]; run a command with -h for its flags"

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	repo := fs.String("repo", ".", "the SlipStream repository")
	prowlarr := fs.String("prowlarr", "", "a clone of Prowlarr's repository")
	ref := fs.String("ref", "develop", "the Prowlarr ref to read")
	asJSON := fs.Bool("json", false, "report: write JSON instead of Markdown")
	unit := fs.String("unit", "", "bump: the unit, such as port:hdbits, or hidden-list")
	to := fs.String("to", "", "bump: the Prowlarr commit to move the unit to")
	site := fs.String("site", "", "new: the site's C# class, such as HDBits")
	id := fs.String("id", "", "new: the port id; the class lowercased with spaces, dots and hyphens dropped by default")
	family := fs.String("family", "", "new: the family the port builds on, such as gazelle")
	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("%s: %w", args[0], err)
	}
	var tree *provenance.Tree
	if *prowlarr != "" {
		tree = &provenance.Tree{Dir: *prowlarr}
	}
	if tree == nil && args[0] != "check" {
		return fmt.Errorf("%s needs --prowlarr, a clone of Prowlarr's repository", args[0])
	}
	switch args[0] {
	case "report":
		return report(ctx, out, *repo, tree, *ref, *asJSON)
	case "check":
		return check(out, *repo, tree)
	case "bump":
		return bump(ctx, out, *repo, tree, *unit, *to)
	case "new":
		return newPort(ctx, out, *repo, tree, *ref, *site, *id, *family)
	}
	return errors.New(usage)
}

func report(ctx context.Context, out io.Writer, repo string, tree *provenance.Tree, ref string, asJSON bool) error {
	rep, err := buildReport(ctx, repo, tree, ref)
	if err != nil {
		return err
	}
	if !asJSON {
		_, err = io.WriteString(out, rep.markdown())
		return err //nolint:wrapcheck // the one write of the command's output
	}
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rep); err != nil {
		return fmt.Errorf("write the report as JSON: %w", err)
	}
	return nil
}

func check(out io.Writer, repo string, tree *provenance.Tree) error {
	problems, err := provenance.Check(repo, tree)
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return problemsError(problems)
	}
	_, err = fmt.Fprintln(out, "Every port's provenance agrees with its code.")
	return err //nolint:wrapcheck // the one write of the command's output
}

func bump(ctx context.Context, out io.Writer, repo string, tree *provenance.Tree, unit, to string) error {
	if unit == "" || to == "" {
		return errors.New("bump needs --unit and --to")
	}
	c, err := resolve(ctx, tree, to)
	if err != nil {
		return err
	}
	if unit == "hidden-list" {
		s, err := provenance.ReadSkipped(repo)
		if err != nil {
			return err
		}
		s.Commit = c.SHA
		if err := provenance.WriteSkipped(repo, s); err != nil {
			return err
		}
	} else {
		dir, err := unitDir(repo, unit)
		if err != nil {
			return err
		}
		if err := provenance.Bump(repo, dir, c.SHA, c.Date); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(out, "Moved %s to %s (%s).\n", unit, c.SHA, c.Date)
	return err //nolint:wrapcheck // the one write of the command's output
}

func unitDir(repo, unit string) (string, error) {
	units, _, err := provenance.Units(repo)
	if err != nil {
		return "", err
	}
	for _, u := range units {
		if u.ID == unit {
			return u.Dir, nil
		}
	}
	ids := make([]string, len(units))
	for i, u := range units {
		ids[i] = u.ID
	}
	return "", fmt.Errorf("no unit is %q; the units are %s", unit, strings.Join(ids, ", "))
}
